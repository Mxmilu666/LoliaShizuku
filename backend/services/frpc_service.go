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

	"github.com/Mxmilu666/LoliaShizuku/backend/api"
	"github.com/Mxmilu666/LoliaShizuku/backend/httpclient"
	"github.com/Mxmilu666/LoliaShizuku/backend/models"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultFrpcRepoOwner       = "Lolia-FRP"
	defaultFrpcRepoName        = "lolia-frp"
	defaultFrpcStatusTimeout   = 20 * time.Second
	defaultFrpcDownloadTimeout = 2 * time.Minute
	defaultFrpcInstallTimeout  = 5 * time.Minute
	frpcVersionProbeTimeout    = 3 * time.Second

	mirrorModeOfficial = "official"
	mirrorModeBuiltin  = "builtin"
	mirrorModeCustom   = "custom"

	frpcInstallProgressEvent = "frpc_install_progress"

	installPhaseResolving   = "resolving"
	installPhaseDownloading = "downloading"
	installPhaseVerifying   = "verifying"
	installPhaseExtracting  = "extracting"
	installPhaseDone        = "done"

	progressEmitInterval = 150 * time.Millisecond
)

type frpcInstallState struct {
	Version     string `json:"version"`
	AssetName   string `json:"asset_name"`
	SHA256      string `json:"sha256"`
	InstalledAt string `json:"installed_at"`
}

type frpcUserSettings struct {
	GitHubMirrorURL string                  `json:"github_mirror_url,omitempty"`
	MirrorConfig    models.FrpcMirrorConfig `json:"mirror_config,omitempty"`
}

var defaultBuiltinFrpcMirrors = []models.FrpcMirrorPreset{
	{
		ID:          "milu",
		Name:        "gh.milu.moe",
		Description: "Milu GitHub 镜像",
		BaseURL:     "https://gh.milu.moe",
	},
	{
		ID:          "akaere",
		Name:        "cdn.akaere.online",
		Description: "Akaere GitHub 路径镜像",
		BaseURL:     "https://cdn.akaere.online/github.com",
	},
	{
		ID:          "xiaomocs",
		Name:        "hub.xiaomocs.com",
		Description: "XiaoMo GitHub 路径镜像",
		BaseURL:     "https://hub.xiaomocs.com/github",
	},
	{
		ID:          "locyan",
		Name:        "mirrors.locyan.cn",
		Description: "乐青云镜像",
		URLTemplate: "https://mirrors.locyan.cn/github-release/{owner}/{repo}/Release%20{tag}/{asset}",
	},
}

