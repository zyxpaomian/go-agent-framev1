package server

import (
	"bytes"
	"encoding/binary"

	"github.com/golang/protobuf/proto"

	"fmt"
	"msg"
	"net"
	"sync"
	"time"
	"util/config"
	ce "util/error"
	log "util/serverlog"
)

type TcpClient struct {
	ClientID              string
	Conn                  net.Conn
	Valid                 bool
	SendLock              *sync.Mutex
	ReadBuf               []byte
	ReadMsgPayloadLth     uint32
	LastHeartbeatSyncTime string
}

func NewClient(conn net.Conn, clientid string) *TcpClient {
	return &TcpClient{
		ClientID:              clientid,
		Conn:                  conn,
		ReadBuf:               []byte{},
		ReadMsgPayloadLth:     0,
		Valid:                 true,
		LastHeartbeatSyncTime: "1970-01-01 00:00:00",
		SendLock:              &sync.Mutex{},
	}
}

func (c *TcpClient) SetLastHeartBeatSyncTime(t string) {
	log.Debugf("%s 的心跳时间更新为%s", c.ClientID, t)
	c.LastHeartbeatSyncTime = t
}

func (c *TcpClient) JudgeValid() bool {
	if c.Valid == false {
		return false
	}
	if c.LastHeartbeatSyncTime == "1970-01-01 00:00:00" {
		return true
	}
	now := time.Now()
	h, _ := time.ParseDuration(fmt.Sprintf("-%ds", config.GlobalConf.GetInt("tcp", "heartbeattimeout")))
	nextTimeout := now.Add(h)
	// nowStr := now.UnixNano()
	nextTimeoutStr := nextTimeout.UnixNano()
	clientLastHeartbeatSynctime, _ := time.Parse("2006-01-02 15:04:05", c.LastHeartbeatSyncTime)
	clientLastHeartbeatSynctimestr := clientLastHeartbeatSynctime.UnixNano()
	if nextTimeoutStr > clientLastHeartbeatSynctimestr {
		log.Errorf("上次心跳包同步时间为 %s ,超时时间为 %s 秒, 超时时间阀值为 %s", c.LastHeartbeatSyncTime, config.GlobalConf.GetInt("tcp", "heartbeattimeout"), clientLastHeartbeatSynctime)
		return false
	}
	return true
}

func (c *TcpClient) SetValid(valid bool) {
	c.Valid = valid
}

