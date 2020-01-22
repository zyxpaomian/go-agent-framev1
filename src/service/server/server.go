package server

import (
	"controller/agentctrl"
	"fmt"
	"math/rand"
	"msg"
	"net"
	"os"
	"service/etcdclient"
	"strings"
	"sync"
	"time"
	"util/config"
	ce "util/error"
	log "util/serverlog"

	"github.com/golang/protobuf/proto"
)

var Tcpserver *TcpServer

type TcpServer struct {
	// 客户端操作锁
	ClientLock *sync.RWMutex
	Clients    map[string]*TcpClient
}

func InitServer() {
	Tcpserver = &TcpServer{
		ClientLock: &sync.RWMutex{},
		Clients:    map[string]*TcpClient{},
	}
}

func (s *TcpServer) agentAliveCheck() {
	time.Sleep(time.Duration(config.GlobalConf.GetInt("tcp", "agentcheckinterval")) * time.Second)
	for {
		shouldagents, err := agentctrl.GetAgentInfo()
		if err != nil {
			log.Errorf("拉取所有Agent信息失败")
			time.Sleep(time.Duration(config.GlobalConf.GetInt("tcp", "agentcheckinterval")) * time.Second)
			continue
		}
		actualagents := s.ListAliveAgents()
		for _, shouldagent := range shouldagents {
			shouldagentip := shouldagent.Agentip
			shouldagent.Alive = 1
			for _, actualagent := range actualagents {
				if shouldagentip == actualagent {
					shouldagent.Alive = 0
					break
				}
			}
			err := agentctrl.UpdateAgentAlive(shouldagent.Alive, shouldagentip)
			if err != nil {
				log.Errorf("更新Agent存活状态失败")
				continue
			}
		}
		time.Sleep(time.Duration(config.GlobalConf.GetInt("tcp", "agentcheckinterval")) * time.Second)
	}
}
func (s *TcpServer) Run() {
	go s.agentAliveCheck()
	bind := config.GlobalConf.GetStr("tcp", "bind")
	l, err := net.Listen("tcp", bind)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Errorf("客户端Acceptc获取Error: %s", err.Error())
			continue
		}
		log.Debugf("收到来自客户端: %s ===> 目的端: %s 的请求", conn.RemoteAddr(), conn.LocalAddr())
		go s.HandleConn(conn)
	}
}

func (s *TcpServer) HandleConn(conn net.Conn) {
	defer conn.Close()

	clientid, err := s.GetConnId(conn)
	if err != nil {
		log.Errorf("获取客户端ConnID失败, 失败原因: %s", err.Error())
		return
	}
	s.ClientLock.Lock()
	if _, exist := s.Clients[clientid]; exist == true {
		log.Warnf("已存在链接，忽略来自 %s 的请求", clientid)
		s.ClientLock.Unlock()
		return
	}
	client := NewClient(conn, clientid)
	s.Clients[clientid] = client
	s.ClientLock.Unlock()
	// 数据处理
	for {
		msg, err := client.GetMsg()
		if err != nil {
			log.Errorf("读取客户端信息失败,客户端: %s 报错内容为: %s ", client.ClientID, err.Error())
			s.InValidClient(client)
			break
		}
		go s.HandleMsg(msg, client)
	}

}

func (s *TcpServer) HandleMsg(agentMsg *msg.Msg, client *TcpClient) error {
	switch agentMsg.MsgType {
	case msg.HeartBeat_Msg:
		s.HandleHeartBeatMsg(agentMsg, client)
	case msg.ShellRun_Rsp:
		s.HandleShellRunRspMsg(agentMsg, client)
	case msg.ShellRun_Rsp:
		s.HandleScriptRunRspMsg(agentMsg, client)
	default:
		return ce.New(fmt.Sprintf("未知的消息类型, %s: %d", client.ClientID, agentMsg.MsgType))
	}
	return nil
}

