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
	ID                     string   `json:"id"`
	Channel                string   `json:"channel"`
	Successful             bool     `json:"successful"`
	Version                string   `json:"version"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
	ClientID               string   `json:"clientId"`
	Advice                 struct {
		Reconnect string `json:"reconnect"`
		Interval  int    `json:"interval"`
		Timeout   int    `json:"timeout"`
	} `json:"advice"`
}

// ConnectResponse 定义 /meta/connect 消息结构
type ConnectResponse struct {
	ID        string `json:"id"`
	ClientID  string `json:"clientId"`
	Channel   string `json:"channel"`
	Successful bool   `json:"successful"`
	Advice    struct {
		Reconnect string `json:"reconnect"`
		Interval  int    `json:"interval"`
		Timeout   int    `json:"timeout"`
	} `json:"advice"`
}

func Qrsign() {
	// WebSocket 服务器地址
	serverURL := "wss://www.teachermate.com.cn/faye"

	// 连接服务器
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	log.Println("Connected to server")

	// 发送握手消息
	handshakeMsg := map[string]interface{}{
		"channel":                  "/meta/handshake",
		"version":                  "1.0",
		"supportedConnectionTypes": []string{"websocket"},
		"id":                       "1",
	}
	if err := conn.WriteJSON(handshakeMsg); err != nil {
		log.Fatalf("Failed to send handshake: %v", err)
	}

	// 读取握手响应
	_, message, err := conn.ReadMessage()
	if err != nil {
		log.Fatalf("Failed to read handshake response: %v", err)
	}

	// 解析握手响应（数组形式）
	var handshakeRes []HandshakeResponse
	if err := json.Unmarshal(message, &handshakeRes); err != nil {
		log.Fatalf("Failed to parse handshake response: %v", err)
	}

	// 检查握手结果
	if len(handshakeRes) == 0 || !handshakeRes[0].Successful {
		log.Fatalf("Handshake failed: %+v", handshakeRes)
	}

	clientID := handshakeRes[0].ClientID
	timeout := handshakeRes[0].Advice.Timeout
	log.Printf("Handshake successful, ClientID: %s", clientID)

	// 发送连接消息
	sendConnect := func() {
		connectMsg := map[string]interface{}{
			"channel":       "/meta/connect",
			"clientId":      clientID,
			"connectionType": "websocket",
			"id":            "2",
		}
		if err := conn.WriteJSON(connectMsg); err != nil {
			log.Fatalf("Failed to send connect message: %v", err)
		}
	}

	// 首次发送连接消息
	sendConnect()

	// 订阅特定频道
	courseID := 1388301 // 替换为实际的 courseId
	signID := 3395080   // 替换为实际的 signId
	subscribeMsg := map[string]interface{}{
		"channel":     "/meta/subscribe",
		"clientId":    clientID,
		"subscription": fmt.Sprintf("/attendance/%d/%d/qr", courseID, signID),
		"id":          "3",
	}
	if err := conn.WriteJSON(subscribeMsg); err != nil {
		log.Fatalf("Failed to send subscribe message: %v", err)
	}
	log.Println("Subscription message sent")

	// 定时发送 /meta/connect 保持心跳
	go func() {
		ticker := time.NewTicker(time.Duration(timeout/2) * time.Millisecond)
		defer ticker.Stop()
		for {
			<-ticker.C
			sendConnect()
		}
	}()

	// 循环接收消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("Failed to read message: %v", err)
		}

		// 打印接收到的消息
		log.Printf("Received message: %s", message)
	}
}
