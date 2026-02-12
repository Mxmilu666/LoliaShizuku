package services

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"loliashizuku/backend/api"
	"loliashizuku/backend/httpclient"
	"loliashizuku/backend/models"
)

const (
	defaultFrpcRepoOwner       = "Lolia-FRP"
	defaultFrpcRepoName        = "lolia-frp"
	defaultFrpcStatusTimeout   = 20 * time.Second
	defaultFrpcDownloadTimeout = 2 * time.Minute
	defaultFrpcInstallTimeout  = 5 * time.Minute
	frpcVersionProbeTimeout    = 3 * time.Second
)

type frpcInstallState struct {
	Version     string `json:"version"`
	AssetName   string `json:"asset_name"`
	SHA256      string `json:"sha256"`
	InstalledAt string `json:"installed_at"`
}

type frpcUserSettings struct {
	GitHubMirrorURL string `json:"github_mirror_url"`
}

type FrpcService struct {
	releaseAPI *api.GitHubReleaseAPI
	httpClient *http.Client
	repoOwner  string
	repoName   string

	installMu     sync.Mutex
	installCancel context.CancelFunc
}

func NewFrpcService() *FrpcService {
	client := &http.Client{Timeout: defaultFrpcInstallTimeout}

	repoOwner := strings.TrimSpace(os.Getenv("LOLIA_FRPC_REPO_OWNER"))
	if repoOwner == "" {
		repoOwner = defaultFrpcRepoOwner
	}
	repoName := strings.TrimSpace(os.Getenv("LOLIA_FRPC_REPO_NAME"))
	if repoName == "" {
		repoName = defaultFrpcRepoName
	}

	return &FrpcService{
		releaseAPI: api.NewGitHubReleaseAPI(client, httpclient.ResolveUserAgent("")),
		httpClient: client,
		repoOwner:  repoOwner,
		repoName:   repoName,
	}
}

func (s *FrpcService) GetFrpcStatus() (*models.FrpcStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFrpcStatusTimeout)
	defer cancel()
	return s.buildStatus(ctx, true, nil)
}

func (s *FrpcService) InstallOrUpdateFrpc() (*models.FrpcInstallResult, error) {
	ctx, err := s.beginInstall()
	if err != nil {
		return nil, err
	}
	defer s.endInstall()

	latest, err := s.resolveLatestRelease(ctx)
	if err != nil {
		return nil, normalizeInstallError(err)
	}

	paths, err := s.paths()
	if err != nil {
		return nil, normalizeInstallError(err)
	}
	if err := ensureDirs(paths.FrpcDir, paths.BinDir, paths.DownloadDir); err != nil {
		return nil, normalizeInstallError(err)
	}

	archivePath := filepath.Join(paths.DownloadDir, latest.Asset.Name)
	downloadedSHA256, err := s.downloadArchive(ctx, latest.Asset.DownloadURL, archivePath)
	if err != nil {
		return nil, normalizeInstallError(err)
	}

	expectedSHA256 := strings.ToLower(strings.TrimSpace(latest.Asset.SHA256))
	if expectedSHA256 == "" {
		return nil, fmt.Errorf("release asset digest is empty: %s", latest.Asset.Name)
	}
	if downloadedSHA256 != expectedSHA256 {
		return nil, fmt.Errorf("sha256 mismatch for %s: expected=%s actual=%s", latest.Asset.Name, expectedSHA256, downloadedSHA256)
	}

	binaryName := filepath.Base(paths.BinaryPath)
	if err := extractBinaryFromArchive(archivePath, latest.Asset.ArchiveFormat, binaryName, paths.BinaryPath); err != nil {
		return nil, normalizeInstallError(err)
	}

	state := frpcInstallState{
		Version:     latest.TagName,
		AssetName:   latest.Asset.Name,
		SHA256:      downloadedSHA256,
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}

	if detectedVersion, detectErr := detectFrpcVersion(ctx, paths.BinaryPath); detectErr == nil && detectedVersion != "" {
		state.Version = detectedVersion
	}

	if err := saveInstallState(paths.StatePath, state); err != nil {
		return nil, normalizeInstallError(err)
	}
	if err := removeIfExists(archivePath); err != nil {
		return nil, normalizeInstallError(err)
	}

	status, err := s.buildStatus(ctx, false, latest)
	if err != nil {
		return nil, normalizeInstallError(err)
	}

	return &models.FrpcInstallResult{
		Release: *latest,
		Status:  *status,
	}, nil
}

