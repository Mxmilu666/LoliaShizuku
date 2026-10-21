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
  getHomeAnnouncements,
  getTrafficDaily,
  type Announcement,
  type DailyTrafficResponse,
} from "@/services/center";
import { Browser } from "@wailsio/runtime";
import AnnouncementContent from "@/components/AnnouncementContent.vue";
import { getAnnouncementLevelMeta, safeExternalURL } from "@/utils/announcement";
import { useGlobalLoadingStore } from "@/stores/globalLoading";
import { isMobile } from "@/utils/platform";

// 统计数值字号：移动端三列并排，使用更小的字号
const statValueClass = isMobile ? "text-title-medium" : "text-headline-small";

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
  trafficLimit: 0,
  trafficUsed: 0,
  tunnelCount: 0,
  tunnelLimit: 0,
  // 中心端单位为 MB/s
  bandwidthLimit: null as number | null,
});

// 中心端以接近 int64 上限的值表示不限流量
const UNLIMITED_TRAFFIC = 2 ** 62;
const trafficUnlimited = computed(() => stats.value.trafficLimit >= UNLIMITED_TRAFFIC);

const trafficUsedPercent = computed(() =>
  stats.value.trafficLimit > 0
    ? Math.min(100, (stats.value.trafficUsed / stats.value.trafficLimit) * 100)
    : 0,
);
const tunnelPercent = computed(() =>
  stats.value.tunnelLimit > 0
    ? Math.min(100, (stats.value.tunnelCount / stats.value.tunnelLimit) * 100)
    : 0,
);

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
    // 以字节为单位绘图，展示时再按大小自适应换算单位（流量较小时按 GB 会全部显示为 0）
    amount: Number(stat.total_traffic || 0),
    tunnels:
      stat.tunnel_stats?.map((tunnel) => ({
        name: tunnel.remark || tunnel.tunnel_name,
        amount: Number(tunnel.total_traffic || 0),
      })) ?? [],
  }));
});

const x = (_: DataRecord, i: number) => i;
const y = (d: DataRecord) => d.amount;

const total = computed(() =>
  data.value.reduce((acc: number, current) => acc + current.amount, 0),
);

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
        <span style="font-weight:600;">${formatBytes(tunnel.amount)}</span>
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
        ${formatBytes(d.amount)}
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
      trafficLimit,
      trafficUsed,
      tunnelCount: Number(
        dashboard.tunnel?.count ?? dashboard.tunnels?.length ?? 0,
      ),
      tunnelLimit: Number(dashboard.user?.max_tunnel_count ?? 0),
      bandwidthLimit:
        dashboard.user?.bandwidth_limit !== undefined
          ? Number(dashboard.user.bandwidth_limit)
          : null,
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

// 公告：置顶横幅 + 公告板；加载失败不影响首页其他内容
const topAnnouncements = ref<Announcement[]>([]);
const boardAnnouncements = ref<Announcement[]>([]);
const announcementsLoading = ref(false);
const announcementsError = ref("");

const loadAnnouncements = async () => {
  announcementsLoading.value = true;
  announcementsError.value = "";
  try {
    const data = await getHomeAnnouncements();
    topAnnouncements.value = data.top;
    boardAnnouncements.value = data.board;
  } catch (error) {
    announcementsError.value = error instanceof Error ? error.message : "公告加载失败";
  } finally {
    announcementsLoading.value = false;
  }
};

const openAnnouncementAction = (value: string) => {
  const url = safeExternalURL(value);
  if (url) {
    void Browser.OpenURL(url);
  }
};

const loadData = async () => {
  errorMessage.value = "";
  await withGlobalLoading(async () => {
    await Promise.all([loadDashboard(), loadDailyTraffic(), loadAnnouncements()]);
  });
};

const chartVars = {
  "--vis-crosshair-line-stroke-color": "rgb(var(--v-theme-primary))",
  "--vis-crosshair-circle-stroke-color": "rgb(var(--v-theme-surface))",
  "--vis-axis-grid-color": "rgba(var(--v-theme-on-surface), 0.08)",
  "--vis-axis-tick-color": "rgba(var(--v-theme-on-surface), 0.12)",
  "--vis-axis-tick-label-color": "rgba(var(--v-theme-on-surface), 0.6)",
  "--vis-tooltip-background-color": "var(--md-surface-container-high)",
  "--vis-tooltip-border-color": "transparent",
  "--vis-tooltip-text-color": "rgb(var(--v-theme-on-surface))",
  "--vis-tooltip-border-radius": "12px",
} as const;

// 格式化带宽 (MB/s 转换为 Mbps)
const bandwidthText = computed(() =>
  stats.value.bandwidthLimit === null
    ? "-"
    : `${(stats.value.bandwidthLimit * 8).toFixed(0)} Mbps`,
);

