package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mxmilu666/LoliaShizuku/backend/models"
)

// 以子进程方式运行已安装的 frpc 二进制（frpc -t id:token）。
// Android 端同样如此：应用 targetSdkVersion 为 28（与 Termux 相同），允许执行应用私有目录中的二进制。

type runnerPlatformState struct {
	cmd *exec.Cmd
}

func (s *CenterService) runnerPIDLocked() int {
	if s.runner.cmd != nil && s.runner.cmd.Process != nil {
		return s.runner.cmd.Process.Pid
	}
	return 0
}

func (s *CenterService) StartRunner(tunnelNames []string) (*models.RunnerRuntimeStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	targets, currentStatus, err := s.prepareRunnerStart(ctx, tunnelNames)
	if err != nil {
		return nil, err
	}
	if targets == nil {
		return currentStatus, nil
	}

	resolvedTunnelNames := make([]string, 0, len(targets))
	nodeAddresses := make([]string, 0, len(targets))
	for _, target := range targets {
		resolvedTunnelNames = append(resolvedTunnelNames, target.Name)
		nodeAddresses = append(nodeAddresses, target.NodeAddress)
	}

	binaryPath, err := resolveTrustedRunnerBinaryPath()
	if err != nil {
		return nil, err
	}

	// 启动参数因平台而异：桌面端 -t id:token；Android 端 -c 配置文件（见 runner_args_*.go）
	cmdArgs, displayCommand, err := s.buildRunnerArgs(ctx, binaryPath, targets)
	if err != nil {
		return nil, err
	}

	s.runnerMu.Lock()
	if s.isRunnerRunningLocked() {
		status := s.buildRunnerStatusLocked()
		s.runnerMu.Unlock()
		return status, fmt.Errorf("runner 已在运行中")
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	// #nosec G204 -- binaryPath is constrained to the managed frpc install path and cmdArgs are built by buildRunnerArgs from validated inputs.
	cmd := exec.CommandContext(runCtx, binaryPath, cmdArgs...)
	// 输出经管道读取时 frpc 默认不着色，强制输出 ANSI 颜色，由前端渲染日志高亮
	cmd.Env = append(os.Environ(), "CLICOLOR_FORCE=1")
	configureBackgroundProcess(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		runCancel()
		s.runnerMu.Unlock()
		return nil, fmt.Errorf("打开 frpc stdout 失败: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		runCancel()
		s.runnerMu.Unlock()
		return nil, fmt.Errorf("打开 frpc stderr 失败: %w", err)
	}

	if err := cmd.Start(); err != nil {
		runCancel()
		s.runnerMu.Unlock()
		return nil, fmt.Errorf("启动 frpc 失败: %w", err)
	}

	s.runner.cmd = cmd
	s.runnerCancel = runCancel
	s.runnerStartedAt = time.Now().UTC()
	s.runnerTunnelName = firstNonEmptyString(resolvedTunnelNames...)
	s.runnerTunnelNames = append([]string(nil), resolvedTunnelNames...)
	s.runnerNodeAddress = firstNonEmptyString(nodeAddresses...)
	s.runnerCommand = displayCommand
	s.runnerLastError = ""
	s.runnerLogs = []string{
		fmt.Sprintf("[runner] started: pid=%d", cmd.Process.Pid),
	}
	s.runnerStopping = false
	status := s.buildRunnerStatusLocked()
	s.runnerMu.Unlock()

	go s.consumeRunnerOutput(stdout)
	go s.consumeRunnerOutput(stderr)
	go s.waitRunnerExit(cmd)
	onRunnerStarted(len(targets))

	return status, nil
}

func (s *CenterService) StopRunner() (*models.RunnerRuntimeStatus, error) {
	s.runnerMu.Lock()
	if !s.isRunnerRunningLocked() {
		status := s.buildRunnerStatusLocked()
		s.runnerMu.Unlock()
		return status, nil
	}

	cmd := s.runner.cmd
	cancel := s.runnerCancel
	s.runnerStopping = true
	s.runnerMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}

	deadline := time.Now().Add(runnerStopTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		s.runnerMu.Lock()
		running := s.isRunnerRunningLocked()
		s.runnerMu.Unlock()
		if !running {
			break
		}
	}

	s.runnerMu.Lock()
	shouldKill := s.isRunnerRunningLocked() && cmd != nil && cmd.Process != nil
	s.runnerMu.Unlock()

	if shouldKill {
		_ = cmd.Process.Kill()
	}

	deadline = time.Now().Add(runnerStopTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		s.runnerMu.Lock()
		running := s.isRunnerRunningLocked()
		s.runnerMu.Unlock()
		if !running {
			break
		}
	}

	s.runnerMu.Lock()
	status := s.buildRunnerStatusLocked()
	s.runnerMu.Unlock()
	return status, nil
}

func (s *CenterService) isRunnerRunningLocked() bool {
	if s.runner.cmd == nil || s.runner.cmd.Process == nil {
		return false
	}
	if s.runner.cmd.ProcessState == nil {
		return true
	}
	return !s.runner.cmd.ProcessState.Exited()
}

func (s *CenterService) waitRunnerExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	s.runnerMu.Lock()
	defer s.runnerMu.Unlock()

	wasStopping := s.runnerStopping
	s.runnerStopping = false

	if err != nil && !wasStopping && !errors.Is(err, context.Canceled) {
		s.runnerLastError = err.Error()
		s.appendRunnerLogLocked("[runner] exited with error: " + err.Error())
	} else {
		s.appendRunnerLogLocked("[runner] exited")
	}

	if s.runner.cmd == cmd {
		s.runner.cmd = nil
	}
	s.runnerCancel = nil
	onRunnerStopped()
}

func (s *CenterService) consumeRunnerOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s.runnerMu.Lock()
		s.appendRunnerLogLocked(line)
		s.runnerMu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		s.runnerMu.Lock()
		s.appendRunnerLogLocked("[runner] log read error: " + err.Error())
		s.runnerMu.Unlock()
	}
}

func resolveTrustedRunnerBinaryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("获取配置目录失败: %w", err)
	}

	installDir := filepath.Join(
		configDir,
		"LoliaShizuku",
		"userdata",
		"frpc",
		"bin",
	)
	binaryName := runnerFrpcBinaryName()
	path := filepath.Clean(filepath.Join(installDir, binaryName))
	expectedDir := filepath.Clean(installDir)
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("frpc 路径必须为绝对路径: %s", path)
	}
	if filepath.Base(path) != binaryName {
		return "", fmt.Errorf("非法的 frpc 文件名: %s", path)
	}
	if filepath.Dir(path) != expectedDir {
		return "", fmt.Errorf("非法的 frpc 安装目录: %s", path)
	}

	ok, err := fileExistsForRunner(path)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("frpc 未安装，请先在设置页面安装: %s", path)
	}

	return path, nil
}

func runnerFrpcBinaryName() string {
	if runtime.GOOS == "windows" {
		return "frpc.exe"
	}
	return "frpc"
}

func fileExistsForRunner(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("runner 可执行文件不能是符号链接: %s", path)
		}
		if info.IsDir() {
			return false, nil
		}
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("检查文件失败 %s: %w", path, err)
}
