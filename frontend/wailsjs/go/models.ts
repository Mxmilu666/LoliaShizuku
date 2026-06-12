export namespace models {
	
	export class AppVersionInfo {
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new AppVersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	    }
	}
	export class HomeStatsData {
	    user_count: number;
	    tunnel_count: number;
	    total_traffic_used: number;
	
	    static createFrom(source: any = {}) {
	        return new HomeStatsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_count = source["user_count"];
	        this.tunnel_count = source["tunnel_count"];
	        this.total_traffic_used = source["total_traffic_used"];
	    }
	}
	export class TunnelItem {
	    bandwidth_limit: number;
	    custom_domain: string;
	    id: number;
	    local_ip: string;
	    local_port: number;
	    name: string;
	    node_address?: string;
	    node_id: number;
	    node_name?: string;
	    remark: string;
	    remote_port: number;
	    status: string;
	    type: string;
	    total_in?: number;
	    total_out?: number;
	    total_traffic?: number;
	
	    static createFrom(source: any = {}) {
	        return new TunnelItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bandwidth_limit = source["bandwidth_limit"];
	        this.custom_domain = source["custom_domain"];
	        this.id = source["id"];
	        this.local_ip = source["local_ip"];
	        this.local_port = source["local_port"];
	        this.name = source["name"];
	        this.node_address = source["node_address"];
	        this.node_id = source["node_id"];
	        this.node_name = source["node_name"];
	        this.remark = source["remark"];
	        this.remote_port = source["remote_port"];
	        this.status = source["status"];
	        this.type = source["type"];
	        this.total_in = source["total_in"];
	        this.total_out = source["total_out"];
	        this.total_traffic = source["total_traffic"];
	    }
	}
	export class UserTunnelSummary {
	    count: number;
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new UserTunnelSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.total = source["total"];
	    }
	}
	export class UserTrafficData {
	    user_id: string;
	    username: string;
	    traffic_limit: number;
	    traffic_used: number;
	    traffic_remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new UserTrafficData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_id = source["user_id"];
	        this.username = source["username"];
	        this.traffic_limit = source["traffic_limit"];
	        this.traffic_used = source["traffic_used"];
	        this.traffic_remaining = source["traffic_remaining"];
	    }
	}
	export class UserInfoData {
	    avatar: string;
	    bandwidth_limit: number;
	    created_at?: string;
	    email: string;
	    has_kyc?: boolean;
	    id: number;
	    is_baned?: boolean;
	    kyc_status?: string;
	    max_tunnel_count: number;
	    role: string;
	    today_checked?: boolean;
	    traffic_limit: number;
	    traffic_used: number;
	    tunnel_token?: string;
	    username: string;
	
	    static createFrom(source: any = {}) {
	        return new UserInfoData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avatar = source["avatar"];
	        this.bandwidth_limit = source["bandwidth_limit"];
	        this.created_at = source["created_at"];
	        this.email = source["email"];
	        this.has_kyc = source["has_kyc"];
	        this.id = source["id"];
	        this.is_baned = source["is_baned"];
	        this.kyc_status = source["kyc_status"];
	        this.max_tunnel_count = source["max_tunnel_count"];
	        this.role = source["role"];
	        this.today_checked = source["today_checked"];
	        this.traffic_limit = source["traffic_limit"];
	        this.traffic_used = source["traffic_used"];
	        this.tunnel_token = source["tunnel_token"];
	        this.username = source["username"];
	    }
	}
	export class CenterDashboardData {
	    user: UserInfoData;
	    traffic: UserTrafficData;
	    tunnel: UserTunnelSummary;
	    tunnels: TunnelItem[];
	    app: AppVersionInfo;
	    home: HomeStatsData;
	
	    static createFrom(source: any = {}) {
	        return new CenterDashboardData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = this.convertValues(source["user"], UserInfoData);
	        this.traffic = this.convertValues(source["traffic"], UserTrafficData);
	        this.tunnel = this.convertValues(source["tunnel"], UserTunnelSummary);
	        this.tunnels = this.convertValues(source["tunnels"], TunnelItem);
	        this.app = this.convertValues(source["app"], AppVersionInfo);
	        this.home = this.convertValues(source["home"], HomeStatsData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DailyTunnelStat {
	    tunnel_name: string;
	    remark: string;
	    total_traffic: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyTunnelStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tunnel_name = source["tunnel_name"];
	        this.remark = source["remark"];
	        this.total_traffic = source["total_traffic"];
	    }
	}
	export class DailyTrafficStat {
	    date: string;
	    total_traffic: number;
	    tunnel_stats: DailyTunnelStat[];
	
	    static createFrom(source: any = {}) {
	        return new DailyTrafficStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.total_traffic = source["total_traffic"];
	        this.tunnel_stats = this.convertValues(source["tunnel_stats"], DailyTunnelStat);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DailyTrafficResponse {
	    days: number;
	    daily_stats: DailyTrafficStat[];
	
	    static createFrom(source: any = {}) {
	        return new DailyTrafficResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.days = source["days"];
	        this.daily_stats = this.convertValues(source["daily_stats"], DailyTrafficStat);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class FrpcConfigData {
	    config: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = source["config"];
	    }
	}
	export class FrpcInstalledInfo {
	    version: string;
	    asset_name: string;
	    sha256: string;
	    installed_at: string;
	    binary_path: string;
	    binary_exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FrpcInstalledInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.asset_name = source["asset_name"];
	        this.sha256 = source["sha256"];
	        this.installed_at = source["installed_at"];
	        this.binary_path = source["binary_path"];
	        this.binary_exists = source["binary_exists"];
	    }
	}
	export class FrpcMirrorPreset {
	    id: string;
	    name: string;
	    description?: string;
	    base_url?: string;
	    url_template?: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcMirrorPreset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.base_url = source["base_url"];
	        this.url_template = source["url_template"];
	    }
	}
	export class FrpcMirrorConfig {
	    mode: string;
	    preset_id?: string;
	    custom_base_url?: string;
	    custom_url_template?: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcMirrorConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.preset_id = source["preset_id"];
	        this.custom_base_url = source["custom_base_url"];
	        this.custom_url_template = source["custom_url_template"];
	    }
	}
	export class FrpcPaths {
	    userdata_dir: string;
	    frpc_dir: string;
	    bin_dir: string;
	    binary_path: string;
	    download_dir: string;
	    state_path: string;
	    settings_path: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userdata_dir = source["userdata_dir"];
	        this.frpc_dir = source["frpc_dir"];
	        this.bin_dir = source["bin_dir"];
	        this.binary_path = source["binary_path"];
	        this.download_dir = source["download_dir"];
	        this.state_path = source["state_path"];
	        this.settings_path = source["settings_path"];
	    }
	}
	export class FrpcStatus {
	    goos: string;
	    goarch: string;
	    paths: FrpcPaths;
	    github_mirror_url: string;
	    mirror_config: FrpcMirrorConfig;
	    builtin_mirrors: FrpcMirrorPreset[];
	    installed?: FrpcInstalledInfo;
	    latest?: FrpcReleaseInfo;
	    update_available: boolean;
	    latest_error?: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.goos = source["goos"];
	        this.goarch = source["goarch"];
	        this.paths = this.convertValues(source["paths"], FrpcPaths);
	        this.github_mirror_url = source["github_mirror_url"];
	        this.mirror_config = this.convertValues(source["mirror_config"], FrpcMirrorConfig);
	        this.builtin_mirrors = this.convertValues(source["builtin_mirrors"], FrpcMirrorPreset);
	        this.installed = this.convertValues(source["installed"], FrpcInstalledInfo);
	        this.latest = this.convertValues(source["latest"], FrpcReleaseInfo);
	        this.update_available = source["update_available"];
	        this.latest_error = source["latest_error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FrpcReleaseAsset {
	    name: string;
	    download_url: string;
	    content_type: string;
	    size: number;
	    digest: string;
	    sha256: string;
	    os: string;
	    arch: string;
	    archive_format: string;
	
	    static createFrom(source: any = {}) {
	        return new FrpcReleaseAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.download_url = source["download_url"];
	        this.content_type = source["content_type"];
	        this.size = source["size"];
	        this.digest = source["digest"];
	        this.sha256 = source["sha256"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.archive_format = source["archive_format"];
	    }
	}
	export class FrpcReleaseInfo {
	    tag_name: string;
	    name: string;
	    html_url: string;
	    published_at: string;
	    asset: FrpcReleaseAsset;
	
	    static createFrom(source: any = {}) {
	        return new FrpcReleaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.name = source["name"];
	        this.html_url = source["html_url"];
	        this.published_at = source["published_at"];
	        this.asset = this.convertValues(source["asset"], FrpcReleaseAsset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FrpcInstallResult {
	    release: FrpcReleaseInfo;
	    status: FrpcStatus;
	
	    static createFrom(source: any = {}) {
	        return new FrpcInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.release = this.convertValues(source["release"], FrpcReleaseInfo);
	        this.status = this.convertValues(source["status"], FrpcStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	export class NodeItem {
	    id: number;
	    name: string;
	    status: string;
	    ip_address: string;
	    supported_protocols: string[];
	    need_kyc: boolean;
	    frps_version: string;
	    agent_version: string;
	    frps_port: number;
	    sponsor: string;
	    bandwidth: number;
	    last_seen: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.ip_address = source["ip_address"];
	        this.supported_protocols = source["supported_protocols"];
	        this.need_kyc = source["need_kyc"];
	        this.frps_version = source["frps_version"];
	        this.agent_version = source["agent_version"];
	        this.frps_port = source["frps_port"];
	        this.sponsor = source["sponsor"];
	        this.bandwidth = source["bandwidth"];
	        this.last_seen = source["last_seen"];
	        this.created_at = source["created_at"];
	    }
	}
	export class NodeListData {
	    nodes: NodeItem[];
	    total: number;
	    page: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new NodeListData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], NodeItem);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.limit = source["limit"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunnerData {
	    config: string;
	    version: string;
	    nodes: NodeItem[];
	    current_tunnel?: TunnelItem;
	
	    static createFrom(source: any = {}) {
	        return new RunnerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = source["config"];
	        this.version = source["version"];
	        this.nodes = this.convertValues(source["nodes"], NodeItem);
	        this.current_tunnel = this.convertValues(source["current_tunnel"], TunnelItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RunnerRuntimeStatus {
	    running: boolean;
	    pid: number;
	    started_at?: string;
	    tunnel_name?: string;
	    tunnel_names?: string[];
	    node_address?: string;
	    command?: string;
	    last_error?: string;
	    log_lines?: string[];
	
	    static createFrom(source: any = {}) {
	        return new RunnerRuntimeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.pid = source["pid"];
	        this.started_at = source["started_at"];
	        this.tunnel_name = source["tunnel_name"];
	        this.tunnel_names = source["tunnel_names"];
	        this.node_address = source["node_address"];
	        this.command = source["command"];
	        this.last_error = source["last_error"];
	        this.log_lines = source["log_lines"];
	    }
	}
	export class TrafficTunnelItem {
	    tunnel_name: string;
	    node_id: string;
	    total_in: number;
	    total_out: number;
	    total_traffic: number;
	    max_connections: number;
	    remark: string;
	
	    static createFrom(source: any = {}) {
	        return new TrafficTunnelItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tunnel_name = source["tunnel_name"];
	        this.node_id = source["node_id"];
	        this.total_in = source["total_in"];
	        this.total_out = source["total_out"];
	        this.total_traffic = source["total_traffic"];
	        this.max_connections = source["max_connections"];
	        this.remark = source["remark"];
	    }
	}
	export class TrafficTunnelData {
	    count: number;
	    days: number;
	    tunnels: TrafficTunnelItem[];
	
	    static createFrom(source: any = {}) {
	        return new TrafficTunnelData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.days = source["days"];
	        this.tunnels = this.convertValues(source["tunnels"], TrafficTunnelItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TunnelDetailData {
	    bandwidth_limit: number;
	    client_version: string;
	    created_at: string;
	    custom_domain: string;
	    id: number;
	    local_ip: string;
	    local_port: number;
	    name: string;
	    node_address: string;
	    node_id: number;
	    node_name: string;
	    remark: string;
	    remote_port: number;
	    status: string;
	    tunnel_token: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelDetailData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bandwidth_limit = source["bandwidth_limit"];
	        this.client_version = source["client_version"];
	        this.created_at = source["created_at"];
	        this.custom_domain = source["custom_domain"];
	        this.id = source["id"];
	        this.local_ip = source["local_ip"];
	        this.local_port = source["local_port"];
	        this.name = source["name"];
	        this.node_address = source["node_address"];
	        this.node_id = source["node_id"];
	        this.node_name = source["node_name"];
	        this.remark = source["remark"];
	        this.remote_port = source["remote_port"];
	        this.status = source["status"];
	        this.tunnel_token = source["tunnel_token"];
	        this.type = source["type"];
	    }
	}
	
	export class TunnelListData {
	    limit: number;
	    list: TunnelItem[];
	    page: number;
	    total: number;
	    total_page: number;
	
	    static createFrom(source: any = {}) {
	        return new TunnelListData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.limit = source["limit"];
	        this.list = this.convertValues(source["list"], TunnelItem);
	        this.page = source["page"];
	        this.total = source["total"];
	        this.total_page = source["total_page"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TunnelOverviewData {
	    list: TunnelItem[];
	    page: number;
	    limit: number;
	    total: number;
	    total_page: number;
	
	    static createFrom(source: any = {}) {
	        return new TunnelOverviewData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.list = this.convertValues(source["list"], TunnelItem);
	        this.page = source["page"];
	        this.limit = source["limit"];
	        this.total = source["total"];
	        this.total_page = source["total_page"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	

}

export namespace version {
	
	export class Info {
	    app_name: string;
	    version: string;
	    git_commit: string;
	    git_branch: string;
	    build_time: string;
	    go_version: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_name = source["app_name"];
	        this.version = source["version"];
	        this.git_commit = source["git_commit"];
	        this.git_branch = source["git_branch"];
	        this.build_time = source["build_time"];
	        this.go_version = source["go_version"];
	        this.platform = source["platform"];
	    }
	}

}

