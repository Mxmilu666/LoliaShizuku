package services

import (
	"sync"
	"time"

	"github.com/Mxmilu666/LoliaShizuku/backend/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type systemService struct {
	window   *application.WebviewWindow
	loopOnce sync.Once

	configManager *config.Manager
}

var system *systemService
var onceSystem sync.Once

func System() *systemService {
	if system == nil {
		onceSystem.Do(func() {
			system = &systemService{}
		})
	}
	return system
}

func (s *systemService) Start(window *application.WebviewWindow, configManager *config.Manager) {
	s.window = window
	s.configManager = configManager
	s.loopOnce.Do(func() {
		go s.loopWindowEvent()
	})
}

func (s *systemService) loopWindowEvent() {
	var fullscreen, maximised, minimised, normal bool
	var width, height int
	var dirty bool

	for {
		time.Sleep(300 * time.Millisecond)
		if s.window == nil {
			continue
		}

		dirty = false

		// Check fullscreen state
		if f := s.window.IsFullscreen(); f != fullscreen {
			fullscreen = f
			dirty = true
		}

		// Check window size
		if w, h := s.window.Size(); w != width || h != height {
			width, height = w, h
			dirty = true
			s.saveWindowSize(width, height)
		}

		// Check maximised state
		if m := s.window.IsMaximised(); m != maximised {
			maximised = m
			dirty = true
			s.saveWindowMaximised(maximised)
		}

		// Check minimised state
		if m := s.window.IsMinimised(); m != minimised {
			minimised = m
			dirty = true
		}

		// Check normal state (v3 has no IsNormal; derive it from the other states)
		if n := !fullscreen && !maximised && !minimised; n != normal {
			normal = n
			dirty = true
		}

		// Emit event if any state changed
		if dirty {
			s.Emit("window_changed", map[string]interface{}{
				"fullscreen": fullscreen,
				"width":      width,
				"height":     height,
				"maximised":  maximised,
				"minimised":  minimised,
				"normal":     normal,
			})
		}
	}
}

// Emit sends a custom event to the frontend. It is a no-op before the
// application has been created.
func (s *systemService) Emit(name string, data any) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit(name, data)
}

func (s *systemService) saveWindowSize(width, height int) {
	if s.configManager == nil {
		return
	}
	_ = s.configManager.UpdateWindowSize(width, height)
}

func (s *systemService) saveWindowMaximised(maximised bool) {
	if s.configManager == nil {
		return
	}
	_ = s.configManager.UpdateWindowMaximised(maximised)
}
