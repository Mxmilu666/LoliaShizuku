package services

import "github.com/Mxmilu666/LoliaShizuku/backend/config"

// PreferencesService exposes persisted app preferences to the frontend.
type PreferencesService struct {
	configManager *config.Manager
}

// NewPreferencesService creates a new PreferencesService.
func NewPreferencesService(configManager *config.Manager) *PreferencesService {
	return &PreferencesService{configManager: configManager}
}

// GetWindowSize returns the stored window size and maximised state.
func (s *PreferencesService) GetWindowSize() (int, int, bool) {
	return s.configManager.GetWindowSize()
}

// SaveWindowSize persists the window size.
func (s *PreferencesService) SaveWindowSize(width, height int) error {
	return s.configManager.UpdateWindowSize(width, height)
}

// SaveWindowMaximised persists the window maximised state.
func (s *PreferencesService) SaveWindowMaximised(maximised bool) error {
	return s.configManager.UpdateWindowMaximised(maximised)
}