onMounted(() => {
  void loadData();
});
</script>

<template>
  <div class="md-page d-flex flex-column ga-4">
    <v-alert v-if="errorMessage" type="error" variant="tonal" rounded="lg">
      {{ errorMessage }}
    </v-alert>

    <!-- 问候：primary container 强调卡片 -->
    <section class="md-card-primary d-flex align-center ga-4">
      <!-- Vuetify 4 中提供默认插槽时会忽略 image 属性，因此在插槽内显式渲染头像 -->
      <v-avatar color="primary" :size="isMobile ? 44 : 56" class="flex-shrink-0">
        <v-img v-if="userInfo.avatarUrl" :src="userInfo.avatarUrl" :alt="userInfo.name" cover />
        <span v-else class="text-title-large">
          {{ userInfo.name.slice(0, 1).toUpperCase() }}
        </span>
      </v-avatar>
      <div class="min-w-0">
        <h1 :class="isMobile ? 'text-title-medium' : 'text-headline-small'">{{ userInfo.name }}，{{ greeting }}</h1>
        <p class="text-body-medium md-on-primary-container mt-1">{{ userInfo.email }}</p>
      </div>
    </section>

    <!-- 置顶公告 -->
    <section
      v-for="item in topAnnouncements"
      :key="`top-${item.id}`"
      class="md-banner"
      :class="getAnnouncementLevelMeta(item.level).color ? `md-banner--${getAnnouncementLevelMeta(item.level).color}` : ''"
    >
      <v-icon size="18" class="md-banner-icon">{{ getAnnouncementLevelMeta(item.level).icon }}</v-icon>
      <div class="min-w-0 flex-grow-1">
        <div class="text-title-small">{{ item.title }}</div>
        <AnnouncementContent :content="item.content" class="text-body-medium mt-1" />
      </div>
      <v-btn
        v-if="item.action_label && item.action_url"
        variant="text"
        class="md-banner-action flex-shrink-0"
        @click="openAnnouncementAction(item.action_url)"
      >
        {{ item.action_label }}
      </v-btn>
    </section>

    <!-- 概览 -->
    <section class="stat-grid">
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-chart-pie</v-icon>
          可用流量
        </div>
        <div :class="statValueClass">
          {{ trafficUnlimited ? "不限" : formatBytes(stats.availableTraffic) }}
        </div>
        <v-progress-linear
          v-if="!trafficUnlimited"
          :model-value="trafficUsedPercent"
          color="primary"
          bg-color="secondary-container"
          bg-opacity="1"
          rounded
          height="4"
        />
        <div class="text-body-small md-on-surface-variant">
          已用 {{ formatBytes(stats.trafficUsed) }}<template v-if="!trafficUnlimited"> / {{ formatBytes(stats.trafficLimit) }}</template>
        </div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-server</v-icon>
          隧道数量
        </div>
        <div :class="statValueClass">
          {{ stats.tunnelCount }}<span class="text-title-medium md-on-surface-variant"> / {{ stats.tunnelLimit }}</span>
        </div>
        <v-progress-linear
          :model-value="tunnelPercent"
          color="primary"
          bg-color="secondary-container"
          bg-opacity="1"
          rounded
          height="4"
        />
        <div class="text-body-small md-on-surface-variant">
          还可创建 {{ Math.max(stats.tunnelLimit - stats.tunnelCount, 0) }} 条
        </div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-gauge-high</v-icon>
          带宽限制
        </div>
        <div :class="statValueClass">{{ bandwidthText }}</div>
        <div class="text-body-small md-on-surface-variant">单条隧道的最大速率</div>
      </div>
    </section>

    <div class="home-grid">
    <!-- 近七天流量 -->
    <section ref="cardRef" class="md-card-outlined chart-card">
      <div class="d-flex align-end justify-space-between flex-wrap ga-2 px-4 pt-4">
        <div>
          <h2 class="text-title-medium">近七天流量</h2>
          <p class="text-body-small md-on-surface-variant mt-1">悬停查看每日各隧道用量</p>
        </div>
        <div class="text-headline-small">{{ formatBytes(total) }}</div>
      </div>

      <div class="pb-3">
        <VisXYContainer
          :data="data"
          :padding="{ top: 40 }"
          :height="isMobile ? 200 : 300"
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
            :opacity="0.12"
          />

          <VisAxis type="x" :x="x" :tick-format="xTicks" />

          <VisCrosshair
            color="rgb(var(--v-theme-primary))"
            :template="template"
          />

          <VisTooltip />
        </VisXYContainer>
      </div>
    </section>

    <!-- 公告板 -->
    <section class="md-card-outlined board-card">
      <div class="d-flex align-center ga-2 mb-3">
        <v-icon size="15" class="md-on-surface-variant">fas fa-bullhorn</v-icon>
        <h2 class="text-title-medium">公告</h2>
        <v-spacer />
        <v-btn
          icon="fas fa-rotate"
          variant="text"
          size="small"
          color="on-surface"
          title="刷新公告"
          :loading="announcementsLoading"
          @click="loadAnnouncements"
        />
      </div>
      <div v-if="announcementsError" class="text-body-medium text-error">{{ announcementsError }}</div>
      <div
        v-else-if="announcementsLoading && boardAnnouncements.length === 0"
        class="text-body-medium md-on-surface-variant"
      >
        正在加载公告…
      </div>
      <div v-else-if="boardAnnouncements.length === 0" class="board-empty text-body-medium md-on-surface-variant">
        暂无公告
      </div>
      <div v-else class="board-list">
        <article v-for="item in boardAnnouncements" :key="`board-${item.id}`" class="board-item">
          <div class="d-flex align-center ga-2">
            <v-icon
              size="13"
              :color="getAnnouncementLevelMeta(item.level).color"
              :class="{ 'md-on-surface-variant': !getAnnouncementLevelMeta(item.level).color }"
            >
              {{ getAnnouncementLevelMeta(item.level).icon }}
            </v-icon>
            <h3 class="text-title-small min-w-0">{{ item.title }}</h3>
          </div>
          <AnnouncementContent :content="item.content" class="text-body-medium md-on-surface-variant mt-1" />
          <v-btn
            v-if="item.action_label && item.action_url"
            variant="text"
            color="primary"
            size="small"
            class="mt-1 ml-n2"
            append-icon="fas fa-arrow-up-right-from-square"
            @click="openAnnouncementAction(item.action_url)"
          >
            {{ item.action_label }}
          </v-btn>
        </article>
      </div>
    </section>
    </div>
  </div>
