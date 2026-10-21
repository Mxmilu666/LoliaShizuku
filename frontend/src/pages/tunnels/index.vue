<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Browser } from "@wailsio/runtime";
import {
  deleteTunnel,
  getRunnerRuntimeStatus,
  getTunnelsPage,
  getUserDomains,
  startRunner,
  type DomainItem,
  type NodeItem,
  type RunnerRuntimeStatus,
  type TunnelItem,
} from "@/services/center";
import { useGlobalLoadingStore } from "@/stores/globalLoading";
import TunnelCard from "./components/TunnelCard.vue";
import MobileFab from "@/components/MobileFab.vue";
import { isMobile } from "@/utils/platform";
import TunnelDetailDialog from "./components/TunnelDetailDialog.vue";
import TunnelFormDialog from "./components/TunnelFormDialog.vue";
import {
  copyText,
  formatBytes,
  getTunnelTitle,
  tunnelTypes,
  type TunnelType,
} from "@/utils/tunnel";

defineOptions({
  name: "TunnelsPage",
});

const TRAFFIC_DAYS = 2;

const router = useRouter();
const globalLoadingStore = useGlobalLoadingStore();

const loaded = ref(false);
const errorMessage = ref("");
const tunnels = ref<TunnelItem[]>([]);
const nodes = ref<NodeItem[]>([]);
const domains = ref<DomainItem[]>([]);
const maxTunnelCount = ref(0);
const runnerStatus = ref<RunnerRuntimeStatus>({
  running: false,
  pid: 0,
  tunnel_names: [],
  log_lines: [],
});
const startingTunnelName = ref("");

// 筛选
const searchQuery = ref("");
const typeFilter = ref<"all" | TunnelType>("all");
const statusFilter = ref<"all" | "online" | "offline" | "running">("all");

const statusFilterItems = [
  { title: "全部状态", value: "all" },
  { title: "在线", value: "online" },
  { title: "离线", value: "offline" },
  { title: "本地运行中", value: "running" },
] as const;
const statusFilterLabel = computed(
  () => statusFilterItems.find((item) => item.value === statusFilter.value)?.title ?? "全部状态",
);

// 提示
const snackbar = ref({ show: false, text: "" });
const notify = (text: string) => {
  snackbar.value = { show: true, text };
};

// 弹窗
const formOpen = ref(false);
const formMode = ref<"create" | "edit">("create");
const formTunnel = ref<TunnelItem | null>(null);
const detailOpen = ref(false);
const detailTunnel = ref<TunnelItem | null>(null);
const deleteOpen = ref(false);
const deleteTarget = ref<TunnelItem | null>(null);
const deleting = ref(false);

const runningNames = computed(() => {
  if (!runnerStatus.value.running) return new Set<string>();
  const names = [...(runnerStatus.value.tunnel_names ?? [])];
  if (runnerStatus.value.tunnel_name) names.push(runnerStatus.value.tunnel_name);
  return new Set(names.map((name) => name.trim()));
});
const isRunning = (tunnel: TunnelItem) => runningNames.value.has(tunnel.name.trim());

const stats = computed(() => {
  const list = tunnels.value;
  return {
    total: list.length,
    online: list.filter((item) => item.status.toLowerCase() === "active").length,
    running: list.filter(isRunning).length,
    traffic: list.reduce((sum, item) => sum + Number(item.total_traffic ?? 0), 0),
  };
});

const quotaFull = computed(
  () => maxTunnelCount.value > 0 && stats.value.total >= maxTunnelCount.value,
);
const quotaPercent = computed(() =>
  maxTunnelCount.value > 0
    ? Math.min(100, (stats.value.total / maxTunnelCount.value) * 100)
    : 0,
);

const typeCounts = computed(() => {
  const counts: Record<string, number> = { all: tunnels.value.length };
  for (const item of tunnels.value) {
    const type = item.type.toLowerCase();
    counts[type] = (counts[type] ?? 0) + 1;
  }
  return counts;
});

const hasFilters = computed(
  () =>
    !!searchQuery.value.trim() ||
    typeFilter.value !== "all" ||
    statusFilter.value !== "all",
);

