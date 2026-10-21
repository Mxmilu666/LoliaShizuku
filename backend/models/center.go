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
	ClientVersion  string `json:"client_version,omitempty"`
	TotalIn        int64  `json:"total_in,omitempty"`
	TotalOut       int64  `json:"total_out,omitempty"`
	TotalTraffic   int64  `json:"total_traffic,omitempty"`

	Config TunnelConfigView `json:"config"`
}

// TunnelConfigView 隧道配置（auto_tls + 高级配置）
type TunnelConfigView struct {
	AutoTLS              bool   `json:"auto_tls"`
	ProxyProtocolVersion string `json:"proxy_protocol_version"`
	Protocol             string `json:"protocol"`
	HTTPRedirect         bool   `json:"http_redirect"`
}

// CreateTunnelRequest 创建隧道请求
type CreateTunnelRequest struct {
	NodeID       int64  `json:"node_id"`
	Type         string `json:"type"`
	LocalIP      string `json:"local_ip"`
	LocalPort    int    `json:"local_port"`
	RemotePort   int    `json:"remote_port,omitempty"`
	CustomDomain string `json:"custom_domain,omitempty"`
	Remark       string `json:"remark,omitempty"`
}

// TunnelConfigUpdate 隧道配置更新，nil 表示不修改该项
type TunnelConfigUpdate struct {
	AutoTLS              *bool   `json:"auto_tls,omitempty"`
	ProxyProtocolVersion *string `json:"proxy_protocol_version,omitempty"`
	Protocol             *string `json:"protocol,omitempty"`
	HTTPRedirect         *bool   `json:"http_redirect,omitempty"`
}

// UpdateTunnelRequest 编辑隧道请求，nil 表示不修改该项
type UpdateTunnelRequest struct {
	LocalIP      *string             `json:"local_ip,omitempty"`
	LocalPort    *int                `json:"local_port,omitempty"`
	CustomDomain *string             `json:"custom_domain,omitempty"`
	Config       *TunnelConfigUpdate `json:"config,omitempty"`
	Remark       *string             `json:"remark,omitempty"`
}

// TunnelRealtimeTraffic 单个隧道的实时流量
type TunnelRealtimeTraffic struct {
	TunnelName   string `json:"tunnel_name"`
	NodeName     string `json:"node_name"`
	TrafficIn    int64  `json:"traffic_in"`
	TrafficOut   int64  `json:"traffic_out"`
	TotalTraffic int64  `json:"total_traffic"`
	Connections  int64  `json:"connections"`
	Status       string `json:"status"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type DomainItem struct {
	ID          int64  `json:"id"`
	Domain      string `json:"domain"`
	IsVerified  bool   `json:"is_verified"`
	Status      string `json:"status"`
	HasBeian    bool   `json:"has_beian"`
	BeianNumber string `json:"beian_number,omitempty"`
	Remark      string `json:"remark,omitempty"`
}

type DomainListData struct {
	Domains []DomainItem `json:"domains"`
	Total   int64        `json:"total"`
}

// TunnelsPageData 隧道页面所需的聚合数据
type TunnelsPageData struct {
	Tunnels        []TunnelItem `json:"tunnels"`
	Total          int64        `json:"total"`
	MaxTunnelCount int64        `json:"max_tunnel_count"`
	Nodes          []NodeItem   `json:"nodes"`
	TrafficDays    int          `json:"traffic_days"`
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

	Config TunnelConfigView `json:"config"`
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
	RegionCode         string   `json:"region_code"`
	BeianRequired      bool     `json:"beian_required"`
	HighTraffic        bool     `json:"high_traffic"`
	TrafficRatio       float64  `json:"traffic_ratio"`
	Remark             string   `json:"remark,omitempty"`
	Load               *float64 `json:"load,omitempty"`
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
	// LoliaConfig 包含 AutoTLS、DoH dnsServer 等 LoliaFRP 扩展字段（base64），仅 LoliaFRP-CLI 可解析
	LoliaConfig string `json:"lolia_config,omitempty"`
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

// Announcement 平台公告。Content 为 Markdown，由前端渲染并清洗。
type Announcement struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Level       string `json:"level"`        // info / warning / error / success / normal
	DisplayType string `json:"display_type"` // top：置顶横幅；board：公告板
	NeedModal   bool   `json:"need_modal"`
	ActionLabel string `json:"action_label"`
	ActionURL   string `json:"action_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type AnnouncementListData struct {
	Items []Announcement `json:"items"`
}

// HomeAnnouncements 首页展示的公告：置顶横幅与公告板
type HomeAnnouncements struct {
	Top   []Announcement `json:"top"`
	Board []Announcement `json:"board"`
}
