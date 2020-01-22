package httpservice

import (
	"net/http"
	"service/etcdclient"
	"service/server"
	"strings"

	"github.com/gin-gonic/gin"
)

func RemoteShellRun(c *gin.Context) {
	type reqContent struct {
		Taskcmd string `json:"taskcmd" binding:"required"`
		TaskIp  string `json:"taskip" binding:"required"`
	}
	var r reqContent
	err := c.ShouldBindJSON(&r)
	if err == nil {
		taskcmd := r.Taskcmd
		taskip := r.TaskIp
		taskiplist := strings.Split(taskip, ",")

		// 服务端发送消息到客户端并获取任务ID
		taskid, _ := server.Tcpserver.RemoteShellRun(taskcmd, taskiplist)
		c.JSON(http.StatusOK, gin.H{
			"Status": "Success",
			"Msg":    taskid,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"Status": "Failed",
			"Msg":    "参数获取失败",
		})
	}
}

func GetShellResult(c *gin.Context) {
	type reqContent struct {
		TaskId string `json:"taskid" binding:"required"`
	}
	var r reqContent
	err := c.ShouldBindJSON(&r)
	if err == nil {
		taskid := r.TaskId

		queryurl := "/task/result/" + taskid + "/iplist"
		ipstr, err := etcdclient.Etcdclient.GetSingleCfg(queryurl)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status": "Failed",
				"Msg":    "拉取任务IP清单失败",
			})
		} else {
			iplist := strings.Split(ipstr, ",")
			resultdict := make(map[string]string)
			var finresult []interface{}
			for _, ip := range iplist {
				queryurl := "/task/result/" + taskid + "/" + ip
				resultdict["ip"] = ip
				resultdict["status"] = "failed"
				resultdict["output"] = "unknown error"
				result, err := etcdclient.Etcdclient.GetSingleCfg(queryurl)
				if err == nil {
					resultdict["ip"] = ip
					resultdict["status"] = "success"
					resultdict["output"] = result
				}
				finresult = append(finresult, resultdict)
			}

			c.JSON(http.StatusOK, gin.H{
				"Status": "Success",
				"Msg":    finresult,
			})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"Status": "Failed",
			"Msg":    "参数获取失败",
		})
	}
}

func FileTrans(c *gin.Context) {
	type reqContent struct {
		Srcfilename string `json:"srcfilename" binding:"required"`
		Dstfilename string `json:"dstfilename" binding:"required"`
		Ipstr       string `json:"ipstr" binding:"required"`
	}
	var r reqContent
	err := c.ShouldBindJSON(&r)
	if err == nil {
		srcfilename := r.Srcfilename
		dstfilename := r.Dstfilename
		ipstr := r.Ipstr

		iplist := strings.Split(ipstr, ",")
		sendresult, _ := server.Tcpserver.RemoteFileSend(srcfilename, dstfilename, iplist)

		c.JSON(http.StatusOK, gin.H{
			"Status": "Success",
			"Msg":    sendresult,
		})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"Status": "Failed",
			"Msg":    "参数获取失败",
		})
	}
}
