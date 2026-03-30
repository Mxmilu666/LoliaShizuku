<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  getRunnerRuntimeStatus,
  getTunnelsOverview,
  startRunner,
  type RunnerRuntimeStatus,
  type TunnelOverviewItem,
} from "@/services/center";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { useGlobalLoadingStore } from "@/stores/globalLoading";

defineOptions({
  name: "TunnelsPage",
});

const errorMessage = ref("");
const searchQuery = ref("");
const tunnels = ref<TunnelOverviewItem[]>([]);
const globalLoadingStore = useGlobalLoadingStore();
const withGlobalLoading = <T,>(task: () => Promise<T>) =>
  globalLoadingStore.withGlobalLoading(task);
const router = useRouter();
const runnerStatus = ref<RunnerRuntimeStatus>({
  running: false,
  pid: 0,
  started_at: "",
  tunnel_name: "",
  tunnel_names: [],
  command: "",
  last_error: "",
  log_lines: [],
});
const startingTunnelName = ref("");

const filteredTunnels = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase();
  if (!keyword) {
    return tunnels.value;
  }

  return tunnels.value.filter((tunnel) => {
    const haystack = [
      tunnel.name,
      tunnel.type,
      tunnel.remark,
      tunnel.custom_domain,
      tunnel.local_ip,
      String(tunnel.local_port),
      String(tunnel.remote_port),
      String(tunnel.id),
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(keyword);
  });
});
const isRunnerRunning = computed(() => runnerStatus.value.running);

const loadTunnels = async () => {
  errorMessage.value = "";

  await withGlobalLoading(async () => {
    try {
      const [response, status] = await Promise.all([
        getTunnelsOverview(1, 100, 2),
        getRunnerRuntimeStatus(),
      ]);
      tunnels.value = response.list ?? [];
      runnerStatus.value = status;
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : "加载隧道列表失败，请稍后重试";
    }
  });
};

const formatBytes = (value: number) => {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index++;
  }
  return `${size.toFixed(2)} ${units[index]}`;
};

const getStatusColor = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized === "active") {
    return "success";
  }
  if (normalized === "inactive") {
    return "grey";
  }
  return "info";
};

const getStatusText = (status: string) => {
  const normalized = status.toLowerCase();
  if (normalized === "active") {
    return "运行中";
  }
  if (normalized === "inactive") {
    return "已停止";
  }
  return status || "未知";
};

const getTunnelTarget = (tunnel: TunnelOverviewItem) => {
  const customDomain = (tunnel.custom_domain || "").trim();
  if (customDomain) {
    return customDomain;
  }

  const nodeAddress = (tunnel.node_address || "").trim();
  if (nodeAddress) {
    return `${nodeAddress}:${tunnel.remote_port}`;
  }

  if (tunnel.remote_port > 0) {
    return `端口 ${tunnel.remote_port}`;
  }

  return "-";
};

const openTunnelDetail = (name: string) => {
  const tunnelName = name.trim();
  if (!tunnelName) {
    return;
  }
  BrowserOpenURL(
    `https://dash.lolia.link/dash/tunnel/${encodeURIComponent(tunnelName)}`,
  );
};

const isStartedTunnel = (tunnelName: string) =>
  isRunnerRunning.value &&
  ((runnerStatus.value.tunnel_names ?? []).some(
    (currentTunnelName) => currentTunnelName.trim() === tunnelName.trim(),
  ) ||
    (runnerStatus.value.tunnel_name || "").trim() === tunnelName.trim());

const handleStartTunnel = async (tunnelName: string) => {
  errorMessage.value = "";
  startingTunnelName.value = tunnelName;
  try {
    runnerStatus.value = await startRunner([tunnelName]);
    await loadTunnels();
    await router.push("/runner");
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "启动隧道失败，请稍后重试";
  } finally {
    startingTunnelName.value = "";
  }
};

onMounted(() => {
  void loadTunnels();
});
</script>

<template>
  <div>
    <v-alert v-if="errorMessage" type="error" variant="tonal" class="mb-3">
      {{ errorMessage }}
    </v-alert>

    <v-card elevation="2" class="mb-4">
      <v-card-text class="d-flex align-center flex-wrap ga-4">
        <v-text-field
          v-model="searchQuery"
          label="搜索隧道"
          prepend-inner-icon="fas fa-search"
          hide-details="auto"
          clearable
          class="flex-grow-1"
        />

        <v-btn color="primary" variant="tonal" @click="loadTunnels">
          <v-icon start>fas fa-rotate</v-icon>
          刷新
        </v-btn>
      </v-card-text>
    </v-card>

    <v-row dense>
      <v-col
        v-for="tunnel in filteredTunnels"
        :key="tunnel.id"
        cols="12"
        sm="6"
        md="4"
      >
        <v-card elevation="2" class="h-100 d-flex flex-column">
          <v-card-title class="d-flex align-center justify-space-between">
            <span class="text-subtitle-1 font-weight-bold">
              {{ tunnel.remark }}
            </span>
            <v-chip
              :color="getStatusColor(tunnel.status)"
              class="rounded-pill"
              size="small"
            >
              <v-icon start size="14">fas fa-circle</v-icon>
              {{ getStatusText(tunnel.status) }}
            </v-chip>
          </v-card-title>

          <v-card-text class="d-flex flex-column ga-2 pt-0">
            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-server</v-icon>
              <span class="text-caption">{{ tunnel.node_name }}</span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-network-wired</v-icon>
              <span class="text-caption">{{ tunnel.type.toUpperCase() }}</span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-earth-africa</v-icon>
              <span class="text-caption">
                {{ tunnel.local_ip }}:{{ tunnel.local_port }}
              </span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-arrow-right</v-icon>
              <span class="text-caption">{{ getTunnelTarget(tunnel) }}</span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-exchange-alt</v-icon>
              <span class="text-caption">
                ↓ {{ formatBytes(Number(tunnel.total_in ?? 0)) }} / ↑
                {{ formatBytes(Number(tunnel.total_out ?? 0)) }}
              </span>
            </div>
          </v-card-text>

          <v-divider />

          <v-card-actions>
            <v-btn
              color="success"
              variant="tonal"
              size="small"
              prepend-icon="fas fa-play"
              :loading="startingTunnelName === tunnel.name"
              :disabled="
                !!startingTunnelName || isStartedTunnel(tunnel.name)
              "
              @click="handleStartTunnel(tunnel.name)"
            >
              {{ isStartedTunnel(tunnel.name) ? "已启动" : "启动" }}
            </v-btn>
            <v-spacer />
            <v-btn
              color="secondary"
              variant="tonal"
              size="small"
              prepend-icon="fas fa-eye"
              @click="openTunnelDetail(tunnel.name)"
            >
              详情
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>

      <v-col v-if="filteredTunnels.length === 0" cols="12">
        <v-card elevation="0" class="text-center py-16">
          <v-icon size="64" color="grey-lighten-1" class="mb-4">
            fas fa-folder-open
          </v-icon>
          <div class="text-h6 font-weight-bold text-medium-emphasis mb-2">
            暂无可展示的隧道
          </div>
          <div class="text-body-2 text-medium-emphasis mb-4">
            你可以尝试刷新或调整搜索条件
          </div>
          <v-btn
            color="primary"
            variant="tonal"
            prepend-inner-icon="fas fa-rotate"
            @click="loadTunnels"
          >
            重新加载
          </v-btn>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>
