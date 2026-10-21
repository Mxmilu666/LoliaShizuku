package backend

import (
	"context"
	"fmt"

	"github.com/Mxmilu666/LoliaShizuku/backend/config"
	"github.com/Mxmilu666/LoliaShizuku/backend/services"
	"github.com/Mxmilu666/LoliaShizuku/backend/version"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App struct
type App struct {
	configManager *config.Manager
}

// NewApp creates a new App application struct
func NewApp(configManager *config.Manager) *App {
	// Initialize system service instance
	services.System()

	// Initialize config manager
	if configManager == nil {
		configManager = config.NewManager()
	}

	return &App{
		configManager: configManager,
	}
}

// ServiceStartup is called by Wails when the application starts.
func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	// Initialize configuration if not already done
	if !a.configManager.IsInitialized() {
		if err := a.configManager.Initialize(); err != nil {
			fmt.Printf("Failed to initialize config: %v\n", err)
		}
	}

	return nil
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetConfigJSON returns the configuration as JSON string
func (a *App) GetConfigJSON() (string, error) {
	return a.configManager.GetConfigJSON()
}

// UpdateConfig updates the configuration from JSON string
func (a *App) UpdateConfig(jsonStr string) error {
	return a.configManager.UpdateConfig(jsonStr)
}

// GetConfigPath returns the configuration file path
func (a *App) GetConfigPath() string {
	return a.configManager.GetConfigPath()
}

// ResetConfig resets configuration to default values
func (a *App) ResetConfig() error {
	return a.configManager.ResetToDefaults()
}

// GetVersionInfo returns build/version metadata for the About page
func (a *App) GetVersionInfo() version.Info {
	return version.GetInfo()
}
