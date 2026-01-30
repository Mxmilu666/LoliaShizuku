<script lang="ts" setup>
import { ref, computed } from "vue";
import {
  VisXYContainer,
  VisLine,
  VisAxis,
  VisArea,
  VisCrosshair,
  VisTooltip,
} from "@unovis/vue";
import { useElementSize } from "@vueuse/core";

defineOptions({
  name: "HomePage",
});

// 定义数据类型
interface TrafficDataPoint {
  date: Date;
  total: number;
}

// 用户信息
const userInfo = ref({
  name: "米露",
  email: "user@example.com",
  avatarUrl: "https://cn.cravatar.com/avatar/ccd1317597a7796d8b5f2b2785e88d5f?s=180&d=mp&r=g",
});

// 用户统计数据
const stats = ref({
  availableTraffic: 125.5,
  totalTraffic: 500,
  tunnelCount: 8,
  tunnelLimit: 20,
  bandwidthLimit: "100 Mbps",
});

// 获取问候语
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
const { width, height } = useElementSize(cardRef);

// 流量数据
const data: TrafficDataPoint[] = [
  { date: new Date(2025, 0, 24, 0), total: 460 },
  { date: new Date(2025, 0, 24, 2), total: 290 },
  { date: new Date(2025, 0, 24, 4), total: 195 },
  { date: new Date(2025, 0, 24, 6), total: 375 },
  { date: new Date(2025, 0, 24, 8), total: 790 },
  { date: new Date(2025, 0, 24, 10), total: 1270 },
  { date: new Date(2025, 0, 24, 12), total: 1720 },
  { date: new Date(2025, 0, 24, 14), total: 2060 },
  { date: new Date(2025, 0, 24, 16), total: 1580 },
  { date: new Date(2025, 0, 24, 18), total: 1910 },
  { date: new Date(2025, 0, 24, 20), total: 2400 },
  { date: new Date(2025, 0, 24, 22), total: 1430 },
];

// Unovis 配置
const x = (_: TrafficDataPoint, i: number) => i;
const y = (d: TrafficDataPoint) => d.total;

// 计算总流量
const total = computed(
  () => data.reduce((acc, { total }) => acc + total, 0) / 1024,
);

const formatNumber = (value: number) => `${value.toFixed(2)} GB`;

const formatTime = (date: Date): string => {
  const hours = date.getHours();
  return `${hours.toString().padStart(2, "0")}:00`;
};

const xTicks = (i: number) => {
  if (i === 0 || i === data.length - 1 || !data[i]) {
    return "";
  }
  return formatTime(data[i].date);
};

const template = (d: TrafficDataPoint) => {
  if (!d) return "";

  return `
    <div>
      <div style="font-weight: 600; margin-bottom: 0.5rem;">
        ${formatTime(d.date)}
      </div>
      <div style="font-weight: 500;">
        ${formatNumber(d.total / 1024)}
      </div>
    </div>
  `;
};
</script>

<template>
  <div class="home-container">
    <!-- 用户问候卡片 -->
    <v-card elevation="2" class="welcome-card mb-4">
      <div class="welcome-content">
        <v-avatar :image="userInfo.avatarUrl" color="primary" size="56" class="user-avatar"/>
        <div class="welcome-text">
          <div class="text-h5 font-weight-bold">
            {{ userInfo.name }}{{ greeting }}
          </div>
          <div class="text-body-2 text-medium-emphasis">
            {{ userInfo.email }}
          </div>
        </div>
      </div>
    </v-card>

    <!-- 统计卡片 -->
    <v-card elevation="2" class="mb-4">
      <div class="stats-container">
        <!-- 可用流量 -->
        <div class="stat-section">
          <div class="card-body">
            <v-avatar color="primary" size="40" class="mb-3">
              <v-icon size="18">fas fa-chart-line</v-icon>
            </v-avatar>
            <div class="text-caption text-medium-emphasis">可用流量</div>
            <div class="text-h5 font-weight-bold">
              {{ formatNumber(stats.availableTraffic) }}
            </div>
          </div>
        </div>

        <v-divider vertical />

        <!-- 隧道数量 -->
        <div class="stat-section">
          <div class="card-body">
            <v-avatar color="success" size="40" class="mb-3">
              <v-icon size="18">fas fa-server</v-icon>
            </v-avatar>
            <div class="text-caption text-medium-emphasis">隧道数量</div>
            <div class="text-h5 font-weight-bold">
              {{ stats.tunnelCount }} / {{ stats.tunnelLimit }}
            </div>
          </div>
        </div>

        <v-divider vertical />

        <!-- 带宽限制 -->
        <div class="stat-section">
          <div class="card-body">
            <v-avatar color="warning" size="40" class="mb-3">
              <v-icon size="18">fas fa-gauge-high</v-icon>
            </v-avatar>
            <div class="text-caption text-medium-emphasis">带宽限制</div>
            <div class="text-h5 font-weight-bold">
              {{ stats.bandwidthLimit }}
            </div>
          </div>
        </div>
      </div>
    </v-card>

    <!-- 图表卡片 -->
    <v-card ref="cardRef" elevation="2" class="chart-card">
      <v-card-title class="card-title-wrapper">
        <div class="card-body">
          <div class="text-caption text-medium-emphasis">
            过去 24 小时流量使用
          </div>
          <div class="text-h5 font-weight-bold">
            {{ formatNumber(total) }}
          </div>
        </div>
      </v-card-title>

      <v-divider />

      <v-card-text class="pa-0 pb-3 chart-content">
        <VisXYContainer
          :data="data"
          :padding="{ top: 40 }"
          class="h-96"
          :width="width"
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

<style scoped>
.home-container {
  padding: 1rem;
}

/* 欢迎卡片 */
.welcome-card {
  padding: 1.5rem;
}

.welcome-content {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.welcome-text {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.user-avatar {
  flex-shrink: 0;
}

/* 统计卡片 */
.stats-container {
  display: flex;
  align-items: stretch;
  height: 100%;
}

.stat-section {
  flex: 1;
  padding: 1rem;
  display: flex;
  justify-content: flex-start;
}

/* 图表卡片 */
.chart-card {
  margin-top: 1rem;
}

/* Unovis 样式 */
:deep(.unovis-xy-container) {
  --vis-crosshair-line-stroke-color: rgb(var(--v-theme-primary));
  --vis-crosshair-circle-stroke-color: rgb(var(--v-theme-surface));

  --vis-axis-grid-color: rgba(var(--v-theme-on-surface), 0.08);
  --vis-axis-tick-color: rgba(var(--v-theme-on-surface), 0.12);
  --vis-axis-tick-label-color: rgba(var(--v-theme-on-surface), 0.6);

  --vis-tooltip-background-color: rgb(var(--v-theme-surface));
  --vis-tooltip-border-color: rgba(var(--v-theme-on-surface), 0.12);
  --vis-tooltip-text-color: rgb(var(--v-theme-on-surface));
  --vis-tooltip-border-radius: 10px;
}
</style>