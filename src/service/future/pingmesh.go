package future

import (
	"bytes"
	"encoding/binary"
	"net"
	"structs"
	"time"
	log "util/agentlog"
	"util/influx"
)

type Icmp struct {
	Type        uint8
	Code        uint8
	CheckSum    uint16
	Identifier  uint16
	SequenceNum uint16
}

var Pingr Icmp

func (i *Icmp) ResetIcmp() {
	i.Type = 8
	i.Code = 0
	i.CheckSum = 0
	i.Identifier = 0
	i.SequenceNum = 1

	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, i)
	i.CheckReturnSum(buffer.Bytes())
	buffer.Reset()
}

func (i *Icmp) StartPing(domainList []string, srcip string) {
	ipList := []string{}
	cli, err := influx.InfluxInit()
	if err != nil {
		log.Errorln("Influxdb初始化失败")
		return
	}
	for _, domain := range domainList {
		ip, err := net.ResolveIPAddr("ip", domain)
		if err != nil {
			log.Warnf("解析 %s 失败", ip.String())
			return
		}
		ipList = append(ipList, ip.String())
	}

	ch := make(chan *structs.PingMesh, len(ipList))
	go i.ProcessPing(ipList, ch)
	for result := range ch {
		tags := map[string]string{"dst": result.Dstip, "src": srcip}
		fields := map[string]interface{}{
			"ttl":    result.Durtime,
			"status": result.Status,
		}
		influx.SimpleInsert(cli, "pingmesh", "pingresult", tags, fields)
	}
}

func (i *Icmp) ProcessPing(ipList []string, ch chan *structs.PingMesh) {
	for _, ip := range ipList {
		pingresult, _ := i.SendICMPRequest(ip)
		ch <- pingresult
	}
	close(ch)
}

func (i *Icmp) CheckReturnSum(data []byte) {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)
	i.CheckSum = uint16(^sum)
}

func (i *Icmp) SendICMPRequest(destAddr string) (*structs.PingMesh, error) {
	result := &structs.PingMesh{}
	conn, err := net.DialTimeout("ip4:icmp", destAddr, time.Duration(2)*time.Second)
	if err != nil {
		log.Warnf("发起Ping测 %s 失败", destAddr)
		result.Dstip = destAddr
		result.Durtime = 0
		result.Status = "Fail"
		result.Exectime = time.Now().Format("2006-01-02 15:04:05")
		return result, err
	}
	defer conn.Close()

	var buffer bytes.Buffer

	i.ResetIcmp()
	binary.Write(&buffer, binary.BigEndian, i)

	if _, err := conn.Write(buffer.Bytes()); err != nil {
		log.Warnf("Ping测 %s 发送报文失败,错误原因: %s", destAddr, err.Error())
		result.Dstip = destAddr
		result.Durtime = 0
		result.Status = "Fail"
		result.Exectime = time.Now().Format("2006-01-02 15:04:05")
		return result, err
	}

	tStart := time.Now()

	conn.SetReadDeadline((time.Now().Add(time.Second * 4)))

	recv := make([]byte, 256)
	receiveCnt, err := conn.Read(recv)

	if err != nil {
		log.Warnf("Ping测 %s 接受回包失败, 失败原因： %s", destAddr, err.Error())
		result.Dstip = destAddr
		result.Durtime = 0
		result.Status = "Fail"
		result.Exectime = time.Now().Format("2006-01-02 15:04:05")
		return result, err
	}

	tEnd := time.Now()
	duration := tEnd.Sub(tStart).Nanoseconds() / 1e6

	log.Debugf("Ping 测 %s, 字节 %d, 耗时 %d ms", destAddr, receiveCnt, duration)
	result.Dstip = destAddr
	result.Durtime = duration
	result.Status = "Good"
	result.Exectime = time.Now().Format("2006-01-02 15:04:05")

	return result, nil
}
