package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// 设备授权模式（RFC 8628）：应用获取用户码，用户在任意设备的浏览器中打开验证页面输入并确认，
// 应用轮询令牌端点直至授权完成。适用于无法接收 localhost 回调的场景（如手机）。

// DeviceLoginInfo 是展示给用户的设备授权信息。
type DeviceLoginInfo struct {
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"` // 秒
}

type deviceLoginSession struct {
	response *oauth2.DeviceAuthResponse
	ctx      context.Context
	cancel   context.CancelFunc
}

var (
	deviceLoginMu sync.Mutex
	deviceLogin   *deviceLoginSession
)

func startDeviceLogin() (*DeviceLoginInfo, error) {
	oauthCfg, err := resolveOAuthConfig()
	if err != nil {
		return nil, err
	}

	requestCtx, cancelRequest := context.WithTimeout(context.Background(), oauthTokenTimeout)
	defer cancelRequest()
	response, err := oauthCfg.DeviceAuth(requestCtx)
	if err != nil {
		return nil, fmt.Errorf("获取设备码失败: %w", err)
	}

	// 设备码过期后停止轮询（未返回 expires_in 时按 RFC 8628 常见的 15 分钟处理）
	if response.Expiry.IsZero() {
		response.Expiry = time.Now().Add(15 * time.Minute)
	}
	ctx, cancel := context.WithDeadline(context.Background(), response.Expiry)

	deviceLoginMu.Lock()
	if deviceLogin != nil {
		deviceLogin.cancel()
	}
	deviceLogin = &deviceLoginSession{response: response, ctx: ctx, cancel: cancel}
	deviceLoginMu.Unlock()

	return &DeviceLoginInfo{
		UserCode:                response.UserCode,
		VerificationURI:         response.VerificationURI,
		VerificationURIComplete: response.VerificationURIComplete,
		ExpiresIn:               int64(time.Until(response.Expiry).Seconds()),
	}, nil
}

// waitDeviceLogin 轮询令牌端点直至用户完成授权，成功后保存 token。
func waitDeviceLogin() error {
	deviceLoginMu.Lock()
	session := deviceLogin
	deviceLoginMu.Unlock()
	if session == nil {
		return errors.New("请先获取设备码")
	}

	defer func() {
		deviceLoginMu.Lock()
		if deviceLogin == session {
			deviceLogin = nil
		}
		deviceLoginMu.Unlock()
		session.cancel()
	}()

	oauthCfg, err := resolveOAuthConfig()
	if err != nil {
		return err
	}

	token, err := oauthCfg.DeviceAccessToken(session.ctx, session.response)
	if err != nil {
		return deviceLoginError(err)
	}
	return SaveOAuthToken(token)
}

func cancelDeviceLogin() {
	deviceLoginMu.Lock()
	defer deviceLoginMu.Unlock()
	if deviceLogin != nil {
		deviceLogin.cancel()
		deviceLogin = nil
	}
}

func deviceLoginError(err error) error {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		switch retrieveErr.ErrorCode {
		case "access_denied":
			return errors.New("已拒绝授权")
		case "expired_token":
			return errors.New("设备码已过期，请重新获取")
		}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errors.New("设备码已过期，请重新获取")
	case errors.Is(err, context.Canceled):
		return errors.New("已取消设备码登录")
	}
	return fmt.Errorf("设备码登录失败: %w", err)
}