func (s *FrpcService) CancelInstallOrUpdateFrpc() error {
	s.installMu.Lock()
	cancel := s.installCancel
	s.installMu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	return nil
}

func (s *FrpcService) RemoveFrpc() error {
	paths, err := s.paths()
	if err != nil {
		return err
	}

	if err := removeIfExists(paths.BinaryPath); err != nil {
		return err
	}
	if err := removeIfExists(paths.StatePath); err != nil {
		return err
	}
	return nil
}

func (s *FrpcService) GetGitHubMirrorURL() (string, error) {
	settings, err := s.loadUserSettings()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(settings.GitHubMirrorURL), nil
}

func (s *FrpcService) SetGitHubMirrorURL(rawURL string) error {
	mirrorURL, err := normalizeMirrorURL(rawURL)
	if err != nil {
		return err
	}

	settings, err := s.loadUserSettings()
	if err != nil {
		return err
	}
	settings.GitHubMirrorURL = mirrorURL
	return s.saveUserSettings(settings)
}

func (s *FrpcService) buildStatus(
	ctx context.Context,
	fetchLatest bool,
	latest *models.FrpcReleaseInfo,
) (*models.FrpcStatus, error) {
	paths, err := s.paths()
	if err != nil {
		return nil, err
	}

	installed, err := loadInstalledInfo(paths.StatePath, paths.BinaryPath)
	if err != nil {
		return nil, err
	}

	status := &models.FrpcStatus{
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Paths:     paths,
		Installed: installed,
	}
	settings, settingsErr := s.loadUserSettings()
	if settingsErr == nil {
		status.GitHubMirrorURL = strings.TrimSpace(settings.GitHubMirrorURL)
	} else {
		status.GitHubMirrorURL = ""
	}

	resolvedLatest := latest
	if fetchLatest && resolvedLatest == nil {
		resolvedLatest, err = s.resolveLatestRelease(ctx)
		if err != nil {
			status.LatestError = err.Error()
			return status, nil
		}
	}
	if resolvedLatest != nil {
		status.Latest = resolvedLatest
		status.UpdateAvailable = isUpdateAvailable(installed, resolvedLatest)
	}

	return status, nil
}

func (s *FrpcService) resolveLatestRelease(ctx context.Context) (*models.FrpcReleaseInfo, error) {
	release, err := s.releaseAPI.GetLatestRelease(ctx, s.repoOwner, s.repoName)
	if err != nil {
		return nil, err
	}

	assetName, archiveFormat, err := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	var selected *models.GitHubReleaseAsset
	for i := range release.Assets {
		asset := &release.Assets[i]
		if strings.EqualFold(strings.TrimSpace(asset.Name), assetName) {
			selected = asset
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("latest release does not contain asset %s", assetName)
	}

	sha256Digest := parseSHA256Digest(selected.Digest)
	asset := models.FrpcReleaseAsset{
		Name:          selected.Name,
		DownloadURL:   selected.BrowserDownloadURL,
		ContentType:   selected.ContentType,
		Size:          selected.Size,
		Digest:        selected.Digest,
		SHA256:        sha256Digest,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		ArchiveFormat: archiveFormat,
	}

	return &models.FrpcReleaseInfo{
		TagName:     release.TagName,
		Name:        release.Name,
		HTMLURL:     release.HTMLURL,
		PublishedAt: release.PublishedAt,
		Asset:       asset,
	}, nil
}

func (s *FrpcService) paths() (models.FrpcPaths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return models.FrpcPaths{}, fmt.Errorf("get config dir: %w", err)
	}

	appDir := filepath.Join(configDir, "LoliaShizuku")
	userDataDir := filepath.Join(appDir, "userdata")
	frpcDir := filepath.Join(userDataDir, "frpc")
	binDir := filepath.Join(frpcDir, "bin")
	downloadDir := filepath.Join(frpcDir, "downloads")
	binaryPath := filepath.Join(binDir, frpcBinaryName())
	statePath := filepath.Join(frpcDir, "installed.json")
	settingsPath := filepath.Join(frpcDir, "settings.json")

	return models.FrpcPaths{
		UserDataDir:  userDataDir,
		FrpcDir:      frpcDir,
		BinDir:       binDir,
		BinaryPath:   binaryPath,
		DownloadDir:  downloadDir,
		StatePath:    statePath,
		SettingsPath: settingsPath,
	}, nil
}

