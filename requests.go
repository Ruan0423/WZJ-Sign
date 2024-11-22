package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var BaseHeader= map[string][]string {
	"User-Agent": {"Mozilla/5.0 (Linux; Android 5.0; SM-N9100 Build/LRX21V) > AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 > Chrome/37.0.0.0 Mobile Safari/537.36 > MicroMessenger/6.0.2.56_r958800.520 NetType/WIFI"},
    "Content-Type": {"application/json"},
    "Accept": {"*/*"},
    "Accept-Language": {"zh-CN,en-US;qbaseHeaders=0.7,en;q=0.3"},
}

const (
	OpenidURL = "https://v18.teachermate.cn/wechat-pro-ssr/?openid=1afa187e0cb54ddb7c7db49ad859f97f&from=wzj"
	Openid = "1afa187e0cb54ddb7c7db49ad859f97f"

)

var (
	APIReferrer = "https://v18.teachermate.cn/wechat-pro/student/edit?openid="+Openid
	APIStudentinfo = "https://v18.teachermate.cn/wechat-api/v2/students"
	APIStudentRole = "https://v18.teachermate.cn/wechat-api/v2/students/role"
	APIActiveSign = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student/active_signs"

	APISignIn = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student-sign-in"
)

//获取个人信息请求
func RequestStudentRole(openid string) {
	//新建请求
	req,err := http.NewRequest("GET", APIStudentRole, nil)
	if err!=nil {
		log.Println("创建RequestsStudentRole失败：",err)
		return
	}
	//添加header
	for k,v :=range BaseHeader {
		req.Header[k] = v
	}
	req.Header.Add("Openid",openid)
	req.Header.Add("referrer",APIReferrer)

	//创建http客户端
	cilent :=&http.Client{

	}
	
	res,err:=cilent.Do(req)
	if err!=nil{
			fmt.Println("qignqiuerr:",err)
	}
	data,_ := ioutil.ReadAll(res.Body)
	fmt.Println(string(data))
	defer res.Body.Close()	

	
}