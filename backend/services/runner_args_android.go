//go:build android

package services

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Android 版 frpc 以纯 Go（CGO_ENABLED=0）编译，读不到系统 DNS：
//   - frpc -t 拉取配置时只能经硬编码的 8.8.8.8 解析 api.lolia.link（国内常不可用）；
//   - 中心端下发的 DoH dnsServer 在解析 DoH 服务器自身域名时又会走 [::1]:53，必然失败。
// 因此 Android 端由应用拉取配置，把 dnsServer 改为系统 DNS 后写入文件，以 frpc -c 启动。

const (
	// 由 MainActivity 写入的系统 DNS 列表文件（每行一个地址）
	systemDNSFile = "system_dns"
	// 读不到系统 DNS 时使用的公共 DNS（与中心端下发的 DoH 同为 AliDNS）
	fallbackDNSServer = "223.5.5.5"
)

var dnsServerLinePattern = regexp.MustCompile(`(?m)^dnsServer\s*=.*$`)

func (s *CenterService) buildRunnerArgs(ctx context.Context, binaryPath string, targets []runnerTarget) ([]string, string, error) {
	// binaryPath = <frpc>/bin/frpc，配置写入 <frpc>/run/
	runDir := filepath.Join(filepath.Dir(filepath.Dir(binaryPath)), "run")
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, "", fmt.Errorf("创建 frpc 配置目录失败: %w", err)
	}

	dnsServer := resolveSystemDNSServer()
	args := make([]string, 0, len(targets)*2)
	for _, target := range targets {
		data, err := s.api.GetFrpcConfig(ctx, target.Name)
		if err != nil {
			return nil, "", apiMessageError(err)
		}
		encoded := strings.TrimSpace(data.LoliaConfig)
		if encoded == "" {
			encoded = strings.TrimSpace(data.Config)
		}
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, "", fmt.Errorf("解码隧道 %s 的配置失败: %w", target.Name, err)
		}

		// 配置包含隧道 token，仅应用自身可读
		path := filepath.Join(runDir, fmt.Sprintf("tunnel-%d.toml", target.ID))
		if err := os.WriteFile(path, []byte(withDNSServer(string(raw), dnsServer)), 0o600); err != nil {
			return nil, "", fmt.Errorf("写入隧道 %s 的配置失败: %w", target.Name, err)
		}
		args = append(args, "-c", path)
	}

	display := fmt.Sprintf("%s -c <%d 个配置文件>（DNS: %s）", filepath.Base(binaryPath), len(targets), dnsServer)
	return args, display, nil
}

// withDNSServer 将配置中的顶层 dnsServer 替换为指定地址；不存在时插入到文件开头
func withDNSServer(config, dnsServer string) string {
	line := fmt.Sprintf("dnsServer = %q", dnsServer)
	if dnsServerLinePattern.MatchString(config) {
		return dnsServerLinePattern.ReplaceAllLiteralString(config, line)
	}
	return line + "\n" + config
}

// resolveSystemDNSServer 读取 MainActivity 写入的系统 DNS，优先 IPv4；读不到时使用公共 DNS
func resolveSystemDNSServer() string {
	file, err := os.Open(filepath.Join(application.Android.StoragePath(), systemDNSFile))
	if err != nil {
		return fallbackDNSServer
	}
	defer func() {
		_ = file.Close()
	}()

	var firstIPv6 string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ip := net.ParseIP(strings.TrimSpace(scanner.Text()))
		// 跳过无法解析与带 zone 的链路本地地址
		if ip == nil || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip.To4() != nil {
			return ip.String()
		}
		if firstIPv6 == "" {
			firstIPv6 = net.JoinHostPort(ip.String(), "53")
		}
	}
	if firstIPv6 != "" {
		return firstIPv6
	}
	return fallbackDNSServer
}
