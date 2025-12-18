package main

import (
	"fmt"
	"net/http"
	"os"
	"sign/settings"
	"strconv"
	"strings"
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

// 添加允许被使用的用户路由
func addUserHandler(c *gin.Context) {
	if c.Request.Method == "POST" {
		action := c.PostForm("action")
		if action == "add" {
			name := c.PostForm("name")
			//向user.txt中添加用户
			if name != "" {
				Mu.Lock()
				f, err := os.OpenFile("user.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err == nil {
					fmt.Fprintln(f, name)
					f.Close()
				}
				Mu.Unlock()
			}
		} else if action == "delete" {
			name := c.PostForm("name")
			if name != "" {
				err := RemoveUserFromFile(name, "user.txt")
				if err != nil {
					fmt.Println("删除用户出错了！！", err)
				}
			}
		} else if action == "toggle_mode" {
			Test = !Test
		}
		// 刷新用户列表
		RefushUsers()
		c.Redirect(http.StatusSeeOther, "/adduser")
		return
	}
	// 显示页面
	mu.Lock()
	defer mu.Unlock()
	c.HTML(http.StatusOK, "adduser.html", gin.H{
		"users": Allow_users,
		"Test":  Test,
	})
}
func indexHandler(c *gin.Context) {
	c.HTML(200, "index.html", nil)
}

// postHandler 处理提交的openid 提交 POST 请求，返回 WebSocket URL
func postHandler(c *gin.Context) {
	var requestData struct {
		OpenID string `json:"openid"`
		Email  string `json:"email"`
		Latlon string `json:"latlon"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil || requestData.OpenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数无效"})
		return
	}

	// 净化openid
	openid := GetopenidFromUrl(requestData.OpenID)
	// 返回 WebSocket URL
	wsURL := "ws://" + c.Request.Host + "/wzjsign/ws?openid=" + openid + "&email=" + requestData.Email + "&latlon=" + requestData.Latlon
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
	if oldconn, ok := clients[clientID]; ok {
		ResponsMsg(oldconn, "之前的任务已暂停，这是新任务。")
		oldconn.Close()
		delete(clients, clientID)
	}
	clients[clientID] = conn
	mu.Unlock()

	defer func() {
		mu.Lock()
		if clients[clientID] == conn {
			delete(clients, clientID)
		}
		mu.Unlock()
		conn.Close()
	}()

	//业务处理
	// 获取 openid
	openid := c.Query("openid")
	email := c.Query("email")
	latlon := c.Query("latlon")

	var lat float64
	var lon float64
	if latlon!="" {
			//解析经纬度
	parts := strings.Split(latlon,",")
	if len(parts) !=2{
		ResponsMsg(conn,"经纬度格式不正确，请重新输入")
		conn.Close()
		return
	}
	lat, err = strconv.ParseFloat(parts[0], 64)
    if err != nil {
		ResponsMsg(conn,"纬度格式不正确，请重新输入")
		conn.Close()
        return
    }

    lon, err = strconv.ParseFloat(parts[1], 64)
    if err != nil {
        ResponsMsg(conn,"经度格式不正确，请重新输入")
		conn.Close()
        return
    }

	}


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
	name := studentinfo.Name

	if !Test {
		allow := false
		// 检查name是否在Allow_users中
		for _, user := range Allow_users {
			if user == name {
				allow = true
				break
			}
		}
		if !allow {
			ResponsMsg(conn, "你没有权限使用此功能，请联系管理员！")
			conn.Close()
			return
		}
	}

	//记录签到日志
	logger.Println("开启签到任务：", studentinfo, email, clientID, c.Request.UserAgent())
	if err := SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("%s %s", name, studentinfo.CollegeName)); err != nil {
		fmt.Println("发送邮件失败！", err)
	}
	ResponsMsg(conn, name+studentinfo.StudentNumber)

	for {
		activesign, err := getActiveSign(openid)
		if err != nil {
			if err := ResponsMsg(conn, err.Error()); err != nil {
				break
			}
			break
		}
		if len(activesign) != 0 {
			Signing := activesign[0]
			if err := ResponsMsg(conn, Signing.Name+"正在签到！！"); err != nil {
				break
			}
			if err := SendEmail(email, fmt.Sprintf("%s正在签到！！！", Signing.Name)); err != nil {
				fmt.Println("发送邮件失败！！！", err)
			}
			//GPS定位和普通签到处理
			if Signing.IsQR == 0 {
				ResponsMsg(conn, fmt.Sprintf("%s正在进行的是GPS或者普通签到！", Signing.Name))
				time.Sleep(10 * time.Second) //10s后签到
				signres, err := GetCommonSignRes(openid, Signing.CourseID, Signing.SignID,lat,lon)
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
						logger.Println("签到成功：", studentinfo, email, clientID, c.Request.UserAgent())
						if err := SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("%s %s 签到成功", name, studentinfo.CollegeName)); err != nil {
							fmt.Println("发送邮件失败！", err)
						}
					}
				}
				//二维码签到
			} else {
				if Signing.IsQR == 1 {
					ResponsMsg(conn, "正在进行二维码签到！若20s内还无二维码，请点重新提交")
					if err := SendEmail(email, fmt.Sprintf("课程%s正在进行二维码签到，请使用微信打开网站签到！", Signing.Name)); err != nil {
						fmt.Println("发送邮件失败！", err)
					}
					logger.Println("签到成功：", studentinfo, email, clientID, c.Request.UserAgent())
					if err := SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("%s %s 签到成功", name, studentinfo.CollegeName)); err != nil {
						fmt.Println("发送邮件失败！", err)
					}
					if err := Qrsign(conn, Signing.CourseID, Signing.SignID); err != nil {
						ResponsMsg(conn, err.Error())
					}

				}
			}

		} else {
			if err := ResponsMsg(conn, "目前木有签到，正在持续监听签到中,请勿关闭网页"); err != nil {
				break
			}
		}

		time.Sleep(10 * time.Second)
	}

}