func (c *TcpClient) GetMsg() (*msg.Msg, error) {
	/*
		4 字节报文长度
		4 字节的报文类型
		报文(暂定protobuf)
	*/
	for {
		if c.JudgeValid() == false {
			log.Errorf("%s 客户端无效, 退出读取循环", c.ClientID)
			return nil, ce.New(fmt.Sprintf("%s 无效", c.ClientID))
		}

		c.Conn.SetReadDeadline(time.Now().Add(time.Duration(config.GlobalConf.GetInt("tcp", "readtimeout")) * time.Second))
		// 当报文长度信息还没有获取到的时候
		if len(c.ReadBuf) < 4 {
			log.Debugf("当前客户端的readbuf长度小于前4字节的报文长度,准备从 %s 读取报文", c.ClientID)
			readBuf := make([]byte, 256)
			n, err := c.Conn.Read(readBuf)
			log.Debugf("从客户端 %s 读取到报文", c.ClientID)
			if err != nil {
				log.Errorf("从客户端 %s 读取报文数据失败,报错信息为: %s", c.ClientID, err.Error())
				return nil, err
			}
			c.Conn.SetReadDeadline(time.Time{})
			log.Debugf("此次从客户端 %s 读取到了 %d 长度的报文数据, 报文内容为 %v", c.ClientID, n, readBuf[:n])
			// 将本次从客户端读取到的数据加载到客户端的readbuf中
			c.ReadBuf = append(c.ReadBuf, readBuf[:n]...)
			// 加载后判断当前客户端readbuf的长度是否满足报文长度的需求，如不满足，等待下次循环
			if len(c.ReadBuf) >= 4 {
				c.ReadMsgPayloadLth = binary.BigEndian.Uint32(c.ReadBuf[0:4])
				log.Debugf("获取到用户的payload长度，长度为 %d", c.ReadMsgPayloadLth)
			}
			// 数据包长度已获取，拿到了一个完整数据包的总长度，但包没传完，重新初始化一个缓冲池继续读报文
		} else if c.ReadMsgPayloadLth+8 > uint32(len(c.ReadBuf)) {
			readBuf := make([]byte, 256)
			n, err := c.Conn.Read(readBuf)
			if err != nil {
				log.Errorf("从客户端 %s 读取报文数据失败,报错信息为: %s", c.ClientID, err.Error())
				return nil, err
			}
			c.Conn.SetReadDeadline(time.Time{})
			log.Debugf("此次从客户端 %s 读取到了 %d 长度的报文数据, 报文内容为 %v", c.ClientID, n, readBuf[:n])
			// 将本次从客户端读取到的数据加载到客户端的readbuf中
			c.ReadBuf = append(c.ReadBuf, readBuf[:n]...)
			// 报文读完了，但客户端的readbuf里可能还有下一个报文的一部分信息,这里处理下粘包以及解析报文数据
		} else {
			c.Conn.SetReadDeadline(time.Time{})
			log.Debugf("本次报文都已读取完成, 客户端: %s", c.ClientID)
			msgLength := 8 + c.ReadMsgPayloadLth
			msgTypeBytes := c.ReadBuf[4:8]
			log.Debugf("本地获取报文总长度 %d , 消息类型为 %d , 消息长度为 %d , 客户端 %s", msgLength, binary.BigEndian.Uint32(msgTypeBytes), c.ReadMsgPayloadLth, c.ClientID)

			//消息是空数据的情况，消息实体为空
			if c.ReadMsgPayloadLth == 0 {
				// 如果客户端readbuf的数据长度和需要读取的数据长度一致，则相当于是个空包，即4+4+0 ，可以直接丢弃，重新初始化客户端的readbuf
				if uint32(len(c.ReadBuf)) == msgLength {
					c.ReadBuf = []byte{}
					// 其他情况则是客户端readbuf的数据长度大于8（不可能小于8，小于8包都没传完），则忽略本次，从客户端的readbuf的下一个数据包开始读
				} else {
					c.ReadBuf = c.ReadBuf[msgLength:len(c.ReadBuf)]
				}

				msg := &msg.Msg{
					// MsgLength: msgLength,
					MsgType:  binary.BigEndian.Uint32(msgTypeBytes),
					MsgData:  []byte{},
					MsgProto: nil,
				}

				// c.ReadTotalBytesLth += msgLength
				c.ReadMsgPayloadLth = 0
				// 重新从剩下的客户端的readbuf里采集，获取下一个数据包的payload长度
				if len(c.ReadBuf) >= 4 {
					c.ReadMsgPayloadLth = binary.BigEndian.Uint32(c.ReadBuf[0:4])
				}
				return msg, nil
			} else {
				// 消息不是空的情况
				ReadMsgPayloadBytes := c.ReadBuf[8:msgLength]
				c.ReadBuf = c.ReadBuf[msgLength:len(c.ReadBuf)]

				msg := &msg.Msg{
					// MsgLength: msgLength,
					MsgType:  binary.BigEndian.Uint32(msgTypeBytes),
					MsgData:  ReadMsgPayloadBytes,
					MsgProto: nil,
				}

				// c.ReadTotalBytesLth += msgLength
				c.ReadMsgPayloadLth = 0
				// 重新采集下一个数据包，获取下一个数据包的payload长度
				if len(c.ReadBuf) >= 4 {
					c.ReadMsgPayloadLth = binary.BigEndian.Uint32(c.ReadBuf[0:4])
					log.Debugf("本地获取报文总长度 %d , 消息类型为 %d , 消息长度为 %d , 客户端 %s", msgLength, binary.BigEndian.Uint32(msgTypeBytes), c.ReadMsgPayloadLth, c.ClientID)
				}
				return msg, nil
			}
		}
	}
}

func (c *TcpClient) SendMsg(msg *msg.Msg) {
	protoMsgBytes := []byte{}
	var err error
	if msg.MsgProto != nil {
		protoMsgBytes, err = proto.Marshal(msg.MsgProto)
		if err != nil {
			log.Errorf("protobuf生成消息失败: %s", err.Error())
		}
	}
	msgLength := len(protoMsgBytes)
	msgType := msg.MsgType

	// 创建客户端缓冲池
	packetBuf := &bytes.Buffer{}

	var msgLengthBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(msgLengthBytes[:], uint32(msgLength))
	packetBuf.Write(msgLengthBytes[:])

	var msgTypeBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(msgTypeBytes, uint32(msgType))
	packetBuf.Write(msgTypeBytes[:])

	packetBuf.Write(protoMsgBytes[:])

	packet := packetBuf.Bytes()

	log.Debugf("发送报文,报文长度 %d,类型 %d, 具体消息长度 %d", msgLength+8, msgType, msgLength)
	log.Debugf("报文内容: %v", packet)

	c.SendLock.Lock()
	c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(config.GlobalConf.GetInt("tcp", "writetimeout")) * time.Second))
	_, err = c.Conn.Write(packet)
	if err != nil {
		log.Errorf("发送信息失败: %s", err.Error())
		c.SetValid(false)
	}
	c.SendLock.Unlock()

}