func (s *TcpServer) GetConnId(conn net.Conn) (string, error) {
	addr := conn.RemoteAddr().String()
	addrs := strings.Split(addr, ":")
	if len(addrs) != 2 {
		log.Errorf("对端地址信息异常: %s", addr)
		return "", ce.New("客户端获取信息异常")
	}
	return strings.TrimSpace(addrs[0]), nil
}

func (s *TcpServer) InValidClient(client *TcpClient) {
	client.Valid = false
	s.ClientLock.Lock()
	if _, exist := s.Clients[client.ClientID]; exist == true {
		log.Infof("移除客户端: %s", client.ClientID)
		delete(s.Clients, client.ClientID)
	}
	s.ClientLock.Unlock()
}

func (s *TcpServer) HandleHeartBeatMsg(agentMsg *msg.Msg, client *TcpClient) {
	heartbeatMsg := &msg.Heartbeat{}
	err := proto.Unmarshal(agentMsg.MsgData, heartbeatMsg)
	if err != nil {
		log.Errorf("解析心跳包失败, 失败原因: %s ", err.Error())
		return
	}
	log.Debugf("收到来自 %s 的心跳包, 心跳时间 %s, 心跳包状态为 %s", client.ClientID, heartbeatMsg.Synctime, heartbeatMsg.Status)
	client.SetLastHeartBeatSyncTime(heartbeatMsg.Synctime)
	heartbeatRspMsg := &msg.Msg{
		MsgType:  msg.HeartBeat_Rsp,
		MsgData:  []byte{},
		MsgProto: nil,
	}
	client.SendMsg(heartbeatRspMsg)

}

// 接受Agent 端的shell命令执行结果
func (s *TcpServer) HandleShellRunRspMsg(agentMsg *msg.Msg, client *TcpClient) {
	shellRunRspMsg := &msg.ShellRunRsp{}
	err := proto.Unmarshal(agentMsg.MsgData, shellRunRspMsg)
	if err != nil {
		log.Errorf("解析shell任务执行回包失败, 失败原因: %s ", err.Error())
		return
	}
	log.Infof("收到来自 %s 的 shell执行结果, 任务traceid: %s, 任务结果: %s ", client.ClientID, shellRunRspMsg.Taskid, shellRunRspMsg.RuncmdRsp)
	keyurl := "/task/result/" + shellRunRspMsg.Taskid + "/" + client.ClientID
	_, err = etcdclient.Etcdclient.ClientPut(keyurl, shellRunRspMsg.RuncmdRsp)
	if err != nil {
		log.Errorf("将执行结果存入Etcd失败,失败原因: %s", err.Error())
		return
	}
}

// 接受Agent 端的脚本命令执行结果
func (s *TcpServer) HandleScriptRunRspMsg(agentMsg *msg.Msg, client *TcpClient) {
	scriptRunRspMsg := &msg.ScriptRunRsp{}
	err := proto.Unmarshal(agentMsg.MsgData, scriptRunRspMsg)
	if err != nil {
		log.Errorf("解析插件任务执行回包失败, 失败原因: %s ", err.Error())
		return
	}
	log.Infof("收到来自 %s 的 插件执行结果, 任务traceid: %s, 任务结果: %s ", client.ClientID, scriptRunRspMsg.Taskid, scriptRunRspMsg.RunRsp)
	keyurl := "/task/result/" + scriptRunRspMsg.Taskid + "/" + client.ClientID
	_, err = etcdclient.Etcdclient.ClientPut(keyurl, scriptRunRspMsg.RunRsp)
	if err != nil {
		log.Errorf("将执行结果存入Etcd失败,失败原因: %s", err.Error())
		return
	}
}

// 远程Shell Or Cmd执行
func (s *TcpServer) RemoteShellRun(cmd string, iplist []string) (string, error) {
	taskid := fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
	shellRunMsg := &msg.Msg{
		MsgType: msg.ShellRun_Msg,
		MsgData: []byte{},
		MsgProto: &msg.ShellRun{
			Runcmd: cmd,
			Taskid: taskid,
		},
	}
	var ipstr string
	for _, ip := range iplist {
		if ipstr == "" {
			ipstr = ip
		} else {
			ipstr = ipstr + "," + ip
		}
	}
	keyurl := "/task/result/" + taskid + "/iplist"
	_, err := etcdclient.Etcdclient.ClientPut(keyurl, ipstr)
	if err != nil {
		log.Errorf("将任务IP清单存入etcd失败,失败原因: %s", err.Error())
		return "", err
	}
	s.Multicast(iplist, shellRunMsg)
	return taskid, nil
}

