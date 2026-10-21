//go:build android

package services

import "github.com/wailsapp/wails/v3/pkg/application"

// Android 通过 Intent.ACTION_VIEW 打开系统浏览器
func openExternalURL(url string) error {
	application.Android.OpenURL(url)
	return nil
}
