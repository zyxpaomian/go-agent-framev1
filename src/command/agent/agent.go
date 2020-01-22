package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"service/agent"
	"service/etcdclient"
	"service/future"
	"strconv"
	"time"
	log "util/agentlog"
	//"service/httpservice"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	// 获取主机底层信息
	future.Osinfo.ColInit()
	localip := future.Osinfo.Localip

	// 开始初始化etcd
	etcdclient.Etcdclient.ClientInit(localip, []string{"192.168.159.133:2379"})
	fmt.Println("启动etcd初始化完成")

	// 获取日志相关配置
	loglevel, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/loglevel")
	logdir, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/logdir")

	//日志初始化
	log.AgentInitLog(logdir, loglevel)
	fmt.Println("日志文件初始化完成....")

	//注册本地初始化信息到etcd
	fmt.Println("开始注册客户端信息到etcd")
	etcdclient.Etcdclient.ClientDiscovery(localip, "ip", localip)
	etcdclient.Etcdclient.ClientDiscovery(localip, "os", future.Osinfo.Os)
	etcdclient.Etcdclient.ClientDiscovery(localip, "hostname", future.Osinfo.Hostname)
	etcdclient.Etcdclient.ClientDiscovery(localip, "cpu", strconv.Itoa(future.Osinfo.Cpu))
	etcdclient.Etcdclient.ClientDiscovery(localip, "mem", future.Osinfo.Mem)
	etcdclient.Etcdclient.ClientDiscovery(localip, "pingswitch", "OFF")
	etcdclient.Etcdclient.ClientDiscovery(localip, "tcpswitch", "OFF")
	etcdclient.Etcdclient.ClientDiscovery(localip, "pcapswitch", "OFF")
	etcdclient.Etcdclient.ClientDiscovery(localip, "pcapiface", "")
	etcdclient.Etcdclient.ClientDiscovery(localip, "pcapfilter", "")

	// agent主程序启动
	fmt.Println("启动Agent....")
	agent := agent.NewAgent(localip)
	go agent.RunAgent(localip)

	select {}
	//var confPath = flag.String("confPath", "conf/agent.ini", "load conf file")
	//flag.Parse()

	//配置文件初始化

	//go future.Pingr.StartPing()
	//select {}

	// 启动TCP

	//fmt.Println(etcdclient.Etcdclient)
	//aa := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/tcp/writetimeout")
	//fmt.Println(aa)
	//wchchan := make(chan string)
	//go etcdclient.Etcdclient.WatchCfg("/server/cfg/tcp/writetimeout", wchchan)
	// fmt.Println(aa)

	//for {
	//	fmt.Println(wchchan)
	//}

	//aa, _ := etcdcfg.NewClient("1.1.1.1", []string{"192.168.159.133:2379"})
	//fmt.Println(aa)
	//go aa.ClientStart()

}