</template>

<style scoped>
.min-w-0 {
  min-width: 0;
}

.md-on-primary-container {
  color: var(--md-on-primary-container);
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

/* 公告横幅：MD3 tonal 容器，按等级着色；normal 使用 secondary container */
.md-banner {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 8px 14px 16px;
  border-radius: 12px;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-banner-icon {
  margin-top: 2px;
}

.md-banner-action {
  margin-top: -6px;
  color: inherit !important;
}

.md-banner--info,
.md-banner--warning,
.md-banner--error,
.md-banner--success {
  --banner-color: var(--v-theme-info);
  background: color-mix(in srgb, rgb(var(--banner-color)) 14%, rgb(var(--v-theme-surface)));
  color: rgb(var(--v-theme-on-surface));
}

.md-banner--warning { --banner-color: var(--v-theme-warning); }
.md-banner--error { --banner-color: var(--v-theme-error); }
.md-banner--success { --banner-color: var(--v-theme-success); }

.md-banner--info .md-banner-icon,
.md-banner--warning .md-banner-icon,
.md-banner--error .md-banner-icon,
.md-banner--success .md-banner-icon {
  color: rgb(var(--banner-color));
}

.home-grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
  gap: 12px;
  align-items: stretch;
}

.board-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.board-list {
  display: flex;
  flex-direction: column;
  max-height: 360px;
  overflow-y: auto;
}

.board-item {
  padding: 12px 0;
}

.board-item:first-child {
  padding-top: 0;
}

.board-item + .board-item {
  border-top: 1px solid var(--md-outline-variant);
}

.board-empty {
  padding: 24px 0;
  text-align: center;
}

@media (max-width: 960px) {
  .home-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  /* 窄屏 / 手机：公告比图表更重要，排在前面 */
  .board-card {
    order: -1;
  }

  .board-list {
    max-height: none;
  }
}

.chart-card {
  padding: 0;
  overflow: hidden;
}

@media (max-width: 760px) {
  .stat-grid {
    grid-template-columns: 1fr;
  }
}

/* 移动端：问候区更紧凑，三项统计并排显示，隐藏次要说明文字 */
.is-mobile .md-card-primary {
  padding: 16px;
  gap: 12px !important;
}

.is-mobile .stat-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.is-mobile .stat-grid .md-stat {
  gap: 6px;
  padding: 12px;
}

.is-mobile .stat-grid .md-stat-label {
  font-size: 0.75rem;
  gap: 6px;
}

.is-mobile .stat-grid .md-stat > div:nth-child(2) {
  white-space: nowrap;
}

.is-mobile .stat-grid .text-body-small {
  display: none;
}
</style>
