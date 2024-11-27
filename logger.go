package main

import (
	"fmt"
	"log"
	"os"
)

//定义新的logger，方便记录自己想要的信息
var logger *log.Logger

func initlogger() {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err!=nil {
		fmt.Println("初始化日志时打开日志文件失败~",err)
		return
	}
	logger = log.New(logFile,"", log.Ldate|log.Ltime|log.Lshortfile)
}