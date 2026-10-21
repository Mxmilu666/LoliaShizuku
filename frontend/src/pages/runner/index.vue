<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import {
  getRunnerData,
  getRunnerRuntimeStatus,
  getTunnelsOverview,
  startRunner,
  stopRunner,
  type RunnerRuntimeStatus,
} from "@/services/center";
import { useGlobalLoadingStore } from "@/stores/globalLoading";
import { getTunnelStatusMeta } from "@/utils/tunnel";
import { formatDateTime } from "@/utils/format";
import { renderAnsiLog } from "@/utils/ansiLog";
import MobileFab from "@/components/MobileFab.vue";
import { isMobile } from "@/utils/platform";

// 概览数值字号：移动端更紧凑（Vuetify 字号类在 CSS layer 中带 !important，无法用普通样式覆盖）
const valueClass = isMobile ? "text-title-medium" : "text-title-large";

defineOptions({
  name: "RunnerPage",
});

const errorMessage = ref("");
const globalLoadingStore = useGlobalLoadingStore();
const withGlobalLoading = <T>(task: () => Promise<T>) =>
  globalLoadingStore.withGlobalLoading(task);
const runningAction = ref(false);
const runtimePolling = ref(false);
let runtimePollTimer: ReturnType<typeof setInterval> | null = null;

const summary = ref({
  server: "-",
  protocol: "-",
  version: "-",
  pid: "-",
  startTime: "-",
});

const tunnels = ref<
  Array<{
    name: string;
    remark: string;
    customDomain: string;
    type: string;
    local: string;
    remote: string;
    remotePort: number;
    status: string;
    statusColor: string;
  }>
>([]);

const logs = ref<string[]>([]);
const selectedTunnelName = ref("");
const runtimeStatus = ref<RunnerRuntimeStatus>({
  running: false,
  pid: 0,
  started_at: "",
  node_address: "",
  tunnel_names: [],
  command: "",
  last_error: "",
  log_lines: [],
});

const isRunning = computed(() => runtimeStatus.value.running);
// frpc 输出带 ANSI 颜色（后端设置 CLICOLOR_FORCE），渲染为高亮 HTML
const logHtml = computed(() => renderAnsiLog(logs.value));
const statusLabel = computed(() => (isRunning.value ? "运行中" : "未运行"));
const statusColor = computed(() => (isRunning.value ? "success" : "medium-emphasis"));
const activeTunnelNames = computed(() => {
  const names = runtimeStatus.value.tunnel_names ?? [];
  if (names.length > 0) {
    return names.filter((item) => item.trim() !== "");
  }

  const fallbackName = (runtimeStatus.value.tunnel_name || "").trim();
  return fallbackName ? [fallbackName] : [];
});
const activeTunnelNameSet = computed(() => new Set(activeTunnelNames.value));
const activeTunnels = computed(() => {
  if (!isRunning.value) {
    return [] as typeof tunnels.value;
  }
  if (activeTunnelNameSet.value.size === 0) {
    return [] as typeof tunnels.value;
  }
  return tunnels.value.filter((tunnel) => activeTunnelNameSet.value.has(tunnel.name));
});

const joinHostPort = (host: string, port?: number | null) => {
  const normalizedHost = host.trim();
  if (!normalizedHost) {
    return "-";
  }
  if (!port || port <= 0) {
    return normalizedHost;
  }
  return `${normalizedHost}:${port}`;
};

const formatRuntimeRemote = (tunnel: {
  customDomain: string;
  remote: string;
  remotePort: number;
}) => {
  const customDomain = (tunnel.customDomain || "").trim();
  if (customDomain) {
    return customDomain;
  }
  const remote = (tunnel.remote || "").trim();
  if (remote && remote !== "-") {
    return remote;
  }
  const runtimeNodeAddress = (runtimeStatus.value.node_address || "").trim();
  if (!runtimeNodeAddress) {
    return remote || "-";
  }
  return joinHostPort(runtimeNodeAddress, tunnel.remotePort);
};

const resolveRuntimeServer = (fallback: string) => {
  const runtimeNodeAddress = (runtimeStatus.value.node_address || "").trim();
  if (activeTunnels.value.length > 1) {
    return `${activeTunnels.value.length} 条隧道`;
  }
  const activeTunnel = activeTunnels.value[0];
  const customDomain = (activeTunnel?.customDomain || "").trim();
  if (customDomain) {
    return customDomain;
  }
  if (!runtimeNodeAddress) {
    return fallback;
  }
  return joinHostPort(runtimeNodeAddress, activeTunnel?.remotePort);
};