type FrpcService struct {
	versionAPI *api.ClientVersionAPI
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

	// 版本信息走中心 API 公开接口，无需携带 OAuth Token
	versionClient := httpclient.New(httpclient.Options{
		BaseURL:    centerAPIBaseURL(),
		HTTPClient: client,
	})

	return &FrpcService{
		versionAPI: api.NewClientVersionAPI(versionClient),
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

	s.emitInstallProgress(installPhaseResolving, 0, 0)
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
	downloadedSHA256, err := s.downloadArchive(ctx, latest.Asset.DownloadURL, archivePath, latest.Asset.Size)
	if err != nil {
		return nil, normalizeDownloadError(err)
	}

	s.emitInstallProgress(installPhaseVerifying, 0, 0)
	expectedSHA256 := strings.ToLower(strings.TrimSpace(latest.Asset.SHA256))
	if expectedSHA256 == "" {
		return nil, fmt.Errorf("release asset digest is empty: %s", latest.Asset.Name)
	}
	if downloadedSHA256 != expectedSHA256 {
		return nil, fmt.Errorf("sha256 mismatch for %s: expected=%s actual=%s", latest.Asset.Name, expectedSHA256, downloadedSHA256)
	}

	s.emitInstallProgress(installPhaseExtracting, 0, 0)
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

	s.emitInstallProgress(installPhaseDone, latest.Asset.Size, latest.Asset.Size)

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
	config, err := s.GetMirrorConfig()
	if err != nil {
		return "", err
	}
	switch config.Mode {
	case mirrorModeBuiltin:
		preset, ok := findBuiltinMirrorPreset(config.PresetID)
		if !ok {
			return "", nil
		}
		return strings.TrimSpace(preset.BaseURL), nil
	case mirrorModeCustom:
		return strings.TrimSpace(config.CustomBaseURL), nil
	default:
		return "", nil
	}
}

func (s *FrpcService) SetGitHubMirrorURL(rawURL string) error {
	mirrorURL := strings.TrimSpace(rawURL)
	if mirrorURL == "" {
		return s.SetMirrorConfig(models.FrpcMirrorConfig{
			Mode: mirrorModeOfficial,
		})
	}

	for _, preset := range builtinMirrorPresets() {
		if strings.TrimSpace(preset.BaseURL) != "" && sameNormalizedURL(preset.BaseURL, mirrorURL) {
			return s.SetMirrorConfig(models.FrpcMirrorConfig{
				Mode:     mirrorModeBuiltin,
				PresetID: preset.ID,
			})
		}
	}

	return s.SetMirrorConfig(models.FrpcMirrorConfig{
		Mode:          mirrorModeCustom,
		CustomBaseURL: mirrorURL,
	})
}

func (s *FrpcService) GetMirrorConfig() (models.FrpcMirrorConfig, error) {
	settings, err := s.loadUserSettings()
	if err != nil {
		return models.FrpcMirrorConfig{}, err
	}
	return normalizeStoredMirrorConfig(settings), nil
}

func (s *FrpcService) SetMirrorConfig(config models.FrpcMirrorConfig) error {
	normalized, err := normalizeMirrorConfig(config)
	if err != nil {
		return err
	}

	settings, err := s.loadUserSettings()
	if err != nil {
		return err
	}
	settings.GitHubMirrorURL = ""
	settings.MirrorConfig = normalized
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
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		Paths:          paths,
		Installed:      installed,
		BuiltinMirrors: builtinMirrorPresets(),
	}
	mirrorConfig, mirrorErr := s.GetMirrorConfig()
	if mirrorErr == nil {
		status.MirrorConfig = mirrorConfig
		status.GitHubMirrorURL = legacyMirrorURLFromConfig(mirrorConfig)
	} else {
		status.MirrorConfig = models.FrpcMirrorConfig{Mode: mirrorModeOfficial}
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
	release, err := s.versionAPI.GetLatestClientVersion(ctx)
	if err != nil {
		return nil, err
	}

	tag := strings.TrimSpace(release.Tag)
	if tag == "" {
		return nil, fmt.Errorf("client version tag is empty")
	}

	assetName, archiveFormat, err := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}

	var selected *models.ClientVersionAsset
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

	// 下载地址仍指向 GitHub Release，保持镜像重写规则可用
	downloadURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/%s",
		s.repoOwner,
		s.repoName,
		neturl.PathEscape(tag),
		neturl.PathEscape(selected.Name),
	)

	sha256Digest := parseSHA256Digest(selected.Hash)
	asset := models.FrpcReleaseAsset{
		Name:          selected.Name,
		DownloadURL:   downloadURL,
		Digest:        selected.Hash,
		SHA256:        sha256Digest,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		ArchiveFormat: archiveFormat,
	}

	return &models.FrpcReleaseInfo{
		TagName: tag,
		Name:    strings.TrimSpace(release.Version),
		HTMLURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", s.repoOwner, s.repoName, neturl.PathEscape(tag)),
		Asset:   asset,
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

func (s *FrpcService) emitInstallProgress(phase string, downloaded, total int64) {
	ctx := System().ctx
	if ctx == nil {
		return
	}

	percent := 0.0
	if total > 0 {
		percent = float64(downloaded) / float64(total) * 100
		if percent > 100 {
			percent = 100
		}
	}

	wruntime.EventsEmit(ctx, frpcInstallProgressEvent, map[string]interface{}{
		"phase":      phase,
		"downloaded": downloaded,
		"total":      total,
		"percent":    percent,
	})
}

type progressWriter struct {
	total      int64
	downloaded int64
	lastPct    int
	lastEmit   time.Time
	onProgress func(downloaded, total int64)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.downloaded += int64(n)

	pct := -1
	if w.total > 0 {
		pct = int(w.downloaded * 100 / w.total)
	}

	now := time.Now()
	if pct != w.lastPct || now.Sub(w.lastEmit) >= progressEmitInterval {
		w.lastPct = pct
		w.lastEmit = now
		if w.onProgress != nil {
			w.onProgress(w.downloaded, w.total)
		}
	}
	return n, nil
}

func (s *FrpcService) downloadArchive(ctx context.Context, url string, outputPath string, expectedSize int64) (string, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, defaultFrpcDownloadTimeout)
	defer cancel()

	downloadURL, err := s.resolveDownloadURL(strings.TrimSpace(url))
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("User-Agent", httpclient.ResolveUserAgent(""))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download release asset: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

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

	total := resp.ContentLength
	if total <= 0 {
		total = expectedSize
	}

	hasher := sha256.New()
	progress := &progressWriter{
		total: total,
		onProgress: func(downloaded, total int64) {
			s.emitInstallProgress(installPhaseDownloading, downloaded, total)
		},
	}
	s.emitInstallProgress(installPhaseDownloading, 0, total)

	if _, err := io.Copy(io.MultiWriter(file, hasher, progress), resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("write archive file: %w", err)
	}

	// Ensure a final 100% frame is delivered even if the last chunk was throttled.
	s.emitInstallProgress(installPhaseDownloading, progress.downloaded, total)
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
	defer func() {
		_ = file.Close()
	}()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read gzip archive: %w", err)
	}
	defer func() {
		_ = gzReader.Close()
	}()

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
	defer func() {
		_ = zipReader.Close()
	}()

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
		defer func() {
			_ = reader.Close()
		}()

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

