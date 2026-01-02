package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"sign/settings"
)

// Mount 管理后台挂载任务
type Mount struct {
	OpenID string
	Name   string
	Email  string
	Lat    float64
	Lon    float64
	stop   chan struct{}
	// 订阅状态，防止重复订阅相同的二维码签到
	subscribedCourseID int
	subscribedSignID   int
	subscribeRunning   bool
}

type MountManager struct {
	mu     sync.Mutex
	mounts map[string]*Mount // keyed by openid
	persistFile string
}

func NewMountManager() *MountManager {
	return &MountManager{mounts: make(map[string]*Mount), persistFile: "mounts.json"}
}

var mountMgr = NewMountManager()

// MaskName 只显示姓，后面打码
func MaskName(name string) string {
	if name == "" {
		return ""
	}
	r := []rune(name)
	if len(r) <= 1 {
		return name
	}
	return string(r[0]) + strings.Repeat("*", len(r)-1)
}

func (m *MountManager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := []string{}
	for _, mm := range m.mounts {
		res = append(res, MaskName(mm.Name))
	}
	return res
}

// ActiveDetails 返回更详细的挂载信息（供管理使用）
func (m *MountManager) ActiveDetails() []map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := []map[string]interface{}{}
	for k, mm := range m.mounts {
		res = append(res, map[string]interface{}{
			"openid": k,
			"name": MaskName(mm.Name),
			"email": mm.Email,
			"lat": mm.Lat,
			"lon": mm.Lon,
		})
	}
	return res
}

// Start 挂载，如果已有则返回错误
func (m *MountManager) Start(openid, email, latlon string) error {
	m.mu.Lock()
	if _, ok := m.mounts[openid]; ok {
		m.mu.Unlock()
		return fmt.Errorf("已存在挂载任务")
	}
	m.mu.Unlock()

	student, err := getstudentInfo(openid)
	if err != nil {
		return err
	}
	name := student.Name
	// 检查是否在允许名单中
	allow := false
	for _, u := range Mount_Allow_users {
		if u == name {
			allow = true
			break
		}
	}
	if !allow && !Test {
		return fmt.Errorf("该用户没有挂载权限，请联系管理员")
	}

	lat := 0.0
	lon := 0.0
	if latlon != "" {
		parts := strings.Split(latlon, ",")
		if len(parts) == 2 {
			lat, _ = strconv.ParseFloat(parts[0], 64)
			lon, _ = strconv.ParseFloat(parts[1], 64)
		}
	}

	mount := &Mount{OpenID: openid, Name: name, Email: email, Lat: lat, Lon: lon, stop: make(chan struct{})}

	m.mu.Lock()
	m.mounts[openid] = mount
	// 持久化当前挂载列表
	m.saveLocked()
	m.mu.Unlock()

	// 启动后台 goroutine
	go func(mm *Mount) {
		log.Println("挂载任务开始：", mm.Name)
		if mm.Email != "" {
			SendEmail(mm.Email, fmt.Sprintf("%s 已开启挂载任务。", mm.Name))
		} else {
			SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("%s 已开启挂载任务（未提供用户邮箱）。", mm.Name))
		}
		for {
			select {
			case <-mm.stop:
				log.Println("挂载任务停止：", mm.Name)
				if mm.Email != "" {
					SendEmail(mm.Email, fmt.Sprintf("%s 的挂载任务已停止。", mm.Name))
				} else {
					SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("%s 的挂载任务已停止（未提供用户邮箱）。", mm.Name))
				}
				return
			default:
				// 轮询检查活动签到
				activesign, err := getActiveSign(mm.OpenID)
				if err != nil {
					// openid 失效或其他异常
					if mm.Email != "" {
						SendEmail(mm.Email, fmt.Sprintf("挂载异常：%s", err.Error()))
					} else {
						SendEmail(settings.Conf.Email.UserName, fmt.Sprintf("挂载异常：%s (用户未提供邮箱)", err.Error()))
					}
					// 终止挂载任务
					m.Stop(mm.OpenID)
					return
				}
				if len(activesign) != 0 {
					Signing := activesign[0]
					// 尽量异步发送通知，避免阻塞
					if mm.Email != "" {
						SendEmailAsync(mm.Email, fmt.Sprintf("%s 正在签到：%s", mm.Name, Signing.Name))
					} else {
						SendEmailAsync(settings.Conf.Email.UserName, fmt.Sprintf("%s 正在签到：%s (未提供用户邮箱)", mm.Name, Signing.Name))
					}
					if Signing.IsQR == 0 {
						// 普通签到
						time.Sleep(10 * time.Second)
						signres, err := GetCommonSignRes(mm.OpenID, Signing.CourseID, Signing.SignID, mm.Lat, mm.Lon)
						if err != nil {
							if mm.Email != "" {
								SendEmailAsync(mm.Email, fmt.Sprintf("签到失败：%s", err.Error()))
							} else {
								SendEmailAsync(settings.Conf.Email.UserName, fmt.Sprintf("签到失败：%s (用户未提供邮箱)", err.Error()))
							}
						} else {
							if signres.MsgClient != "" {
								SendEmailAsync(mm.Email, signres.MsgClient)
							} else {
								// 成功后发通知，但不停止挂载
								SendEmailAsync(mm.Email, fmt.Sprintf("签到成功，你是第%d个签到！", signres.SignRank))
								// 额外通知管理员备份一份
								SendEmailAsync(settings.Conf.Email.UserName, fmt.Sprintf("%s（%s）签到成功：课程 %s", mm.Name, mm.OpenID, Signing.Name))
							}
						}
					} else {
					// 二维码签到：避免重复订阅相同课程/签到
					m.mu.Lock()
					already := mm.subscribeRunning && mm.subscribedCourseID == Signing.CourseID && mm.subscribedSignID == Signing.SignID
					if !already {
						// 标记为正在订阅并持久化
						mm.subscribedCourseID = Signing.CourseID
						mm.subscribedSignID = Signing.SignID
						mm.subscribeRunning = true
						_ = m.saveLocked()
						m.mu.Unlock()

						// 使用目标邮箱（若用户未提供则使用管理员邮箱）
						var targetEmail = mm.Email
						if targetEmail == "" {
							targetEmail = settings.Conf.Email.UserName
						}
						SendEmailAsync(targetEmail, fmt.Sprintf("检测到二维码签到，开始订阅二维码（课程：%s）", Signing.Name))
						logger.Println("开始订阅二维码：", mm.Name, Signing.CourseID, Signing.SignID)

						// 异步调用 Qrsign，回调中使用异步发送邮件，避免阻塞 mount 主循环
						go func(openid, email string, courseID, signID int) {
							logger.Println("Qrsign goroutine 启动：", openid, courseID, signID)
							err := Qrsign(nil, courseID, signID, func(qr string) {
								// 尽快异步发送二维码链接
								SendEmailAsync(email, fmt.Sprintf("%s", qr))
							})
							logger.Println("Qrsign goroutine 结束：", openid, courseID, signID, "err:", err)
							// 结束后回写状态并持久化
							m.mu.Lock()
							if mm2, ok := m.mounts[openid]; ok {
								mm2.subscribeRunning = false
								mm2.subscribedCourseID = 0
								mm2.subscribedSignID = 0
							}
							_ = m.saveLocked()
							m.mu.Unlock()

							if err != nil {
								SendEmailAsync(email, fmt.Sprintf("订阅二维码失败：%s", err.Error()))
							} else {
								SendEmailAsync(email, "二维码订阅已结束")
							}
						}(mm.OpenID, targetEmail, Signing.CourseID, Signing.SignID)
					} else {
						m.mu.Unlock()
					}
					}
				} else {
					// 没有签到，继续监听
				}
				// 轮询间隔
				time.Sleep(10 * time.Second)
			}
		}
	}(mount)

	return nil
}

