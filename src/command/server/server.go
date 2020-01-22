package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"service/etcdclient"
	"service/httpservice"
	"service/server"
	"time"
	"util/config"
	"util/mysql"
	log "util/serverlog"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	var confPath = flag.String("confPath", "conf/default.ini", "load conf file")
	flag.Parse()

	//配置文件初始化
	config.GlobalConf.CfgInit(*confPath)
	fmt.Println("配置文件初始化完成....")

	// 开始初始化etcd
	etcdclient.Etcdclient.ClientInit("127.0.0.1", []string{"192.168.159.133:2379"})
	fmt.Println("启动etcd初始化完成")

	//日志初始化
	log.ServerInitLog()
	fmt.Println("日志文件初始化完成....")

	//DB初始化
	mysql.DB.DbInit()
	fmt.Println("数据库初始化完成....")

	fmt.Println("TCP服务启动....")
	server.InitServer()
	go server.Tcpserver.Run()

	// 启动http 服务器
	fmt.Println("HTTP服务器启动....")
	router := httpservice.InitRouter()
	httpserver := &http.Server{
		Addr:           "172.20.100.191:8080",
		Handler:        router,
		ReadTimeout:    3600 * time.Second,
		WriteTimeout:   3600 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go httpserver.ListenAndServe()

	// 启动TCP

	//测试
	//fmt.Println("####")

	//select {}
	//启动HTTP
	//fmt.Println("开始启动HTTP服务")

	//router.Run(":8000")
	//fmt.Println("启动HTTP服务器成功")
	//fmt.Println(http_srv.ListenAndServe())
	select {}
}
