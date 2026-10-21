<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  getFrpcConfig,
  getTunnelDetail,
  getTunnelRealtimeTraffic,
  type TunnelDetailData,
  type TunnelItem,
  type TunnelRealtimeTraffic,
} from "@/services/center";
import {
  formatBytes,
  formatDateTime,
  getAccessAddress,
  getLocalAddress,
  getTunnelStatusMeta,
  getTunnelTitle,
  getTypeMeta,
  isHTTPType,
} from "@/utils/tunnel";
import { isMobile, useFullscreenSystemBars } from "@/utils/platform";

const props = defineProps<{
  tunnel: TunnelItem | null;
  running: boolean;
  starting: boolean;
  startDisabled: boolean;
}>();

const open = defineModel<boolean>({ required: true });

const emit = defineEmits<{
  start: [];
  edit: [];
  delete: [];
  copy: [text: string, label: string];
}>();

const tab = ref<"overview" | "config">("overview");
const loading = ref(false);
const errorMessage = ref("");
const detail = ref<TunnelDetailData | null>(null);
const traffic = ref<TunnelRealtimeTraffic | null>(null);
const showToken = ref(false);

const configText = ref("");
const configLoading = ref(false);
const configError = ref("");

const loadDetail = async () => {
  const tunnel = props.tunnel;
  if (!tunnel) return;

  loading.value = true;
  errorMessage.value = "";
  try {
    const [detailResult, trafficResult] = await Promise.allSettled([
      getTunnelDetail(tunnel.name),
      getTunnelRealtimeTraffic(tunnel.name),
    ]);
    if (detailResult.status === "fulfilled") {
      detail.value = detailResult.value;
    } else {
      errorMessage.value =
        detailResult.reason instanceof Error
          ? detailResult.reason.message
          : "获取隧道详情失败";
    }
    traffic.value = trafficResult.status === "fulfilled" ? trafficResult.value : null;
  } finally {
    loading.value = false;
  }
};

const loadConfig = async () => {
  const tunnel = props.tunnel;
  if (!tunnel || configLoading.value) return;
  configLoading.value = true;
  configError.value = "";
  try {
    configText.value = await getFrpcConfig(tunnel.name);
  } catch (error) {
    configError.value = error instanceof Error ? error.message : "获取配置文件失败";
  } finally {
    configLoading.value = false;
  }
};

watch(open, (value) => {
  if (!value) return;
  tab.value = "overview";
  detail.value = null;
  traffic.value = null;
  showToken.value = false;
  configText.value = "";
  configError.value = "";
  void loadDetail();
});

useFullscreenSystemBars(computed(() => open.value && isMobile));

// 内容区是独立滚动容器：切换标签页时回到顶部，避免新内容沿用上一页的滚动位置而出现空白
const bodyRef = ref<{ $el: HTMLElement } | null>(null);

watch(tab, (value) => {
  bodyRef.value?.$el?.scrollTo({ top: 0 });
  if (value === "config" && !configText.value) {
    void loadConfig();
  }
});

// 合并列表数据与详情接口数据，详情未返回前先展示列表数据
const merged = computed(() => {
  if (!props.tunnel) return null;
  return { ...props.tunnel, ...(detail.value ?? {}) } as TunnelItem &
    Partial<TunnelDetailData>;
});

const typeMeta = computed(() => getTypeMeta(merged.value?.type || "tcp"));
const statusMeta = computed(() => getTunnelStatusMeta(merged.value?.status || ""));
const accessAddress = computed(() => (merged.value ? getAccessAddress(merged.value) : ""));

const maskedToken = computed(() => {
  const token = detail.value?.tunnel_token || "";
  if (!token) return "-";
  if (showToken.value) return token;
  return token.length <= 8 ? "••••••••" : `${token.slice(0, 4)}••••••••${token.slice(-4)}`;
});

const bandwidthText = computed(() => {
  const limit = Number(merged.value?.bandwidth_limit ?? 0);
  return limit > 0 ? `${(limit * 8).toFixed(0)} Mbps` : "不限";
});

