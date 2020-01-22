package httpservice

import (
	"os"
	"util/config"

	"github.com/gin-gonic/gin"
	// log "util/serverlog"
)

func InitRouter() *gin.Engine {
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)
	logdir := config.GlobalConf.GetStr("server", "logdir")
	logfile := config.GlobalConf.GetStr("server", "httplogname")
	file, err := os.OpenFile(logdir+"/"+logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic("打开Gin HTTP日志文件失败")
	}
	gin.DefaultWriter = file

	router := gin.Default()
	// 远程操作相关接口
	router.POST("/api/v1/agent/remoteshell", RemoteShellRun)
	router.GET("/api/v1/agent/getShellResult", GetShellResult)
	router.POST("/api/v1/agent/fileTrans", FileTrans)
	return router
}
