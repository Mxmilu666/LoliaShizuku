package api

import (
	"context"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"

	"github.com/Mxmilu666/LoliaShizuku/backend/httpclient"
	"github.com/Mxmilu666/LoliaShizuku/backend/models"
)

type CenterAPI struct {
	client *httpclient.Client
}

func NewCenterAPI(client *httpclient.Client) *CenterAPI {
	return &CenterAPI{client: client}
}

func (a *CenterAPI) GetUserInfo(ctx context.Context) (*models.UserInfoData, error) {
	var data models.UserInfoData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/info", nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetUserTrafficStats(ctx context.Context) (*models.UserTrafficData, error) {
	var data models.UserTrafficData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/traffic/stats", nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetUserTunnels(ctx context.Context, page, limit int) (*models.TunnelListData, error) {
	query := map[string]string{
		"page":  strconv.Itoa(page),
		"limit": strconv.Itoa(limit),
	}
	var data models.TunnelListData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/tunnel", query, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetTrafficTunnels(ctx context.Context, days int) (*models.TrafficTunnelData, error) {
	query := map[string]string{
		"days": strconv.Itoa(days),
	}
	var data models.TrafficTunnelData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/traffic/tunnels", query, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetTrafficDaily(ctx context.Context, days int) (*models.DailyTrafficResponse, error) {
	query := map[string]string{
		"days": strconv.Itoa(days),
	}
	var data models.DailyTrafficResponse
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/traffic/daily", query, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetNodes(ctx context.Context) (*models.NodeListData, error) {
	var data models.NodeListData
	if err := a.client.DoJSON(ctx, http.MethodPost, "/user/nodes", nil, map[string]any{}, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetFrpcConfig(ctx context.Context, tunnel string) (*models.FrpcConfigData, error) {
	query := map[string]string{
		"tunnel": strings.TrimSpace(tunnel),
	}
	var data models.FrpcConfigData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/user/frpc/config", query, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetTunnelDetail(ctx context.Context, tunnelName string) (*models.TunnelDetailData, error) {
	trimmedName := strings.TrimSpace(tunnelName)
	if trimmedName == "" {
		return nil, nil
	}

	path := "/user/tunnel/" + neturl.PathEscape(trimmedName)
	var data models.TunnelDetailData
	if err := a.client.DoJSON(ctx, http.MethodGet, path, nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetClientVersion(ctx context.Context) (*models.AppVersionInfo, error) {
	var data models.AppVersionInfo
	if err := a.client.DoJSON(ctx, http.MethodGet, "/client/version", nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (a *CenterAPI) GetHomeStats(ctx context.Context) (*models.HomeStatsData, error) {
	var data models.HomeStatsData
	if err := a.client.DoJSON(ctx, http.MethodGet, "/home", nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
