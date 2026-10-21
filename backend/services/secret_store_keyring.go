//go:build !android

package services

import "github.com/zalando/go-keyring"

// 桌面端使用系统钥匙串（macOS Keychain / Windows Credential Manager / Secret Service）

var errSecretNotFound = keyring.ErrNotFound

func secretGet(service, key string) (string, error) {
	return keyring.Get(service, key)
}

func secretSet(service, key, value string) error {
	return keyring.Set(service, key, value)
}

func secretDelete(service, key string) error {
	return keyring.Delete(service, key)
}
