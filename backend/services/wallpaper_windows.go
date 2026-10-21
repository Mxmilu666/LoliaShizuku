//go:build windows

package services

import (
	"errors"
	"image"
	"syscall"
	"unsafe"
)

const spiGetDeskWallpaper = 0x0073

var procSystemParametersInfoW = syscall.NewLazyDLL("user32.dll").NewProc("SystemParametersInfoW")

func wallpaperPath() (string, error) {
	buf := make([]uint16, syscall.MAX_PATH)
	ret, _, err := procSystemParametersInfoW.Call(
		spiGetDeskWallpaper,
		uintptr(len(buf)),
		uintptr(unsafe.Pointer(&buf[0])),
		0,
	)
	if ret == 0 {
		return "", err
	}
	path := syscall.UTF16ToString(buf)
	if path == "" {
		return "", errors.New("未设置桌面壁纸")
	}
	return path, nil
}

func decodeWallpaper(path string) (image.Image, error) {
	return decodeImageFile(path)
}

// Windows 的强调色直接使用 Wails 提供的实现
func platformAccentColor() string {
	return ""
}