func builtinMirrorPresets() []models.FrpcMirrorPreset {
	presets := make([]models.FrpcMirrorPreset, 0, len(defaultBuiltinFrpcMirrors))
	for _, preset := range defaultBuiltinFrpcMirrors {
		presets = append(presets, models.FrpcMirrorPreset{
			ID:          strings.TrimSpace(preset.ID),
			Name:        strings.TrimSpace(preset.Name),
			Description: strings.TrimSpace(preset.Description),
			BaseURL:     strings.TrimSpace(preset.BaseURL),
			URLTemplate: strings.TrimSpace(preset.URLTemplate),
		})
	}
	return presets
}

func findBuiltinMirrorPreset(id string) (models.FrpcMirrorPreset, bool) {
	needle := strings.TrimSpace(id)
	if needle == "" {
		return models.FrpcMirrorPreset{}, false
	}

	for _, preset := range builtinMirrorPresets() {
		if preset.ID == needle {
			return preset, true
		}
	}
	return models.FrpcMirrorPreset{}, false
}

func normalizeStoredMirrorConfig(settings frpcUserSettings) models.FrpcMirrorConfig {
	if strings.TrimSpace(settings.MirrorConfig.Mode) == "" {
		return legacyMirrorConfig(settings.GitHubMirrorURL)
	}

	config, err := normalizeMirrorConfig(settings.MirrorConfig)
	if err != nil {
		return models.FrpcMirrorConfig{Mode: mirrorModeOfficial}
	}
	return config
}

func legacyMirrorConfig(rawURL string) models.FrpcMirrorConfig {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return models.FrpcMirrorConfig{Mode: mirrorModeOfficial}
	}

	for _, preset := range builtinMirrorPresets() {
		if strings.TrimSpace(preset.BaseURL) != "" && sameNormalizedURL(preset.BaseURL, trimmed) {
			return models.FrpcMirrorConfig{
				Mode:     mirrorModeBuiltin,
				PresetID: preset.ID,
			}
		}
	}

	return models.FrpcMirrorConfig{
		Mode:          mirrorModeCustom,
		CustomBaseURL: trimmed,
	}
}

