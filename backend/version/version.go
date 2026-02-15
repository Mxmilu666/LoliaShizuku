package version

import (
	"fmt"
	"runtime"
	"strings"
)

// These variables are expected to be overridden by -ldflags during build.
var (
	AppName   = "LoliaShizuku"
	Version   = "0.0.2"
	GitCommit = "dev"
	GitBranch = "dev"
	BuildTime = "unknown"
)

type Info struct {
	AppName   string `json:"app_name"`
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

func GetInfo() Info {
	return Info{
		AppName:   AppName,
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func ShortCommit() string {
	commit := strings.TrimSpace(GitCommit)
	if len(commit) > 7 {
		return commit[:7]
	}
	if commit == "" {
		return "unknown"
	}
	return commit
}

func FullVersion() string {
	info := GetInfo()
	return fmt.Sprintf("%s %s (%s)", info.AppName, info.Version, ShortCommit())
}

func UserAgent() string {
	info := GetInfo()
	return fmt.Sprintf("%s/%s (%s; %s)", info.AppName, info.Version, info.Platform, ShortCommit())
}
