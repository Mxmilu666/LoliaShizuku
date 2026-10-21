import { Clipboard } from "@wailsio/runtime";
import type { NodeItem, TunnelItem } from "@/services/center";

export type TunnelType = "tcp" | "udp" | "http" | "https";

export interface TunnelTypeMeta {
  value: TunnelType;
  label: string;
  icon: string;
  color: string;
  description: string;
}

export const tunnelTypes: TunnelTypeMeta[] = [
  {
    value: "tcp",
    label: "TCP",
    icon: "fas fa-plug",
    color: "primary",
    description: "网站、数据库、远程桌面、MC Java 版等需要稳定连接的场景（最常用）",
  },
  {
    value: "udp",
    label: "UDP",
    icon: "fas fa-wave-square",
    color: "secondary",
    description: "游戏、语音、MC 基岩版等实时性强的场景，延迟低",
  },
  {
    value: "http",
    label: "HTTP",
    icon: "fas fa-globe",
    color: "success",
    description: "通过域名公开访问本地 Web 服务（建站 / 反向代理）",
  },
  {
    value: "https",
    label: "HTTPS",
    icon: "fas fa-lock",
    color: "warning",
    description: "通过域名加密访问，可开启 Auto TLS 自动签发证书",
  },
];

export const getTypeMeta = (type: string): TunnelTypeMeta =>
  tunnelTypes.find((item) => item.value === type.toLowerCase()) ??
  tunnelTypes[0];

export const isHTTPType = (type: string) => {
  const normalized = type.toLowerCase();
  return normalized === "http" || normalized === "https";
};

export interface StatusMeta {
  text: string;
  color: string;
}

export const getTunnelStatusMeta = (status: string): StatusMeta => {
  switch ((status || "").toLowerCase()) {
    case "active":
      return { text: "在线", color: "success" };
    case "inactive":
      return { text: "离线", color: "grey" };
    case "error":
      return { text: "异常", color: "error" };
    default:
      return { text: status || "未知", color: "info" };
  }
};

export const isNodeOnline = (node?: NodeItem | null) =>
  (node?.status || "").toLowerCase() === "online";

// 节点未单独上报 https，https 隧道与 http 共用 vhost 能力
export const nodeSupportsType = (node: NodeItem, type: TunnelType) => {
  const protocols = (node.supported_protocols ?? []).map((item) =>
    item.toLowerCase(),
  );
  return protocols.includes(type === "https" ? "http" : type);
};

export const getLoadColor = (load?: number | null) => {
  if (load === null || load === undefined || !Number.isFinite(load)) {
    return "grey";
  }
  if (load >= 80) return "error";
  if (load >= 50) return "warning";
  return "success";
};

export const formatBytes = (value?: number | null) => {
  const size = Number(value ?? 0);
  if (!Number.isFinite(size) || size <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let current = size;
  let index = 0;
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024;
    index++;
  }
  return `${current.toFixed(index === 0 ? 0 : 2)} ${units[index]}`;
};

export { formatDateTime } from "./format";

// 访问地址：HTTP/HTTPS 为域名，TCP/UDP 为节点地址:远程端口
export const getAccessAddress = (tunnel: TunnelItem) => {
  if (isHTTPType(tunnel.type)) {
    const domain = (tunnel.custom_domain || "").trim();
    return domain ? `${tunnel.type.toLowerCase()}://${domain}` : "";
  }
  const nodeAddress = (tunnel.node_address || "").trim();
  if (nodeAddress && tunnel.remote_port > 0) {
    return `${nodeAddress}:${tunnel.remote_port}`;
  }
  return tunnel.remote_port > 0 ? `:${tunnel.remote_port}` : "";
};

export const getLocalAddress = (tunnel: TunnelItem) =>
  `${tunnel.local_ip}:${tunnel.local_port}`;

export const getTunnelTitle = (tunnel: TunnelItem) =>
  (tunnel.remark || "").trim() || tunnel.name;

export const copyText = (text: string) => Clipboard.SetText(text);
