//go:build !android

package services

import "github.com/pkg/browser"

func openExternalURL(url string) error {
	return browser.OpenURL(url)
}
