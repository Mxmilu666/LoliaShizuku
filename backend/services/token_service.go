package services

// TokenService exposes token helpers to the frontend via Wails binding.
type TokenService struct{}

// NewTokenService creates a new TokenService instance.
func NewTokenService() *TokenService {
	return &TokenService{}
}

// HasOAuthToken checks whether an OAuth token exists in the system keyring.
func (s *TokenService) HasOAuthToken() (bool, error) {
	return HasOAuthToken()
}
