<script setup lang="ts">
import { computed, ref } from "vue";

defineOptions({
  name: "RunnerPage",
});

const isRunning = ref(true);

const summary = ref({
  server: "frp.lolia.cn:7000",
  protocol: "tcp",
  version: "v0.54.0",
  pid: 24860,
  startTime: "2026-02-04 10:12:14",
});

const tunnels = ref([
  {
    name: "web-prod",
    local: "127.0.0.1:8080",
    remote: "frp.lolia.cn:31080",
    status: "在线",
    statusColor: "success",
  },
  {
    name: "ssh-dev",
    local: "127.0.0.1:22",
    remote: "frp.lolia.cn:31022",
    status: "在线",
    statusColor: "success",
  },
  {
    name: "grafana",
    local: "127.0.0.1:3000",
    remote: "frp.lolia.cn:31090",
    status: "等待重连",
    statusColor: "warning",
  },
]);

const logs = ref([
  "2026/02/04 10:12:14 [I] [service.go:316] start frpc service for config file",
  "2026/02/04 10:12:14 [I] [root.go:105] frpc version 0.54.0",
  "2026/02/04 10:12:15 [I] [login.go:92] login to server success, server address: frp.lolia.cn:7000",
  "2026/02/04 10:12:15 [I] [proxy_manager.go:173] proxy added: web-prod [tcp]",
  "2026/02/04 10:12:15 [I] [proxy_manager.go:173] proxy added: ssh-dev [tcp]",
  "2026/02/04 10:12:16 [I] [proxy_manager.go:173] proxy added: grafana [tcp]",
  "2026/02/04 10:12:22 [I] [control.go:224] [web-prod] proxy status: online",
  "2026/02/04 10:12:22 [I] [control.go:224] [ssh-dev] proxy status: online",
  "2026/02/04 10:12:25 [W] [control.go:224] [grafana] proxy status: reconnecting",
  "2026/02/04 10:12:33 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:12:43 [I] [control.go:148] ping server success, latency=21ms",
  "2026/02/04 10:12:53 [I] [control.go:148] ping server success, latency=24ms",
  "2026/02/04 10:13:03 [I] [control.go:148] ping server success, latency=22ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
  "2026/02/04 10:13:13 [I] [control.go:148] ping server success, latency=23ms",
]);

const logText = computed(() => logs.value.join("\n"));
const statusLabel = computed(() => (isRunning.value ? "运行中" : "已停止"));
const statusColor = computed(() => (isRunning.value ? "success" : "error"));
</script>

<template>
  <v-card elevation="2" class="pa-6 mb-4">
    <div class="d-flex align-center justify-space-between flex-wrap ga-6">
      <div class="flex-grow-1">
        <div class="text-h5 font-weight-bold">LoliaCLI Runner</div>
        <div class="text-caption text-medium-emphasis">
          已连接到 {{ summary.server }}（{{ summary.protocol.toUpperCase() }}）
        </div>
        <div class="d-flex flex-wrap ga-2 mt-3">
          <v-chip :color="statusColor" size="small" variant="tonal">
            {{ statusLabel }}
          </v-chip>
          <v-chip color="primary" size="small" variant="outlined">
            PID {{ summary.pid }}
          </v-chip>
          <v-chip color="info" size="small" variant="outlined">
            {{ summary.version }}
          </v-chip>
          <v-chip color="secondary" size="small" variant="outlined">
            启动于 {{ summary.startTime }}
          </v-chip>
        </div>
      </div>

      <div class="d-flex flex-wrap ga-3">
        <v-btn color="primary" prepend-icon="fas fa-stop"> 停止 </v-btn>
        <v-btn variant="tonal" prepend-icon="fas fa-rotate"> 重启 </v-btn>
      </div>
    </div>
  </v-card>

  <v-row>
    <v-col cols="12" md="4">
      <v-card elevation="2" class="h-100 d-flex flex-column">
        <v-card-title class="d-flex align-center justify-space-between">
          <div class="text-h6 font-weight-bold">隧道状态</div>
          <v-chip size="x-small" color="primary" variant="outlined">
            {{ tunnels.length }} 条规则
          </v-chip>
        </v-card-title>
        <v-divider />
        <v-card-text class="d-flex flex-column ga-3 flex-grow-1 overflow-auto">
          <v-sheet
            v-for="tunnel in tunnels"
            :key="tunnel.name"
            class="pa-3 d-flex align-center justify-space-between"
            rounded="lg"
            border
          >
            <div>
              <div class="text-subtitle-1 font-weight-bold">
                {{ tunnel.name }}
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ tunnel.local }} → {{ tunnel.remote }}
              </div>
            </div>
            <v-chip :color="tunnel.statusColor" size="x-small" variant="tonal">
              {{ tunnel.status }}
            </v-chip>
          </v-sheet>
        </v-card-text>
      </v-card>
    </v-col>

    <v-col cols="12" md="8">
      <v-card elevation="2" class="h-100">
        <v-card-title
          class="d-flex align-center justify-space-between flex-wrap ga-4"
        >
          <div>
            <div class="text-h6 font-weight-bold">LoliaCLI 输出</div>
            <div class="text-caption text-medium-emphasis">
              实时显示 LoliaCLI 的日志输出
            </div>
          </div>
          <div class="d-flex align-center ga-2">
            <v-chip size="x-small" color="success" variant="tonal">
              自动滚动：开
            </v-chip>
            <v-btn
              variant="text"
              size="small"
              prepend-icon="fas fa-trash"
              disabled
            >
              清空
            </v-btn>
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
