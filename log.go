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

// 日志等级
type LevelLog uint8

const (
	DEBUG LevelLog = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// 输出类型
type OutputType uint8

const (
	ONLY_TERMINAL          OutputType                  = iota // 输出到终端
	ONLY_FILE                                                 // 输出到文件
	BOTH_TERMINAL_AND_FILE = ONLY_TERMINAL | ONLY_FILE        // 既输出到终端也输出到文件
)

// 日志输出字段定制
type LogFlag uint8

const (
	FLAG_NONE     LogFlag = 0b00000000 // 无前缀标识
	FLAG_TIME     LogFlag = 0b00000001 // 有时间标识
	FLAG_THREADID LogFlag = 0b00000010 // 有线程ID标识
	FLAG_LEVEL    LogFlag = 0b00000010 // 有等级标识
	FLAG_FILENAME LogFlag = 0b00000100 // 有文件名标识
	FLAG_FUNCNAME LogFlag = 0b00001000 // 有函数名标识
	FLAG_LINENO   LogFlag = 0b00010000 // 有行号标识
	FLAG_ALL      LogFlag = 0b00011111 // 上述标识均有
)

//单条日志信息结构体
type logMsg struct {
	level    LevelLog
	msg      string
	time     string
	fileName string
	funcName string
	lineNo   int
}

// 日志对象结构体
type Logger struct {
	Level      LevelLog            // 日志等级
	LevelStr   map[LevelLog]string // 日志标识map
	OutputType OutputType          // 输出类型
	Flags      LogFlag             // 输出字段定义
	fileName   string              // 文件名
	filePath   string              // 日志路径
	fileObj    *os.File            // 日志对象
	msg        chan *logMsg        // 存储日志msg的通道
}

var once1 sync.Once // 实现日志单例对象
var once2 sync.Once // 实现只打开一次文件
var logger *Logger  // 定义单例日志指针

// 获取单例Logger对象
func getInstance() *Logger {
	if logger == nil {
		once1.Do(func() {
			logger = &Logger{
				Level: DEBUG,
				LevelStr: map[LevelLog]string{
					DEBUG:   "DEBUG  ",
					INFO:    "INFO   ",
					WARNING: "WARNING",
					ERROR:   "ERROR  ",
					FATAL:   "FATAL  ",
				},
				OutputType: BOTH_TERMINAL_AND_FILE,
				Flags:      FLAG_ALL,
				fileName:   "test.log",
				msg:        make(chan *logMsg, 1000),
			}
		})
	}
	return logger
}

func init() {
	// 初始化Log单例
	getInstance()

	// 初始化日志文件保存路径
	curPath, err := os.Getwd()
	logger.filePath = curPath

	if err != nil {
		fmt.Println("get current file path failed! err:", err)
		return
	}

	// 运行goroutine实现日志的写入打印操作
	go outPut()

	fmt.Println("Logger init Success!")
}

// 日志输出函数
func outPut() {

	var content string
	for {
		select {
		case log := <-logger.msg:

			content = logger.formatPrefix(*log) + log.msg

			//content = fmt.Sprintf("[%s] [%s] [%s %s() line%d] %v", log.time, logger.LevelStr[log.level], log.fileName, log.funcName, log.lineNo, log.msg)

			//判断是否输出到终端
			if logger.OutputType&ONLY_TERMINAL == ONLY_TERMINAL {
				fmt.Println(content)
			}

			//判断是否输出到文件
			if logger.OutputType&ONLY_FILE == ONLY_FILE {
				fmt.Fprintln(logger.fileObj, content)
			}
		default:
			break
		}
	}

}

func (l *Logger) handleLogMsg(logLevel LevelLog, msg interface{}) {

	// 第一次收到消息时判断是否需要打开文件
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

	// 处理收到的消息，填充结构体
	log := &logMsg{
		level: logLevel,
		msg:   fmt.Sprint(msg),
		time:  time.Now().Format("2006-01-02 15:04:05"),
	}

	// 填充函数名和行号
	fileName, funName, lineNo := getFuncCallerInfo()
	log.fileName = fileName
	log.funcName = funName
	log.lineNo = lineNo

	// 放入通道中
	l.msg <- log
}

// 设置输出类型
func SetOutputType(outputType OutputType) {
	logger.OutputType = outputType
}

// 设置输出类型
func SetFlags(flags LogFlag) {
	logger.Flags = flags
}

// 设置log文件名称
func SetFileName(name string) {
	logger.fileName = name
}

// 信息输出
func Info(msg interface{}) {
	logger.handleLogMsg(INFO, msg)
}

// 调试信息输出
func Debug(msg interface{}) {
	logger.handleLogMsg(DEBUG, msg)
}

// 警告信息输出
func Warning(msg interface{}) {
	logger.handleLogMsg(WARNING, msg)
}

// 严重错误信息输出
func Fatal(msg interface{}) {
	logger.handleLogMsg(FATAL, msg)
}

// 错误信息输出
func Error(msg interface{}) {
	logger.handleLogMsg(ERROR, msg)
}

// 获取打印日志语句所在函数的信息（文件名 函数名 行号）
func getFuncCallerInfo() (fileName string, funcName string, lineNo int) {
	pc, fileName, lineNo, ok := runtime.Caller(3)
	if !ok {
		fmt.Println("get FuncCaller Info failed")
	}

	// 获取到的是完整文件名，需要去除文件路径
	_, fileName = path.Split(fileName)

	// 获取函数名
	funcName = runtime.FuncForPC(pc).Name()
	temp := strings.Split(funcName, ".")
	funcName = strings.Join(temp[1:], "")

	return fileName, funcName, lineNo
}

// 通过falgs形成前缀
func (l *Logger) formatPrefix(log logMsg) string {
	//判断无标志则返回为空
	if logger.Flags == FLAG_NONE {
		return ""
	}

	//标识全有则按照固定格式输出所有信息
	if logger.Flags == FLAG_ALL {
		return fmt.Sprintf("[%s] [%s] [%s %s() line%d] ", log.time, logger.LevelStr[log.level], log.fileName, log.funcName, log.lineNo)
	}

	// 否则按照标识进行组合
	var prefix string
	if logger.Flags&FLAG_TIME == FLAG_TIME {
		prefix += fmt.Sprintf("[%s]", log.time)
	}

	if logger.Flags&FLAG_LEVEL == FLAG_LEVEL {
		if len(prefix) > 0 {
			prefix += " " + fmt.Sprintf("[%s]", logger.LevelStr[log.level])
		} else {
			prefix += fmt.Sprintf("[%s]", logger.LevelStr[log.level])
		}
	}

	if len(prefix) > 0 {
		prefix = fmt.Sprintf("%s ", prefix)
	}
	//线程ID

	//获取调用函数信息
	var funcInfo string
	if logger.Flags&FLAG_FILENAME == FLAG_FILENAME {
		funcInfo += log.fileName
	}

	if logger.Flags&FLAG_FUNCNAME == FLAG_FUNCNAME {
		if len(funcInfo) > 0 {
			funcInfo = " " + log.funcName + "()"
		} else {
			funcInfo += log.funcName + "()"
		}
	}

	if logger.Flags&FLAG_LINENO == FLAG_LINENO {
		if len(funcInfo) > 0 {
			funcInfo += " " + fmt.Sprintf("line%d", log.lineNo)
		} else {
			funcInfo += fmt.Sprintf("line%d", log.lineNo)
		}
	}

	if len(funcInfo) > 0 {
		funcInfo = fmt.Sprintf("[%s] ", funcInfo)
	}

	if len(prefix) > 0 && len(funcInfo) > 0 {
		return prefix + "" + funcInfo + ""
	}

	return prefix + funcInfo
}
