//go:build !darwin && !windows && (!linux || android)

package services

import (
	"errors"
	"image"
)

func wallpaperPath() (string, error) {
	return "", errors.New("当前平台不支持读取桌面壁纸")
}

func decodeWallpaper(path string) (image.Image, error) {
	return decodeImageFile(path)
}

func platformAccentColor() string {
	return ""
}
