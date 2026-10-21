package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	runner            runnerPlatformState
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
		RefreshAccessToken: func(ctx context.Context, rejected string) (string, error) {
			refreshCtx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
			defer cancel()
			return refreshRejectedAccessToken(refreshCtx, rejected)
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

// runnerTarget 是一条待启动隧道的运行参数
type runnerTarget struct {
	ID          int64
	Name        string
	Token       string
	NodeAddress string
}

// prepareRunnerStart 处理与已运行隧道的合并、默认隧道选择，并获取各隧道详情。
// 返回 nil targets 表示请求的隧道均已在运行，调用方应直接返回 currentStatus。
func (s *CenterService) prepareRunnerStart(
	ctx context.Context,
	tunnelNames []string,
) ([]runnerTarget, *models.RunnerRuntimeStatus, error) {
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
			return nil, currentStatus, nil
		}
		if _, err := s.StopRunner(); err != nil {
			return nil, nil, err
		}
		selectedTunnelNames = mergedTunnelNames
	}

	if len(selectedTunnelNames) == 0 {
		tunnels, err := s.api.GetUserTunnels(ctx, 1, 100)
		if err != nil {
			return nil, nil, err
		}
		if len(tunnels.List) == 0 {
			return nil, nil, fmt.Errorf("当前账号暂无隧道，无法启动 frpc")
		}
		selectedTunnelNames = []string{strings.TrimSpace(tunnels.List[0].Name)}
	}
	if len(selectedTunnelNames) == 0 {
		return nil, nil, fmt.Errorf("无效的隧道名称")
	}

	targets := make([]runnerTarget, 0, len(selectedTunnelNames))
	for _, selectedTunnelName := range selectedTunnelNames {
		tunnelDetail, err := s.api.GetTunnelDetail(ctx, selectedTunnelName)
		if err != nil {
			return nil, nil, err
		}
		if tunnelDetail == nil {
			return nil, nil, fmt.Errorf("获取隧道详情失败：%s", selectedTunnelName)
		}
		if tunnelDetail.ID <= 0 {
			return nil, nil, fmt.Errorf("隧道详情缺少有效 id：%s", selectedTunnelName)
		}

		token := strings.TrimSpace(tunnelDetail.TunnelToken)
		if token == "" {
			return nil, nil, fmt.Errorf("隧道详情未返回 tunnel_token：%s", selectedTunnelName)
		}

		targets = append(targets, runnerTarget{
			ID:          tunnelDetail.ID,
			Name:        strings.TrimSpace(tunnelDetail.Name),
			Token:       token,
			NodeAddress: strings.TrimSpace(tunnelDetail.NodeAddress),
		})
	}
	return targets, currentStatus, nil
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

// GetTunnelsPage 聚合隧道页面所需数据：隧道列表、配额、节点与近 days 天流量
func (s *CenterService) GetTunnelsPage(days int) (*models.TunnelsPageData, error) {
	ctx := context.Background()

	var (
		wg         sync.WaitGroup
		tunnelList *models.TunnelListData
		tunnelErr  error
		userInfo   *models.UserInfoData
		nodes      *models.NodeListData
		traffic    *models.TrafficTunnelData
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		tunnelList, tunnelErr = s.api.GetUserTunnels(ctx, 1, 100)
	}()
	go func() {
		defer wg.Done()
		userInfo, _ = s.api.GetUserInfo(ctx)
	}()
	go func() {
		defer wg.Done()
		nodes, _ = s.api.GetNodes(ctx)
	}()
	if days > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			traffic, _ = s.api.GetTrafficTunnels(ctx, days)
		}()
	}
	wg.Wait()

	if tunnelErr != nil {
		return nil, tunnelErr
	}

	trafficByName := map[string]models.TrafficTunnelItem{}
	if traffic != nil {
		for _, item := range traffic.Tunnels {
			trafficByName[strings.TrimSpace(item.TunnelName)] = item
		}
	}
	nodeMetaByID := map[int64]models.NodeItem{}
	var nodeItems []models.NodeItem
	if nodes != nil {
		nodeItems = nodes.Nodes
		for _, node := range nodes.Nodes {
			nodeMetaByID[node.ID] = node
		}
	}

	tunnels := enrichTunnelNodeMeta(tunnelList.List, nodeMetaByID)
	for i := range tunnels {
		if item, ok := trafficByName[strings.TrimSpace(tunnels[i].Name)]; ok {
			tunnels[i].TotalIn = item.TotalIn
			tunnels[i].TotalOut = item.TotalOut
			tunnels[i].TotalTraffic = item.TotalTraffic
		}
	}

	data := &models.TunnelsPageData{
		Tunnels:     tunnels,
		Total:       tunnelList.Total,
		Nodes:       nodeItems,
		TrafficDays: days,
	}
	if userInfo != nil {
		data.MaxTunnelCount = userInfo.MaxTunnelCount
	}
	return data, nil
}

