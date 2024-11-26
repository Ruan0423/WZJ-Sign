package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sign/settings"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
	"gopkg.in/gomail.v2"
)

var errreq = errors.New("openid失效啦")

// Verifyopenid 验证openid是否有效
func Verifyopenid(status string) bool {
	if status == "401 Unauthorized" {
		return false
	}
	return true
}

// 获取学生姓名
func getstudentName(openid string) (interface{}, error) {
	type Item struct {
		ItemName    string      `json:"item_name"`
		ItemComment string      `json:"item_comment"`
		ItemValue   interface{} `json:"item_value"`
	}
	status, jsondata := RequestStudentinfo(openid)
	if !Verifyopenid(status) {
		return "nil", errreq
	}
	var data [][]Item
	if err := json.Unmarshal([]byte(jsondata), &data); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return "nil", errreq
	}

	return data[0][2].ItemValue, nil
}

// getActiveSign 获取正在进行的签到，返回的是签到课程列表。
func getActiveSign(openid string) ([]ActiveSign, error) {

	status, jsondata := RequestActiveSign(openid)
	if !Verifyopenid(status) {
		return nil, errreq
	}
	var data []ActiveSign
	if err := json.Unmarshal([]byte(jsondata), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// 获取普通签到结果
func GetCommonSignRes(openid string, courseid int, signid int) (SignResult, error) {
	status, jsondata := RequestSign(openid, courseid, signid)
	if !Verifyopenid(status) {
		return SignResult{}, errreq
	}
	var data SignResult
	if err := json.Unmarshal([]byte(jsondata), &data); err != nil {
		return data, err
	}
	return data, nil
}

// GetopenidFromUrl 提取精华openid
func GetopenidFromUrl(url string) string {
	if strings.Contains(url, "openid") {
		start := strings.Index(url, "openid") + 7
		end := strings.Index(url, "&")
		return url[start:end]
	} else {
		return url
	}
}

// 向已经连接ws通讯的客户端发送消息
func ResponsMsg(conn *websocket.Conn, msg string) error {
	return conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// 向ws客户端发送二维码数据
func ResponseQR(conn *websocket.Conn, qrdata string) error {
	qrbyte, err := qrcode.Encode(qrdata, qrcode.Medium, 256)
	if err != nil {
		return err
	}
	resdatawithbase64 := base64.RawStdEncoding.EncodeToString(qrbyte)
	err = conn.WriteJSON(map[string]string{
		"type":  "qrcode",
		"data":  resdatawithbase64,
		"qrUrl": qrdata,
	})
	if err != nil {
		return err
	}
	return nil
}

// 配置邮箱

func SendEmail(to string, msg string) error {
	e := gomail.NewMessage()
	e.SetHeader("From",settings.Conf.Email.UserName)
	e.SetHeader("To", to)
	e.SetHeader("Subject", "微助教自动签到通知")

	e.SetBody("text/plain", msg)

	d := gomail.NewDialer(
		settings.Conf.Email.Host,
		settings.Conf.Email.Port,
		settings.Conf.Email.UserName,
		settings.Conf.Email.PassWord,
	)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(e); err != nil {
		return err
	}
	return nil
}
