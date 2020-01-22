package agent

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"msg"
	"net"
	"os"
	"service/etcdclient"
	"service/future"
	"strconv"
	"strings"
	"sync"
	"time"
	log "util/agentlog"
	ce "util/error"

	"github.com/golang/protobuf/proto"
)

type Agent struct {
	Conn              net.Conn
	Valid             bool
	SendLock          *sync.Mutex
	ReadBuf           []byte
	ReadMsgPayloadLth uint32
	PingSwitch        string
	TcpSwitch         string
	PcapSwitch        string
	LocalIp           string
	//ReadTotalBytesLth     uint64
	//LastHeartbeatSyncTime string
}

// 创建新个客户端，初始化所有信息
func NewAgent(localip string) *Agent {
	agent := &Agent{}
	agent.ResetAgent(localip)
	return agent
}

// 初始化所有信息
func (a *Agent) ResetAgent(localip string) {
	log.Infoln("初始化Agent信息")
	a.Conn = nil
	a.Valid = true
	a.SendLock = &sync.Mutex{}
	a.ReadBuf = []byte{}
	a.ReadMsgPayloadLth = 0
	a.PingSwitch = "OFF"
	a.TcpSwitch = "OFF"
	a.PcapSwitch = "OFF"
	a.LocalIp = localip
}

// 运行客户端
func (a *Agent) RunAgent(localip string) {
	for {
		log.Infoln("启动Agent")
		a.StartAgent()
		log.Errorln("agent 发生异常，等待30s后重置客户端再次链接")
		a.ResetAgent(localip)
		time.Sleep(time.Duration(10) * time.Second)
	}
}

// 开启客户端监听服务
func (a *Agent) StartAgent() {
	agentbind, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/agentbind")
	conn, err := net.Dial("tcp", agentbind)
	if err != nil {
		log.Errorf("无法链接到服务器, 报错信息为: %s", err.Error())
		return
	}
	defer conn.Close()
	a.Conn = conn

	var wg sync.WaitGroup

	wg.Add(1)
	go a.Heartbeat(&wg)

	wg.Add(1)
	go a.Listen(&wg)

	wg.Add(1)
	go a.PingDetct(&wg)

	wg.Add(1)
	go a.TcpDetct(&wg)

	wg.Add(1)
	go a.PcapDump(&wg)

	wg.Add(1)
	go a.DataCollect(&wg)

	wg.Wait()
}

// 心跳服务,每5s发送一次心跳包
func (a *Agent) Heartbeat(wg *sync.WaitGroup) {
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		heartbeatMsg := &msg.Msg{
			MsgType: msg.HeartBeat_Msg,
			MsgProto: &msg.Heartbeat{
				Status:   "GOOD",
				Synctime: time.Now().Format("2006-01-02 15:04:05"),
			},
		}
		a.SendMsg(heartbeatMsg)
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		time.Sleep(time.Duration(5) * time.Second)
	}
	wg.Done()
}

// 客户端接口消息使用
func (a *Agent) Listen(wg *sync.WaitGroup) {
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}
		msg, err := a.GetMsg()
		if err != nil {
			log.Errorf("从服务端读取消息失败,失败原因为: %s", err.Error())
			a.Valid = false
			break
		}
		a.HandleMsg(msg)
	}
	wg.Done()
}

// 网络Ping探测服务
func (a *Agent) PingDetct(wg *sync.WaitGroup) {
	sleeptimestr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/detctinterval")
	sleeptime, _ := strconv.Atoi(sleeptimestr)
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		url := "/server/" + a.LocalIp + "/pingswitch"
		a.PingSwitch, _ = etcdclient.Etcdclient.GetSingleCfg(url)

		if a.PingSwitch == "OFF" {
			log.Debugln("PingMesh 开关关闭, 无需进行Ping探测, 30s 后再次进行检测开关是否打开")
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		domainListstr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/pingiprange")
		domainList := strings.Split(domainListstr, ",")
		future.Pingr.StartPing(domainList, a.LocalIp)
		time.Sleep(time.Duration(sleeptime) * time.Second)
	}
	wg.Done()
}

