package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Mxmilu666/LoliaShizuku/backend/models"
)

type GitHubReleaseAPI struct {
	httpClient *http.Client
	userAgent  string
}

func NewGitHubReleaseAPI(httpClient *http.Client, userAgent string) *GitHubReleaseAPI {
	return &GitHubReleaseAPI{
		httpClient: httpClient,
		userAgent:  strings.TrimSpace(userAgent),
	}
}

func (a *GitHubReleaseAPI) GetLatestRelease(ctx context.Context, owner, repo string) (*models.GitHubRelease, error) {
	trimmedOwner := strings.TrimSpace(owner)
	trimmedRepo := strings.TrimSpace(repo)
	if trimmedOwner == "" || trimmedRepo == "" {
		return nil, fmt.Errorf("owner/repo is empty")
	}

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest",
		trimmedOwner,
		trimmedRepo,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build github release request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if a.userAgent != "" {
		req.Header.Set("User-Agent", a.userAgent)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request github latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github latest release failed: status=%d", resp.StatusCode)
	}

	var release models.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode github latest release: %w", err)
	}
	return &release, nil
}