const filteredTunnels = computed(() => {
  const keyword = searchQuery.value.trim().toLowerCase();
  return tunnels.value.filter((tunnel) => {
    if (typeFilter.value !== "all" && tunnel.type.toLowerCase() !== typeFilter.value) {
      return false;
    }
    const online = tunnel.status.toLowerCase() === "active";
    if (statusFilter.value === "online" && !online) return false;
    if (statusFilter.value === "offline" && online) return false;
    if (statusFilter.value === "running" && !isRunning(tunnel)) return false;
    if (!keyword) return true;
    return [
      tunnel.name,
      tunnel.remark,
      tunnel.type,
      tunnel.custom_domain,
      tunnel.node_name,
      tunnel.local_ip,
      String(tunnel.local_port),
      String(tunnel.remote_port),
      String(tunnel.id),
    ]
      .join(" ")
      .toLowerCase()
      .includes(keyword);
  });
});

const clearFilters = () => {
  searchQuery.value = "";
  typeFilter.value = "all";
  statusFilter.value = "all";
};

const loadTunnels = async () => {
  errorMessage.value = "";
  await globalLoadingStore.withGlobalLoading(async () => {
    try {
      const [page, status] = await Promise.all([
        getTunnelsPage(TRAFFIC_DAYS),
        getRunnerRuntimeStatus(),
      ]);
      tunnels.value = page.tunnels ?? [];
      nodes.value = page.nodes ?? [];
      maxTunnelCount.value = page.max_tunnel_count;
      runnerStatus.value = status;
    } catch (error) {
      errorMessage.value =
        error instanceof Error ? error.message : "加载隧道列表失败，请稍后重试";
    } finally {
      loaded.value = true;
    }
  });
};

// 域名仅在创建/编辑 HTTP(S) 隧道时用于补全，失败不影响页面
const loadDomains = async () => {
  try {
    domains.value = await getUserDomains();
  } catch {
    domains.value = [];
  }
};

const handleCopy = async (text: string, label = "内容") => {
  try {
    await copyText(text);
    notify(`已复制${label}`);
  } catch {
    notify("复制失败");
  }
};

const openWeb = (tunnel: TunnelItem) => {
  Browser.OpenURL(
    `https://dash.lolia.link/dash/tunnel/${encodeURIComponent(tunnel.name.trim())}`,
  );
};

const handleStart = async (tunnel: TunnelItem) => {
  if (isRunning(tunnel)) {
    await router.push("/runner");
    return;
  }
  startingTunnelName.value = tunnel.name;
  try {
    runnerStatus.value = await startRunner([tunnel.name]);
    await router.push("/runner");
  } catch (error) {
    notify(error instanceof Error ? error.message : "启动隧道失败，请稍后重试");
  } finally {
    startingTunnelName.value = "";
  }
};

const openCreate = () => {
  formMode.value = "create";
  formTunnel.value = null;
  formOpen.value = true;
  void loadDomains();
};

const openEdit = (tunnel: TunnelItem) => {
  detailOpen.value = false;
  formMode.value = "edit";
  formTunnel.value = tunnel;
  formOpen.value = true;
  if (["http", "https"].includes(tunnel.type.toLowerCase())) {
    void loadDomains();
  }
};

const openDetail = (tunnel: TunnelItem) => {
  detailTunnel.value = tunnel;
  detailOpen.value = true;
};

const openDelete = (tunnel: TunnelItem) => {
  deleteTarget.value = tunnel;
  deleteOpen.value = true;
};

const handleSaved = async (_tunnel: TunnelItem | null, message: string) => {
  notify(message);
  await loadTunnels();
};

const confirmDelete = async () => {
  const target = deleteTarget.value;
  if (!target) return;
  deleting.value = true;
  try {
    await deleteTunnel(target.name);
    deleteOpen.value = false;
    detailOpen.value = false;
    notify(`已删除隧道「${getTunnelTitle(target)}」`);
    await loadTunnels();
  } catch (error) {
    notify(error instanceof Error ? error.message : "删除隧道失败");
  } finally {
    deleting.value = false;
  }
};

onMounted(() => {
  void loadTunnels();
});
</script>