func (s *FrpcService) downloadArchive(ctx context.Context, url string, outputPath string) (string, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, defaultFrpcDownloadTimeout)
	defer cancel()

	mirrorURL, mirrorErr := s.GetGitHubMirrorURL()
	if mirrorErr != nil {
		return "", mirrorErr
	}

	downloadURL := applyMirrorURL(strings.TrimSpace(url), mirrorURL)
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("User-Agent", httpclient.ResolveUserAgent(""))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download release asset failed: status=%d", resp.StatusCode)
	}

	if err := ensureDirs(filepath.Dir(outputPath)); err != nil {
		return "", err
	}

	tempPath := outputPath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create temp archive file: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hasher), resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("write archive file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("close archive file: %w", err)
	}

	if err := os.Rename(tempPath, outputPath); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("rename archive file: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func loadInstalledInfo(statePath string, binaryPath string) (*models.FrpcInstalledInfo, error) {
	exists, err := fileExists(binaryPath)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !exists {
				return nil, nil
			}
			version := "unknown"
			if detectedVersion, detectErr := detectFrpcVersionWithTimeout(binaryPath); detectErr == nil && detectedVersion != "" {
				version = detectedVersion
			}
			return &models.FrpcInstalledInfo{
				Version:      version,
				BinaryPath:   binaryPath,
				BinaryExists: true,
			}, nil
		}
		return nil, fmt.Errorf("read frpc install state: %w", err)
	}

	var state frpcInstallState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("decode frpc install state: %w", err)
	}

	version := strings.TrimSpace(state.Version)
	if version == "" {
		version = "unknown"
	}
	if exists && (version == "unknown" || version == "") {
		if detectedVersion, detectErr := detectFrpcVersionWithTimeout(binaryPath); detectErr == nil && detectedVersion != "" {
			version = detectedVersion
		}
	}

	return &models.FrpcInstalledInfo{
		Version:      version,
		AssetName:    state.AssetName,
		SHA256:       strings.ToLower(strings.TrimSpace(state.SHA256)),
		InstalledAt:  state.InstalledAt,
		BinaryPath:   binaryPath,
		BinaryExists: exists,
	}, nil
}

func saveInstallState(path string, state frpcInstallState) error {
	if err := ensureDirs(filepath.Dir(path)); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode frpc install state: %w", err)
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write frpc install state: %w", err)
	}
	return nil
}

func isUpdateAvailable(installed *models.FrpcInstalledInfo, latest *models.FrpcReleaseInfo) bool {
	if latest == nil {
		return false
	}
	if installed == nil {
		return true
	}

	installedVersion := normalizeInstalledVersionForCompare(installed.Version)
	latestVersion := normalizeGitHubTagForCompare(latest.TagName)
	if installedVersion == "" || installedVersion == "unknown" {
		return true
	}
	if latestVersion == "" {
		return true
	}
	return installedVersion != latestVersion
}

