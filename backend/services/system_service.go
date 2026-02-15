package services

import (
	"context"
	"sync"
	"time"

	"github.com/Mxmilu666/LoliaShizuku/backend/config"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type systemService struct {
	ctx      context.Context
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

func (s *systemService) Start(ctx context.Context, configManager *config.Manager) {
	s.ctx = ctx
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
		if s.ctx == nil {
			continue
		}

		dirty = false

		// Check fullscreen state
		if f := runtime.WindowIsFullscreen(s.ctx); f != fullscreen {
			fullscreen = f
			dirty = true
		}

		// Check window size
		if w, h := runtime.WindowGetSize(s.ctx); w != width || h != height {
			width, height = w, h
			dirty = true
			s.saveWindowSize(width, height)
		}

		// Check maximised state
		if m := runtime.WindowIsMaximised(s.ctx); m != maximised {
			maximised = m
			dirty = true
			s.saveWindowMaximised(maximised)
		}

		// Check minimised state
		if m := runtime.WindowIsMinimised(s.ctx); m != minimised {
			minimised = m
			dirty = true
		}

		// Check normal state
		if n := runtime.WindowIsNormal(s.ctx); n != normal {
			normal = n
			dirty = true
		}

		// Emit event if any state changed
		if dirty {
			runtime.EventsEmit(s.ctx, "window_changed", map[string]interface{}{
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
