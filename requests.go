package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var BaseHeader = map[string][]string{
	"User-Agent":      {"Mozilla/5.0 (Linux; Android 5.0; SM-N9100 Build/LRX21V) > AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 > Chrome/37.0.0.0 Mobile Safari/537.36 > MicroMessenger/6.0.2.56_r958800.520 NetType/WIFI"},
	"Content-Type":    {"application/json"},
	"Accept":          {"*/*"},
	"Accept-Language": {"zh-CN,en-US;qbaseHeaders=0.7,en;q=0.3"},
}

var (
	OpenidURL = "https://v18.teachermate.cn/wechat-pro-ssr/?openid=1afa187e0cb54ddb7c7db49ad859f97f&from=wzj"
	Openid    = "e32b7d07fc718217b1ccf724b0083df2"
	lat = "30.520517"
	lon = "114.423792"
)

var (
	APIReferrer    = "https://v18.teachermate.cn/wechat-pro/student/edit?openid=" + Openid
	APIStudentinfo = "https://v18.teachermate.cn/wechat-api/v2/students"
	APIStudentRole = "https://v18.teachermate.cn/wechat-api/v2/students/role"
	APIActiveSign  = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student/active_signs"

	APISignIn = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student-sign-in"
)

// RequestStudentRole 获取个人信息请求
func RequestStudentRole(openid string) (status string, Resdata string) {
	//新建请求
	req, err := http.NewRequest("GET", APIStudentRole, nil)
	if err != nil {
		log.Println("创建RequestsStudentRole失败：", err)
		return
	}
	//添加header
	for k, v := range BaseHeader {
		req.Header[k] = v
	}
	req.Header.Add("Openid", openid)
	req.Header.Add("referrer", APIReferrer)

	//创建http客户端
	cilent := &http.Client{}

	res, err := cilent.Do(req)
	if err != nil {
		fmt.Println("qignqiuerr:", err)
	}
	defer res.Body.Close()
	data, _ := ioutil.ReadAll(res.Body)
	status = res.Status
	Resdata = string(data)

	return
}

// RequestStudentinfo 获取学生信息
func RequestStudentinfo(openid string) (status string, data string) {

	req, err := http.NewRequest("GET", APIStudentinfo, nil)
	if err != nil {
		log.Println("Create RequestStudentinfo Req err!", err)
		return
	}
	for k, v := range BaseHeader {
		req.Header[k] = v
	}
	req.Header.Add("Openid", openid)
	req.Header.Add("referrer", APIReferrer)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("client Do err :", err)
		return
	}

	status = res.Status
	databyte, _ := ioutil.ReadAll(res.Body)

	return status, string(databyte)
}

// RequestActiveSign 获取签到信息
func RequestActiveSign(openid string) (status string, data string) {
	req, err := http.NewRequest("GET", APIActiveSign, nil)
	if err != nil {
		log.Println("Create RequestActiveSign req err :", err)
		return
	}
	for k, v := range BaseHeader {
		req.Header[k] = v
	}
	req.Header.Add("Openid", openid)
	req.Header.Add("referrer", APIReferrer)
	req.Header.Add("If-None-Match", "38-djBNGTNDrEJXNs9DekumVQ")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println("client req err ", err)
		return
	}
	defer res.Body.Close()
	databyte, _ := ioutil.ReadAll(res.Body)
	status = res.Status
	data = string(databyte)
	return
}

//RequestSign 普通签到
func RequestSign(openid string ,courseid int, signid int)(status string,data string){

	//构建请求体
	bodydata := map[string]interface{}{
		"courseId": courseid,
        "signId":   signid,
		"lat": lat,
		"lon": lon,
	}
	// 转化为json格式数据
	jsondata, err := json.Marshal(bodydata)
	if err!= nil {
        log.Println("Marshal json err :", err)
        return
    }

	// 定制post请求
	req , err := http.NewRequest("POST", APISignIn, bytes.NewBuffer(jsondata))
	if err != nil {
		log.Println("Create RequestSign req err :", err)
        return
	}
	for k, v := range BaseHeader {
        req.Header[k] = v
    }
	req.Header.Add("Openid", openid)
	req.Header.Add("referrer", APIReferrer)

	client := &http.Client{

	}
	res ,err := client.Do(req)
	if err !=nil {
		log.Println("client req err ", err)
        return
	}
	status = res.Status
	databyte, _ := ioutil.ReadAll(res.Body)
	data = string(databyte)
	return	
	
}