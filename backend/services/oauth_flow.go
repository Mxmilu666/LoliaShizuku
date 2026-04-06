package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const (
	defaultOAuthAuthorizeURL = "https://dash.lolia.link/oauth/authorize"
	defaultOAuthTokenURL     = "https://api.lolia.link/api/v1/oauth2/token"
	defaultOAuthRedirectURL  = "http://localhost:1145"
	defaultOAuthScope        = "all"
	defaultOAuthClientID     = "mdn2kiogechzveez"

	oauthAuthTimeout  = 3 * time.Minute
	oauthTokenTimeout = 20 * time.Second
)

type oauthCallbackResult struct {
	code string
	err  error
}

func resolveOAuthConfig() (*oauth2.Config, error) {
	resolvedClientID := strings.TrimSpace(os.Getenv("LOLIA_OAUTH_CLIENT_ID"))
	if resolvedClientID == "" {
		resolvedClientID = defaultOAuthClientID
	}

	authorizeURL := strings.TrimSpace(os.Getenv("LOLIA_OAUTH_AUTHORIZE_URL"))
	if authorizeURL == "" {
		authorizeURL = defaultOAuthAuthorizeURL
	}

	tokenURL := strings.TrimSpace(os.Getenv("LOLIA_OAUTH_TOKEN_URL"))
	if tokenURL == "" {
		tokenURL = defaultOAuthTokenURL
	}

	redirectURL := strings.TrimSpace(os.Getenv("LOLIA_OAUTH_REDIRECT_URL"))
	if redirectURL == "" {
		redirectURL = defaultOAuthRedirectURL
	}

	return &oauth2.Config{
		ClientID:    resolvedClientID,
		RedirectURL: redirectURL,
		Scopes:      strings.Fields(defaultOAuthScope),
		Endpoint: oauth2.Endpoint{
			AuthURL:   authorizeURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}, nil
}

func randomURLSafeString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func beginOAuthLogin() error {
	oauthCfg, err := resolveOAuthConfig()
	if err != nil {
		return err
	}

	redirectURI, err := url.Parse(oauthCfg.RedirectURL)
	if err != nil {
		return fmt.Errorf("invalid redirect_uri: %w", err)
	}
	if redirectURI.Scheme != "http" {
		return fmt.Errorf("redirect_uri must use http for desktop loopback callback")
	}
	if strings.TrimSpace(redirectURI.Host) == "" {
		return fmt.Errorf("redirect_uri host is empty")
	}

	state, err := randomURLSafeString(32)
	if err != nil {
		return err
	}

	codeVerifier, err := randomURLSafeString(64)
	if err != nil {
		return err
	}

	resultCh := make(chan oauthCallbackResult, 1)
	handlerPath := redirectURI.Path
	if handlerPath == "" {
		handlerPath = "/"
	}

	mux := http.NewServeMux()
	mux.HandleFunc(handlerPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		if query.Get("state") != state {
			http.Error(w, "OAuth state mismatch", http.StatusBadRequest)
			select {
			case resultCh <- oauthCallbackResult{err: fmt.Errorf("oauth state mismatch")}:
			default:
			}
			return
		}

		if oauthErr := strings.TrimSpace(query.Get("error")); oauthErr != "" {
			message := oauthErr
			if desc := strings.TrimSpace(query.Get("error_description")); desc != "" {
				message = fmt.Sprintf("%s: %s", oauthErr, desc)
			}
			http.Error(w, message, http.StatusBadRequest)
			select {
			case resultCh <- oauthCallbackResult{err: fmt.Errorf("oauth authorize failed: %s", message)}:
			default:
			}
			return
		}

		code := strings.TrimSpace(query.Get("code"))
		if code == "" {
			http.Error(w, "missing OAuth code", http.StatusBadRequest)
			select {
			case resultCh <- oauthCallbackResult{err: fmt.Errorf("missing oauth code")}:
			default:
			}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><h3>Login Success</h3><p>You can return to the app now.</p></body></html>`))

		select {
		case resultCh <- oauthCallbackResult{code: code}:
		default:
		}
	})

	listener, err := net.Listen("tcp", redirectURI.Host)
	if err != nil {
		return fmt.Errorf("listen oauth callback %s: %w", redirectURI.Host, err)
	}

	server := &http.Server{
		Handler: mux,
	}

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case resultCh <- oauthCallbackResult{err: fmt.Errorf("oauth callback server error: %w", serveErr)}:
			default:
			}
		}
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authCodeOptions := []oauth2.AuthCodeOption{
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(codeVerifier),
	}

	authURL := oauthCfg.AuthCodeURL(state, authCodeOptions...)

	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("open authorize url: %w", err)
	}

	var result oauthCallbackResult
	select {
	case result = <-resultCh:
	case <-time.After(oauthAuthTimeout):
		return fmt.Errorf("oauth authorization timed out after %s", oauthAuthTimeout.String())
	}

	if result.err != nil {
		return result.err
	}

	tokenCtx, cancel := context.WithTimeout(context.Background(), oauthTokenTimeout)
	defer cancel()

	exchangeOptions := []oauth2.AuthCodeOption{oauth2.VerifierOption(codeVerifier)}

	token, err := oauthCfg.Exchange(tokenCtx, result.code, exchangeOptions...)
	if err != nil {
		return fmt.Errorf("exchange oauth code for token: %w", err)
	}

	if err := SaveOAuthToken(token); err != nil {
		return err
	}
	return nil
}
