//go:build linux && !android

package services

import (
	"context"
	"errors"
	"image"
	neturl "net/url"
	"os/exec"
	"strings"
	"time"
)

// GNOME (libadwaita 1.6+) 强调色名称对应的颜色
var gnomeAccentColors = map[string]string{
	"blue":   "#3584e4",
	"teal":   "#2190a4",
	"green":  "#3a944a",
	"yellow": "#c88800",
	"orange": "#ed5b00",
	"red":    "#e62d42",
	"pink":   "#d56199",
	"purple": "#9141ac",
	"slate":  "#6f8396",
}

func gsettingsGet(schema, key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gsettings", "get", schema, key).Output()
	if err != nil {
		return ""
	}
	return strings.Trim(strings.TrimSpace(string(out)), "'")
}

// 目前支持 GNOME 系桌面；其他桌面环境返回错误，由前端回退到系统强调色
func wallpaperPath() (string, error) {
	key := "picture-uri"
	if strings.Contains(gsettingsGet("org.gnome.desktop.interface", "color-scheme"), "dark") {
		key = "picture-uri-dark"
	}
	uri := gsettingsGet("org.gnome.desktop.background", key)
	if uri == "" && key != "picture-uri" {
		uri = gsettingsGet("org.gnome.desktop.background", "picture-uri")
	}
	if uri == "" {
		return "", errors.New("无法获取桌面壁纸")
	}
	if !strings.HasPrefix(uri, "file://") {
		return uri, nil
	}
	parsed, err := neturl.Parse(uri)
	if err != nil {
		return "", err
	}
	return parsed.Path, nil
}

func decodeWallpaper(path string) (image.Image, error) {
	return decodeImageFile(path)
}

func platformAccentColor() string {
	return gnomeAccentColors[gsettingsGet("org.gnome.desktop.interface", "accent-color")]
}