// 网络Tcp探测服务
func (a *Agent) TcpDetct(wg *sync.WaitGroup) {
	sleeptimestr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/detctinterval")
	sleeptime, _ := strconv.Atoi(sleeptimestr)
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		url := "/server/" + a.LocalIp + "/tcpswitch"
		a.TcpSwitch, _ = etcdclient.Etcdclient.GetSingleCfg(url)

		if a.TcpSwitch == "OFF" {
			log.Debugln("TcpMesh 开关关闭, 无需进行Ping探测, 30s 后再次进行检测开关是否打开")
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}
		domainListstr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/nciprange")
		domainList := strings.Split(domainListstr, ",")
		future.StartTcp(domainList, a.LocalIp)
		time.Sleep(time.Duration(sleeptime) * time.Second)
	}
	wg.Done()
}

// 在线抓包服务
func (a *Agent) PcapDump(wg *sync.WaitGroup) {
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		url := "/server/" + a.LocalIp + "/pcapswitch"
		a.PcapSwitch, _ = etcdclient.Etcdclient.GetSingleCfg(url)

		if a.PcapSwitch == "OFF" {
			log.Debugln("TcpDump 开关关闭, 无需进行在线抓包, 30s 后再次进行检测开关是否打开")
			time.Sleep(time.Duration(10) * time.Second)
			continue
		}

		ifaceurl := "/server/" + a.LocalIp + "/pcapiface"
		iface, _ := etcdclient.Etcdclient.GetSingleCfg(ifaceurl)

		filterurl := "/server/" + a.LocalIp + "/pcapfilter"
		filter, _ := etcdclient.Etcdclient.GetSingleCfg(filterurl)

		future.Capdump.CaptureInit(iface, filter)
		future.Capdump.CaptureStart(a.LocalIp)

		time.Sleep(time.Duration(5) * time.Second)
	}
	wg.Done()
}

// 主机信息收集
func (a *Agent) DataCollect(wg *sync.WaitGroup) {
	for {
		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		itemsurl := "/server/cfg/common/collectitems"
		itemsstr, _ := etcdclient.Etcdclient.GetSingleCfg(itemsurl)
		items := strings.Split(itemsstr, ",")

		future.MannualCollect(items)

		if a.Valid == false {
			log.Errorln("当前客户端不可用")
			break
		}

		time.Sleep(time.Duration(30) * time.Second)
	}
	wg.Done()
}

// 客户端处理消息使用
func (a *Agent) HandleMsg(serverMsg *msg.Msg) error {
	switch serverMsg.MsgType {
	case msg.HeartBeat_Rsp:
		a.HandleHeartBeatRspMsg(serverMsg)
	case msg.ShellRun_Msg:
		a.HandleShellRunMsg(serverMsg)
	case msg.FileTrans_Msg:
		a.HandleFileTransMsg(serverMsg)
	case msg.ScriptRun_Msg:
		a.HandleScriptRunMsg(serverMsg)
	default:
		return ce.New(fmt.Sprintf("未知的消息类型, %d", serverMsg.MsgType))
	}
	return nil
}

// 处理收到的心跳包
func (a *Agent) HandleHeartBeatRspMsg(serverMsg *msg.Msg) {
	log.Debugln("收到了来自客户端的心跳响应包")
}

// 处理收到的shell命令执行请求
func (a *Agent) HandleShellRunMsg(serverMsg *msg.Msg) {
	shellRunMsg := &msg.ShellRun{}
	err := proto.Unmarshal(serverMsg.MsgData, shellRunMsg)
	if err != nil {
		log.Errorf("解析需要执行的shell命令失败, 失败原因: %s", err.Error())
		return
	}
	log.Infof("收到来自服务端发来的shell命令: %s, 任务traceid : %s", shellRunMsg.Runcmd, shellRunMsg.Taskid)
	shellResult, err := future.ExecShell(shellRunMsg.Runcmd)
	if err != nil {
		log.Errorf("shell命令执行失败,shell 命令: %s ,失败原因: %s", shellRunMsg.Runcmd, err.Error())
		return
	}
	log.Infof("shell命令执行成功, shell 命令: %s, 执行结果: %s, 任务traceid : %s", shellRunMsg.Runcmd, shellResult, shellRunMsg.Taskid)

	shellRunRspMsg := &msg.Msg{
		MsgType: msg.ShellRun_Rsp,
		MsgProto: &msg.ShellRunRsp{
			RuncmdRsp: shellResult,
			Taskid:    shellRunMsg.Taskid,
		},
	}
	a.SendMsg(shellRunRspMsg)

}

