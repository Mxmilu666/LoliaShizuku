//go:build android

package main

import (
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Mxmilu666/LoliaShizuku/backend/config"
)

// prepareEnvironment 让 os.UserConfigDir 指向应用私有目录（Android 上 HOME / XDG 变量通常未设置）
func prepareEnvironment() {
	if dir := strings.TrimSpace(application.Android.StoragePath()); dir != "" {
		_ = os.Setenv("XDG_CONFIG_HOME", dir)
	}
}

// 移动端没有窗口尺寸/最大化等桌面窗口状态，无需轮询保存
func startWindowStateSync(_ *application.WebviewWindow, _ *config.Manager) {}
