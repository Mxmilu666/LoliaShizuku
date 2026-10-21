//go:build darwin

package services

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit
#include <stdlib.h>
#import <AppKit/AppKit.h>

static char* desktopImagePath(void) {
	@autoreleasepool {
		NSScreen *screen = [NSScreen mainScreen];
		if (screen == nil) {
			return NULL;
		}
		NSURL *url = [[NSWorkspace sharedWorkspace] desktopImageURLForScreen:screen];
		if (url == nil || ![url isFileURL]) {
			return NULL;
		}
		return strdup([[url path] fileSystemRepresentation]);
	}
}
*/
import "C"

import (
	"context"
	"errors"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func wallpaperPath() (string, error) {
	// NSScreen / NSWorkspace 需要在主线程访问
	path := application.InvokeSyncWithResult(func() string {
		cPath := C.desktopImagePath()
		if cPath == nil {
			return ""
		}
		defer C.free(unsafe.Pointer(cPath))
		return C.GoString(cPath)
	})
	if path == "" {
		return "", errors.New("无法获取桌面壁纸路径")
	}
	return path, nil
}

// decodeWallpaper 借助系统自带的 sips 转为 PNG 缩略图，以支持 HEIC 等 Go 无法直接解码的格式。
func decodeWallpaper(path string) (image.Image, error) {
	tmpDir, err := os.MkdirTemp("", "loliashizuku-wallpaper-")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	output := filepath.Join(tmpDir, "wallpaper.png")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/sips", "-s", "format", "png", "-Z", "256", path, "--out", output)
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return decodeImageFile(output)
}

// macOS 的强调色直接使用 Wails 提供的实现
func platformAccentColor() string {
	return ""
}