<template>
  <div class="md-page">
    <!-- 页头 -->
    <header class="md-page-header">
      <div class="min-w-0">
        <h1 class="text-headline-small">隧道</h1>
        <p class="text-body-medium md-on-surface-variant mt-1">
          管理你的 Lolia FRP 隧道，一键在本地启动
        </p>
      </div>
      <v-spacer />
      <v-btn
        icon="fas fa-rotate"
        variant="text"
        color="on-surface"
        title="刷新"
        @click="loadTunnels"
      />
      <v-tooltip :disabled="!quotaFull" location="bottom" text="隧道数量已达上限">
        <template #activator="{ props: tooltipProps }">
          <span v-bind="tooltipProps">
            <v-btn
              color="primary"
              variant="flat"
              size="large"
              prepend-icon="fas fa-plus"
              :disabled="quotaFull"
              @click="openCreate"
            >
              新建隧道
            </v-btn>
          </span>
        </template>
      </v-tooltip>
    </header>

    <v-alert
      v-if="errorMessage"
      type="error"
      variant="tonal"
      rounded="lg"
      class="mb-4"
      closable
    >
      {{ errorMessage }}
    </v-alert>

    <!-- 概览：MD3 filled cards -->
    <!-- 移动端：刷新放到顶栏，新建使用 FAB，统计压缩为一张摘要卡 -->
    <template v-if="isMobile">
      <Teleport defer to="#app-bar-actions">
        <v-btn icon="fas fa-rotate" variant="text" color="on-surface" title="刷新" @click="loadTunnels" />
      </Teleport>
      <MobileFab icon="fas fa-plus" label="新建" :disabled="quotaFull" @click="openCreate" />

      <section class="md-card-filled mobile-summary mb-4">
        <div class="mobile-summary-item">
          <div class="text-title-medium">{{ stats.total }}<span class="md-on-surface-variant">/{{ maxTunnelCount || "-" }}</span></div>
          <div class="text-body-small md-on-surface-variant">配额</div>
        </div>
        <div class="mobile-summary-item">
          <div class="text-title-medium">{{ stats.online }}</div>
          <div class="text-body-small md-on-surface-variant">在线</div>
        </div>
        <div class="mobile-summary-item">
          <div class="text-title-medium">{{ stats.running }}</div>
          <div class="text-body-small md-on-surface-variant">本地运行</div>
        </div>
        <div class="mobile-summary-item">
          <div class="text-title-medium">{{ formatBytes(stats.traffic) }}</div>
          <div class="text-body-small md-on-surface-variant">近 {{ TRAFFIC_DAYS }} 天</div>
        </div>
      </section>
    </template>

    <section v-else class="stat-grid mb-6">
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-network-wired</v-icon>
          隧道配额
        </div>
        <div class="text-headline-small">
          {{ stats.total }}<span class="text-title-medium md-on-surface-variant"> / {{ maxTunnelCount || "-" }}</span>
        </div>
        <v-progress-linear
          :model-value="quotaPercent"
          :color="quotaFull ? 'error' : 'primary'"
          bg-color="secondary-container"
          bg-opacity="1"
          rounded
          height="4"
        />
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-signal</v-icon>
          在线隧道
        </div>
        <div class="text-headline-small">{{ stats.online }}</div>
        <div class="text-body-small md-on-surface-variant">离线 {{ stats.total - stats.online }}</div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-terminal</v-icon>
          本地运行中
        </div>
        <div class="text-headline-small">{{ stats.running }}</div>
        <div class="text-body-small md-on-surface-variant">
          {{ runnerStatus.running ? `frpc PID ${runnerStatus.pid}` : "frpc 未运行" }}
        </div>
      </div>
      <div class="md-stat">
        <div class="md-stat-label text-label-large">
          <v-icon size="14">fas fa-chart-line</v-icon>
          近 {{ TRAFFIC_DAYS }} 天流量
        </div>
        <div class="text-headline-small">{{ formatBytes(stats.traffic) }}</div>
        <div class="text-body-small md-on-surface-variant">所有隧道合计</div>
      </div>
    </section>

    <!-- 搜索 + 筛选 -->
    <section class="d-flex flex-column ga-3 mb-4">
      <!-- MD3 search bar：56dp 高、全圆角、surface-container-high -->
      <label class="md-search">
        <v-icon size="16" class="md-on-surface-variant">fas fa-magnifying-glass</v-icon>
        <input
          v-model="searchQuery"
          type="text"
          class="md-search-input text-body-large"
          placeholder="搜索名称、域名、端口、节点…"
        />
        <v-btn
          v-if="searchQuery"
          icon="fas fa-xmark"
          variant="text"
          size="small"
          color="on-surface"
          title="清除"
          @click="searchQuery = ''"
        />
      </label>

      <!-- MD3 filter chips -->
      <div class="filter-row d-flex align-center flex-wrap ga-2">
        <v-chip-group v-model="typeFilter" mandatory class="py-0 md-chip-group">
          <v-chip value="all" variant="outlined" filter class="md-filter-chip">
            全部 · {{ typeCounts.all ?? 0 }}
          </v-chip>
          <v-chip
            v-for="item in tunnelTypes"
            :key="item.value"
            :value="item.value"
            variant="outlined"
            filter
            class="md-filter-chip"
            :disabled="!typeCounts[item.value]"
          >
            {{ item.label }} · {{ typeCounts[item.value] ?? 0 }}
          </v-chip>
        </v-chip-group>
        <v-spacer />
        <v-menu location="bottom end">
          <template #activator="{ props: menuProps }">
            <v-chip
              v-bind="menuProps"
              variant="outlined"
              class="md-filter-chip"
              :class="{ 'md-filter-chip--selected': statusFilter !== 'all' }"
              append-icon="fas fa-caret-down"
            >
              {{ statusFilterLabel }}
            </v-chip>
          </template>
          <v-list density="comfortable" min-width="180">
            <v-list-item
              v-for="item in statusFilterItems"
              :key="item.value"
              :title="item.title"
              :active="statusFilter === item.value"
              color="primary"
              @click="statusFilter = item.value"
            />
          </v-list>
        </v-menu>
      </div>
    </section>

    <!-- 列表 -->
    <div v-if="!loaded" class="tunnel-grid">
      <v-skeleton-loader
        v-for="n in 6"
        :key="n"
        type="list-item-avatar-two-line, paragraph, actions"
        class="md-skeleton"
      />
    </div>

    <div v-else-if="filteredTunnels.length > 0" class="tunnel-grid">
      <TunnelCard
        v-for="tunnel in filteredTunnels"
        :key="tunnel.id"
        :tunnel="tunnel"
        :running="isRunning(tunnel)"
        :starting="startingTunnelName === tunnel.name"
        :start-disabled="!!startingTunnelName"
        :traffic-days="TRAFFIC_DAYS"
        @start="handleStart(tunnel)"
        @detail="openDetail(tunnel)"
        @edit="openEdit(tunnel)"
        @delete="openDelete(tunnel)"
        @copy="(text) => handleCopy(text, '访问地址')"
        @open-web="openWeb(tunnel)"
      />
    </div>

    <section v-else class="md-empty">
      <template v-if="tunnels.length === 0 && !errorMessage">
        <div class="md-empty-icon"><v-icon size="28">fas fa-route</v-icon></div>
        <h2 class="text-title-large">还没有隧道</h2>
        <p class="text-body-medium md-on-surface-variant">创建一条隧道，把本地服务映射到公网</p>
        <v-btn color="primary" variant="flat" prepend-icon="fas fa-plus" :disabled="quotaFull" @click="openCreate">
          新建隧道
        </v-btn>
      </template>
      <template v-else-if="hasFilters">
        <div class="md-empty-icon"><v-icon size="28">fas fa-filter-circle-xmark</v-icon></div>
        <h2 class="text-title-large">没有匹配的隧道</h2>
        <p class="text-body-medium md-on-surface-variant">试试调整搜索关键词或筛选条件</p>
        <v-btn class="md-btn-tonal" variant="flat" prepend-icon="fas fa-eraser" @click="clearFilters">
          清除筛选
        </v-btn>
      </template>
      <template v-else>
        <div class="md-empty-icon"><v-icon size="28">fas fa-plug-circle-exclamation</v-icon></div>
        <h2 class="text-title-large">加载失败</h2>
        <p class="text-body-medium md-on-surface-variant">请检查网络后重试</p>
        <v-btn class="md-btn-tonal" variant="flat" prepend-icon="fas fa-rotate" @click="loadTunnels">
          重新加载
        </v-btn>
      </template>
    </section>

    <TunnelFormDialog
      v-model="formOpen"
      :mode="formMode"
      :tunnel="formTunnel"
      :nodes="nodes"
      :domains="domains"
      @saved="handleSaved"
    />

    <TunnelDetailDialog
      v-model="detailOpen"
      :tunnel="detailTunnel"
      :running="detailTunnel ? isRunning(detailTunnel) : false"
      :starting="!!detailTunnel && startingTunnelName === detailTunnel.name"
      :start-disabled="!!startingTunnelName"
      @start="detailTunnel && handleStart(detailTunnel)"
      @edit="detailTunnel && openEdit(detailTunnel)"
      @delete="detailTunnel && openDelete(detailTunnel)"
      @copy="handleCopy"
    />

    <!-- MD3 basic dialog：带图标的确认对话框 -->
    <v-dialog v-model="deleteOpen" max-width="400" :persistent="deleting">
      <div v-if="deleteTarget" class="md-dialog text-center">
        <v-icon size="24" color="error">fas fa-trash</v-icon>
        <h2 class="text-headline-small">删除隧道？</h2>
        <p class="text-body-medium md-on-surface-variant text-left">
          隧道「{{ getTunnelTitle(deleteTarget) }}」将立即下线且无法恢复。
          <template v-if="isRunning(deleteTarget)">
            该隧道正在本地运行，请先在运行页停止。
          </template>
        </p>
        <div class="md-dialog-actions">
          <v-btn variant="text" color="primary" :disabled="deleting" @click="deleteOpen = false">
            取消
          </v-btn>
          <v-btn
            variant="text"
            color="error"
            :loading="deleting"
            :disabled="isRunning(deleteTarget)"
            @click="confirmDelete"
          >
            删除
          </v-btn>
        </div>
      </div>
    </v-dialog>

    <v-snackbar v-model="snackbar.show" timeout="2400" location="bottom">
      {{ snackbar.text }}
    </v-snackbar>
  </div>