// 处理收到的插件脚本执行请求
func (a *Agent) HandleScriptRunMsg(serverMsg *msg.Msg) {
	ScriptRunMsg := &msg.ScriptRun{}
	err := proto.Unmarshal(serverMsg.MsgData, ScriptRunMsg)
	if err != nil {
		log.Errorf("解析需要执行的插件失败, 失败原因: %s", err.Error())
		return
	}
	log.Infof("收到来自服务端发来的插件执行请求: %s, 任务traceid : %s", ScriptRunMsg.Scriptname, ScriptRunMsg.Taskid)
	scriptResult, err := future.ScriptExec(ScriptRunMsg.Scriptname, ScriptRunMsg.Argvs)
	if err != nil {
		log.Errorf("插件执行失败,插件 名称: %s ,失败原因: %s", ScriptRunMsg.Scriptname, err.Error())
		return
	}
	log.Infof("插件执行成功, 插件名称: %s, 执行结果: %s, 任务traceid : %s", ScriptRunMsg.Scriptname, scriptResult, ScriptRunMsg.Taskid)

	ScriptRunRspMsg := &msg.Msg{
		MsgType: msg.ScriptRun_Rsp,
		MsgProto: &msg.ScriptRunRsp{
			RunRsp: scriptResult,
			Taskid: ScriptRunMsg.Taskid,
		},
	}
	a.SendMsg(ScriptRunRspMsg)
}

// 处理收到的文件
func (a *Agent) HandleFileTransMsg(serverMsg *msg.Msg) {
	fileTransMsg := &msg.FileTrans{}
	err := proto.Unmarshal(serverMsg.MsgData, fileTransMsg)
	if err != nil {
		log.Errorf("解析收到的数据包失败, 失败原因: %s", err.Error())
		return
	}
	log.Infof("收到来自服务器推送的文件, 文件名: %s ", fileTransMsg.DstFilename)
	fileObj, err := os.Create(fileTransMsg.DstFilename)
	if err != nil {
		log.Errorf("打开文件: %s 句柄失败, 失败原因: %s", fileTransMsg.DstFilename, err.Error())
		return
	}
	writer := bufio.NewWriter(fileObj)
	defer writer.Flush()
	writer.WriteString(string(fileTransMsg.FileContent))
	log.Infof("保存文件: %s 成功", fileTransMsg.DstFilename)
}