// 远程插件执行
func (s *TcpServer) RemoteScriptRun(scriptname string, argvs []string, iplist []string) (string, error) {
	taskid := fmt.Sprintf("%06v", rand.New(rand.NewSource(time.Now().UnixNano())).Int31n(1000000))
	ScriptRunMsg := &msg.Msg{
		MsgType: msg.ScriptRun_Msg,
		MsgData: []byte{},
		MsgProto: &msg.ShellRun{
			Scriptname: scriptname,
			Argvs:      argvs,
			Taskid:     taskid,
		},
	}
	var ipstr string
	for _, ip := range iplist {
		if ipstr == "" {
			ipstr = ip
		} else {
			ipstr = ipstr + "," + ip
		}
	}
	keyurl := "/task/result/" + taskid + "/iplist"
	_, err := etcdclient.Etcdclient.ClientPut(keyurl, ipstr)
	if err != nil {
		log.Errorf("将任务IP清单存入etcd失败,失败原因: %s", err.Error())
		return "", err
	}
	s.Multicast(iplist, ScriptRunMsg)
	return taskid, nil
}

// 文件传输
func (s *TcpServer) RemoteFileSend(srcfilename string, dstfilename string, iplist []string) (string, error) {
	fp, err := os.Open(srcfilename)
	if err != nil {
		log.Errorf("读取需要发送的文件失败,失败原因: %s", err.Error())
		return "", err
	}
	defer fp.Close()

	fileInfo, err := fp.Stat()
	if err != nil {
		return "", err
	}

	buffer := make([]byte, fileInfo.Size())
	_, err = fp.Read(buffer) // 文件内容读取到buffer中
	if err != nil {
		return "", err
	}

	FileTransMsg := &msg.Msg{
		MsgType: msg.FileTrans_Msg,
		MsgData: []byte{},
		MsgProto: &msg.FileTrans{
			SrcFilename: srcfilename,
			DstFilename: dstfilename,
			FileContent: buffer,
		},
	}
	var ipstr string
	for _, ip := range iplist {
		if ipstr == "" {
			ipstr = ip
		} else {
			ipstr = ipstr + "," + ip
		}
	}
	s.Multicast(iplist, FileTransMsg)
	return "Send File Success", nil

}

// 指定IP发送消息
func (s *TcpServer) Multicast(iplist []string, msg *msg.Msg) {
	s.ClientLock.Lock()
	for _, c := range s.Clients {
		for _, a := range iplist {
			if a == c.ClientID {
				c.SendMsg(msg)
			} else {
				continue
			}
		}
	}
	s.ClientLock.Unlock()
}

func (s *TcpServer) ListAliveAgents() []string {
	agents := []string{}
	s.ClientLock.Lock()
	for _, k := range s.Clients {
		agents = append(agents, k.ClientID)
	}
	s.ClientLock.Unlock()
	return agents
}

/*func test() {
	heartbeatmsg := &msg.Heartbeat{
		Id:   1,
		Time: "1970-01-01 00:00:00",
	}
	data, err := proto.Marshal(heartbeatmsg)
	if err != nil {
		fmt.Printf("marshaling error: ", err)
	}
	fmt.Println(data)
	heartbeatmsg1 := &msg.Heartbeat{}
	err = proto.Unmarshal(data, heartbeatmsg1)
	if err != nil {
		fmt.Printf("unmarshaling error: ", err)
	}
	fmt.Println(heartbeatmsg1)
	// Now test and newTest contain the same data.
	//test.GetOptionalgroup().GetRequiredField()
	//etc
}*/
