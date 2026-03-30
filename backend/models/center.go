package models

import "encoding/json"

// APIEnvelope is the unified response wrapper from LoliaFRP Center API.
type APIEnvelope struct {
	Code   int             `json:"code"`
	Status int             `json:"status"`
	Msg    string          `json:"msg"`
	Data   json.RawMessage `json:"data"`
}

type CenterDashboardData struct {
	User      UserInfoData      `json:"user"`
	Traffic   UserTrafficData   `json:"traffic"`
	Tunnel    UserTunnelSummary `json:"tunnel"`
	Tunnels   []TunnelItem      `json:"tunnels"`
	App       AppVersionInfo    `json:"app"`
	HomeStats HomeStatsData     `json:"home"`
}

type TunnelOverviewData struct {
	List      []TunnelItem `json:"list"`
	Page      int64        `json:"page"`
	Limit     int64        `json:"limit"`
	Total     int64        `json:"total"`
	TotalPage int64        `json:"total_page"`
}

type RunnerData struct {
	Config        string      `json:"config"`
	Version       string      `json:"version"`
	Nodes         []NodeItem  `json:"nodes"`
	CurrentTunnel *TunnelItem `json:"current_tunnel,omitempty"`
}

type RunnerRuntimeStatus struct {
	Running     bool     `json:"running"`
	PID         int      `json:"pid"`
	StartedAt   string   `json:"started_at,omitempty"`
	TunnelName  string   `json:"tunnel_name,omitempty"`
	TunnelNames []string `json:"tunnel_names,omitempty"`
	NodeAddress string   `json:"node_address,omitempty"`
	Command     string   `json:"command,omitempty"`
	LastError   string   `json:"last_error,omitempty"`
	LogLines    []string `json:"log_lines,omitempty"`
}

type UserInfoData struct {
	Avatar         string `json:"avatar"`
	BandwidthLimit int64  `json:"bandwidth_limit"`
	CreatedAt      string `json:"created_at,omitempty"`
	Email          string `json:"email"`
	HasKYC         bool   `json:"has_kyc,omitempty"`
	ID             int64  `json:"id"`
	IsBaned        bool   `json:"is_baned,omitempty"`
	KYCStatus      string `json:"kyc_status,omitempty"`
	MaxTunnelCount int64  `json:"max_tunnel_count"`
	Role           string `json:"role"`
	TodayChecked   bool   `json:"today_checked,omitempty"`
	TrafficLimit   int64  `json:"traffic_limit"`
	TrafficUsed    int64  `json:"traffic_used"`
	TunnelToken    string `json:"tunnel_token,omitempty"`
	Username       string `json:"username"`
}

type UserTrafficData struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	TrafficLimit     int64  `json:"traffic_limit"`
	TrafficUsed      int64  `json:"traffic_used"`
	TrafficRemaining int64  `json:"traffic_remaining"`
}

type UserTunnelSummary struct {
	Count int64 `json:"count"`
	Total int64 `json:"total"`
}

type TunnelItem struct {
	BandwidthLimit int64  `json:"bandwidth_limit"`
	CustomDomain   string `json:"custom_domain"`
	ID             int64  `json:"id"`
	LocalIP        string `json:"local_ip"`
	LocalPort      int64  `json:"local_port"`
	Name           string `json:"name"`
	NodeAddress    string `json:"node_address,omitempty"`
	NodeID         int64  `json:"node_id"`
	NodeName       string `json:"node_name,omitempty"`
	Remark         string `json:"remark"`
	RemotePort     int64  `json:"remote_port"`
	Status         string `json:"status"`
	Type           string `json:"type"`
	TotalIn        int64  `json:"total_in,omitempty"`
	TotalOut       int64  `json:"total_out,omitempty"`
	TotalTraffic   int64  `json:"total_traffic,omitempty"`
}

type TunnelDetailData struct {
	BandwidthLimit int64  `json:"bandwidth_limit"`
	ClientVersion  string `json:"client_version"`
	CreatedAt      string `json:"created_at"`
	CustomDomain   string `json:"custom_domain"`
	ID             int64  `json:"id"`
	LocalIP        string `json:"local_ip"`
	LocalPort      int64  `json:"local_port"`
	Name           string `json:"name"`
	NodeAddress    string `json:"node_address"`
	NodeID         int64  `json:"node_id"`
	NodeName       string `json:"node_name"`
	Remark         string `json:"remark"`
	RemotePort     int64  `json:"remote_port"`
	Status         string `json:"status"`
	TunnelToken    string `json:"tunnel_token"`
	Type           string `json:"type"`
}

type AppVersionInfo struct {
	Version string `json:"version"`
}

type HomeStatsData struct {
	UserCount        int64 `json:"user_count"`
	TunnelCount      int64 `json:"tunnel_count"`
	TotalTrafficUsed int64 `json:"total_traffic_used"`
}

type TunnelListData struct {
	Limit     int64        `json:"limit"`
	List      []TunnelItem `json:"list"`
	Page      int64        `json:"page"`
	Total     int64        `json:"total"`
	TotalPage int64        `json:"total_page"`
}

type NodeItem struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	IPAddress          string   `json:"ip_address"`
	SupportedProtocols []string `json:"supported_protocols"`
	NeedKYC            bool     `json:"need_kyc"`
	FrpsVersion        string   `json:"frps_version"`
	AgentVersion       string   `json:"agent_version"`
	FrpsPort           int64    `json:"frps_port"`
	Sponsor            string   `json:"sponsor"`
	Bandwidth          int64    `json:"bandwidth"`
	LastSeen           string   `json:"last_seen"`
	CreatedAt          string   `json:"created_at"`
}

type NodeListData struct {
	Nodes []NodeItem `json:"nodes"`
	Total int64      `json:"total"`
	Page  int64      `json:"page"`
	Limit int64      `json:"limit"`
}

type TrafficTunnelItem struct {
	TunnelName    string `json:"tunnel_name"`
	NodeID        string `json:"node_id"`
	TotalIn       int64  `json:"total_in"`
	TotalOut      int64  `json:"total_out"`
	TotalTraffic  int64  `json:"total_traffic"`
	MaxConnection int64  `json:"max_connections"`
	Remark        string `json:"remark"`
}

type TrafficTunnelData struct {
	Count   int64               `json:"count"`
	Days    int64               `json:"days"`
	Tunnels []TrafficTunnelItem `json:"tunnels"`
}

type FrpcConfigData struct {
	Config string `json:"config"`
}

type DailyTunnelStat struct {
	TunnelName   string `json:"tunnel_name"`
	Remark       string `json:"remark"`
	TotalTraffic int64  `json:"total_traffic"`
}

type DailyTrafficStat struct {
	Date         string            `json:"date"`
	TotalTraffic int64             `json:"total_traffic"`
	TunnelStats  []DailyTunnelStat `json:"tunnel_stats"`
}

type DailyTrafficResponse struct {
	Days       int64              `json:"days"`
	DailyStats []DailyTrafficStat `json:"daily_stats"`
}