func (m *MountManager) Stop(openid string) error {
	m.mu.Lock()
	if mm, ok := m.mounts[openid]; ok {
		close(mm.stop)
		delete(m.mounts, openid)
		// 持久化
		if err := m.saveLocked(); err != nil {
			m.mu.Unlock()
			return err
		}
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	return fmt.Errorf("未找到挂载任务")
}

func (m *MountManager) saveLocked() error {
	// m.mu must be held by caller
	p := []map[string]interface{}{}
	for _, mm := range m.mounts {
		p = append(p, map[string]interface{}{
			"openid": mm.OpenID,
			"email": mm.Email,
			"lat": mm.Lat,
			"lon": mm.Lon,
		})
	}
	// 写入临时文件然后重命名
	f, err := os.CreateTemp(".", "mounts-*.tmp")
	if err != nil {
		return err
	}
	d := json.NewEncoder(f)
	d.SetIndent("", "  ")
	if err := d.Encode(p); err != nil {
		f.Close()
		return err
	}
	f.Close()
	if err := os.Rename(f.Name(), m.persistFile); err != nil {
		return err
	}
	return nil
}

func (m *MountManager) Restore() error {
	// 从持久化文件加载并尝试恢复
	if _, err := os.Stat(m.persistFile); os.IsNotExist(err) {
		return nil
	}
	b, err := os.ReadFile(m.persistFile)
	if err != nil {
		return err
	}
	var p []struct{
		OpenID string `json:"openid"`
		Email string `json:"email"`
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	for _, item := range p {
		// 尝试恢复（Start 会自行检查重复和权限）
		if err := m.Start(item.OpenID, item.Email, fmt.Sprintf("%f,%f", item.Lat, item.Lon)); err != nil {
			log.Println("恢复挂载失败：", item.OpenID, err)
		}
	}
	return nil
}