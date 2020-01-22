package future

import (
	"net"
	"strings"
	"structs"
	"time"
	log "util/agentlog"
	"util/influx"
)

func SendTcpRequest(destAddr string, destPort string) (*structs.TcpMesh, error) {
	result := &structs.TcpMesh{}
	conn, err := net.DialTimeout("tcp4", destAddr, time.Duration(2)*time.Second)

	if err != nil {
		log.Warnf("发起TCP探测 %s 失败", destAddr)
		result.Dstip = destAddr
		result.Status = "Fail"
		result.Exectime = time.Now().Format("2006-01-02 15:04:05")
		result.Port = destPort
		return result, err
	}
	result.Dstip = destAddr
	result.Status = "Good"
	result.Exectime = time.Now().Format("2006-01-02 15:04:05")
	result.Port = destPort
	defer conn.Close()
	return result, nil
}

func ProcessTcp(ipList []string, ch chan *structs.TcpMesh) {
	for _, ip := range ipList {
		port := strings.Split(ip, ":")[1]
		tcpresult, _ := SendTcpRequest(ip, port)
		ch <- tcpresult
	}
	close(ch)
}

func StartTcp(domainList []string, srcip string) {
	ipList := []string{}
	cli, err := influx.InfluxInit()
	if err != nil {
		log.Errorln("Influxdb初始化失败")
		return
	}
	for _, domainport := range domainList {
		domain := strings.Split(domainport, ":")[0]
		port := strings.Split(domainport, ":")[1]
		ip, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			log.Warnf("解析 %s 失败", ip.String())
			return
		}
		dectedip := ip.String() + ":" + port
		ipList = append(ipList, dectedip)
	}

	ch := make(chan *structs.TcpMesh, len(ipList))
	go ProcessTcp(ipList, ch)
	for result := range ch {
		tags := map[string]string{"dst": result.Dstip, "src": srcip}
		fields := map[string]interface{}{
			"port":   result.Port,
			"status": result.Status,
		}
		influx.SimpleInsert(cli, "pingmesh", "ncresult", tags, fields)
	}
}
