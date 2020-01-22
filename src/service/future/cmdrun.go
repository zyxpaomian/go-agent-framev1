package future

import (
	"bytes"
	"runtime"

	"os/exec"
	"strings"
	log "util/agentlog"
)

func ExecShell(cmd string) (string, error) {
	if runtime.GOOS == "windows" {
		return WindowsCmdRun(cmd)
	} else {
		return LinuxCmdRun(cmd)
	}

}

func WindowsCmdRun(cmd string) (string, error) {
	result, err := exec.Command("cmd", "/c", cmd).Output()
	if err != nil {
		log.Errorf("Windows下执行命令失败,失败原因: %s", err.Error())
		return "", err
	}
	return strings.TrimSpace(string(result)), nil
}

func LinuxCmdRun(s string) (string, error) {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command("/bin/bash", "-c", s)

	//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
	var out bytes.Buffer
	cmd.Stdout = &out

	//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
	err := cmd.Run()
	if err != nil {
		log.Errorf("shell命令执行失败,shell 命令: %s ,失败原因: %", s, err.Error())
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
