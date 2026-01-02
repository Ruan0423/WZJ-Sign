package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
func getstudentInfo(openid string) (StudenetInfo, error) {
	var data_role []StudenetInfo
	//获取姓名
	type Item struct {
		ItemName    string      `json:"item_name"`
		ItemComment string      `json:"item_comment"`
		ItemValue   interface{} `json:"item_value"`
	}
	status, jsondata_name := RequestStudentinfo(openid)
	if !Verifyopenid(status) {
		return StudenetInfo{}, errreq
	}
	var data [][]Item
	if err := json.Unmarshal([]byte(jsondata_name), &data); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return StudenetInfo{}, errreq
	}
	student_name := data[0][2].ItemValue

	//获取其他信息
	status, jsondata_role := RequestStudentRole(openid)
	if !Verifyopenid(status) {
		return StudenetInfo{}, errreq
	}
	if err := json.Unmarshal([]byte(jsondata_role), &data_role); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return StudenetInfo{}, errreq
	}
	data_role[0].Name = student_name.(string)
	return data_role[0], nil
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
func GetCommonSignRes(openid string, courseid int, signid int,lat float64,lon float64) (SignResult, error) {
	status, jsondata := RequestSign(openid, courseid, signid,lat,lon)
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

// SendEmailAsync 异步发送邮件（不阻塞调用者），仅记录错误
func SendEmailAsync(to string, msg string) {
	go func() {
		if err := SendEmail(to, msg); err != nil {
			if settings.Conf != nil {
				// 如果用户邮箱发送失败，发给管理员并在日志记录
				logger.Println("SendEmailAsync 发送失败：", err, "to:", to)
				_ = SendEmail(settings.Conf.Email.UserName, "异步发送邮件失败："+err.Error()+" to:"+to)
			} else {
				logger.Println("SendEmailAsync 发送失败：", err)
			}
		}
	}()
}

// 获取所有课程的作业

func GetHomeworks(openid string) (Homeworks , error) {
	status , data := RequsetHomwork(openid)
	if status == "304 Not Modified" {
		GetHomeworks(openid)
	} else {
		if !Verifyopenid(status) {
			return Homeworks{},errreq
		}else {
			var Homeworks Homeworks
			err := json.Unmarshal([]byte(data),&Homeworks)
			if err!=nil {
				return Homeworks,err
			}
			return Homeworks,nil
		}
	}
	return Homeworks{},nil
}

// 获取某个课程的答题
func GetQuestions(openid string, CourseID int)(Question,error){
	status , data := RequestQuestions(openid,CourseID)
	if status == "304 Not Modified" {
		GetQuestions(openid,CourseID)
	} else {
		if !Verifyopenid(status) {
			return Question{},errreq
		}else {
			var Question Question
			err := json.Unmarshal([]byte(data),&Question)
			if err!=nil {
				return Question,err
			}
			return Question,nil
		}
	}
	return Question{},nil
}


// 从user.txt中删除某个用户
func RemoveUserFromFile(userName, fileName string) error {
	// 确认输入文件存在
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return fmt.Errorf("文件 %s 不存在", fileName)
	}

	// 打开文件进行读取
	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("打开文件时出错: %v", err)
	}
	defer file.Close()

	// 获取文件所在目录
	dir := filepath.Dir(fileName)
	// 创建临时文件用于保存结果，放在同一目录下
	tempFile, err := os.CreateTemp(dir, "temp*.txt")
	if err != nil {
		return fmt.Errorf("创建临时文件时出错: %v", err)
	}
	defer tempFile.Close()

	removed := false
	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(tempFile)

	// 遍历每一行，并将不匹配的行写入临时文件
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(line, userName) {
			removed = true
			continue // 跳过当前行(即删除指定用户)
		}
		_, _ = writer.WriteString(line + "\n")
	}

	// 检查错误
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取文件时出错: %v", err)
	}

	// 刷新缓冲区
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新写入缓冲区时出错: %v", err)
	}

	// 关闭原始文件和临时文件以确保它们不再被占用
	file.Close()
	tempFile.Close()

	// 如果找到了并移除了指定的姓名，则替换原文件
	if removed {
		// 删除原文件
		if err := os.Remove(fileName); err != nil {
			return fmt.Errorf("删除原文件时出错: %v", err)
		}

		// 重命名临时文件为原文件名
		if err := os.Rename(tempFile.Name(), fileName); err != nil {
			return fmt.Errorf("重命名临时文件时出错: %v", err)
		}
	} else {
		os.Remove(tempFile.Name()) // 删除临时文件
		return fmt.Errorf("未找到姓名: %s", userName)
	}

	return nil
}