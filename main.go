package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	lock         sync.Mutex
	currPath     string
	config       Config
	files        []string //监听的文件目录
	cmd          *exec.Cmd
	exit         chan bool
	eventTime    = make(map[string]int64)
	scheduleTime time.Time
	infoLogger   log.Logger
	errLogger    log.Logger
)

type Config struct {
	AppName    string   `json:"app_name"`
	ListenExts []string `json:"listen_exts"`
}

func main() {
	infoLogger.SetOutput(os.Stdout)
	infoLogger.SetPrefix("[INFO]:")
	infoLogger.SetFlags(log.LstdFlags)

	infoLogger.SetOutput(os.Stdout)
	infoLogger.SetPrefix("[ERROR]:")
	infoLogger.SetFlags(log.LstdFlags)
	//获取当前目录地址
	_currPath, err := os.Getwd()
	currPath = _currPath
	if err != nil {
		errLogger.Fatalln(err.Error())
	}
	config.ListenExts = []string{".go"}
	outputExt := ""
	if runtime.GOOS == "windows" {
		outputExt = ".exe"
	}
	//生成可执行文件的名称
	config.AppName = filepath.Base(currPath) + outputExt
	//运行执行内容
	runApp()
}
