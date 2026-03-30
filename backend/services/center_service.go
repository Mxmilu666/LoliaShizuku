package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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
)

const (
	defaultCenterAPIBaseURL = "https://api.lolia.link/api/v1"
	defaultHTTPTimeout      = 20 * time.Second
	runnerLogMaxLines       = 300
	runnerStopTimeout       = 3 * time.Second
)

type CenterService struct {
	api *api.CenterAPI

	runnerMu          sync.Mutex
	runnerCmd         *exec.Cmd
	runnerCancel      context.CancelFunc
	runnerStartedAt   time.Time
	runnerTunnelName  string
	runnerTunnelNames []string
	runnerNodeAddress string
	runnerCommand     string
	runnerLastError   string
	runnerLogs        []string
	runnerStopping    bool
}

func NewCenterService() *CenterService {
	service := &CenterService{}

	client := httpclient.New(httpclient.Options{
		BaseURL:    centerAPIBaseURL(),
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
		GetAccessToken: func(ctx context.Context) (string, error) {
			return service.getValidAccessToken(ctx)
		},
		OnUnauthorized: func(ctx context.Context) error {
			return ClearOAuthToken()
		},
	})

	service.api = api.NewCenterAPI(client)
	return service
}

func centerAPIBaseURL() string {
	baseURL := strings.TrimSpace(os.Getenv("LOLIA_CENTER_API_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultCenterAPIBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

func (s *CenterService) getValidAccessToken(ctx context.Context) (string, error) {
	refreshCtx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()

	token, err := loadOrRefreshOAuthToken(refreshCtx)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func (s *CenterService) GetDashboard() (*models.CenterDashboardData, error) {
	ctx := context.Background()

	user, err := s.api.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	traffic, err := s.api.GetUserTrafficStats(ctx)
	if err != nil {
		return nil, err
	}
	tunnelList, err := s.api.GetUserTunnels(ctx, 1, 20)
	if err != nil {
		return nil, err
	}
	nodeMetaByID := map[int64]models.NodeItem{}
	nodes, nodesErr := s.api.GetNodes(ctx)
	if nodesErr == nil {
		for _, node := range nodes.Nodes {
			nodeMetaByID[node.ID] = node
		}
	}
	version, err := s.api.GetClientVersion(ctx)
	if err != nil {
		return nil, err
	}
	homeStats, err := s.api.GetHomeStats(ctx)
	if err != nil {
		return nil, err
	}

	data := &models.CenterDashboardData{
		User:    *user,
		Traffic: *traffic,
		Tunnel: models.UserTunnelSummary{
			Count: int64(len(tunnelList.List)),
			Total: tunnelList.Total,
		},
		Tunnels:   enrichTunnelNodeMeta(tunnelList.List, nodeMetaByID),
		App:       *version,
		HomeStats: *homeStats,
	}
	return data, nil
}

func (s *CenterService) GetTunnelsOverview(page, limit, days int) (*models.TunnelOverviewData, error) {
	ctx := context.Background()

	tunnelList, err := s.api.GetUserTunnels(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	trafficByName := map[string]models.TrafficTunnelItem{}
	nodeMetaByID := map[int64]models.NodeItem{}
	if days > 0 {
		traffic, trafficErr := s.api.GetTrafficTunnels(ctx, days)
		if trafficErr == nil {
			for _, item := range traffic.Tunnels {
				trafficByName[strings.TrimSpace(item.TunnelName)] = item
			}
		}
	}
	nodes, nodesErr := s.api.GetNodes(ctx)
	if nodesErr == nil {
		for _, node := range nodes.Nodes {
			nodeMetaByID[node.ID] = node
		}
	}

	enriched := make([]models.TunnelItem, 0, len(tunnelList.List))
	for _, tunnel := range tunnelList.List {
		current := tunnel
		if node, ok := nodeMetaByID[tunnel.NodeID]; ok {
			if strings.TrimSpace(current.NodeAddress) == "" {
				current.NodeAddress = strings.TrimSpace(node.IPAddress)
			}
			if strings.TrimSpace(current.NodeName) == "" {
				current.NodeName = strings.TrimSpace(node.Name)
			}
		}
		if traffic, ok := trafficByName[strings.TrimSpace(tunnel.Name)]; ok {
			current.TotalIn = traffic.TotalIn
			current.TotalOut = traffic.TotalOut
			current.TotalTraffic = traffic.TotalTraffic
		}
		enriched = append(enriched, current)
	}

	return &models.TunnelOverviewData{
		List:      enriched,
		Page:      tunnelList.Page,
		Limit:     tunnelList.Limit,
		Total:     tunnelList.Total,
		TotalPage: tunnelList.TotalPage,
	}, nil
}

func (s *CenterService) GetRunnerData(tunnelID int64) (*models.RunnerData, error) {
	ctx := context.Background()

	version, err := s.api.GetClientVersion(ctx)
	if err != nil {
		return nil, err
	}
	nodes, err := s.api.GetNodes(ctx)
	if err != nil {
		return nil, err
	}
	tunnels, err := s.api.GetUserTunnels(ctx, 1, 100)
	if err != nil {
		return nil, err
	}

	var selectedTunnel *models.TunnelItem
	if tunnelID > 0 {
		for _, item := range tunnels.List {
			if item.ID == tunnelID {
				copyItem := item
				selectedTunnel = &copyItem
				break
			}
		}
	}
	if selectedTunnel == nil && len(tunnels.List) > 0 {
		copyItem := tunnels.List[0]
		selectedTunnel = &copyItem
	}
	if selectedTunnel != nil {
		for _, node := range nodes.Nodes {
			if node.ID == selectedTunnel.NodeID {
				if strings.TrimSpace(selectedTunnel.NodeAddress) == "" {
					selectedTunnel.NodeAddress = strings.TrimSpace(node.IPAddress)
				}
				if strings.TrimSpace(selectedTunnel.NodeName) == "" {
					selectedTunnel.NodeName = strings.TrimSpace(node.Name)
				}
				break
			}
		}
	}

	return &models.RunnerData{
		Config:        "",
		Version:       version.Version,
		Nodes:         nodes.Nodes,
		CurrentTunnel: selectedTunnel,
	}, nil
}

func (s *CenterService) StartRunner(tunnelNames []string) (*models.RunnerRuntimeStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHTTPTimeout)
	defer cancel()

	selectedTunnelNames := normalizeTunnelNames(tunnelNames)
	s.runnerMu.Lock()
	currentlyRunning := s.isRunnerRunningLocked()
	existingTunnelNames := append([]string(nil), s.runnerTunnelNames...)
	if len(existingTunnelNames) == 0 && strings.TrimSpace(s.runnerTunnelName) != "" {
		existingTunnelNames = []string{strings.TrimSpace(s.runnerTunnelName)}
	}
	currentStatus := s.buildRunnerStatusLocked()
	s.runnerMu.Unlock()

	if currentlyRunning {
		mergedTunnelNames := mergeTunnelNames(existingTunnelNames, selectedTunnelNames)
		if len(mergedTunnelNames) == len(existingTunnelNames) {
			return currentStatus, nil
		}
		if _, err := s.StopRunner(); err != nil {
			return nil, err
		}
		selectedTunnelNames = mergedTunnelNames
	}

	if len(selectedTunnelNames) == 0 {
		tunnels, err := s.api.GetUserTunnels(ctx, 1, 100)
		if err != nil {
			return nil, err
		}
		if len(tunnels.List) == 0 {
			return nil, fmt.Errorf("当前账号暂无隧道，无法启动 frpc")
		}
		selectedTunnelNames = []string{strings.TrimSpace(tunnels.List[0].Name)}
	}
	if len(selectedTunnelNames) == 0 {
		return nil, fmt.Errorf("无效的隧道名称")
	}

	tokenArgs := make([]string, 0, len(selectedTunnelNames))
	resolvedTunnelNames := make([]string, 0, len(selectedTunnelNames))
	nodeAddresses := make([]string, 0, len(selectedTunnelNames))
	for _, selectedTunnelName := range selectedTunnelNames {
		tunnelDetail, err := s.api.GetTunnelDetail(ctx, selectedTunnelName)
		if err != nil {
			return nil, err
		}
		if tunnelDetail == nil {
			return nil, fmt.Errorf("获取隧道详情失败：%s", selectedTunnelName)
		}
		if tunnelDetail.ID <= 0 {
			return nil, fmt.Errorf("隧道详情缺少有效 id：%s", selectedTunnelName)
		}

		token := strings.TrimSpace(tunnelDetail.TunnelToken)
		if token == "" {
			return nil, fmt.Errorf("隧道详情未返回 tunnel_token：%s", selectedTunnelName)
		}

		tokenArgs = append(tokenArgs, fmt.Sprintf("%d:%s", tunnelDetail.ID, token))
		resolvedTunnelNames = append(resolvedTunnelNames, strings.TrimSpace(tunnelDetail.Name))
		nodeAddresses = append(nodeAddresses, strings.TrimSpace(tunnelDetail.NodeAddress))
	}

	binaryPath, err := resolveLocalFrpcBinaryPath()
	if err != nil {
		return nil, err
	}
	exists, err := fileExistsForRunner(binaryPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("frpc 未安装，请先在设置页面安装: %s", binaryPath)
	}

	s.runnerMu.Lock()
	if s.isRunnerRunningLocked() {
		status := s.buildRunnerStatusLocked()
		s.runnerMu.Unlock()
		return status, fmt.Errorf("runner 已在运行中")
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	cmdArgs := make([]string, 0, len(tokenArgs)*2)
	for _, tokenArg := range tokenArgs {
		cmdArgs = append(cmdArgs, "-t", tokenArg)
	}
	cmd := exec.CommandContext(runCtx, binaryPath, cmdArgs...)
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

	s.runnerCmd = cmd
	s.runnerCancel = runCancel
	s.runnerStartedAt = time.Now().UTC()
	s.runnerTunnelName = firstNonEmptyString(resolvedTunnelNames...)
	s.runnerTunnelNames = append([]string(nil), resolvedTunnelNames...)
	s.runnerNodeAddress = firstNonEmptyString(nodeAddresses...)
	s.runnerCommand = buildMaskedRunnerCommand(binaryPath, tokenArgs)
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

	return status, nil
}

func (s *CenterService) StopRunner() (*models.RunnerRuntimeStatus, error) {
	s.runnerMu.Lock()
	if !s.isRunnerRunningLocked() {
		status := s.buildRunnerStatusLocked()
		s.runnerMu.Unlock()
		return status, nil
	}

	cmd := s.runnerCmd
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

func (s *CenterService) GetRunnerRuntimeStatus() (*models.RunnerRuntimeStatus, error) {
	s.runnerMu.Lock()
	defer s.runnerMu.Unlock()
	return s.buildRunnerStatusLocked(), nil
}

func (s *CenterService) GetUserInfo() (*models.UserInfoData, error) {
	return s.api.GetUserInfo(context.Background())
}

func (s *CenterService) GetUserTrafficStats() (*models.UserTrafficData, error) {
	return s.api.GetUserTrafficStats(context.Background())
}

func (s *CenterService) GetUserTunnels(page, limit int) (*models.TunnelListData, error) {
	return s.api.GetUserTunnels(context.Background(), page, limit)
}

func (s *CenterService) GetTrafficTunnels(days int) (*models.TrafficTunnelData, error) {
	return s.api.GetTrafficTunnels(context.Background(), days)
}

func (s *CenterService) GetTrafficDaily(days int) (*models.DailyTrafficResponse, error) {
	if days <= 0 {
		days = 7
	}
	return s.api.GetTrafficDaily(context.Background(), days)
}

func (s *CenterService) GetNodes() (*models.NodeListData, error) {
	return s.api.GetNodes(context.Background())
}

func (s *CenterService) GetFrpcConfig(tunnel string) (*models.FrpcConfigData, error) {
	return s.api.GetFrpcConfig(context.Background(), tunnel)
}

func (s *CenterService) GetTunnelDetail(tunnelName string) (*models.TunnelDetailData, error) {
	return s.api.GetTunnelDetail(context.Background(), tunnelName)
}

func (s *CenterService) GetClientVersion() (*models.AppVersionInfo, error) {
	return s.api.GetClientVersion(context.Background())
}

func (s *CenterService) GetHomeStats() (*models.HomeStatsData, error) {
	return s.api.GetHomeStats(context.Background())
}

func enrichTunnelNodeMeta(tunnels []models.TunnelItem, nodeMetaByID map[int64]models.NodeItem) []models.TunnelItem {
	if len(tunnels) == 0 {
		return nil
	}

	enriched := make([]models.TunnelItem, 0, len(tunnels))
	for _, tunnel := range tunnels {
		current := tunnel
		if node, ok := nodeMetaByID[tunnel.NodeID]; ok {
			if strings.TrimSpace(current.NodeAddress) == "" {
				current.NodeAddress = strings.TrimSpace(node.IPAddress)
			}
			if strings.TrimSpace(current.NodeName) == "" {
				current.NodeName = strings.TrimSpace(node.Name)
			}
		}
		enriched = append(enriched, current)
	}
	return enriched
}

func (s *CenterService) isRunnerRunningLocked() bool {
	if s.runnerCmd == nil || s.runnerCmd.Process == nil {
		return false
	}
	if s.runnerCmd.ProcessState == nil {
		return true
	}
	return !s.runnerCmd.ProcessState.Exited()
}

func (s *CenterService) buildRunnerStatusLocked() *models.RunnerRuntimeStatus {
	status := &models.RunnerRuntimeStatus{
		Running:     s.isRunnerRunningLocked(),
		Command:     s.runnerCommand,
		LastError:   s.runnerLastError,
		TunnelName:  s.runnerTunnelName,
		TunnelNames: append([]string(nil), s.runnerTunnelNames...),
		NodeAddress: s.runnerNodeAddress,
	}

	if !s.runnerStartedAt.IsZero() {
		status.StartedAt = s.runnerStartedAt.Format(time.RFC3339)
	}
	if s.runnerCmd != nil && s.runnerCmd.Process != nil {
		status.PID = s.runnerCmd.Process.Pid
	}
	if len(s.runnerLogs) > 0 {
		status.LogLines = append([]string(nil), s.runnerLogs...)
	}
	return status
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

	if s.runnerCmd == cmd {
		s.runnerCmd = nil
	}
	s.runnerCancel = nil
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

func (s *CenterService) appendRunnerLogLocked(line string) {
	s.runnerLogs = append(s.runnerLogs, line)
	if len(s.runnerLogs) > runnerLogMaxLines {
		s.runnerLogs = append([]string(nil), s.runnerLogs[len(s.runnerLogs)-runnerLogMaxLines:]...)
	}
}

func resolveLocalFrpcBinaryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("获取配置目录失败: %w", err)
	}

	path := filepath.Join(
		configDir,
		"LoliaShizuku",
		"userdata",
		"frpc",
		"bin",
		runnerFrpcBinaryName(),
	)
	return path, nil
}

func runnerFrpcBinaryName() string {
	if runtime.GOOS == "windows" {
		return "frpc.exe"
	}
	return "frpc"
}

func fileExistsForRunner(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("检查文件失败 %s: %w", path, err)
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

func normalizeTunnelNames(tunnelNames []string) []string {
	if len(tunnelNames) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(tunnelNames))
	seen := make(map[string]struct{}, len(tunnelNames))
	for _, tunnelName := range tunnelNames {
		trimmed := strings.TrimSpace(tunnelName)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func mergeTunnelNames(current []string, requested []string) []string {
	merged := make([]string, 0, len(current)+len(requested))
	seen := make(map[string]struct{}, len(current)+len(requested))

	appendNames := func(names []string) {
		for _, name := range names {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			merged = append(merged, trimmed)
		}
	}

	appendNames(current)
	appendNames(requested)
	return merged
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