func legacyMirrorURLFromConfig(config models.FrpcMirrorConfig) string {
	switch strings.TrimSpace(config.Mode) {
	case mirrorModeBuiltin:
		preset, ok := findBuiltinMirrorPreset(config.PresetID)
		if !ok {
			return ""
		}
		return strings.TrimSpace(preset.BaseURL)
	case mirrorModeCustom:
		return strings.TrimSpace(config.CustomBaseURL)
	default:
		return ""
	}
}

func normalizeMirrorURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", nil
	}

	parsed, err := neturl.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid mirror base url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("mirror base url must start with http:// or https://")
	}
	return strings.TrimRight(trimmed, "/"), nil
}

func normalizeMirrorTemplate(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return "", fmt.Errorf("mirror url template must start with http:// or https://")
	}
	if !containsSupportedTemplateToken(trimmed) {
		return "", fmt.Errorf("mirror url template must contain at least one supported placeholder")
	}
	if _, err := applyMirrorTemplate("https://github.com/example/repo/releases/download/test.zip?download=1", trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

func normalizeMirrorConfig(config models.FrpcMirrorConfig) (models.FrpcMirrorConfig, error) {
	mode := strings.TrimSpace(config.Mode)
	if mode == "" {
		mode = mirrorModeOfficial
	}

	normalized := models.FrpcMirrorConfig{
		Mode: mode,
	}

	switch mode {
	case mirrorModeOfficial:
		return normalized, nil
	case mirrorModeBuiltin:
		normalized.PresetID = strings.TrimSpace(config.PresetID)
		if normalized.PresetID == "" {
			return models.FrpcMirrorConfig{}, fmt.Errorf("builtin mirror preset is required")
		}
		if _, ok := findBuiltinMirrorPreset(normalized.PresetID); !ok {
			return models.FrpcMirrorConfig{}, fmt.Errorf("unknown builtin mirror preset: %s", normalized.PresetID)
		}
		return normalized, nil
	case mirrorModeCustom:
		baseURL, err := normalizeMirrorURL(config.CustomBaseURL)
		if err != nil {
			return models.FrpcMirrorConfig{}, err
		}
		urlTemplate, err := normalizeMirrorTemplate(config.CustomURLTemplate)
		if err != nil {
			return models.FrpcMirrorConfig{}, err
		}
		if baseURL == "" && urlTemplate == "" {
			return models.FrpcMirrorConfig{}, fmt.Errorf("custom mirror base url or url template is required")
		}
		if baseURL != "" && urlTemplate != "" {
			return models.FrpcMirrorConfig{}, fmt.Errorf("custom mirror base url and url template cannot both be set")
		}
		normalized.CustomBaseURL = baseURL
		normalized.CustomURLTemplate = urlTemplate
		return normalized, nil
	default:
		return models.FrpcMirrorConfig{}, fmt.Errorf("unsupported mirror mode: %s", mode)
	}
}

func (s *FrpcService) resolveDownloadURL(rawURL string) (string, error) {
	config, err := s.GetMirrorConfig()
	if err != nil {
		return "", err
	}
	return resolveMirrorURL(rawURL, config)
}

func resolveMirrorURL(rawURL string, config models.FrpcMirrorConfig) (string, error) {
	urlValue := strings.TrimSpace(rawURL)
	if urlValue == "" {
		return "", fmt.Errorf("download url is empty")
	}

	switch strings.TrimSpace(config.Mode) {
	case "", mirrorModeOfficial:
		return urlValue, nil
	case mirrorModeBuiltin:
		preset, ok := findBuiltinMirrorPreset(config.PresetID)
		if !ok {
			return "", fmt.Errorf("unknown builtin mirror preset: %s", config.PresetID)
		}
		return applyMirrorDefinition(urlValue, preset.BaseURL, preset.URLTemplate)
	case mirrorModeCustom:
		return applyMirrorDefinition(urlValue, config.CustomBaseURL, config.CustomURLTemplate)
	default:
		return "", fmt.Errorf("unsupported mirror mode: %s", config.Mode)
	}
}

func applyMirrorDefinition(rawURL, baseURL, urlTemplate string) (string, error) {
	if strings.TrimSpace(urlTemplate) != "" {
		return applyMirrorTemplate(rawURL, urlTemplate)
	}
	if strings.TrimSpace(baseURL) != "" {
		return applyMirrorURL(rawURL, baseURL), nil
	}
	return "", fmt.Errorf("mirror definition is empty")
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

func applyMirrorTemplate(rawURL, urlTemplate string) (string, error) {
	urlValue := strings.TrimSpace(rawURL)
	template := strings.TrimSpace(urlTemplate)
	if urlValue == "" {
		return "", fmt.Errorf("download url is empty")
	}
	if template == "" {
		return "", fmt.Errorf("mirror url template is empty")
	}

	parsedURL, err := neturl.Parse(urlValue)
	if err != nil || parsedURL == nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid download url")
	}

	owner := ""
	repo := ""
	tag := ""
	asset := ""
	if releaseInfo, ok := parseGitHubReleaseDownloadURL(parsedURL); ok {
		owner = releaseInfo.Owner
		repo = releaseInfo.Repo
		tag = releaseInfo.Tag
		asset = releaseInfo.Asset
	}

	replaced := strings.NewReplacer(
		"{owner}", owner,
		"{repo}", repo,
		"{tag}", tag,
		"{asset}", asset,
	).Replace(template)

	parsedResult, parseErr := neturl.Parse(replaced)
	if parseErr != nil || parsedResult == nil || parsedResult.Scheme == "" || parsedResult.Host == "" {
		return "", fmt.Errorf("mirror url template resolved to invalid url")
	}

	if parsedResult.Scheme != "http" && parsedResult.Scheme != "https" {
		return "", fmt.Errorf("mirror url template resolved to non-http url")
	}

	return replaced, nil
}

func containsSupportedTemplateToken(template string) bool {
	supportedTokens := []string{
		"{owner}",
		"{repo}",
		"{tag}",
		"{asset}",
	}
	for _, token := range supportedTokens {
		if strings.Contains(template, token) {
			return true
		}
	}
	return false
}

type gitHubReleaseDownloadInfo struct {
	Owner string
	Repo  string
	Tag   string
	Asset string
}

func parseGitHubReleaseDownloadURL(parsedURL *neturl.URL) (gitHubReleaseDownloadInfo, bool) {
	if parsedURL == nil {
		return gitHubReleaseDownloadInfo{}, false
	}

	segments := strings.Split(strings.Trim(parsedURL.EscapedPath(), "/"), "/")
	if len(segments) < 5 {
		return gitHubReleaseDownloadInfo{}, false
	}
	if segments[2] != "releases" || segments[3] != "download" {
		return gitHubReleaseDownloadInfo{}, false
	}

	return gitHubReleaseDownloadInfo{
		Owner: segments[0],
		Repo:  segments[1],
		Tag:   segments[4],
		Asset: segments[len(segments)-1],
	}, true
}

func sameNormalizedURL(left, right string) bool {
	leftURL, leftErr := normalizeMirrorURL(left)
	rightURL, rightErr := normalizeMirrorURL(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftURL == rightURL
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
		return fmt.Errorf("frpc 下载超时，可尝试切换下载源后重试")
	}
	return err
}

// normalizeDownloadError 专用于下载阶段的错误：除终止外，一律附带切换下载源的提示。
func normalizeDownloadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("frpc 下载已终止")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("frpc 下载超时，可尝试切换下载源后重试")
	}
	return fmt.Errorf("frpc 下载失败：%v，可尝试切换下载源后重试", err)
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
