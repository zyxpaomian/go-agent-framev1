package future

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type OsInfo struct {
	Os       string
	Localip  string
	Hostname string
	Cpu      int
	Mem      string
}

var Osinfo OsInfo

func (o *OsInfo) ColInit() {
	o.GetOS()
	o.GetLocalIp()
	o.GetHostname()
	o.GetCpu()
	o.GetMem()
}

func (o *OsInfo) GetOS() {
	o.Os = runtime.GOOS
}

func (o *OsInfo) GetLocalIp() {
	conn, err := net.Dial("udp4", "www.baidu.com:80")
	if err != nil {
		panic("无法获取本机IP地址")
	}
	defer conn.Close()
	o.Localip = strings.Split(conn.LocalAddr().String(), ":")[0]
}

func (o *OsInfo) GetHostname() {
	name, err := os.Hostname()
	if err != nil {
		o.Hostname = o.Localip
	} else {
		o.Hostname = name
	}
}

func (o *OsInfo) GetCpu() {
	o.Cpu = runtime.GOMAXPROCS(0)
}

func (o *OsInfo) GetMem() {
	if o.Os == "windows" {
		type memoryStatusEx struct {
			cbSize                  uint32
			dwMemoryLoad            uint32
			ullTotalPhys            uint64 // in bytes
			ullAvailPhys            uint64
			ullTotalPageFile        uint64
			ullAvailPageFile        uint64
			ullTotalVirtual         uint64
			ullAvailVirtual         uint64
			ullAvailExtendedVirtual uint64
		}
		kernel := syscall.NewLazyDLL("Kernel32.dll")
		GlobalMemoryStatusEx := kernel.NewProc("GlobalMemoryStatusEx")
		var memInfo memoryStatusEx
		memInfo.cbSize = uint32(unsafe.Sizeof(memInfo))
		mem, _, _ := GlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))
		if mem == 0 {
			o.Mem = "0"
		}
		o.Mem = fmt.Sprint(memInfo.ullTotalPhys / (1024 * 1024 * 1024))
	} else if o.Os == "linux" {
		command := "lsmem | grep 'Total online memory'"
		cmd := exec.Command("/bin/bash", "-c", command)
		bytes, err := cmd.Output()
		if err != nil {
			o.Mem = "0"
		}
		resp := strings.Split(string(bytes), "\n")[0]
		resp1 := strings.Split(resp, ":")[1]
		resp2 := strings.Split(resp1, " ")[7]
		resp3 := strings.Split(resp2, "G")[0]
		o.Mem = resp3
	} else {
		o.Mem = "Unsupport OS"
	}
}
