package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// HandshakeResponse 定义握手消息的结构
type HandshakeResponse struct {
	ID                       string   `json:"id"`
	Channel                  string   `json:"channel"`
	Successful               bool     `json:"successful"`
	Version                  string   `json:"version"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
	ClientID                 string   `json:"clientId"`
	Advice                   struct {
		Reconnect string `json:"reconnect"`
		Interval  int    `json:"interval"`
		Timeout   int    `json:"timeout"`
	} `json:"advice"`
}

// ConnectResponse 定义 /meta/connect 消息结构
type ConnectResponse struct {
	ID         string `json:"id"`
	ClientID   string `json:"clientId"`
	Channel    string `json:"channel"`
	Successful bool   `json:"successful"`
	Advice     struct {
		Reconnect string `json:"reconnect"`
		Interval  int    `json:"interval"`
		Timeout   int    `json:"timeout"`
	} `json:"advice"`
}

// subscribeResponse 订阅接受二维码信息
type SubscribeResponse struct {
	ClientID string `json:"clientId"`
	Data     struct {
		QrUrl string `json:"qrUrl"`
	} `json:"data"`
	Ext struct {
		InnerFayeToken string `json:"innerFayeToken"`
	} `json:"ext"`
}

func Qrsign(clientconn *websocket.Conn, courseID int, signID int) error {
	// WebSocket 服务器地址
	serverURL := "wss://www.teachermate.com.cn/faye"

	// 连接服务器
	subconn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Printf("Failed to connect: %v", err)
		return err
	}
	defer subconn.Close()

	log.Println("Connected to server")

	// 发送握手消息
	handshakeMsg := map[string]interface{}{
		"channel":                  "/meta/handshake",
		"version":                  "1.0",
		"supportedConnectionTypes": []string{"websocket"},
		"id":                       "1",
	}
	if err := subconn.WriteJSON(handshakeMsg); err != nil {
		log.Printf("Failed to send handshake: %v", err)
		return err
	}

	// 读取握手响应
	_, message, err := subconn.ReadMessage()
	if err != nil {
		log.Printf("Failed to read handshake response: %v", err)
		return err
	}

	// 解析握手响应（数组形式）
	var handshakeRes []HandshakeResponse
	if err := json.Unmarshal(message, &handshakeRes); err != nil {
		log.Printf("Failed to parse handshake response: %v", err)
		return err
	}

	// 检查握手结果
	if len(handshakeRes) == 0 || !handshakeRes[0].Successful {
		log.Printf("Handshake failed: %+v", handshakeRes)
		return err
	}

	clientID := handshakeRes[0].ClientID
	timeout := handshakeRes[0].Advice.Timeout
	log.Printf("Handshake successful, ClientID: %s", clientID)

	// 发送连接消息
	sendConnect := func() {
		connectMsg := map[string]interface{}{
			"channel":        "/meta/connect",
			"clientId":       clientID,
			"connectionType": "websocket",
			"id":             "2",
		}
		if err := subconn.WriteJSON(connectMsg); err != nil {
			log.Printf("Failed to send connect message: %v", err)
			return
		}
	}

	// 首次发送连接消息
	sendConnect()

	// 订阅特定频道

	subscribeMsg := map[string]interface{}{
		"channel":      "/meta/subscribe",
		"clientId":     clientID,
		"subscription": fmt.Sprintf("/attendance/%d/%d/qr", courseID, signID),
		"id":           "3",
	}
	if err := subconn.WriteJSON(subscribeMsg); err != nil {
		log.Printf("Failed to send subscribe message: %v", err)
	}
	log.Println("Subscription message sent")

	stop := make(chan struct{})

	// 启动一个新的 goroutine
	go func() {
		ticker := time.NewTicker(time.Duration(timeout/2) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sendConnect()
			case <-stop:
				fmt.Println("收到停止信号，准备退出协程")
				return
			}
		}
	}()

	stopflag := 0
	// 循环接收消息
	for {
		_, message, err := subconn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read message: %v", err)
		}
		var submsg []SubscribeResponse
		if err := json.Unmarshal(message, &submsg); err != nil {
			fmt.Println(err)
		} else {
			if submsg[0].ClientID == "" {
				// ResponsMsg(clientconn, "二维码签到链接："+submsg[0].Data.QrUrl)
				ResponseQR(clientconn, submsg[0].Data.QrUrl)
			} else {
				if submsg[0].ClientID != "" {
					stopflag += 1
					fmt.Println("stopflag:", stopflag)
				}
			}

		}
		if stopflag == 20 {
			close(stop)
			return fmt.Errorf("二维码签到结束！")
		}
	}
}

//有开启的二维码签到：
//[{"channel":"/attendance/1388301/3401562/qr","data":{"type":1,"qrUrl":"https://www.teachermate.com.cn/api/v1/qr/attendance/31e2ee622ed10ac3697b506d6df9bf3d6dbbf419c2a8bf237cd072b0271d2bff1d1df918963fa5aad8a02d6f4d103469"},"id":"e1r","ext":{"innerFayeToken":"f6c0fba75e72f8c0cd05f8386704605a"}}]
