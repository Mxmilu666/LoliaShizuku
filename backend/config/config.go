package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 表示应用程序配置
type Config struct {
	Version  string         `json:"version"`  // 配置版本号
	App      AppConfig      `json:"app"`      // 应用程序相关设置
	Theme    ThemeConfig    `json:"theme"`    // 主题相关设置
	Window   WindowConfig   `json:"window"`   // 窗口相关设置
	Advanced AdvancedConfig `json:"advanced"` // 高级设置
}

// AppConfig 包含应用程序特定的设置
type AppConfig struct {
	AutoStart bool `json:"autoStart"` // 是否自动启动
}

// ThemeConfig 包含主题设置
type ThemeConfig struct {
	Mode        string `json:"mode"`        // 主题模式：light, dark, auto
	AccentColor string `json:"accentColor"` // 强调颜色
}

// WindowConfig 包含窗口尺寸设置
type WindowConfig struct {
	Width     int  `json:"width"`     // 窗口宽度
	Height    int  `json:"height"`    // 窗口高度
	Maximised bool `json:"maximised"` // 是否最大化
}

// AdvancedConfig 包含高级设置
type AdvancedConfig struct {
	LogLevel  string `json:"logLevel"`  // 日志级别
	DebugMode bool   `json:"debugMode"` // 是否启用调试模式
}

// Manager 处理配置操作
type Manager struct {
	configPath string
	config     *Config
}

// NewManager 创建一个新的配置管理器
func NewManager() *Manager {
	return &Manager{
		config: getDefaultConfig(),
	}
}

// getDefaultConfig 返回默认配置
func getDefaultConfig() *Config {
	return &Config{
		Version: "0.0.1",
		App: AppConfig{
			AutoStart: false,
		},
		Theme: ThemeConfig{
			Mode:        "auto",
			AccentColor: "#6200EE",
		},
		Window: WindowConfig{
			Width:     950,
			Height:    600,
			Maximised: false,
		},
		Advanced: AdvancedConfig{
			LogLevel:  "info",
			DebugMode: false,
		},
	}
}

// Initialize 设置配置管理器并加载配置
func (m *Manager) Initialize() error {
	// 获取配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("无法获取配置目录: %w", err)
	}

	// 创建应用程序特定的配置目录
	appConfigDir := filepath.Join(configDir, "LoliaShizuku")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return fmt.Errorf("无法创建配置目录: %w", err)
	}

	m.configPath = filepath.Join(appConfigDir, "config.json")

	// 如果配置文件存在则加载，否则创建默认配置
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		if err := m.Save(); err != nil {
			return fmt.Errorf("无法创建默认配置: %w", err)
		}
	} else {
		// 加载现有配置
		if err := m.Load(); err != nil {
			return fmt.Errorf("无法加载配置: %w", err)
		}
	}

	return nil
}

// Load 从文件中读取配置
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("无法读取配置文件: %w", err)
	}

	config := getDefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("无法解析配置文件: %w", err)
	}

	m.config = config
	return nil
}

// Save 将配置写入文件
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("无法序列化配置: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("无法写入配置文件: %w", err)
	}

	return nil
}

// IsInitialized 判断配置管理器是否已初始化
func (m *Manager) IsInitialized() bool {
	return m.configPath != ""
}

// GetConfig 返回当前配置
func (m *Manager) GetConfig() *Config {
	return m.config
}

// GetConfigJSON 以 JSON 字符串形式返回配置
func (m *Manager) GetConfigJSON() (string, error) {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("无法将配置序列化为 JSON: %w", err)
	}
	return string(data), nil
}

// UpdateConfig 从 JSON 字符串更新配置
func (m *Manager) UpdateConfig(jsonStr string) error {
	config := getDefaultConfig()
	if err := json.Unmarshal([]byte(jsonStr), config); err != nil {
		return fmt.Errorf("无法解析配置 JSON: %w", err)
	}

	m.config = config
	return m.Save()
}

// UpdateWindowSize 更新窗口尺寸配置
func (m *Manager) UpdateWindowSize(width, height int) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	if m.configPath == "" {
		return nil
	}
	if m.config == nil {
		m.config = getDefaultConfig()
	}
	if m.config.Window.Width == width && m.config.Window.Height == height {
		return nil
	}

	m.config.Window.Width = width
	m.config.Window.Height = height
	return m.Save()
}

// UpdateWindowMaximised updates the window maximised state.
func (m *Manager) UpdateWindowMaximised(maximised bool) error {
	if m.configPath == "" {
		return nil
	}
	if m.config == nil {
		m.config = getDefaultConfig()
	}
	if m.config.Window.Maximised == maximised {
		return nil
	}

	m.config.Window.Maximised = maximised
	return m.Save()
}

// GetWindowSize returns the window size and maximised state.
func (m *Manager) GetWindowSize() (int, int, bool) {
	if m.config == nil {
		m.config = getDefaultConfig()
	}
	return m.config.Window.Width, m.config.Window.Height, m.config.Window.Maximised
}

// GetConfigPath 返回配置文件路径
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// ResetToDefaults 重置配置为默认值
func (m *Manager) ResetToDefaults() error {
	m.config = getDefaultConfig()
	return m.Save()
}