const infoRows = computed(() => {
  const tunnel = merged.value;
  if (!tunnel) return [];
  const config = tunnel.config;
  const rows: Array<{ label: string; value: string; copy?: boolean; mono?: boolean }> = [
    { label: "隧道 ID", value: String(tunnel.id) },
    { label: "节点", value: `${tunnel.node_name || "-"} (#${tunnel.node_id})` },
    { label: "节点地址", value: tunnel.node_address || "-", copy: !!tunnel.node_address, mono: true },
    { label: "本地地址", value: getLocalAddress(tunnel), copy: true, mono: true },
  ];
  if (isHTTPType(tunnel.type)) {
    rows.push({ label: "访问域名", value: tunnel.custom_domain || "-", copy: !!tunnel.custom_domain, mono: true });
  } else {
    rows.push({ label: "远程端口", value: tunnel.remote_port > 0 ? String(tunnel.remote_port) : "-", mono: true });
  }
  rows.push(
    { label: "带宽限制", value: bandwidthText.value },
    { label: "连接协议", value: (config?.protocol || "tcp").toUpperCase() },
    { label: "Proxy Protocol", value: config?.proxy_protocol_version || "关闭" },
  );
  if (tunnel.type.toLowerCase() === "https") {
    rows.push(
      { label: "Auto TLS", value: config?.auto_tls ? "开启" : "关闭" },
      { label: "HTTP 跳转 HTTPS", value: config?.http_redirect ? "开启" : "关闭" },
    );
  }
  rows.push(
    { label: "客户端版本", value: tunnel.client_version || "-" },
    { label: "创建时间", value: formatDateTime(detail.value?.created_at) },
  );
  return rows;
});
</script>

<template>
  <v-dialog v-model="open" max-width="720" scrollable :fullscreen="isMobile">
    <v-card v-if="merged" class="md-dialog" :class="{ 'md-dialog--fullscreen': isMobile }">
      <!-- 标题 -->
      <div class="d-flex align-center ga-4 md-dialog-header">
        <div class="md-icon-container">
          <v-icon size="20">{{ typeMeta.icon }}</v-icon>
        </div>
        <div class="min-w-0 flex-grow-1">
          <h2 class="text-truncate" :class="isMobile ? 'text-title-large' : 'text-headline-small'">{{ getTunnelTitle(merged) }}</h2>
          <div class="d-flex align-center ga-2 text-body-medium md-on-surface-variant mt-1">
            <span>{{ typeMeta.label }}</span>
            <span class="md-sep" />
            <span
              class="d-inline-flex align-center ga-1"
              :class="`text-${running ? 'primary' : statusMeta.color}`"
            >
              <span class="md-status-dot" />
              {{ running ? "本地运行中" : statusMeta.text }}
            </span>
          </div>
        </div>
        <v-btn icon="fas fa-xmark" variant="text" color="on-surface" title="关闭" @click="open = false" />
      </div>

      <!-- MD3 primary tabs -->
      <!-- MD3 primary tabs：指示器由 CSS 绘制（对话框打开动画期间 Vuetify 测得宽度为 0，自带滑块不显示） -->
      <v-tabs v-model="tab" color="primary" grow hide-slider class="md-tabs">
        <v-tab value="overview" class="text-title-small">概览</v-tab>
        <v-tab value="config" class="text-title-small">frpc 配置</v-tab>
      </v-tabs>

      <v-card-text ref="bodyRef" class="md-dialog-body">
        <v-progress-linear v-if="loading" indeterminate color="primary" rounded class="mb-4" />

        <!-- 移动端关闭切换动画：动画期间两页同时占位，会造成内容区高度跳动 -->
        <v-window v-model="tab" :transition="!isMobile" :reverse-transition="!isMobile">
          <v-window-item value="overview">
            <v-alert
              v-if="errorMessage"
              type="warning"
              variant="tonal"
              rounded="lg"
              density="compact"
              class="mb-4"
            >
              {{ errorMessage }}
            </v-alert>

            <!-- 访问地址：primary container -->
            <div class="md-access mb-4">
              <div class="min-w-0 flex-grow-1">
                <div class="text-label-medium md-on-primary-container">访问地址</div>
                <div class="text-title-medium mono text-truncate">{{ accessAddress || "-" }}</div>
              </div>
              <v-btn
                v-if="accessAddress"
                prepend-icon="fas fa-copy"
                variant="flat"
                color="primary"
                @click="emit('copy', accessAddress, '访问地址')"
              >
                复制
              </v-btn>
            </div>

            <!-- 实时流量 -->
            <div class="stat-grid mb-4">
              <div class="md-stat">
                <div class="text-label-medium md-on-surface-variant">入站</div>
                <div class="text-title-medium">{{ formatBytes(traffic?.traffic_in) }}</div>
              </div>
              <div class="md-stat">
                <div class="text-label-medium md-on-surface-variant">出站</div>
                <div class="text-title-medium">{{ formatBytes(traffic?.traffic_out) }}</div>
              </div>
              <div class="md-stat">
                <div class="text-label-medium md-on-surface-variant">已用流量</div>
                <div class="text-title-medium">{{ formatBytes(traffic?.total_traffic) }}</div>
              </div>
              <div class="md-stat">
                <div class="text-label-medium md-on-surface-variant">当前连接</div>
                <div class="text-title-medium">{{ traffic?.connections ?? 0 }}</div>
              </div>
            </div>

            <!-- 基本信息：MD3 two-line list -->
            <div class="info-grid">
              <div v-for="row in infoRows" :key="row.label" class="info-item">
                <div class="min-w-0 flex-grow-1">
                  <div class="text-body-small md-on-surface-variant">{{ row.label }}</div>
                  <div class="text-body-large text-truncate" :class="{ mono: row.mono }" :title="row.value">
                    {{ row.value }}
                  </div>
                </div>
                <v-btn
                  v-if="row.copy"
                  icon="fas fa-copy"
                  variant="text"
                  size="small"
                  color="on-surface"
                  :title="`复制${row.label}`"
                  @click="emit('copy', row.value, row.label)"
                />
              </div>
              <div class="info-item">
                <div class="min-w-0 flex-grow-1">
                  <div class="text-body-small md-on-surface-variant">节点 Token</div>
                  <div class="text-body-large mono text-truncate">{{ maskedToken }}</div>
                </div>
                <template v-if="detail?.tunnel_token">
                  <v-btn
                    :icon="showToken ? 'fas fa-eye-slash' : 'fas fa-eye'"
                    variant="text"
                    size="small"
                    color="on-surface"
                    :title="showToken ? '隐藏' : '显示'"
                    @click="showToken = !showToken"
                  />
                  <v-btn
                    icon="fas fa-copy"
                    variant="text"
                    size="small"
                    color="on-surface"
                    title="复制节点 Token"
                    @click="emit('copy', detail.tunnel_token, '节点 Token')"
                  />
                </template>
              </div>
            </div>
          </v-window-item>

          <v-window-item value="config">
            <div class="d-flex align-center ga-2 mb-3">
              <span class="text-body-medium md-on-surface-variant">
                该隧道的 frpc 配置文件，可用于在其他设备上手动运行
              </span>
              <v-spacer />
              <v-btn
                icon="fas fa-rotate"
                variant="text"
                color="on-surface"
                title="重新获取"
                :loading="configLoading"
                @click="loadConfig"
              />
              <v-btn class="md-btn-tonal"
                prepend-icon="fas fa-copy"
                variant="flat"
                :disabled="!configText"
                @click="emit('copy', configText, '配置文件')"
              >
                复制
              </v-btn>
            </div>
            <v-alert
              v-if="configError"
              type="error"
              variant="tonal"
              rounded="lg"
              density="compact"
              class="mb-3"
            >
              {{ configError }}
            </v-alert>
            <div v-if="configLoading && !configText" class="d-flex justify-center py-10">
              <v-progress-circular indeterminate color="primary" />
            </div>
            <pre v-else class="config-block mono">{{ configText || "暂无配置" }}</pre>
          </v-window-item>
        </v-window>
      </v-card-text>

      <!-- MD3 对话框操作 -->
      <div class="md-dialog-actions">
        <v-btn variant="text" color="error" @click="emit('delete')">删除</v-btn>
        <v-spacer />
        <v-btn variant="text" color="primary" @click="emit('edit')">编辑</v-btn>
        <v-btn
          variant="text"
          color="primary"
          :loading="starting"
          :disabled="!running && startDisabled"
          @click="emit('start')"
        >
          {{ running ? "查看运行" : "启动隧道" }}
        </v-btn>
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.md-dialog-header {
  padding: 24px 16px 16px 24px;
}

