//go:build !android

package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var runnerTokenArgPattern = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9._~+/=-]+$`)

// buildRunnerArgs 桌面端：frpc -t id:token，由 frpc 自行向中心端拉取配置。
func (s *CenterService) buildRunnerArgs(_ context.Context, binaryPath string, targets []runnerTarget) ([]string, string, error) {
	tokenArgs := make([]string, 0, len(targets))
	args := make([]string, 0, len(targets)*2)
	for _, target := range targets {
		tokenArg := fmt.Sprintf("%d:%s", target.ID, target.Token)
		if !runnerTokenArgPattern.MatchString(tokenArg) {
			return nil, "", fmt.Errorf("非法的 tunnel token 参数")
		}
		tokenArgs = append(tokenArgs, tokenArg)
		args = append(args, "-t", tokenArg)
	}
	return args, buildMaskedRunnerCommand(binaryPath, tokenArgs), nil
}

func maskRunnerTokenArg(tokenArg string) string {
	parts := strings.SplitN(tokenArg, ":", 2)
	if len(parts) != 2 {
		return tokenArg
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return parts[0] + ":***"
	}
	if len(token) <= 8 {
		return parts[0] + ":***"
	}

	return fmt.Sprintf("%s:%s***%s", parts[0], token[:4], token[len(token)-4:])
}

func buildMaskedRunnerCommand(binaryPath string, tokenArgs []string) string {
	if len(tokenArgs) == 0 {
		return binaryPath
	}

	parts := make([]string, 0, 1+len(tokenArgs)*2)
	parts = append(parts, binaryPath)
	for _, tokenArg := range tokenArgs {
		parts = append(parts, "-t", maskRunnerTokenArg(tokenArg))
	}
	return strings.Join(parts, " ")
}
