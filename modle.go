package main

type ActiveSign struct {
	CourseID int    `json:"courseId"`
	SignID   int    `json:"signId"`
	IsGPS    int    `json:"isGPS"`
	IsQR     int    `json:"isQR"`
	Name     string `json:"name"`
}


//签到结果 : 签到关闭：{"errorCode":301,"msg":"no sign course now","msgClient":"当前签到已关闭"}。成功{"signRank":14,"studentRank":1}
type SignResult struct {
	SignRank int `json:"signRank"`
	MsgClient string `json:"msgClient"`
}