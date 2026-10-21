//go:build !android

package services

func onRunnerStarted(int) {}

func onRunnerStopped() {}

// runnerExtraEnv 桌面端 frpc 可直接读取系统 DNS，无需额外环境变量。
func runnerExtraEnv(string) []string { return nil }
