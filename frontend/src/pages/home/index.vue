<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import {
  VisXYContainer,
  VisLine,
  VisAxis,
  VisArea,
  VisCrosshair,
  VisTooltip,
} from "@unovis/vue";
import { useElementSize } from "@vueuse/core";
import {
  getDashboard,
  getTrafficDaily,
  type DailyTrafficResponse,
} from "@/services/center";
import { useGlobalLoadingStore } from "@/stores/globalLoading";

defineOptions({
  name: "HomePage",
});

type TunnelData = {
  name: string;
  amount: number;
};

type DataRecord = {
  date: Date;
  amount: number;
  tunnels: TunnelData[];
};

const errorMessage = ref("");
const globalLoadingStore = useGlobalLoadingStore();
const withGlobalLoading = <T>(task: () => Promise<T>) =>
  globalLoadingStore.withGlobalLoading(task);

const userInfo = ref({
  name: "-",
  email: "-",
  avatarUrl: "",
});

const stats = ref({
  availableTraffic: 0,
  tunnelCount: 0,
  tunnelLimit: 0,
  bandwidthLimit: "-",
});

const dailyTraffic = ref<DailyTrafficResponse | null>(null);

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 6) return "夜深了，早点休息喵";
  if (hour < 9) return "早上好~ 又是元气满满的一天呢";
  if (hour < 12) return "上午好，加油喵";
  if (hour < 14) return "中午好，记得吃饭哦";
  if (hour < 18) return "下午好，继续加油w";
  if (hour < 22) return "晚上好，记得放松一下喵";
  return "夜深了，早点休息喵";
});

const cardRef = ref<HTMLElement | null>(null);
const { width } = useElementSize(cardRef);

const data = computed<DataRecord[]>(() => {
  if (!dailyTraffic.value?.daily_stats) {
    return [];
  }

  return dailyTraffic.value.daily_stats.map((stat) => ({
    date: new Date(stat.date),
    amount: Number(stat.total_traffic || 0) / (1024 * 1024 * 1024),
    tunnels:
      stat.tunnel_stats?.map((tunnel) => ({
        name: tunnel.remark || tunnel.tunnel_name,
        amount: Number(tunnel.total_traffic || 0) / (1024 * 1024 * 1024),
      })) ?? [],
  }));
});

const x = (_: DataRecord, i: number) => i;
const y = (d: DataRecord) => d.amount;

const total = computed(() =>
  data.value.reduce((acc: number, current) => acc + current.amount, 0),
);

const formatNumber = (value: number) => `${value.toFixed(2)} GB`;

