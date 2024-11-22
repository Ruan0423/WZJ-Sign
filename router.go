package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // 允许跨域请求
}

func Start() {

}

func 
func wsHandler(c *gin.Context) {
	//将gin http升级为websocket
	conn ,err := upgrader.Upgrade(c.Writer, c.Request,nil)
	if err!= nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
	defer conn.Close()

	//获取参数
	
}