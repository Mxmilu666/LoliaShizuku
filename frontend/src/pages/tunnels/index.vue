<script setup lang="ts">
import { ref } from "vue";

defineOptions({
  name: "TunnelsPage",
});

// 隧道数据类型
interface Tunnel {
  id: string;
  name: string;
  type: string;
  localAddress: string;
  localPort: number;
  remotePort: number;
  status: "running" | "stopped" | "error";
  createdAt: string;
  traffic: {
    in: number;
    out: number;
  };
}

// 隧道列表
const tunnels = ref<Tunnel[]>([
  {
    id: "1",
    name: "SSH 隧道",
    type: "tcp",
    localAddress: "127.0.0.1",
    localPort: 22,
    remotePort: 6001,
    status: "running",
    createdAt: "2025-01-20 10:30",
    traffic: { in: 125.6, out: 89.3 },
  },
  {
    id: "2",
    name: "Web 服务",
    type: "http",
    localAddress: "127.0.0.1",
    localPort: 8080,
    remotePort: 80,
    status: "running",
    createdAt: "2025-01-21 14:20",
    traffic: { in: 1567.8, out: 2341.2 },
  },
  {
    id: "3",
    name: "数据库访问",
    type: "tcp",
    localAddress: "127.0.0.1",
    localPort: 3306,
    remotePort: 6003,
    status: "stopped",
    createdAt: "2025-01-22 09:15",
    traffic: { in: 45.2, out: 32.1 },
  },
  {
    id: "4",
    name: "开发服务器",
    type: "tcp",
    localAddress: "192.168.1.100",
    localPort: 3000,
    remotePort: 6004,
    status: "stopped",
    createdAt: "2025-01-23 16:45",
    traffic: { in: 0, out: 0 },
  },
  {
    id: "5",
    name: "开发服务器",
    type: "tcp",
    localAddress: "192.168.1.100",
    localPort: 3000,
    remotePort: 6004,
    status: "stopped",
    createdAt: "2025-01-23 16:45",
    traffic: { in: 0, out: 0 },
  },
  {
    id: "6",
    name: "开发服务器",
    type: "tcp",
    localAddress: "192.168.1.100",
    localPort: 3000,
    remotePort: 6004,
    status: "stopped",
    createdAt: "2025-01-23 16:45",
    traffic: { in: 0, out: 0 },
  },
  {
    id: "7",
    name: "开发服务器",
    type: "tcp",
    localAddress: "192.168.1.100",
    localPort: 3000,
    remotePort: 6004,
    status: "stopped",
    createdAt: "2025-01-23 16:45",
    traffic: { in: 0, out: 0 },
  },
  {
    id: "8",
    name: "开发服务器",
    type: "tcp",
    localAddress: "192.168.1.100",
    localPort: 3000,
    remotePort: 6004,
    status: "stopped",
    createdAt: "2025-01-23 16:45",
    traffic: { in: 0, out: 0 },
  },
]);

// 搜索查询
const searchQuery = ref("");

// 格式化流量显示
const formatTraffic = (mb: number) => {
  if (mb < 1024) return `${mb.toFixed(2)} MB`;
  return `${(mb / 1024).toFixed(2)} GB`;
};

// 获取状态颜色
const getStatusColor = (status: string) => {
  switch (status) {
    case "running":
      return "success";
    case "stopped":
      return "grey";
    case "error":
      return "error";
    default:
      return "grey";
  }
};

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case "running":
      return "运行中";
    case "stopped":
      return "已停止";
    case "error":
      return "错误";
    default:
      return "未知";
  }
};

// 切换隧道状态
const toggleTunnel = (tunnel: Tunnel) => {
  const newStatus = tunnel.status === "running" ? "stopped" : "running";
  tunnel.status = newStatus;
};
</script>

<template>
  <div>
    <!-- 顶部操作栏 -->
    <v-card elevation="2" class="mb-4">
      <v-card-text class="d-flex align-center flex-wrap ga-4">
        <v-text-field
          v-model="searchQuery"
          label="搜索"
          prepend-inner-icon="fas fa-search"
          hide-details="auto"
          clearable
          class="flex-grow-1"
        />

        <v-btn color="primary" variant="tonal">
          <v-icon start>fas fa-plus</v-icon>
          创建隧道
        </v-btn>
      </v-card-text>
    </v-card>

    <!-- 隧道卡片网格 -->
    <v-row dense>
      <v-col v-for="tunnel in tunnels" :key="tunnel.id" cols="12" sm="6" md="4">
        <v-card elevation="2" class="h-100 d-flex flex-column">
          <v-card-title class="d-flex align-center justify-space-between">
            <span class="text-subtitle-1 font-weight-bold">
              {{ tunnel.name }}
            </span>
            <v-chip :color="getStatusColor(tunnel.status)" class="rounded-pill" size="small">
              <v-icon start size="14">fas fa-circle</v-icon>
              {{ getStatusText(tunnel.status) }}
            </v-chip>
          </v-card-title>

          <v-card-text class="d-flex flex-column ga-2 pt-0">
            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-server</v-icon>
              <span class="text-caption">
                {{ tunnel.localAddress }}:{{ tunnel.localPort }}
              </span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-arrow-right</v-icon>
              <span class="text-caption">端口 {{ tunnel.remotePort }}</span>
            </div>

            <div class="d-flex align-center">
              <v-icon size="14" class="me-2">fas fa-exchange-alt</v-icon>
              <span class="text-caption">
                ↓ {{ formatTraffic(tunnel.traffic.in) }} / ↑
                {{ formatTraffic(tunnel.traffic.out) }}
              </span>
            </div>
          </v-card-text>

          <v-divider />

          <v-card-actions class="px-4 pb-4">
            <v-spacer />
            <v-btn
              :color="tunnel.status === 'running' ? 'warning' : 'success'"
              variant="tonal"
              @click="toggleTunnel(tunnel)"
              :prepend-icon="
                tunnel.status === 'running' ? 'fas fa-stop' : 'fas fa-play'
              "
              size="small"
            >
              {{ tunnel.status === "running" ? "停止" : "启动" }}
            </v-btn>

            <v-btn
              color="info"
              variant="tonal"
              size="small"
              prepend-icon="fas fa-cog"
            >
              设置
            </v-btn>
          </v-card-actions>
        </v-card>
      </v-col>

      <!-- 空状态 -->
      <v-col v-if="tunnels.length === 0" cols="12">
        <v-card elevation="0" class="text-center py-16">
          <v-icon size="64" color="grey-lighten-1" class="mb-4">
            fas fa-folder-open
          </v-icon>
          <div class="text-h6 font-weight-bold text-medium-emphasis mb-2">
            还没有隧道
          </div>
          <div class="text-body-2 text-medium-emphasis mb-4">
            创建你的第一个隧道开始使用吧
          </div>
          <v-btn
            color="primary"
            variant="tonal"
            prepend-inner-icon="fas fa-plus"
          >
            创建隧道
          </v-btn>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>
