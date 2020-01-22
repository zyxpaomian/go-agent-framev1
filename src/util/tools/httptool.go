package tools

import (
	"util/log"
	"github.com/asmcos/requests"
)

func HttpGetClient(url string) {
	resp,err := requests.Get(url)
	if err != nil{
		log.Errorf("调用计划任务接口失败,失败URL: %s",url)
		return 
	  }
	log.Infof("调用计划任务接口成功,成功URL： %s, 返回值: %s", url,resp.Text())
}