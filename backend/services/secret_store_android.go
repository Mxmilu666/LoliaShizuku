//go:build android

package services

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Android 使用 Wails 提供的安全存储（基于 Android Keystore）

var errSecretNotFound = errors.New("secret not found")

func secretStorageKey(service, key string) string {
	return service + "." + key
}

func secretGet(service, key string) (string, error) {
	value, ok, err := application.Android.SecureGet(secretStorageKey(service, key))
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errSecretNotFound
	}
	return value, nil
}

func secretSet(service, key, value string) error {
	return application.Android.SecureSet(secretStorageKey(service, key), value)
}

func secretDelete(service, key string) error {
	return application.Android.SecureDelete(secretStorageKey(service, key))
}
