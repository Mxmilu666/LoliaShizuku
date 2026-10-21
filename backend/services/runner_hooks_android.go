//go:build android

package services

import (
	"fmt"
	"os"
	"path/filepath"

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

// runnerExtraEnv 清理旧版本以 frpc -c 启动时写入的配置文件（含隧道 token），现已统一使用 frpc -t。
func runnerExtraEnv(binaryPath string) []string {
	_ = os.RemoveAll(filepath.Join(filepath.Dir(filepath.Dir(binaryPath)), "run"))
	return nil
}