</template>

<style scoped>
.min-w-0 {
  min-width: 0;
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.md-search {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 56px;
  padding: 0 8px 0 20px;
  border-radius: 28px;
  background: var(--md-surface-container-high);
  cursor: text;
}

.md-search-input {
  flex: 1;
  min-width: 0;
  height: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: rgb(var(--v-theme-on-surface));
}

.md-search-input::placeholder {
  color: var(--md-on-surface-variant);
}

.md-chip-group :deep(.v-slide-group__content) {
  gap: 8px;
}

/* MD3 filter chip：未选中为 outline-variant 描边，选中为 secondary-container 填充 */
.md-filter-chip {
  border-color: var(--md-outline-variant) !important;
  color: var(--md-on-surface-variant) !important;
}

.md-filter-chip.v-chip--selected,
.md-filter-chip--selected {
  background: var(--md-secondary-container) !important;
  border-color: transparent !important;
  color: var(--md-on-secondary-container) !important;
}

.mobile-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  padding: 12px 8px;
}

.mobile-summary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 0;
  text-align: center;
  white-space: nowrap;
}

/* 移动端：搜索栏收窄，筛选 chip 与状态筛选合并为一行横向滚动 */
.is-mobile .md-search {
  height: 48px;
  padding-left: 16px;
}

.is-mobile .filter-row {
  flex-wrap: nowrap;
}

.is-mobile .filter-row .md-chip-group {
  flex: 1;
  min-width: 0;
}

.is-mobile .filter-row > .v-spacer {
  display: none;
}

.tunnel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(400px, 100%), 1fr));
  gap: 12px;
}

.md-skeleton {
  border-radius: 12px;
  background: var(--md-surface-container-low);
}

.md-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 56px 24px;
  border-radius: 12px;
  background: var(--md-surface-container-low);
  text-align: center;
}

.md-empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  margin-bottom: 4px;
  border-radius: 20px;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-empty .v-btn {
  margin-top: 8px;
}

/* MD3 dialog：28dp 圆角、24dp 内边距、surface-container-high */
.md-dialog {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 24px;
  border-radius: 28px;
  background: var(--md-surface-container-high);
  color: rgb(var(--v-theme-on-surface));
}

.md-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  width: 100%;
  margin-top: 8px;
}

@media (max-width: 960px) {
  .stat-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
