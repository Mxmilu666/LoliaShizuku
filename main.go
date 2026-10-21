package main

import (
	"embed"
	"log"

	"github.com/Mxmilu666/LoliaShizuku/backend"
	"github.com/Mxmilu666/LoliaShizuku/backend/config"
	"github.com/Mxmilu666/LoliaShizuku/backend/services"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	minWindowWidth  = 800
	minWindowHeight = 600
)

func main() {
	prepareEnvironment()

	// Initialize configuration early to restore window size
	configManager := config.NewManager()
	if err := configManager.Initialize(); err != nil {
		println("Failed to initialize config:", err.Error())
	}

	prefSvc := services.NewPreferencesService(configManager)

	width, height, maximised := prefSvc.GetWindowSize()
	windowStartState := application.WindowStateNormal
	if maximised {
		windowStartState = application.WindowStateMaximised
	}

	// Create an instance of the app structure
	app := backend.NewApp(configManager)
	tokenService := services.NewTokenService()
	centerService := services.NewCenterService()
	frpcService := services.NewFrpcService()
	systemColorService := services.NewSystemColorService()
	browserService := services.NewBrowserService()

	// Create application with options
	wailsApp := application.New(application.Options{
		Name:        "LoliaShizuku",
		Description: "LoliaShizuku desktop client for Lolia FRP",
		Services: []application.Service{
			application.NewService(app),
			application.NewService(prefSvc),
			application.NewService(tokenService),
			application.NewService(centerService),
			application.NewService(frpcService),
			application.NewService(systemColorService),
			application.NewService(browserService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			_, _ = centerService.StopRunner()
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:      "LoliaShizuku",
		Width:      width,
		Height:     height,
		MinWidth:   minWindowWidth,
		MinHeight:  minWindowHeight,
		StartState: windowStartState,
		Frameless:  true,
		URL:        "/",
	})

	startWindowStateSync(window, configManager)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