// Agent接受消息处理，处理粘包
func (a *Agent) GetMsg() (*msg.Msg, error) {
	/*
		4 字节报文长度
		4 字节的报文类型
		报文(暂定protobuf)
	*/
	for {
		if a.Valid == false {
			log.Errorf("客户端无效, 退出读取循环")
			return nil, ce.New("Agent状态错误,退出循环，等待下次重连")
		}
		rtimeoutstr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/readtimeout")
		rtimeout, _ := strconv.Atoi(rtimeoutstr)
		a.Conn.SetReadDeadline(time.Now().Add(time.Duration(rtimeout) * time.Second))
		// 当报文长度信息还没有获取到的时候
		if len(a.ReadBuf) < 4 {
			log.Debugln("当前客户端的readbuf长度小于前4字节的报文长度,准备读取报文")
			readBuf := make([]byte, 256)
			n, err := a.Conn.Read(readBuf)
			log.Debugln("从服务端读取到报文")
			if err != nil {
				log.Errorf("从服务端读取报文数据失败,报错信息为: %s", err.Error())
				return nil, err
			}
			a.Conn.SetReadDeadline(time.Time{})
			log.Debugf("此次从服务端读取到了 %d 长度的报文数据, 报文内容为 %v", n, readBuf[:n])
			// 将本次从客户端读取到的数据加载到客户端的readbuf中
			a.ReadBuf = append(a.ReadBuf, readBuf[:n]...)
			// 加载后判断当前客户端readbuf的长度是否满足报文长度的需求，如不满足，等待下次循环
			if len(a.ReadBuf) >= 4 {
				a.ReadMsgPayloadLth = binary.BigEndian.Uint32(a.ReadBuf[0:4])
				log.Debugf("获取到的payload长度，长度为 %d", a.ReadMsgPayloadLth)
			}
			// 数据包长度已获取，拿到了一个完整数据包的总长度，但包没传完，重新初始化一个缓冲池继续读报文
		} else if a.ReadMsgPayloadLth+8 > uint32(len(a.ReadBuf)) {
			readBuf := make([]byte, 256)
			n, err := a.Conn.Read(readBuf)
			if err != nil {
				log.Errorf("读取报文数据失败,报错信息为: %s", err.Error())
				return nil, err
			}
			a.Conn.SetReadDeadline(time.Time{})
			log.Debugf("此次读取到了 %d 长度的报文数据, 报文内容为 %v", n, readBuf[:n])
			// 将本次从客户端读取到的数据加载到客户端的readbuf中
			a.ReadBuf = append(a.ReadBuf, readBuf[:n]...)
			// 报文读完了，但客户端的readbuf里可能还有下一个报文的一部分信息,这里处理下粘包以及解析报文数据
		} else {
			a.Conn.SetReadDeadline(time.Time{})
			log.Debugln("本次报文都已读取完成")
			msgLength := 8 + a.ReadMsgPayloadLth
			msgTypeBytes := a.ReadBuf[4:8]
			log.Debugf("本地获取报文总长度 %d , 消息类型为 %d , 消息长度为 %d ", msgLength, binary.BigEndian.Uint32(msgTypeBytes), a.ReadMsgPayloadLth)

			//消息是空数据的情况，消息实体为空
			if a.ReadMsgPayloadLth == 0 {
				// 如果客户端readbuf的数据长度和需要读取的数据长度一致，则相当于是个空包，即4+4+0 ，可以直接丢弃，重新初始化客户端的readbuf
				if uint32(len(a.ReadBuf)) == msgLength {
					a.ReadBuf = []byte{}
					// 其他情况则是客户端readbuf的数据长度大于8（不可能小于8，小于8包都没传完），则忽略本次，从客户端的readbuf的下一个数据包开始读
				} else {
					a.ReadBuf = a.ReadBuf[msgLength:len(a.ReadBuf)]
				}

				msg := &msg.Msg{
					MsgType:  binary.BigEndian.Uint32(msgTypeBytes),
					MsgData:  []byte{},
					MsgProto: nil,
				}

				a.ReadMsgPayloadLth = 0
				// 重新从剩下的客户端的readbuf里采集，获取下一个数据包的payload长度
				if len(a.ReadBuf) >= 4 {
					a.ReadMsgPayloadLth = binary.BigEndian.Uint32(a.ReadBuf[0:4])
				}
				return msg, nil
			} else {
				// 消息不是空的情况
				ReadMsgPayloadBytes := a.ReadBuf[8:msgLength]
				a.ReadBuf = a.ReadBuf[msgLength:len(a.ReadBuf)]

				msg := &msg.Msg{
					MsgType:  binary.BigEndian.Uint32(msgTypeBytes),
					MsgData:  ReadMsgPayloadBytes,
					MsgProto: nil,
				}

				a.ReadMsgPayloadLth = 0
				// 重新采集下一个数据包，获取下一个数据包的payload长度
				if len(a.ReadBuf) >= 4 {
					a.ReadMsgPayloadLth = binary.BigEndian.Uint32(a.ReadBuf[0:4])
					log.Debugf("本地获取报文总长度 %d , 消息类型为 %d , 消息长度为 %d ", msgLength, binary.BigEndian.Uint32(msgTypeBytes), a.ReadMsgPayloadLth)
				}
				return msg, nil
			}
		}
	}
}

func (a *Agent) SendMsg(msg *msg.Msg) {
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

	a.SendLock.Lock()
	wtimeoutstr, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/writetimeout")
	wtimeout, _ := strconv.Atoi(wtimeoutstr)
	a.Conn.SetWriteDeadline(time.Now().Add(time.Duration(wtimeout) * time.Second))
	_, err = a.Conn.Write(packet)
	if err != nil {
		log.Errorf("发送信息失败: %s", err.Error())
		a.Valid = false
	}
	a.SendLock.Unlock()
}
