package services

import (
	"fmt"
	"net/url"
	"strings"
)

// BrowserService 在系统浏览器中打开外部链接。
// 前端 @wailsio/runtime 的 Browser.OpenURL 在 Android 上不可用（Wails 内部调用 xdg-open），
// 因此统一经由此服务打开：Android 使用 Intent，桌面端使用系统默认浏览器。
type BrowserService struct{}

func NewBrowserService() *BrowserService {
	return &BrowserService{}
}

// OpenURL 打开 http(s) 链接。
func (s *BrowserService) OpenURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("无效的链接: %s", rawURL)
	}
	return openExternalURL(parsed.String())
}
