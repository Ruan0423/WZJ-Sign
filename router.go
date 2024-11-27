package main

import (
	"fmt"
	"net/http"
	"sign/settings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // 允许跨域请求
}

var clients = make(map[string]*websocket.Conn)
var mu sync.Mutex

func indexHandler(c *gin.Context) {
	c.HTML(200, "index.html", nil)
}

// postHandler 处理提交的openid 提交 POST 请求，返回 WebSocket URL
func postHandler(c *gin.Context) {
	var requestData struct {
		OpenID string `json:"openid"`
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil || requestData.OpenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 净化openid
	openid := GetopenidFromUrl(requestData.OpenID)
	// 返回 WebSocket URL
	wsURL := "ws://" + c.Request.Host + "/wzjsign/ws?openid=" + openid + "&email=" + requestData.Email
	c.JSON(http.StatusOK, gin.H{"wsUrl": wsURL})
}
func wsHandlernew(c *gin.Context) {
	//检查cilents里面是否有连接
	clientID := c.RemoteIP()

	//将gin http升级为websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	mu.Lock()
	if oldconn,ok :=clients[clientID];ok {
		ResponsMsg(oldconn,"之前的任务已暂停，这是新任务。")
		oldconn.Close()
		delete(clients,clientID)
	}
	clients[clientID] =conn 
	mu.Unlock()

	defer func ()  {
		mu.Lock()
		if clients[clientID]==conn{
			delete(clients,clientID)
		}
		mu.Unlock()
		conn.Close()
	}()

	//业务处理
	// 获取 openid
	openid := c.Query("openid")
	email := c.Query("email")
	if openid == "" {
		conn.WriteMessage(websocket.TextMessage, []byte("Error: openid 参数缺失"))
		return
	}
	//获取个人信息并返回
	studentinfo, err := getstudentInfo(openid)
	if err != nil {
		ResponsMsg(conn, err.Error())
		conn.Close()
		return
	}
	name :=studentinfo.Name
	//记录签到日志
	logger.Println(studentinfo,email,clientID,c.Request.UserAgent())
	if err :=SendEmail(settings.Conf.Email.UserName,fmt.Sprintf("%s %s",name,studentinfo.CollegeName));err!=nil{
		fmt.Println("发送邮件失败！",err)
	}
	ResponsMsg(conn, name + studentinfo.StudentNumber)


	for {
		activesign, err := getActiveSign(openid)
		if err != nil {
			if err:=ResponsMsg(conn, err.Error());err!=nil{
				break
			}
			break
		}
		if len(activesign) != 0 {
			Signing := activesign[0]
			if err:=ResponsMsg(conn, Signing.Name+"正在签到！！");err!=nil{
				break
			}
			if err := SendEmail(email, fmt.Sprintf("%s正在签到！！！", Signing.Name)); err != nil {
				fmt.Println("发送邮件失败！！！", err)
			}
			//GPS定位和普通签到处理
			if Signing.IsQR == 0 {
				ResponsMsg(conn, fmt.Sprintf("%s正在进行的是GPS或者普通签到！",Signing.Name))
				time.Sleep(10 * time.Second) //10s后签到
				signres, err := GetCommonSignRes(openid, Signing.CourseID, Signing.SignID)
				if err != nil {
					ResponsMsg(conn, err.Error())
				} else {
					
					if signres.MsgClient != "" {
						//签到结果
						ResponsMsg(conn, signres.MsgClient)
					} else {

						if err := SendEmail(email, fmt.Sprintf("恭喜你！！%s签到成功了！", Signing.Name)); err != nil {
							fmt.Println("发送邮件失败！", err)
						}
						ResponsMsg(conn, fmt.Sprintf("签到成功，你是第%d个签到！", signres.SignRank))
					}
				}
				//二维码签到
			} else {
				if Signing.IsQR == 1 {
					ResponsMsg(conn, "正在进行二维码签到！")
					if err := SendEmail(email, fmt.Sprintf("课程%s正在进行二维码签到，请使用微信打开网站签到！", Signing.Name)); err != nil {
						fmt.Println("发送邮件失败！", err)
					}
					if err := Qrsign(conn, Signing.CourseID, Signing.SignID); err != nil {
						ResponsMsg(conn, err.Error())
					}
				}
			}

		} else {
			if err :=ResponsMsg(conn, "目前木有签到，正在持续监听签到中");err!=nil{
				break
			}
		}

		time.Sleep(10 * time.Second)
	}

}


