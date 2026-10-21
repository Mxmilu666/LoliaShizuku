package services

import (
	"context"
	"errors"
)

// TokenService exposes token helpers to the frontend via Wails binding.
type TokenService struct{}

// NewTokenService creates a new TokenService instance.
func NewTokenService() *TokenService {
	return &TokenService{}
}

// HasOAuthToken checks whether an OAuth token exists in the system keyring.
func (s *TokenService) HasOAuthToken() (bool, error) {
	ctx := context.Background()
	token, err := loadOrRefreshOAuthToken(ctx)
	if err == nil {
		return token != nil, nil
	}
	if errors.Is(err, errSecretNotFound) {
		return false, nil
	}
	return false, err
}

// BeginOAuthLogin starts OAuth2 Authorization Code login and stores token in keyring.
func (s *TokenService) BeginOAuthLogin() (bool, error) {
	if err := beginOAuthLogin(); err != nil {
		return false, err
	}
	return true, nil
}

// ClearOAuthToken removes OAuth token from keyring.
func (s *TokenService) ClearOAuthToken() error {
	return ClearOAuthToken()
}
