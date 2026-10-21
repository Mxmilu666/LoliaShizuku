package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	tokenService  = "LoliaShizuku"
	oauthTokenKey = "oauth_token"
)

type storedOAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

func toStoredOAuthToken(token *oauth2.Token) *storedOAuthToken {
	if token == nil {
		return nil
	}

	return &storedOAuthToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

func fromStoredOAuthToken(token *storedOAuthToken) *oauth2.Token {
	if token == nil {
		return nil
	}

	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
}

// SaveOAuthToken stores an OAuth token in the OS keyring.
func SaveOAuthToken(token *oauth2.Token) error {
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return fmt.Errorf("oauth token is empty")
	}

	stored := toStoredOAuthToken(token)
	payload, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshal oauth token: %w", err)
	}

	if err := secretSet(tokenService, oauthTokenKey, string(payload)); err != nil {
		return fmt.Errorf("save oauth token to secure storage: %w", err)
	}
	return nil
}

// LoadOAuthToken loads the OAuth token from the OS keyring.
func LoadOAuthToken() (*oauth2.Token, error) {
	raw, err := secretGet(tokenService, oauthTokenKey)
	if err != nil {
		return nil, err
	}

	var stored storedOAuthToken
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		legacyAccessToken := strings.TrimSpace(raw)
		if legacyAccessToken == "" {
			return nil, errSecretNotFound
		}
		// Backward compatibility with old plain-text token value.
		return &oauth2.Token{
			AccessToken: legacyAccessToken,
			TokenType:   "Bearer",
		}, nil
	}

	if strings.TrimSpace(stored.AccessToken) == "" {
		return nil, fmt.Errorf("oauth token access_token is empty")
	}

	return fromStoredOAuthToken(&stored), nil
}

// ClearOAuthToken removes OAuth token from the OS keyring.
func ClearOAuthToken() error {
	err := secretDelete(tokenService, oauthTokenKey)
	if err == nil || errors.Is(err, errSecretNotFound) {
		return nil
	}
	return fmt.Errorf("clear oauth token from secure storage: %w", err)
}

// HasOAuthToken returns true when a token exists in the OS keyring.
func HasOAuthToken() (bool, error) {
	token, err := LoadOAuthToken()
	if err == nil {
		return token != nil && strings.TrimSpace(token.AccessToken) != "", nil
	}
	if errors.Is(err, errSecretNotFound) {
		return false, nil
	}
	return false, err
}
