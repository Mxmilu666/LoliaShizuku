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
const logText = computed(() => logs.value.join("\n"));
const statusLabel = computed(() => (isRunning.value ? "运行中" : "未运行"));
const statusColor = computed(() => (isRunning.value ? "success" : "warning"));
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
  const runtimeNodeAddress = (runtimeStatus.value.node_address || "").trim();
  if (!runtimeNodeAddress) {
    return tunnel.remote;
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

const statusToColor = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized.includes("run") || normalized.includes("online")) {
    return "success";
  }
  if (normalized.includes("error") || normalized.includes("fail")) {
    return "error";
  }
  if (normalized.includes("stop") || normalized.includes("offline")) {
    return "grey";
  }
  return "info";
};

const statusToText = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized.includes("run") || normalized.includes("online")) {
    return "在线";
  }
  if (normalized.includes("error") || normalized.includes("fail")) {
    return "异常";
  }
  if (normalized.includes("stop") || normalized.includes("offline")) {
    return "离线";
  }
  return status || "未知";
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
        const remotePort = item.remote_port || node?.frps_port || 0;
        const customDomain = (item.custom_domain || "").trim();

        return {
          name: item.name,
          remark: item.remark || item.name,
          type: item.type || "-",
          customDomain,
          local: `${item.local_ip}:${item.local_port}`,
          remote: customDomain || `${remoteHost}:${remotePort}`,
          remotePort,
          status: statusToText(item.status),
          statusColor: statusToColor(item.status),
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
          ? new Date(runtimeStatus.value.started_at).toLocaleString()
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
        ? new Date(status.started_at).toLocaleString()
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
  <v-alert v-if="errorMessage" type="error" variant="tonal" class="mb-3">
    {{ errorMessage }}
  </v-alert>
  <v-alert v-else-if="runtimeStatus.last_error" type="warning" variant="tonal" class="mb-3">
    上次运行错误：{{ runtimeStatus.last_error }}
  </v-alert>

  <v-card elevation="2" class="pa-6 mb-4">
    <div class="d-flex align-center justify-space-between flex-wrap ga-6">
      <div class="flex-grow-1">
        <div class="text-h5 font-weight-bold">LoliaCLI Runner</div>
        <div class="text-caption text-medium-emphasis">
          已连接至 Runner {{ summary.server !== "-" ? `(${summary.server})` : "" }}（{{ summary.protocol }}）
        </div>
        <div class="d-flex flex-wrap ga-2 mt-3">
          <v-chip :color="statusColor" size="small" variant="tonal" class="rounded-pill">
            {{ statusLabel }}
          </v-chip>
          <v-chip color="primary" size="small" variant="outlined" class="rounded-pill">
            PID {{ summary.pid }}
          </v-chip>
          <v-chip color="info" size="small" variant="outlined" class="rounded-pill">
            {{ summary.version }}
          </v-chip>
          <v-chip color="secondary" size="small" variant="outlined" class="rounded-pill">
            启动于 {{ summary.startTime }}
          </v-chip>
        </div>
      </div>

      <div class="d-flex flex-wrap ga-3">
        <v-btn
          color="success"
          prepend-icon="fas fa-play"
          :loading="runningAction"
          :disabled="isRunning || runningAction || !selectedTunnelName"
          @click="handleStartRunner"
        >
          启动
        </v-btn>
        <v-btn
          color="error"
          prepend-icon="fas fa-stop"
          :loading="runningAction"
          :disabled="!isRunning || runningAction"
          @click="handleStopRunner"
        >
          停止
        </v-btn>
        <v-btn color="primary" prepend-icon="fas fa-rotate" @click="loadRunnerData">
          刷新
        </v-btn>
      </div>
    </div>
  </v-card>

  <v-row>
    <v-col cols="12" md="4">
      <v-card elevation="2" class="h-100 d-flex flex-column">
        <v-card-title class="d-flex align-center justify-space-between">
          <div class="text-h6 font-weight-bold">隧道状态</div>
          <v-chip size="x-small" color="primary" variant="outlined">
            {{ activeTunnels.length }} 条规则
          </v-chip>
        </v-card-title>
        <v-divider />
        <v-card-text class="d-flex flex-column ga-3 flex-grow-1 overflow-auto">
          <v-sheet
            v-for="tunnel in activeTunnels"
            :key="`${tunnel.name}-${tunnel.local}`"
            class="pa-3 d-flex align-center justify-space-between"
            rounded="lg"
            border
          >
            <div>
              <div class="text-subtitle-1 font-weight-bold">
                {{ tunnel.remark }}
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ tunnel.local }} → {{ formatRuntimeRemote(tunnel) }}
              </div>
            </div>
            <v-chip :color="tunnel.statusColor" size="x-small" variant="tonal">
              {{ tunnel.status }}
            </v-chip>
          </v-sheet>
          <v-sheet
            v-if="activeTunnels.length === 0"
            class="pa-4 text-caption text-medium-emphasis"
            rounded="lg"
            border
          >
            Runner 未运行，暂无已启动隧道。
          </v-sheet>
        </v-card-text>
      </v-card>
    </v-col>

    <v-col cols="12" md="8">
      <v-card elevation="2" class="h-100">
        <v-card-title class="d-flex align-center justify-space-between flex-wrap ga-4">
          <div>
            <div class="text-h6 font-weight-bold">frpc 运行日志</div>
            <div class="text-caption text-medium-emphasis">
              启动命令：{{ runtimeStatus.command || "-" }}
            </div>
          </div>
        </v-card-title>
        <v-divider />
        <v-card-text class="pa-4">
          <v-sheet
            class="pa-4 overflow-auto bg-grey-darken-4 text-grey-lighten-4"
            rounded="lg"
            border
            style="min-height: 360px; max-height: 460px"
          >
            <pre class="ma-0 text-body-2 log-text-mono" v-text="logText" />
          </v-sheet>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>
<style scoped>
.log-text-mono {
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
}
</style>
