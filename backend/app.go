package backend

import (
	"context"
	"fmt"

	"loliashizuku/backend/config"
	"loliashizuku/backend/services"
)

// App struct
type App struct {
	ctx           context.Context
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

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration if not already done
	if !a.configManager.IsInitialized() {
		if err := a.configManager.Initialize(); err != nil {
			fmt.Printf("Failed to initialize config: %v\n", err)
		}
	}

	// Start system service with context and config manager
	services.System().Start(ctx, a.configManager)
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
