package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sign/settings"
	"time"
)

// Tencent map geocode response subset
type tencentGeocodeResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Location struct {
			Lng float64 `json:"lng"`
			Lat float64 `json:"lat"`
		} `json:"location"`
		AddressComponents struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Street   string `json:"street"`
			StreetNo string `json:"street_number"`
		} `json:"address_components"`
		Title string `json:"title"`
	} `json:"result"`
}

// Geocode 调用腾讯地图接口获取坐标和规范化地址
func Geocode(address string) (lat float64, lon float64, formatted string, err error) {
	key := settings.Conf.MapConfig.Key
	if key == "" {
		err = fmt.Errorf("没有配置腾讯地图Key")
		return
	}
	u := fmt.Sprintf("https://apis.map.qq.com/ws/geocoder/v1/?address=%s&key=%s", url.QueryEscape(address), url.QueryEscape(key))
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		err = fmt.Errorf("请求地图服务失败：%w", err)
		return
	}
	defer resp.Body.Close()
	var gresp tencentGeocodeResp
	if err = json.NewDecoder(resp.Body).Decode(&gresp); err != nil {
		// 读取原始响应以便调试
		return lat, lon, "", fmt.Errorf("解析地图响应失败：%w", err)
	}
	if resp.StatusCode != 200 {
		return lat, lon, "", fmt.Errorf("地图服务 HTTP 错误：%d", resp.StatusCode)
	}
	if gresp.Status != 0 {
		// 返回更明确的错误信息并记录 URL，方便排查 key 或参数问题
		err = fmt.Errorf("地址解析失败: %s ，需要输入完整的有效的地址（可以从地图上复制）", gresp.Message)
		return
	}
	lat = gresp.Result.Location.Lat
	lon = gresp.Result.Location.Lng
	// 拼接完整地址展示
	formatted = fmt.Sprintf("%s %s %s %s %s", gresp.Result.Title, gresp.Result.AddressComponents.Province, gresp.Result.AddressComponents.City, gresp.Result.AddressComponents.District, gresp.Result.AddressComponents.Street+gresp.Result.AddressComponents.StreetNo)
	return
}