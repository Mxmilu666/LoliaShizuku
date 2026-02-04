package main

import (
	"embed"

	"loliashizuku/backend"
	"loliashizuku/backend/config"
	"loliashizuku/backend/services"

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
		Bind: []interface{}{
			app,
			prefSvc,
			tokenService,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