func (s *CenterService) CreateTunnel(req models.CreateTunnelRequest) (*models.TunnelItem, error) {
	tunnel, err := s.api.CreateTunnel(context.Background(), req)
	return tunnel, apiMessageError(err)
}

func (s *CenterService) UpdateTunnel(tunnelName string, req models.UpdateTunnelRequest) (*models.TunnelItem, error) {
	tunnel, err := s.api.UpdateTunnel(context.Background(), tunnelName, req)
	return tunnel, apiMessageError(err)
}

// DeleteTunnel 删除隧道；隧道正在本地运行时拒绝删除，避免 frpc 持续重连
func (s *CenterService) DeleteTunnel(tunnelName string) error {
	name := strings.TrimSpace(tunnelName)

	s.runnerMu.Lock()
	running := s.isRunnerRunningLocked()
	inUse := strings.TrimSpace(s.runnerTunnelName) == name
	for _, item := range s.runnerTunnelNames {
		if strings.TrimSpace(item) == name {
			inUse = true
			break
		}
	}
	s.runnerMu.Unlock()

	if running && inUse {
		return fmt.Errorf("隧道正在本地运行，请先停止后再删除")
	}
	return apiMessageError(s.api.DeleteTunnel(context.Background(), name))
}

// apiMessageError 将中心端返回的业务错误转换为可直接展示给用户的提示
func apiMessageError(err error) error {
	var apiErr *httpclient.APIError
	if errors.As(err, &apiErr) && strings.TrimSpace(apiErr.Message) != "" {
		return errors.New(strings.TrimSpace(apiErr.Message))
	}
	return err
}

func (s *CenterService) GetTunnelRealtimeTraffic(tunnelName string) (*models.TunnelRealtimeTraffic, error) {
	return s.api.GetTunnelRealtimeTraffic(context.Background(), tunnelName)
}

func (s *CenterService) GetUserDomains() (*models.DomainListData, error) {
	return s.api.GetUserDomains(context.Background())
}

// GetHomeAnnouncements 并行获取首页的置顶公告与公告板
func (s *CenterService) GetHomeAnnouncements() (*models.HomeAnnouncements, error) {
	ctx := context.Background()

	var (
		wg               sync.WaitGroup
		top, board       *models.AnnouncementListData
		topErr, boardErr error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		top, topErr = s.api.GetAnnouncements(ctx, "top")
	}()
	go func() {
		defer wg.Done()
		board, boardErr = s.api.GetAnnouncements(ctx, "board")
	}()
	wg.Wait()

	if topErr != nil && boardErr != nil {
		return nil, apiMessageError(topErr)
	}

	result := &models.HomeAnnouncements{
		Top:   []models.Announcement{},
		Board: []models.Announcement{},
	}
	if top != nil && top.Items != nil {
		result.Top = top.Items
	}
	if board != nil && board.Items != nil {
		result.Board = board.Items
	}
	return result, nil
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
	status.PID = s.runnerPIDLocked()
	if len(s.runnerLogs) > 0 {
		status.LogLines = append([]string(nil), s.runnerLogs...)
	}
	return status
}

func (s *CenterService) appendRunnerLogLocked(line string) {
	s.runnerLogs = append(s.runnerLogs, line)
	if len(s.runnerLogs) > runnerLogMaxLines {
		s.runnerLogs = append([]string(nil), s.runnerLogs[len(s.runnerLogs)-runnerLogMaxLines:]...)
	}
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
