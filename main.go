package main

import (
	"fmt"
	"sign/settings"

	"github.com/gin-gonic/gin"
)

func main() {

	//初始化配置
	if err := settings.Init(); err != nil {
		fmt.Println("初始化配置失败！", err)
		return
	}

	//开始
	Start()

}
func Start() {

	//配置路由
	r := gin.Default()
	r.LoadHTMLFiles("templates/index.html")
	r.GET("/wzjsign", indexHandler)
	r.POST("/wzjsign", postHandler)
	r.GET("/wzjsign/ws", wsHandlernew) // WebSocket 连接处理
	r.Run(fmt.Sprintf(":%d", settings.Conf.APP.Port))

}