func normalizeInstalledVersionForCompare(raw string) string {
	version := strings.TrimSpace(raw)
	if version == "" {
		return ""
	}
	if strings.EqualFold(version, "unknown") {
		return "unknown"
	}

	fields := strings.Fields(version)
	if len(fields) >= 2 && strings.EqualFold(fields[0], "LoliaFRP-CLI") {
		version = strings.TrimSpace(fields[1])
	} else if len(fields) >= 2 {
		version = strings.TrimSpace(fields[len(fields)-1])
	}

	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	if version == "" {
		return ""
	}
	return "LoliaFRP-CLI " + version
}

func normalizeGitHubTagForCompare(tag string) string {
	version := strings.TrimSpace(tag)
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	if version == "" {
		return ""
	}
	return "LoliaFRP-CLI " + version
}

func extractBinaryFromArchive(archivePath, archiveFormat, binaryName, outputPath string) error {
	format := strings.ToLower(strings.TrimSpace(archiveFormat))
	switch format {
	case "tar.gz":
		return extractBinaryFromTarGz(archivePath, binaryName, outputPath)
	case "zip":
		return extractBinaryFromZip(archivePath, binaryName, outputPath)
	default:
		return fmt.Errorf("unsupported archive format: %s", archiveFormat)
	}
}

func extractBinaryFromTarGz(archivePath, binaryName, outputPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open tar.gz archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read gzip archive: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read tar archive: %w", err)
		}
		if header == nil || !header.FileInfo().Mode().IsRegular() {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		return writeExecutableFile(outputPath, tarReader)
	}
	return fmt.Errorf("binary %s not found in tar.gz archive", binaryName)
}

func extractBinaryFromZip(archivePath, binaryName, outputPath string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if !file.FileInfo().Mode().IsRegular() {
			continue
		}
		if filepath.Base(file.Name) != binaryName {
			continue
		}

		reader, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip file entry %s: %w", file.Name, err)
		}
		defer reader.Close()

		return writeExecutableFile(outputPath, reader)
	}
	return fmt.Errorf("binary %s not found in zip archive", binaryName)
}

func writeExecutableFile(outputPath string, reader io.Reader) error {
	if err := ensureDirs(filepath.Dir(outputPath)); err != nil {
		return err
	}

	tempPath := outputPath + ".tmp"
	fileMode := os.FileMode(0o644)
	if runtime.GOOS != "windows" {
		fileMode = 0o755
	}

	outputFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("create output binary: %w", err)
	}
	if _, err := io.Copy(outputFile, reader); err != nil {
		_ = outputFile.Close()
		return fmt.Errorf("write output binary: %w", err)
	}
	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("close output binary: %w", err)
	}

	_ = os.Remove(outputPath)
	if err := os.Rename(tempPath, outputPath); err != nil {
		return fmt.Errorf("replace output binary: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(outputPath, 0o755); err != nil {
			return fmt.Errorf("chmod output binary: %w", err)
		}
	}
	return nil
}

func frpcBinaryName() string {
	if runtime.GOOS == "windows" {
		return "frpc.exe"
	}
	return "frpc"
}

func detectFrpcVersion(ctx context.Context, binaryPath string) (string, error) {
	cmd := exec.CommandContext(ctx, binaryPath, "-v")
	configureBackgroundProcess(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run frpc -v: %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return "", fmt.Errorf("frpc -v output is empty")
	}

	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return strings.TrimSpace(fields[1]), nil
	}
	return "", fmt.Errorf("failed to parse frpc version output: %s", line)
}

func detectFrpcVersionWithTimeout(binaryPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), frpcVersionProbeTimeout)
	defer cancel()
	return detectFrpcVersion(ctx, binaryPath)
}

func (s *FrpcService) loadUserSettings() (frpcUserSettings, error) {
	paths, err := s.paths()
	if err != nil {
		return frpcUserSettings{}, err
	}

	raw, err := os.ReadFile(paths.SettingsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return frpcUserSettings{}, nil
		}
		return frpcUserSettings{}, fmt.Errorf("read frpc settings: %w", err)
	}

	var settings frpcUserSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return frpcUserSettings{}, fmt.Errorf("decode frpc settings: %w", err)
	}
	return settings, nil
}

