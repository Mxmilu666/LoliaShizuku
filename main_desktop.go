//go:build !android

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Mxmilu666/LoliaShizuku/backend/config"
	"github.com/Mxmilu666/LoliaShizuku/backend/services"
)

func prepareEnvironment() {}

// 轮询窗口状态：保存窗口尺寸 / 最大化状态，并通知前端自绘标题栏
func startWindowStateSync(window *application.WebviewWindow, configManager *config.Manager) {
	services.System().Start(window, configManager)
}