const resolveSummaryProtocol = (fallback: string) => {
  if (activeTunnels.value.length === 0) {
    return fallback;
  }

  const protocols = new Set(
    activeTunnels.value
      .map((tunnel) => tunnel.type.trim().toUpperCase())
      .filter((protocol) => protocol !== ""),
  );

  if (protocols.size === 1) {
    return Array.from(protocols)[0];
  }
  if (protocols.size > 1) {
    return "MULTI";
  }
  return fallback;
};


const loadRunnerData = async () => {
  errorMessage.value = "";

  await withGlobalLoading(async () => {
    try {
      const [runnerData, tunnelData, runnerRuntime] = await Promise.all([
        getRunnerData(0),
        getTunnelsOverview(1, 100, 2),
        getRunnerRuntimeStatus(),
      ]);
      runtimeStatus.value = runnerRuntime;

      const nodeMap = new Map<
        number,
        { ip_address: string; frps_port: number }
      >();
      for (const node of runnerData.nodes ?? []) {
        nodeMap.set(Number(node.id), {
          ip_address: node.ip_address || "-",
          frps_port: Number(node.frps_port || 0),
        });
      }

      const tunnelList = tunnelData.list ?? [];
      tunnels.value = tunnelList.map((item) => {
        const node = nodeMap.get(Number(item.node_id));
        const remoteHost = (item.node_address || "").trim() || node?.ip_address || "node";
        const remotePort = item.remote_port || 0;
        const customDomain = (item.custom_domain || "").trim();

        return {
          name: item.name,
          remark: item.remark || item.name,
          type: item.type || "-",
          customDomain,
          local: `${item.local_ip}:${item.local_port}`,
          remote: customDomain || joinHostPort(remoteHost, remotePort),
          remotePort,
          status: getTunnelStatusMeta(item.status).text,
          statusColor: getTunnelStatusMeta(item.status).color,
        };
      });
      const currentTunnelName = runnerData.current_tunnel?.name || "";
      const tunnelNames = new Set(tunnels.value.map((item) => item.name));
      if (!tunnelNames.has(selectedTunnelName.value)) {
        selectedTunnelName.value = tunnelNames.has(currentTunnelName)
          ? currentTunnelName
          : (tunnels.value[0]?.name ?? "");
      }

      const currentNode = runnerData.current_tunnel
        ? nodeMap.get(Number(runnerData.current_tunnel.node_id))
        : undefined;
      const fallbackNodeAddress = currentNode
        ? `${currentNode.ip_address}:${currentNode.frps_port || runnerData.current_tunnel?.remote_port || "-"}`
        : runnerData.nodes?.[0]
          ? `${runnerData.nodes[0].ip_address}:${runnerData.nodes[0].frps_port}`
          : "-";

      summary.value = {
        server: resolveRuntimeServer(fallbackNodeAddress),
        protocol: resolveSummaryProtocol((runnerData.current_tunnel?.type || "-").toUpperCase()),
        version: runnerData.version || "-",
        pid: runtimeStatus.value.pid > 0 ? String(runtimeStatus.value.pid) : "-",
        startTime: runtimeStatus.value.started_at
          ? formatDateTime(runtimeStatus.value.started_at)
          : "-",
      };

      const runtimeLines = runtimeStatus.value.log_lines || [];
      logs.value =
        runtimeLines.length > 0
          ? runtimeLines
          : ["暂无日志，点击“启动”后可查看 frpc 输出。"];
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : "加载 Runner 数据失败，请稍后重试";
    }
  });
};

const syncRuntimeStatus = async () => {
  if (runtimePolling.value) {
    return;
  }
  runtimePolling.value = true;
  try {
    const status = await getRunnerRuntimeStatus();
    runtimeStatus.value = status;
    const fallbackServer = summary.value.server;
    summary.value = {
      ...summary.value,
      server: resolveRuntimeServer(fallbackServer),
      protocol: resolveSummaryProtocol(summary.value.protocol),
      pid: status.pid > 0 ? String(status.pid) : "-",
      startTime: status.started_at
        ? formatDateTime(status.started_at)
        : "-",
    };

    const runtimeLines = status.log_lines || [];
    logs.value =
      runtimeLines.length > 0
        ? runtimeLines
        : ["暂无日志，点击“启动”后可查看 frpc 输出。"];
  } finally {
    runtimePolling.value = false;
  }
};

