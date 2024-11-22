package main

import (
	"fmt"
	"image"
	"os"
	"sign/settings"

	"github.com/gin-gonic/gin"
	"github.com/liyue201/goqr"
)

func main() {
	//初始化配置
	if err:=settings.Init();err!=nil {
		fmt.Println("初始化配置失败！", err)
        return
    }

	//配置路由
	r:=gin.Default()
	r.GET("/wzjsign",indexHandler)



	fmt.Println("个人信息：")
	fmt.Println(RequestStudentRole(Openid))
	// fmt.Println(RequestStudentinfo(Openid))
	fmt.Println(
		"正在进行的签到：")
	fmt.Println(RequestActiveSign(Openid))
	// RequestSign(Openid,"1388301","3401023")
	fmt.Println("签到结果：")
	fmt.Println(RequestSign(Openid,1388301,3401023))

}

func ScanQrByfile(file string) (qr_content string, err error) {
	f, err := os.Open(file)
	if err != nil {
		fmt.Println("读取二维码文件失败！", err)
		return
	}
	defer f.Close()
	img, _, _ := image.Decode(f)

	qrCodes, err := goqr.Recognize(img)
	if err != nil {
		fmt.Printf("Recognize failed: %v\n", err)
		return
	}
	if len(qrCodes) <= 0 {
		fmt.Printf("识别失败")
		return
	}
	fmt.Println(string(qrCodes[0].Payload))
	qr_content = string(qrCodes[0].Payload)
	return
}

//