package MyLog

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel uint8

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

type OutputType uint8

const (
	ONLY_TERMINAL          OutputType                  = iota //输出到终端
	ONLY_FILE                                                 //输出到文件
	BOTH_TERMINAL_AND_FILE = ONLY_TERMINAL | ONLY_FILE        //既输出到终端也输出到文件
)

type logMsg struct {
	level    LogLevel
	msg      string
	time     string
	fileName string
	funcName string
	lineNo   int
}

type Logger struct {
	Level      LogLevel            //日志等级
	LevelStr   map[LogLevel]string //日志标识map
	OutputType OutputType          //输出类型

	fileName string       //文件名
	filePath string       //日志路径
	fileObj  *os.File     //日志对象
	msg      chan *logMsg //存储日志msg的通道
}

var once1 sync.Once
var once2 sync.Once
var logger *Logger

//获取单例Logger对象
func getInstance() *Logger {
	if logger == nil {
		once1.Do(func() {
			logger = &Logger{
				Level: DEBUG,
				LevelStr: map[LogLevel]string{
					DEBUG:   "DEBUG  ",
					INFO:    "INFO   ",
					WARNING: "WARNING",
					ERROR:   "ERROR  ",
					FATAL:   "FATAL  ",
				},
				OutputType: BOTH_TERMINAL_AND_FILE,
				fileName:   "test.log",
				msg:        make(chan *logMsg, 1000),
			}
		})
	}

	return logger
}

//初始化Log单例
func init() {
	getInstance()

	curPath, _ := os.Getwd()
	logger.filePath = curPath

	//打开文件

	go outPut()

}

func outPut() {

	var content string
	for {
		select {
		case log := <-logger.msg:
			content = fmt.Sprintf("[%s] [%s] [%s %s() line%d]%v", log.time, logger.LevelStr[log.level], log.fileName, log.funcName, log.lineNo, log.msg)
			if logger.OutputType&ONLY_TERMINAL == ONLY_TERMINAL {
				fmt.Println(content)
			}

			if logger.OutputType&ONLY_FILE == ONLY_FILE {
				fmt.Fprintln(logger.fileObj, content)
			}
		default:
			break
		}
	}

}

func (l *Logger) handleLogMsg(logLevel LogLevel, msg interface{}) {

	//第一次收到消息时判断是否需要打开文件
	once2.Do(func() {
		if l.OutputType&ONLY_FILE == ONLY_FILE {
			fileObj, err := os.OpenFile(path.Join(logger.filePath, logger.fileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Println("open file failed, err:", err)
				return
			}

			logger.fileObj = fileObj
		}

	})

	//处理收到的消息，填充结构体
	log := &logMsg{
		level: logLevel,
		msg:   fmt.Sprint(msg),
		time:  time.Now().Format("2006-01-02 15:04:05"),
	}

	//填充函数名和行号
	fileName,funName, lineNo := getFuncCallerInfo()
	log.fileName = fileName
	log.funcName = funName
	log.lineNo = lineNo

	l.msg <- log
}

func SetOutputType(outputType OutputType) {
	logger.OutputType = outputType
}

//设置log文件名称
func SetFileName(name string) {
	logger.fileName = name
}

//信息输出
func Info(msg interface{}) {
	logger.handleLogMsg(INFO, msg)
}

//调试信息输出
func Debug(msg interface{}) {
	logger.handleLogMsg(DEBUG, msg)
}

//警告信息输出
func Warning(msg interface{}) {
	logger.handleLogMsg(WARNING, msg)
}

//严重错误信息输出
func Fatal(msg interface{}) {
	logger.handleLogMsg(FATAL, msg)
}

//错误信息输出
func Error(msg interface{}) {
	logger.handleLogMsg(ERROR, msg)
}

//获取调用函数的信息（文件名 函数名 行号）
func getFuncCallerInfo() (fileName string, funcName string, lineNo int) {
	pc, fileName, lineNo, ok := runtime.Caller(3)
	if !ok {
		fmt.Println("get FuncCaller Info failed")
	}

	//获取到的是完整文件名，需要去除文件路径
	_, fileName = path.Split(fileName)

	//获取函数名
	funcName = runtime.FuncForPC(pc).Name()
	temp := strings.Split(funcName, ".")
	funcName = strings.Join(temp[1:], "")

	return fileName, funcName, lineNo
}
