package future

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	log "util/agentlog"
)

func ScriptExec(scriptname string, argvs []string) (string, error) {
	if runtime.GOOS == "windows" {
		return WindowsScriptRun(scriptname, argvs)
	} else {
		return LinuxScriptRun(scriptname, argvs)
	}

}

func WindowsScriptRun(scriptname string, argvs []string) (string, error) {
	fullscriptpath := "D:\\Coding\\rinck_tcp\\plugin\\windows\\" + scriptname
	var cmdargv string
	for _, argv := range argvs {
		cmdargv = cmdargv + argv
	}
	fullcmd := fullscriptpath + " " + cmdargv
	result, err := exec.Command("cmd", "/c", fullcmd).Output()
	if err != nil {
		log.Errorf("Windows下执行脚本失败,失败原因: %s", err.Error())
	}
	return strings.TrimSpace(string(result)), nil
}

func LinuxScriptRun(scriptname string, argvs []string) (string, error) {
	fullscriptpath := "/etc/rinck/plugin/" + scriptname

	var cmdargv string
	var fullcmd string
	for _, argv := range argvs {
		cmdargv = cmdargv + argv
	}

	scripttype := strings.Split(scriptname, ".")[1]
	if scripttype == "py" {
		fullcmd = "`which python`" + " " + fullscriptpath + " " + cmdargv
	} else {
		fullcmd = fullscriptpath + " " + cmdargv
	}

	cmd := exec.Command("/bin/bash", "-c", fullcmd)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		log.Errorf("Linux下脚本执行失败, 脚本名: %s ,失败原因: %", scriptname, err.Error())
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
