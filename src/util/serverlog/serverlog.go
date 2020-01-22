package serverlog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"util/config"

	"github.com/Sirupsen/logrus"
)

// 封装logrus.Fields
type Fields logrus.Fields


func ServerInitLog() {
	//格式化输出
	logfile := config.GlobalConf.GetStr("server", "logname")
	loglevel := config.GlobalConf.GetStr("server", "loglevel")
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	//设备日志级别
	switch strings.ToUpper(loglevel) {
	case "DEBUG":
		logrus.SetLevel(logrus.DebugLevel)
	case "INFO":
		logrus.SetLevel(logrus.InfoLevel)
	case "WARN":
		logrus.SetLevel(logrus.WarnLevel)
	case "ERROR":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.DebugLevel)
	}

	//设置输出路径
	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic("打开日志文件失败")
	}
	logrus.SetOutput(file)

}

func Debugln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Debugln(args...)
}

func Debugf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Debugf(format, args...)
}

func Warnln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Warnln(args...)
}

func Warnf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Warnf(format, args...)
}

func Infoln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Infoln(args...)
}

func Infof(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Infof(format, args...)
}

func Errorln(args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Errorln(args...)
}

func Errorf(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	logrus.WithField("gofile", fmt.Sprintf("[%s:%d] ", file, line)).Errorf(format, args...)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		file = file[slash+1:]
	}
	fields["gofile"] = fmt.Sprintf("[%s:%d] ", file, line)
	return logrus.WithFields(fields)
}

/*package util

import (
	"log"
	"os"
	"io"
	"fmt"
)

type MyLogger struct {
	File *os.File
	Name string
	Logger *log.Logger
}

var GlobalLog MyLogger


func (l *MyLogger)LogInit() {
	logdir := GlobalConf.GetStr("server","logdir")
	logfile := GlobalConf.GetStr("server","logname")
	file, err := os.OpenFile(logdir + "/" + logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(log.LstdFlags|log.Lshortfile)
	log.SetOutput(file)
	log.Println("default 日志初始化完成")
}

func (l *MyLogger)GetLogger(logfile string)*MyLogger{
	if logfile != GlobalConf.GetStr("server","logname") {
		logdir := GlobalConf.GetStr("server","logdir")
		fmt.Println(logdir)
		file, err := os.OpenFile(logdir + "/" + logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
			return nil
		}
		dlogger := log.New(io.MultiWriter(file, os.Stderr), "", log.LstdFlags|log.Lshortfile)
		slogger := new(MyLogger)
		slogger.Logger = dlogger
		slogger.Name = logfile
		slogger.File = file
		slogger.Logger.SetOutput(file)

		return slogger
	}
	return nil
}

func (l *MyLogger)Close(){
	l.File.Close()
}

func (l *MyLogger) Info(s ...string) {
	log.Println(l.Name, s)
	l.Logger.SetPrefix("[INFO] ")
	l.Logger.Println(s)
}

func (l *MyLogger) Warning(s ...string) {
	log.Println(l.Name, s)
	l.Logger.SetPrefix("[WARN] ")
	l.Logger.Println(s)
}

func (l *MyLogger) Error(s ...string) {
	log.Println(l.Name, s)
	l.Logger.SetPrefix("[ERROR] ")
	l.Logger.Println(s)
}

func (l *MyLogger) Debug(s ...string) {
	log.Println(l.Name, s)
	l.Logger.SetPrefix("[DEBUG] ")
	l.Logger.Println(s)
}*/
