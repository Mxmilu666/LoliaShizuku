import {
  PortScanService,
  type ListeningPort as ListeningPortBinding,
} from "../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";

export type ListeningPort = Omit<ListeningPortBinding, "protocol"> & {
  protocol: "tcp" | "udp";
};

export interface ServiceGuess {
  name: string;
  icon: string;
  /** 适合 HTTP(S) 隧道的网页服务 */
  web?: boolean;
}

/** 扫描本机监听中的端口（Go 侧 PortScanService）。 */
export async function listListeningPorts(): Promise<ListeningPort[]> {
  try {
    const ports = await PortScanService.ListListeningPorts();
    return (ports ?? []).map((item) => ({
      ...item,
      protocol: item.protocol === "udp" ? "udp" : "tcp",
    }));
  } catch (error) {
    throw new Error(error instanceof Error && error.message ? error.message : "扫描本地端口失败");
  }
}

const knownPorts: Record<string, ServiceGuess> = {
  "tcp/21": { name: "FTP", icon: "fas fa-folder-tree" },
  "tcp/22": { name: "SSH", icon: "fas fa-terminal" },
  "tcp/80": { name: "网站 (HTTP)", icon: "fas fa-globe", web: true },
  "tcp/443": { name: "网站 (HTTPS)", icon: "fas fa-lock", web: true },
  "tcp/1433": { name: "SQL Server", icon: "fas fa-database" },
  "tcp/3000": { name: "Web 开发服务", icon: "fas fa-code", web: true },
  "tcp/3306": { name: "MySQL", icon: "fas fa-database" },
  "tcp/3389": { name: "远程桌面 (RDP)", icon: "fas fa-display" },
  "tcp/5173": { name: "Vite 开发服务", icon: "fas fa-code", web: true },
  "tcp/5432": { name: "PostgreSQL", icon: "fas fa-database" },
  "tcp/5900": { name: "VNC 远程桌面", icon: "fas fa-display" },
  "tcp/6379": { name: "Redis", icon: "fas fa-database" },
  "tcp/7777": { name: "Terraria", icon: "fas fa-gamepad" },
  "tcp/8000": { name: "网站 (HTTP)", icon: "fas fa-globe", web: true },
  "tcp/8080": { name: "网站 (HTTP)", icon: "fas fa-globe", web: true },
  "tcp/8443": { name: "网站 (HTTPS)", icon: "fas fa-lock", web: true },
  "tcp/8888": { name: "网站 (HTTP)", icon: "fas fa-globe", web: true },
  "tcp/25565": { name: "Minecraft Java 版", icon: "fas fa-cube" },
  "tcp/25575": { name: "Minecraft RCON", icon: "fas fa-terminal" },
  "tcp/27015": { name: "Steam 游戏服务器", icon: "fas fa-gamepad" },
  "tcp/27017": { name: "MongoDB", icon: "fas fa-database" },
  "udp/2456": { name: "Valheim", icon: "fas fa-gamepad" },
  "udp/7777": { name: "Terraria / 游戏服务器", icon: "fas fa-gamepad" },
  "udp/8211": { name: "幻兽帕鲁", icon: "fas fa-gamepad" },
  "udp/19132": { name: "Minecraft 基岩版", icon: "fas fa-cube" },
  "udp/27015": { name: "Steam 游戏服务器", icon: "fas fa-gamepad" },
  "udp/34197": { name: "Factorio", icon: "fas fa-gamepad" },
};

// 进程名（小写、去扩展名）→ 服务；用于非默认端口
const knownProcesses: Record<string, ServiceGuess> = {
  nginx: { name: "Nginx", icon: "fas fa-globe", web: true },
  httpd: { name: "Apache", icon: "fas fa-globe", web: true },
  apache2: { name: "Apache", icon: "fas fa-globe", web: true },
  caddy: { name: "Caddy", icon: "fas fa-globe", web: true },
  sshd: { name: "SSH", icon: "fas fa-terminal" },
  mysqld: { name: "MySQL", icon: "fas fa-database" },
  mariadbd: { name: "MariaDB", icon: "fas fa-database" },
  postgres: { name: "PostgreSQL", icon: "fas fa-database" },
  "redis-server": { name: "Redis", icon: "fas fa-database" },
  mongod: { name: "MongoDB", icon: "fas fa-database" },
  bedrock_server: { name: "Minecraft 基岩版", icon: "fas fa-cube" },
  terrariaserver: { name: "Terraria", icon: "fas fa-gamepad" },
  palserver: { name: "幻兽帕鲁", icon: "fas fa-gamepad" },
  "palserver-win64-shipping-cmd": { name: "幻兽帕鲁", icon: "fas fa-gamepad" },
  valheim_server: { name: "Valheim", icon: "fas fa-gamepad" },
  factorio: { name: "Factorio", icon: "fas fa-gamepad" },
};

// 系统常驻进程，几乎不会用来做隧道，默认收起
const noiseProcesses = new Set([
  "rapportd",
  "controlcenter",
  "sharingd",
  "identityservicesd",
  "mdnsresponder",
  "avahi-daemon",
  "cupsd",
  "systemd-resolved",
  "chronyd",
  "lsass",
  "wininit",
  "spoolsv",
]);

const processKey = (item: ListeningPort) => item.process.trim().toLowerCase();

// 通过协议发现的服务（Go 侧 minecraft_discovery.go）
const discoveredServices: Record<string, ServiceGuess> = {
  "minecraft-java": { name: "Minecraft Java 版", icon: "fas fa-cube" },
  "minecraft-bedrock": { name: "Minecraft 基岩版", icon: "fas fa-cube" },
};

export function guessService(item: ListeningPort): ServiceGuess | null {
  const discovered = item.service ? discoveredServices[item.service] : undefined;
  if (discovered) return discovered;
  const byPort = knownPorts[`${item.protocol}/${item.port}`];
  const byProcess = knownProcesses[processKey(item)];
  // 进程名更可靠（例如 nginx 监听在 8081）；java 等通用进程以端口为准
  return byProcess ?? byPort ?? null;
}

/** 是否默认展示：识别出的服务，或非临时端口且非系统常驻进程。 */
export function isLikelyService(item: ListeningPort): boolean {
  if (guessService(item)) return true;
  if (noiseProcesses.has(processKey(item))) return false;
  return item.port < 49152;
}

/** 列表展示用的服务名称。 */
export function describeService(item: ListeningPort): string {
  return guessService(item)?.name || item.process || (item.probed ? "本机服务" : "未知进程");
}

/** 列表副标题：服务自报名称（如 Minecraft 世界名）、进程、位置。 */
export function describeServiceDetail(item: ListeningPort): string {
  const parts: string[] = [];
  if (item.title) parts.push(item.title);
  if (item.process && guessService(item)) parts.push(item.process);
  if (item.pid > 0) parts.push(`PID ${item.pid}`);
  if (item.remote) {
    parts.push(`局域网 ${item.address}`);
  } else if (item.service) {
    parts.push("局域网发现");
  } else {
    parts.push(item.probed ? "连接探测" : `监听 ${item.address || "-"}`);
  }
  return parts.join(" · ");
}
