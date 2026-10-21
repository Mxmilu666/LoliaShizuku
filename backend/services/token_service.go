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
	// 路由守卫每次导航都会调用，限制刷新耗时，避免网络异常时界面卡住
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()
	token, err := loadOrRefreshOAuthToken(ctx)
	if err == nil {
		return token != nil, nil
	}
	if errors.Is(err, errOAuthLoginRequired) {
		return false, nil
	}
	// 刷新遇到网络等临时错误时 token 仍保留：视为已登录，避免离线启动就被踢到登录页，
	// 页面请求会显示具体错误，恢复网络后自动刷新
	if stored, loadErr := LoadOAuthToken(); loadErr == nil && stored != nil && stored.RefreshToken != "" {
		return true, nil
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

// StartDeviceLogin 发起设备授权（RFC 8628），返回需要用户在浏览器中输入的用户码。
func (s *TokenService) StartDeviceLogin() (*DeviceLoginInfo, error) {
	return startDeviceLogin()
}

// WaitDeviceLogin 等待用户在浏览器中完成设备授权，成功后保存 token。
func (s *TokenService) WaitDeviceLogin() (bool, error) {
	if err := waitDeviceLogin(); err != nil {
		return false, err
	}
	return true, nil
}

// CancelDeviceLogin 取消进行中的设备授权。
func (s *TokenService) CancelDeviceLogin() {
	cancelDeviceLogin()
}

// ClearOAuthToken removes OAuth token from keyring.
func (s *TokenService) ClearOAuthToken() error {
	return ClearOAuthToken()
}
