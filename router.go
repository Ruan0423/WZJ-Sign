package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // 允许跨域请求
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
		if len(activesign)!=0 {
			Signing := activesign[0]
			ResponsMsg(conn,Signing.Name+"正在签到！！")
			if Signing.IsQR == 0 {
				ResponsMsg(conn, "正在进行的是GPS或者普通签到！")
				time.Sleep(5*time.Second)
				signres , err := GetCommonSignRes(openid, Signing.CourseID, Signing.SignID)
				if err != nil {
					ResponsMsg(conn, err.Error())
				} else {
					if signres.MsgClient!="" {
						ResponsMsg(conn,signres.MsgClient)
					} else {
						ResponsMsg(conn,fmt.Sprintf("签到成功，你是第%d个签到！",signres.SignRank))

					}
				}

			} else {
				if Signing.IsQR ==1 {
					ResponsMsg(conn,"正在进行二维码签到！")
					if err :=Qrsign(conn,Signing.CourseID,Signing.SignID) ;err!=nil {
						ResponsMsg(conn,err.Error())
					}
				}
			}
			
		} else {
			ResponsMsg(conn,"目前木有签到，正在持续监听签到中")
		}


		time.Sleep(10*time.Second)
	}



}



