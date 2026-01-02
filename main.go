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
	// 加载挂载权限
	LoadMountAllowed()
	// 恢复之前持久化的挂载任务（如果有）
	if err := mountMgr.Restore(); err != nil {
		fmt.Println("恢复挂载任务出错：", err)
	}
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

	// 挂载相关 API
	r.POST("/wzjsign/mount/start", mountStartHandler)
	r.POST("/wzjsign/mount/stop", mountStopHandler)
	r.GET("/wzjsign/mount/list", mountListHandler)
	// 管理接口
	r.GET("/wzjsign/mount/active", mountActiveDetailsHandler)
	r.POST("/wzjsign/mount/force_stop", mountForceStopHandler)

	// 地图坐标查询
	r.GET("/map/geocode", geocodeHandler)

	// 入口占位（仅作为导航入口）
	r.GET("/dfyautosign", func(c *gin.Context){ c.String(200, "分易签到入口（占位）") })
	r.GET("/xxtautosign", func(c *gin.Context){ c.String(200, "学习通签到入口（占位）") })

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
			// 刷新挂载权限列表
			LoadMountAllowed()
}