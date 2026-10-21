package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// errOAuthLoginRequired 表示登录已失效（无 token、refresh token 失效或过期），需要重新登录。
var errOAuthLoginRequired = errors.New("oauth login required")

// oauthRefreshMu 串行化 token 刷新。中心端每次刷新都会撤销旧 access token，
// 并发刷新会互相撤销，导致最终保存的 token 不可用。
var oauthRefreshMu sync.Mutex

func loadOrRefreshOAuthToken(ctx context.Context) (*oauth2.Token, error) {
	return loadValidOAuthToken(ctx, "")
}

// refreshRejectedAccessToken 在 access token 被服务端拒绝（401）时强制刷新，返回新的 access token。
// 若其他请求已完成刷新（存储的 token 与 rejected 不同），直接使用新 token。
func refreshRejectedAccessToken(ctx context.Context, rejected string) (string, error) {
	token, err := loadValidOAuthToken(ctx, rejected)
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

func loadValidOAuthToken(ctx context.Context, rejected string) (*oauth2.Token, error) {
	oauthRefreshMu.Lock()
	defer oauthRefreshMu.Unlock()

	// 加锁后重新读取：等待期间其他请求可能已经刷新并保存了新 token
	token, err := LoadOAuthToken()
	if err != nil {
		if errors.Is(err, errSecretNotFound) {
			return nil, errors.Join(errOAuthLoginRequired, err)
		}
		return nil, fmt.Errorf("load oauth token: %w", err)
	}
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return nil, fmt.Errorf("%w: oauth token is empty", errOAuthLoginRequired)
	}

	rejectedByServer := rejected != "" && token.AccessToken == rejected
	if token.Valid() && !rejectedByServer {
		return token, nil
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		_ = ClearOAuthToken()
		return nil, fmt.Errorf("%w: access token expired and no refresh token available", errOAuthLoginRequired)
	}

	oauthCfg, cfgErr := resolveOAuthConfig()
	if cfgErr != nil {
		return nil, fmt.Errorf("load oauth config for refresh: %w", cfgErr)
	}

	// 只携带 refresh token，强制 TokenSource 走刷新流程（即使本地认为 access token 未过期）
	refreshedToken, refreshErr := oauthCfg.TokenSource(ctx, &oauth2.Token{RefreshToken: token.RefreshToken}).Token()
	if refreshErr != nil {
		if isOAuthGrantRejected(refreshErr) {
			// refresh token 已失效 / 过期（中心端有效期 30 天，不随刷新延长）
			_ = ClearOAuthToken()
			return nil, errors.Join(errOAuthLoginRequired, refreshErr)
		}
		// 网络等临时错误：保留 token，稍后重试
		return nil, fmt.Errorf("refresh oauth token: %w", refreshErr)
	}

	if saveErr := SaveOAuthToken(refreshedToken); saveErr != nil {
		return nil, fmt.Errorf("save refreshed oauth token: %w", saveErr)
	}

	return refreshedToken, nil
}

// isOAuthGrantRejected 判断刷新失败是否因为 refresh token 本身被拒绝（需重新登录）。
func isOAuthGrantRejected(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) {
		return false
	}
	switch retrieveErr.ErrorCode {
	case "invalid_grant", "invalid_client", "unauthorized_client":
		return true
	}
	if retrieveErr.Response != nil {
		status := retrieveErr.Response.StatusCode
		return status == http.StatusBadRequest || status == http.StatusUnauthorized
	}
	return false
}
