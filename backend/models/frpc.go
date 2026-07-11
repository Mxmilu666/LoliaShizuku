package models

type ClientVersionInfo struct {
	Version string               `json:"version"`
	Tag     string               `json:"tag"`
	Assets  []ClientVersionAsset `json:"assets"`
}

type ClientVersionAsset struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

type FrpcReleaseAsset struct {
	Name          string `json:"name"`
	DownloadURL   string `json:"download_url"`
	ContentType   string `json:"content_type"`
	Size          int64  `json:"size"`
	Digest        string `json:"digest"`
	SHA256        string `json:"sha256"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	ArchiveFormat string `json:"archive_format"`
}

type FrpcReleaseInfo struct {
	TagName string           `json:"tag_name"`
	Name    string           `json:"name"`
	HTMLURL string           `json:"html_url"`
	Asset   FrpcReleaseAsset `json:"asset"`
}

type FrpcMirrorPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	BaseURL     string `json:"base_url,omitempty"`
	URLTemplate string `json:"url_template,omitempty"`
}

type FrpcMirrorConfig struct {
	Mode              string `json:"mode"`
	PresetID          string `json:"preset_id,omitempty"`
	CustomBaseURL     string `json:"custom_base_url,omitempty"`
	CustomURLTemplate string `json:"custom_url_template,omitempty"`
}

type FrpcInstalledInfo struct {
	Version      string `json:"version"`
	AssetName    string `json:"asset_name"`
	SHA256       string `json:"sha256"`
	InstalledAt  string `json:"installed_at"`
	BinaryPath   string `json:"binary_path"`
	BinaryExists bool   `json:"binary_exists"`
}

type FrpcPaths struct {
	UserDataDir  string `json:"userdata_dir"`
	FrpcDir      string `json:"frpc_dir"`
	BinDir       string `json:"bin_dir"`
	BinaryPath   string `json:"binary_path"`
	DownloadDir  string `json:"download_dir"`
	StatePath    string `json:"state_path"`
	SettingsPath string `json:"settings_path"`
}

type FrpcStatus struct {
	GOOS            string             `json:"goos"`
	GOARCH          string             `json:"goarch"`
	Paths           FrpcPaths          `json:"paths"`
	GitHubMirrorURL string             `json:"github_mirror_url"`
	MirrorConfig    FrpcMirrorConfig   `json:"mirror_config"`
	BuiltinMirrors  []FrpcMirrorPreset `json:"builtin_mirrors"`
	Installed       *FrpcInstalledInfo `json:"installed,omitempty"`
	Latest          *FrpcReleaseInfo   `json:"latest,omitempty"`
	UpdateAvailable bool               `json:"update_available"`
	LatestError     string             `json:"latest_error,omitempty"`
}

type FrpcInstallResult struct {
	Release FrpcReleaseInfo `json:"release"`
	Status  FrpcStatus      `json:"status"`
}
