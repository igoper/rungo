package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/howeyc/fsnotify"
)

//runApp 运行application
func runApp() {
	readAppDirectories(currPath)
	NewWatcher()
	Autobuild()
	//阻塞进程防止执行一次就退出
	for {
		select {
		case <-exit:
			runtime.Goexit()
		}
	}
}

//readAppDirectories 读取所有需要监听的文件
func readAppDirectories(currentPath string) {
	fileinfos, err := ioutil.ReadDir(currentPath)
	if err != nil {
		fmt.Println(err.Error())
	}
	sp := "/"
	if runtime.GOOS == "windows" {
		sp = "\\"
	}
	for _, file := range fileinfos {
		if file.IsDir() {
			readAppDirectories(currentPath + sp + file.Name())
		} else {
			for _, ext := range config.ListenExts {
				if strings.HasSuffix(file.Name(), ext) {
					files = append(files, currentPath+sp+file.Name())
				}
			}
		}
	}
}

//NewWatcher 生成监听文件状态的监听器
func NewWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errLogger.Fatalf(" Fail to create new Watcher[ %s ]\n", err)
	}

	for {
		select {
		case ev := <-watcher.Event:
			isbuild := true
			//判断是否需要监听
			if !checkFileIsListen(ev.Name) {
				continue
			}
			mt := getFileModTime(ev.Name)
			if t := eventTime[ev.Name]; mt == t { //进程可能会一次返回多个修改状态.
				isbuild = false
			}
			eventTime[ev.Name] = mt
			if isbuild {
				go func() {
					// 防止段时间内触发
					scheduleTime = time.Now().Add(1 * time.Second)
					for {
						time.Sleep(scheduleTime.Sub(time.Now()))
						if time.Now().After(scheduleTime) {
							break
						}
						return
					}
					go Autobuild()
				}()
			}
		case err := <-watcher.Error:
			log.Fatal(err)
		}
	}
	infoLogger.Println("Start listening...")
	//添加监听的文件
	for _, file := range files {
		err = watcher.Watch(file)
		if err != nil {
			errLogger.Fatal(err)
		}
		infoLogger.Printf("Listening file %s\n", file)
	}
}

// getFileModTime 返回给定文件的修改时间
func getFileModTime(path string) int64 {
	path = strings.Replace(path, "\\", "/", -1)
	f, err := os.Open(path)
	if err != nil {
		errLogger.Printf("Fail to open file[ %s ]\n", err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		errLogger.Printf("Fail to get file information[ %s ]\n", err)
		return time.Now().Unix()
	}

	return fi.ModTime().Unix()
}

//checkFileIsListen 检查文件是否是需要监听
func checkFileIsListen(fileName string) bool {
	for _, v := range files {
		if v == fileName {
			return true
		}
	}
	return false
}

//Autobuild 构建APP
func Autobuild() {
	lock.Lock()
	defer lock.Unlock()
	infoLogger.Println("Start building...")
	err := os.Chdir(currPath) //改变目录到当前目录
	if err != nil {
		errLogger.Println("[ERROR] : ", err.Error())
	}
	avgs := []string{"build", "-o", config.AppName}
	bcmd := exec.Command("go", avgs...)
	bcmd.Stdout = os.Stdout
	bcmd.Stderr = os.Stderr
	bcmd.Env = append(os.Environ(), "GOGC=off")
	err = bcmd.Run()
	if err != nil {
		errLogger.Println("Build application failed")
		return
	}
	infoLogger.Println("Build application successful")
	Restart(config.AppName)
}

//Restart 重启应用
func Restart(appName string) {
	Kill()
	go Start(appName)
}

//Kill 杀死cmd子进程
func Kill() {
	//停止现在运行的进程
	defer func() {
		if e := recover(); e != nil {
			errLogger.Println("kill cmd process ", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		err := cmd.Process.Kill()
		if err != nil {
			errLogger.Println("kill cmd process ", err)
		}
	}
}

//Start 启动应用
func Start(appName string) {
	cmd = exec.Command("./" + appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Run()
	infoLogger.Printf("%s is runing...\n", appName)
}
