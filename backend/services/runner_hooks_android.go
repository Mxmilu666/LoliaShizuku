//go:build android

package services

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// 前台服务让 Android 在应用切到后台后仍保持进程（及其 frpc 子进程）存活
func onRunnerStarted(tunnelCount int) {
	application.Android.StartForegroundService(fmt.Sprintf(
		`{"title":"LoliaShizuku","text":"正在运行 %d 条隧道"}`, tunnelCount,
	))
}

func onRunnerStopped() {
	application.Android.StopForegroundService()
}
