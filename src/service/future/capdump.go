package future

import (
	"encoding/binary"
	"net"
	"service/etcdclient"
	"time"
	log "util/agentlog"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

type CapDump struct {
	Handle *pcap.Handle
	Iface  string
	Filter string
}

var Capdump CapDump

func (c *CapDump) CaptureInit(iface string, filter string) {
	var err error

	c.Iface = iface
	c.Filter = filter
	c.Handle, err = pcap.OpenLive(c.Iface, 70000, true, pcap.BlockForever)
	if err != nil {
		log.Errorf("初始化抓包失败,失败原因:%s", err.Error())
	}
}

func (c *CapDump) CaptureStart(localip string) {
	dumpserver, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/common/dumpserver")
	conn, err := net.Dial("tcp4", dumpserver)
	if err != nil {
		log.Errorf("发送抓包结果失败，失败原因:%s", err.Error())

	}

	err1 := c.Handle.SetBPFFilter(c.Filter)
	if err1 != nil {
		log.Errorf("设置抓包过滤条件失败,失败原因:%s", err1.Error())
	}
	packetSource := gopacket.NewPacketSource(c.Handle, c.Handle.LinkType())
	packets := packetSource.Packets()

	for packet := range packets {

		if conn == nil {
			log.Errorln("TCP连接到服务端失败，等待10S后重试")
			time.Sleep(time.Duration(10) * time.Second)
			break
		}

		url := "/server/" + localip + "/pcapswitch"
		pcapswitch, _ := etcdclient.Etcdclient.GetSingleCfg(url)
		if pcapswitch == "OFF" {
			c.CloseCapture()
			conn.Close()
			break
		}
		// 发送包长度
		dataLength := len(packet.Data())
		var result [4]byte
		binary.LittleEndian.PutUint32(result[:], uint32(dataLength))
		_, err = conn.Write([]byte{result[0], result[1], result[2], result[3]})
		if err != nil {
			log.Errorf("发送包长度失败,失败原因: %s", err.Error())
			break
		}

		// 发送包类型
		typeContent := 1
		var result2 [4]byte
		binary.LittleEndian.PutUint32(result[2:], uint32(typeContent))
		_, err = conn.Write([]byte{result2[3]})
		if err != nil {
			log.Errorf("发送包类型失败,失败原因: %s", err.Error())
			break
		}

		// 发送数据内容
		_, err = conn.Write(packet.Data())
		if err != nil {
			log.Errorf("发送包内容失败,失败原因: %s", err.Error())
			break
		}
	}
}

func (c *CapDump) CloseCapture() {
	c.Handle.Close()
}
