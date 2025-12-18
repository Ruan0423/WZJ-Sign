package main

import (
	"bufio"
	"fmt"
	"os"
	"sign/settings"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	Allow_users []string
    Test = false
	Mu sync.Mutex
)
func main() {

	//初始化配置
	if err := settings.Init(); err != nil {
		fmt.Println("初始化配置失败！", err)
		return
	}
	//配置日志
	initlogger()
	RefushUsers()
	//开始
	Start()

}
func Start() {

	//配置路由
	r := gin.Default()
	r.LoadHTMLFiles("templates/index.html","templates/adduser.html")
	r.GET("/wzjsign", indexHandler)
	r.POST("/wzjsign", postHandler)

	r.GET("/adduser",addUserHandler)
	r.POST("/adduser",addUserHandler)
	r.GET("/wzjsign/ws", wsHandlernew) // WebSocket 连接处理


	r.Run(fmt.Sprintf(":%d", settings.Conf.APP.Port))

}

func RefushUsers(){
			// 刷新用户列表
			mu.Lock()
			Allow_users = []string{} // 清空当前列表
			if file, err := os.Open("user.txt"); err == nil {
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					Allow_users = append(Allow_users, scanner.Text())
				}
				file.Close()
			}
			mu.Unlock()
}