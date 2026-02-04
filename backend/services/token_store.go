package services

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	tokenService  = "LoliaShizuku"
	oauthTokenKey = "oauth_token"
)

// HasOAuthToken returns true when a token exists in the OS keyring.
func HasOAuthToken() (bool, error) {
	_, err := keyring.Get(tokenService, oauthTokenKey)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return false, nil
	}
	return false, err
}