const handleStartRunner = async () => {
  errorMessage.value = "";
  if (!selectedTunnelName.value) {
    errorMessage.value = "请先选择隧道";
    return;
  }
  runningAction.value = true;
  try {
    runtimeStatus.value = await startRunner(selectedTunnelName.value);
    await loadRunnerData();
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "启动 runner 失败，请稍后重试";
  } finally {
    runningAction.value = false;
  }
};

const handleStopRunner = async () => {
  errorMessage.value = "";
  runningAction.value = true;
  try {
    runtimeStatus.value = await stopRunner();
    await loadRunnerData();
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "停止 runner 失败，请稍后重试";
  } finally {
    runningAction.value = false;
  }
};

onMounted(() => {
  void loadRunnerData();
  runtimePollTimer = setInterval(() => {
    void syncRuntimeStatus();
  }, 1200);
});

onBeforeUnmount(() => {
  if (runtimePollTimer) {
    clearInterval(runtimePollTimer);
    runtimePollTimer = null;
  }
});
</script>

<template>
  <div class="md-page">
    <!-- 页头 -->
    <header class="md-page-header">
      <div class="min-w-0">
        <h1 class="text-headline-small">运行</h1>
        <p class="text-body-medium md-on-surface-variant mt-1">
          本地 frpc 进程状态与实时日志
        </p>
      </div>
      <v-spacer />
      <v-btn
        icon="fas fa-rotate"
        variant="text"
        color="on-surface"
        title="刷新"
        @click="loadRunnerData"
      />
      <v-btn
        v-if="isRunning"
        color="error-container"
        variant="flat"
        size="large"
        prepend-icon="fas fa-stop"
        :loading="runningAction"
        :disabled="runningAction"
        @click="handleStopRunner"
      >
        停止
      </v-btn>
      <v-btn
        v-else
        color="primary"
        variant="flat"
        size="large"
        prepend-icon="fas fa-play"
        :loading="runningAction"
        :disabled="runningAction || !selectedTunnelName"
        @click="handleStartRunner"
      >
        启动
      </v-btn>
    </header>

    <!-- 移动端：刷新放到顶栏，启动 / 停止使用 FAB -->
    <template v-if="isMobile">
      <Teleport defer to="#app-bar-actions">
        <v-btn icon="fas fa-rotate" variant="text" color="on-surface" title="刷新" @click="loadRunnerData" />
      </Teleport>
      <MobileFab
        v-if="isRunning"
        icon="fas fa-stop"
        label="停止"
        danger
        :loading="runningAction"
        :disabled="runningAction"
        @click="handleStopRunner"
      />
      <MobileFab
        v-else
        icon="fas fa-play"
        label="启动"
        :loading="runningAction"
        :disabled="runningAction || !selectedTunnelName"
        @click="handleStartRunner"
      />
    </template>

    <v-alert v-if="errorMessage" type="error" variant="tonal" rounded="lg" class="mb-4">
      {{ errorMessage }}
    </v-alert>
    <v-alert
      v-else-if="runtimeStatus.last_error"
      type="warning"
      variant="tonal"
      rounded="lg"
      class="mb-4"
    >
      上次运行错误：{{ runtimeStatus.last_error }}
    </v-alert>

    <!-- 运行概览 -->
    <section class="summary-grid mb-4">
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-circle-dot</v-icon>
          状态
        </div>
        <div class="d-flex align-center ga-2" :class="[valueClass, `text-${statusColor}`]">
          <span class="status-dot" :class="{ 'status-dot--pulse': isRunning }" />
          {{ statusLabel }}
        </div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-network-wired</v-icon>
          服务器
        </div>
        <div class="text-truncate md-mono" :class="valueClass" :title="summary.server">
          {{ summary.server }}
        </div>
        <div class="text-body-small md-on-surface-variant">协议 {{ summary.protocol }}</div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-microchip</v-icon>
          进程
        </div>
        <div :class="valueClass">PID {{ summary.pid }}</div>
        <div class="text-body-small md-on-surface-variant">frpc {{ summary.version }}</div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-clock</v-icon>
          启动时间
        </div>
        <div class="text-truncate" :class="valueClass">{{ summary.startTime }}</div>
      </div>
    </section>

    <section class="content-grid">
      <!-- 运行中的隧道 -->
      <div class="md-card-outlined d-flex flex-column">
        <div class="d-flex align-center mb-3">
          <h2 class="text-title-medium">运行中的隧道</h2>
          <v-spacer />
          <span class="text-label-large md-on-surface-variant">{{ activeTunnels.length }} 条</span>
        </div>
        <div class="d-flex flex-column ga-2 overflow-auto">
          <div
            v-for="tunnel in activeTunnels"
            :key="`${tunnel.name}-${tunnel.local}`"
            class="tunnel-row"
          >
            <div class="md-icon-container tunnel-row-icon">
              <v-icon size="14">fas fa-route</v-icon>
            </div>
            <div class="min-w-0 flex-grow-1">
              <div class="text-body-large text-truncate">{{ tunnel.remark }}</div>
              <div class="text-body-small md-on-surface-variant md-mono text-truncate">
                {{ tunnel.local }} → {{ formatRuntimeRemote(tunnel) }}
              </div>
            </div>
            <span class="d-flex align-center ga-1 flex-shrink-0 text-no-wrap text-label-medium" :class="`text-${tunnel.statusColor}`">
              <span class="status-dot" />
              {{ tunnel.status }}
            </span>
          </div>
          <div v-if="activeTunnels.length === 0" class="empty-hint text-body-medium md-on-surface-variant">
            <v-icon size="20" class="mb-2">fas fa-moon</v-icon>
            <div>frpc 未运行，暂无已启动的隧道</div>
          </div>
        </div>
      </div>

      <!-- 日志 -->
      <div class="md-card-outlined d-flex flex-column min-w-0">
        <div class="mb-3">
          <h2 class="text-title-medium">frpc 运行日志</h2>
          <p class="text-body-small md-on-surface-variant md-mono text-truncate mt-1" :title="runtimeStatus.command">
            {{ runtimeStatus.command || "尚未启动" }}
          </p>
        </div>
        <!-- 仅插入经 ansi_up 转义后的内容 -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <pre class="log-block md-mono" v-html="logHtml" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.min-w-0 {
  min-width: 0;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(0, 2fr);
  gap: 12px;
  align-items: stretch;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}