.md-dialog-body {
  padding: 16px 24px 0 !important;
}

.md-dialog-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 24px 24px;
}

/* v-tabs 默认 flex: 1 1 auto，在可滚动对话框（纵向 flex）中会随内容区伸缩：
   内容长时被压扁、指示器被挤出可见区域；内容短时被拉高、出现大片空白。固定为自身高度。 */
.md-tabs {
  flex: 0 0 auto;
  border-bottom: 1px solid var(--md-outline-variant);
}

.md-tabs :deep(.v-tab) {
  position: relative;
}

.md-tabs :deep(.v-tab--selected .v-btn__overlay) {
  opacity: 0;
}

/* MD3 primary tab 指示器：3dp 高、顶部圆角，宽度跟随标签文字 */
.md-tabs :deep(.v-tab--selected .v-btn__content)::after {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  bottom: -14px;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: rgb(var(--v-theme-primary));
}

.md-tabs :deep(.v-btn__content) {
  position: relative;
}

.md-on-surface-variant {
  color: var(--md-on-surface-variant);
}

.md-on-primary-container {
  color: var(--md-on-primary-container);
}

.min-w-0 {
  min-width: 0;
}

.mono {
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
}

.md-icon-container {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 16px;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-sep {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
}

.md-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.md-access {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border-radius: 16px;
  background: var(--md-primary-container);
  color: var(--md-on-primary-container);
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.md-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  border-radius: 12px;
  background: var(--md-surface-container-highest);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 24px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 64px;
  border-bottom: 1px solid var(--md-outline-variant);
}

.config-block {
  margin: 0;
  padding: 16px;
  border-radius: 12px;
  max-height: 360px;
  overflow: auto;
  font-size: 0.8rem;
  line-height: 1.6;
  white-space: pre;
  background: var(--md-surface-container-highest);
}

@media (max-width: 600px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
