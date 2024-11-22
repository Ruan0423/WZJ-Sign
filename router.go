package main

import (
	"fmt"
	"net/http"
	"sign/settings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // 允许跨域请求
}

func Start() {
		//初始化配置
		if err:=settings.Init();err!=nil {
			fmt.Println("初始化配置失败！", err)
			return
		}
	
		//配置路由
		r:=gin.Default()
		r.LoadHTMLFiles("templates/index.html")
		r.GET("/wzjsign",indexHandler)
		r.POST("/wzjsign",postHandler)
		r.GET("/wzjsign/ws", wsHandler)  // WebSocket 连接处理
		r.Run(fmt.Sprintf(":%d",settings.Conf.Port))

}

func indexHandler(c *gin.Context) {
	c.HTML(200,"index.html",nil)
}
// postHandler 处理提交的openid 提交 POST 请求，返回 WebSocket URL
func postHandler(c *gin.Context) {
    var requestData struct {
        OpenID string `json:"openid"`
    }

    if err := c.ShouldBindJSON(&requestData); err != nil || requestData.OpenID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
        return
    }

	// 净化openid
	openid := GetopenidFromUrl(requestData.OpenID)

    // 返回 WebSocket URL
    wsURL := "ws://" + c.Request.Host + "/wzjsign/ws?openid=" + openid
    c.JSON(http.StatusOK, gin.H{"wsUrl": wsURL})
}
func wsHandler(c *gin.Context) {
	//将gin http升级为websocket
	conn ,err := upgrader.Upgrade(c.Writer, c.Request,nil)
	if err!= nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
	defer conn.Close()

	//业务处理
	// 获取 openid
    openid := c.Query("openid")
    if openid == "" {
        conn.WriteMessage(websocket.TextMessage, []byte("Error: openid 参数缺失"))
        return
    }
	//获取个人信息并返回
	name,err :=getstudentName(openid)
	if err!=nil {
		ResponsMsg(conn,err.Error())
		conn.Close()
		return
	}
	ResponsMsg(conn,name.(string))

	//
	for {
		activesign, err :=getActiveSign(openid)
		if err!=nil {
			ResponsMsg(conn,err.Error())
			break
		}
		if activesign!=nil {
			ResponsMsg(conn,activesign[0].Name+"正在签到！！")
		}
		time.Sleep(2*time.Second)
	}



}