.status-dot--pulse {
  animation: pulse 1.6s ease-in-out infinite;
}

.tunnel-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px 8px 8px;
  border-radius: 12px;
  background: var(--md-surface-container-high);
}

.tunnel-row-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 16px;
  text-align: center;
}

.log-block {
  flex: 1;
  margin: 0;
  min-height: 360px;
  max-height: 460px;
  padding: 16px;
  border-radius: 12px;
  overflow: auto;
  font-size: 0.8rem;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--md-surface-container-highest);
  color: rgb(var(--v-theme-on-surface));
}

/* 移动端：概览更紧凑，日志区域降低高度 */
.is-mobile .summary-grid .md-stat {
  gap: 4px;
  padding: 12px;
}

/* ANSI 颜色映射到 MD3 主题色，亮/暗主题下均保持可读 */
.log-block :deep(.log-time) {
  color: var(--md-on-surface-variant);
}

.log-block :deep(.log-runner) {
  color: rgb(var(--v-theme-primary));
}

.log-block :deep(.ansi-red-fg),
.log-block :deep(.ansi-bright-red-fg) {
  color: rgb(var(--v-theme-error));
}

.log-block :deep(.ansi-green-fg),
.log-block :deep(.ansi-bright-green-fg) {
  color: rgb(var(--v-theme-success));
}

.log-block :deep(.ansi-yellow-fg),
.log-block :deep(.ansi-bright-yellow-fg) {
  color: rgb(var(--v-theme-warning));
}

.log-block :deep(.ansi-blue-fg),
.log-block :deep(.ansi-bright-blue-fg),
.log-block :deep(.ansi-cyan-fg),
.log-block :deep(.ansi-bright-cyan-fg) {
  color: rgb(var(--v-theme-info));
}

.log-block :deep(.ansi-magenta-fg),
.log-block :deep(.ansi-bright-magenta-fg) {
  color: rgb(var(--v-theme-tertiary));
}

.log-block :deep(.ansi-black-fg),
.log-block :deep(.ansi-bright-black-fg),
.log-block :deep(.ansi-white-fg),
.log-block :deep(.ansi-bright-white-fg) {
  color: var(--md-on-surface-variant);
}

.is-mobile .log-block {
  min-height: 240px;
  max-height: 50vh;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

@media (max-width: 960px) {
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .content-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