const formatBytes = (value: number): string => {
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

const formatDate = (date: Date): string => {
  const month = date.getMonth() + 1;
  const day = date.getDate();
  return `${month}月${day}日`;
};

const xTicks = (i: number) => {
  if (i === 0 || i === data.value.length - 1 || !data.value[i]) {
    return "";
  }
  return formatDate(data.value[i].date);
};

const template = (d: DataRecord) => {
  if (!d) return "";

  const tunnelItems = (d.tunnels || [])
    .map(
      (tunnel) => `
      <div style="display:flex; justify-content:space-between; gap:1rem; padding:0.25rem 0;">
        <span style="opacity:0.8;">${tunnel.name}</span>
        <span style="font-weight:600;">${tunnel.amount.toFixed(2)}GB</span>
      </div>
    `,
    )
    .join("");

  return `
    <div>
      <div style="font-weight:600;">
        ${formatDate(d.date)}
      </div>
      <div style="font-weight:500; margin-bottom:0.5rem; padding-bottom:0.5rem; border-bottom:1px solid rgba(127,127,127,0.25);">
        ${formatNumber(d.amount)}
      </div>
      ${tunnelItems}
    </div>
  `;
};

const loadDashboard = async () => {
  try {
    const dashboard = await getDashboard();
    const trafficLimit = Number(
      dashboard.traffic?.traffic_limit ?? dashboard.user?.traffic_limit ?? 0,
    );
    const trafficUsed = Number(
      dashboard.traffic?.traffic_used ?? dashboard.user?.traffic_used ?? 0,
    );
    const trafficRemaining = Number(
      dashboard.traffic?.traffic_remaining ??
        Math.max(trafficLimit - trafficUsed, 0),
    );

    userInfo.value = {
      name: dashboard.user?.username || "-",
      email: dashboard.user?.email || "-",
      avatarUrl: dashboard.user?.avatar || "",
    };

    stats.value = {
      availableTraffic: trafficRemaining,
      tunnelCount: Number(
        dashboard.tunnel?.count ?? dashboard.tunnels?.length ?? 0,
      ),
      tunnelLimit: Number(dashboard.user?.max_tunnel_count ?? 0),
      bandwidthLimit:
        dashboard.user?.bandwidth_limit !== undefined
          ? `${dashboard.user.bandwidth_limit} Mbps`
          : "-",
    };
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "加载主页数据失败";
  }
};

const loadDailyTraffic = async () => {
  try {
    dailyTraffic.value = await getTrafficDaily(7);
  } catch (error) {
    errorMessage.value =
      error instanceof Error ? error.message : "加载近七天流量失败";
  }
};

const loadData = async () => {
  errorMessage.value = "";
  await withGlobalLoading(async () => {
    await Promise.all([loadDashboard(), loadDailyTraffic()]);
  });
};

const chartVars = {
  "--vis-crosshair-line-stroke-color": "rgb(var(--v-theme-primary))",
  "--vis-crosshair-circle-stroke-color": "rgb(var(--v-theme-surface))",
  "--vis-axis-grid-color": "rgba(var(--v-theme-on-surface), 0.08)",
  "--vis-axis-tick-color": "rgba(var(--v-theme-on-surface), 0.12)",
  "--vis-axis-tick-label-color": "rgba(var(--v-theme-on-surface), 0.6)",
  "--vis-tooltip-background-color": "rgb(var(--v-theme-surface))",
  "--vis-tooltip-border-color": "rgba(var(--v-theme-on-surface), 0.12)",
  "--vis-tooltip-text-color": "rgb(var(--v-theme-on-surface))",
  "--vis-tooltip-border-radius": "10px",
} as const;

// 格式化带宽 (MB/s 转换为 Mbps)
const formatBandwidth = (mbps: number) => {
    return (mbps * 8).toFixed(0)
}

onMounted(() => {
  void loadData();
});
</script>

<template>
  <div class="d-flex flex-column ga-4">
    <v-alert v-if="errorMessage" type="error" variant="tonal" class="mb-1">
      {{ errorMessage }}
    </v-alert>

    <v-card elevation="2" class="pa-6">
      <div class="d-flex align-center ga-4">
        <v-avatar
          :image="userInfo.avatarUrl"
          color="primary"
          size="56"
          class="flex-shrink-0"
        />
        <div class="d-flex flex-column ga-1">
          <div class="text-h5 font-weight-bold">
            {{ userInfo.name }}{{ greeting }}
          </div>
          <div class="text-body-2 text-medium-emphasis">
            {{ userInfo.email }}
          </div>
        </div>
      </div>
    </v-card>

    <v-card elevation="2">
      <v-row dense>
        <v-col cols="12" md="4">
          <div class="pa-4 d-flex flex-column">
            <v-avatar color="primary" size="40" class="mb-2">
              <v-icon size="18">fas fa-chart-line</v-icon>
            </v-avatar>
            <div class="d-flex flex-column">
              <div class="text-caption text-medium-emphasis">可用流量</div>
              <div class="text-h5 font-weight-bold">
                {{ formatBytes(stats.availableTraffic) }}
              </div>
            </div>
          </div>
        </v-col>

        <v-divider vertical />

        <v-col cols="12" md="4">
          <div class="pa-4 d-flex flex-column">
            <v-avatar color="success" size="40" class="mb-2">
              <v-icon size="18">fas fa-server</v-icon>
            </v-avatar>
            <div class="d-flex flex-column">
              <div class="text-caption text-medium-emphasis">隧道数量</div>
              <div class="text-h5 font-weight-bold">
                {{ stats.tunnelCount }} / {{ stats.tunnelLimit }}
              </div>
            </div>
          </div>
        </v-col>

        <v-divider vertical />

        <v-col cols="12" md="4">
          <div class="pa-4 d-flex flex-column">
            <v-avatar color="warning" size="40" class="mb-2">
              <v-icon size="18">fas fa-gauge-high</v-icon>
            </v-avatar>
            <div class="d-flex flex-column">
              <div class="text-caption text-medium-emphasis">带宽限制</div>
              <div class="text-h5 font-weight-bold">
                {{ stats.bandwidthLimit === "-" ? "-" : formatBandwidth(Number(stats.bandwidthLimit.split(" ")[0])) + " Mbps" }}
              </div>
            </div>
          </div>
        </v-col>
      </v-row>
    </v-card>

    <v-card ref="cardRef" elevation="2">
      <v-card-title>
        <div class="d-flex flex-column ga-1">
          <div class="text-caption text-medium-emphasis">近七天流量使用</div>
          <div class="text-h5 font-weight-bold">
            {{ formatNumber(total) }}
          </div>
        </div>
      </v-card-title>

      <v-divider />

      <v-card-text class="pa-0 pb-3">
        <VisXYContainer
          :data="data"
          :padding="{ top: 40 }"
          class="h-96"
          :width="width"
          :style="chartVars"
        >
          <VisLine
            :x="x"
            :y="y"
            color="rgb(var(--v-theme-primary))"
            :lineWidth="3"
          />
          <VisArea
            :x="x"
            :y="y"
            color="rgb(var(--v-theme-primary))"
            :opacity="0.1"
          />

          <VisAxis type="x" :x="x" :tick-format="xTicks" />

          <VisCrosshair
            color="rgb(var(--v-theme-primary))"
            :template="template"
          />

          <VisTooltip />
        </VisXYContainer>
      </v-card-text>
    </v-card>
  </div>
</template>