func (s *FrpcService) saveUserSettings(settings frpcUserSettings) error {
	paths, err := s.paths()
	if err != nil {
		return err
	}
	if err := ensureDirs(paths.FrpcDir); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode frpc settings: %w", err)
	}
	if err := os.WriteFile(paths.SettingsPath, payload, 0o644); err != nil {
		return fmt.Errorf("write frpc settings: %w", err)
	}
	return nil
}

func normalizeMirrorURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := neturl.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid github mirror url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("github mirror url must start with http:// or https://")
	}
	return trimmed, nil
}

func applyMirrorURL(rawURL, mirrorURL string) string {
	urlValue := strings.TrimSpace(rawURL)
	if urlValue == "" {
		return urlValue
	}

	mirror := strings.TrimSpace(mirrorURL)
	if mirror == "" {
		return urlValue
	}

	parsedURL, err := neturl.Parse(urlValue)
	if err == nil && parsedURL != nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		base := strings.TrimRight(mirror, "/")
		path := strings.TrimLeft(parsedURL.EscapedPath(), "/")
		if path == "" {
			path = strings.TrimLeft(parsedURL.Path, "/")
		}

		rebuilt := base
		if path != "" {
			rebuilt += "/" + path
		}
		if parsedURL.RawQuery != "" {
			rebuilt += "?" + parsedURL.RawQuery
		}
		if parsedURL.Fragment != "" {
			rebuilt += "#" + parsedURL.Fragment
		}
		return rebuilt
	}

	if !strings.HasSuffix(mirror, "/") {
		mirror += "/"
	}
	return mirror + strings.TrimLeft(urlValue, "/")
}

func releaseAssetName(goos, goarch string) (string, string, error) {
	normalizedOS := strings.ToLower(strings.TrimSpace(goos))
	normalizedArch := strings.ToLower(strings.TrimSpace(goarch))

	switch normalizedOS {
	case "windows":
		switch normalizedArch {
		case "386", "amd64", "arm", "arm64":
			return fmt.Sprintf("LoliaFrp_%s_%s.zip", normalizedOS, normalizedArch), "zip", nil
		}
	case "linux", "darwin", "freebsd", "openbsd":
		switch normalizedArch {
		case "386", "amd64", "arm", "arm64":
			return fmt.Sprintf("LoliaFrp_%s_%s.tar.gz", normalizedOS, normalizedArch), "tar.gz", nil
		}
	case "android":
		switch normalizedArch {
		case "arm", "arm64":
			return fmt.Sprintf("LoliaFrp_%s_%s.tar.gz", normalizedOS, normalizedArch), "tar.gz", nil
		}
	}

	return "", "", fmt.Errorf("unsupported platform: %s/%s", normalizedOS, normalizedArch)
}

func parseSHA256Digest(raw string) string {
	digest := strings.ToLower(strings.TrimSpace(raw))
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) != 64 {
		return ""
	}
	for _, r := range digest {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ""
		}
	}
	return digest
}

func (s *FrpcService) beginInstall() (context.Context, error) {
	s.installMu.Lock()
	defer s.installMu.Unlock()

	if s.installCancel != nil {
		return nil, fmt.Errorf("frpc 下载/安装正在进行中")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultFrpcInstallTimeout)
	s.installCancel = cancel
	return ctx, nil
}

func (s *FrpcService) endInstall() {
	s.installMu.Lock()
	cancel := s.installCancel
	s.installCancel = nil
	s.installMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func normalizeInstallError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("frpc 下载已终止")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("frpc 下载超时，请稍后重试")
	}
	return err
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("remove %s: %w", path, err)
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat %s: %w", path, err)
}

func ensureDirs(dirs ...string) error {
	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		if err := os.MkdirAll(trimmed, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", trimmed, err)
		}
	}
	return nil
}
