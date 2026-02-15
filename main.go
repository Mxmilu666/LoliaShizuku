package main

import (
	"context"
	"embed"

	"github.com/Mxmilu666/LoliaShizuku/backend"
	"github.com/Mxmilu666/LoliaShizuku/backend/config"
	"github.com/Mxmilu666/LoliaShizuku/backend/services"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	minWindowWidth  = 800
	minWindowHeight = 600
)

func main() {
	// Initialize configuration early to restore window size
	configManager := config.NewManager()
	if err := configManager.Initialize(); err != nil {
		println("Failed to initialize config:", err.Error())
	}

	prefSvc := services.NewPreferencesService(configManager)

	width, height, maximised := prefSvc.GetWindowSize()
	windowStartState := options.Normal
	if maximised {
		windowStartState = options.Maximised
	}

	// Create an instance of the app structure
	app := backend.NewApp(configManager)
	tokenService := services.NewTokenService()
	centerService := services.NewCenterService()
	frpcService := services.NewFrpcService()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "LoliaShizuku",
		Width:            width,
		Height:           height,
		MinWidth:         minWindowWidth,
		MinHeight:        minWindowHeight,
		WindowStartState: windowStartState,
		Frameless:        true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		OnBeforeClose: func(ctx context.Context) bool {
			_, _ = centerService.StopRunner()
			return false
		},
		OnShutdown: func(ctx context.Context) {
			_, _ = centerService.StopRunner()
		},
		Bind: []interface{}{
			app,
			prefSvc,
			tokenService,
			centerService,
			frpcService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
