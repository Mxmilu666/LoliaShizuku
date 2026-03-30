type CenterServiceBinding = {
  GetDashboard: () => Promise<any>;
  GetRunnerRuntimeStatus: () => Promise<any>;
  GetTunnelsOverview: (page: number, limit: number, days: number) => Promise<any>;
  GetRunnerData: (tunnelID: number) => Promise<any>;
  GetTunnelDetail: (tunnelName: string) => Promise<any>;
  StartRunner: (tunnelNames: string[]) => Promise<RunnerRuntimeStatus>;
  StopRunner: () => Promise<any>;
  GetTrafficDaily: (days: number) => Promise<any>;
};

function getCenterServiceBinding(): CenterServiceBinding {
  const svc = (window as any).go?.services?.CenterService;
  if (!svc) {
    throw new Error("CenterService 未绑定，请重启应用。");
  }
  return svc as CenterServiceBinding;
}

function parseError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }
  if (typeof error === "string") {
    return new Error(error);
  }
  if (typeof error === "object" && error !== null && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string") {
      return new Error(message);
    }
  }
  return new Error("请求失败");
}

export interface DashboardData {
  user: {
    avatar: string;
    bandwidth_limit: number;
    email: string;
    id: number;
    max_tunnel_count: number;
    role: string;
    traffic_limit: number;
    traffic_used: number;
    username: string;
  };
  traffic: {
    user_id: string;
    username: string;
    traffic_limit: number;
    traffic_used: number;
    traffic_remaining: number;
  };
  tunnel: {
    count: number;
    total: number;
  };
  tunnels: TunnelOverviewItem[];
  app: {
    version: string;
  };
  home: {
    user_count: number;
    tunnel_count: number;
    total_traffic_used: number;
  };
}

export interface TunnelOverviewItem {
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
}

export interface DailyTrafficResponse {
  days: number;
  daily_stats: Array<{
    date: string;
    total_traffic: number;
    tunnel_stats?: Array<{
      tunnel_name: string;
      remark: string;
      total_traffic: number;
    }>;
  }>;
}

export interface TunnelsOverviewData {
  list: TunnelOverviewItem[];
  page: number;
  limit: number;
  total: number;
  total_page: number;
}

export interface RunnerData {
  config: string;
  version: string;
  nodes: Array<{
    id: number;
    name: string;
    status: string;
    ip_address: string;
    frps_port: number;
  }>;
  current_tunnel?: TunnelOverviewItem;
}

export interface TunnelDetailData {
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
}

export interface RunnerRuntimeStatus {
  running: boolean;
  pid: number;
  started_at?: string;
  tunnel_name?: string;
  tunnel_names?: string[];
  node_address?: string;
  command?: string;
  last_error?: string;
  log_lines?: string[];
}

export async function getDashboard(): Promise<DashboardData> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetDashboard()) as DashboardData;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getTunnelsOverview(
  page = 1,
  limit = 50,
  days = 2,
): Promise<TunnelsOverviewData> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetTunnelsOverview(page, limit, days)) as TunnelsOverviewData;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getRunnerData(tunnelID = 0): Promise<RunnerData> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetRunnerData(tunnelID)) as RunnerData;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getTunnelDetail(tunnelName: string): Promise<TunnelDetailData> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetTunnelDetail(tunnelName)) as TunnelDetailData;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getRunnerRuntimeStatus(): Promise<RunnerRuntimeStatus> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetRunnerRuntimeStatus()) as RunnerRuntimeStatus;
  } catch (error) {
    throw parseError(error);
  }
}

export async function startRunner(
  tunnelNames: string | string[] = [],
): Promise<RunnerRuntimeStatus> {
  try {
    const svc = getCenterServiceBinding();
    const normalizedTunnelNames = Array.isArray(tunnelNames)
      ? tunnelNames
      : tunnelNames.trim()
        ? [tunnelNames]
        : [];
    return (await svc.StartRunner(normalizedTunnelNames)) as RunnerRuntimeStatus;
  } catch (error) {
    throw parseError(error);
  }
}

export async function stopRunner(): Promise<RunnerRuntimeStatus> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.StopRunner()) as RunnerRuntimeStatus;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getTrafficDaily(days = 7): Promise<DailyTrafficResponse> {
  try {
    const svc = getCenterServiceBinding();
    return (await svc.GetTrafficDaily(days)) as DailyTrafficResponse;
  } catch (error) {
    throw parseError(error);
  }
}
