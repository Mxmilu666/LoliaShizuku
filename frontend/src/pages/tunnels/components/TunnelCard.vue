<script setup lang="ts">
import { computed } from "vue";
import type { TunnelItem } from "@/services/center";
import {
  formatBytes,
  getAccessAddress,
  getLocalAddress,
  getTunnelStatusMeta,
  getTunnelTitle,
  getTypeMeta,
} from "@/utils/tunnel";
import { isMobile } from "@/utils/platform";

const props = defineProps<{
  tunnel: TunnelItem;
  running: boolean;
  starting: boolean;
  startDisabled: boolean;
  trafficDays: number;
}>();

const emit = defineEmits<{
  start: [];
  detail: [];
  edit: [];
  delete: [];
  copy: [text: string];
  openWeb: [];
}>();

const typeMeta = computed(() => getTypeMeta(props.tunnel.type));
const statusMeta = computed(() =>
  props.running
    ? { text: "本地运行中", color: "primary" }
    : getTunnelStatusMeta(props.tunnel.status),
);
const title = computed(() => getTunnelTitle(props.tunnel));
const accessAddress = computed(() => getAccessAddress(props.tunnel));
</script>

<template>
  <!-- 移动端紧凑卡片：标题 / 访问地址 / 流量 + 操作，整体高度约为桌面卡片的一半 -->
  <div
    v-if="isMobile"
    class="md-card md-card--compact"
    :class="{ 'md-card--running': running }"
    role="button"
    tabindex="0"
    @click="emit('detail')"
  >
    <div class="d-flex align-center ga-3">
      <div class="md-icon-container md-icon-container--small">
        <v-icon size="15">{{ typeMeta.icon }}</v-icon>
      </div>
      <div class="min-w-0 flex-grow-1">
        <div class="text-title-medium font-weight-medium text-truncate">{{ title }}</div>
        <div class="text-body-small md-on-surface-variant text-truncate">
          {{ typeMeta.label }} · {{ tunnel.node_name || `节点 #${tunnel.node_id}` }} · {{ getLocalAddress(tunnel) }}
        </div>
      </div>
      <div class="md-status text-label-medium" :class="`text-${statusMeta.color}`">
        <span class="md-status-dot" :class="{ 'md-status-dot--pulse': running }" />
        {{ statusMeta.text }}
      </div>
    </div>

    <div class="md-address">
      <span class="text-body-medium mono text-truncate flex-grow-1">{{ accessAddress || "-" }}</span>
      <v-btn
        v-if="accessAddress"
        icon="fas fa-copy"
        variant="text"
        size="small"
        color="on-surface"
        title="复制访问地址"
        @click.stop="emit('copy', accessAddress)"
      />
    </div>

    <div class="d-flex align-center ga-1">
      <span class="d-flex align-center ga-3 text-body-small md-on-surface-variant flex-grow-1">
        <span class="d-flex align-center ga-1">
          <v-icon size="10">fas fa-arrow-down</v-icon>{{ formatBytes(tunnel.total_in) }}
        </span>
        <span class="d-flex align-center ga-1">
          <v-icon size="10">fas fa-arrow-up</v-icon>{{ formatBytes(tunnel.total_out) }}
        </span>
      </span>
      <v-btn
        icon="fas fa-pen"
        variant="text"
        size="small"
        color="on-surface"
        title="编辑"
        @click.stop="emit('edit')"
      />
      <v-menu location="bottom end">
        <template #activator="{ props: menuProps }">
          <v-btn
            v-bind="menuProps"
            icon="fas fa-ellipsis-vertical"
            variant="text"
            size="small"
            color="on-surface"
            title="更多"
            @click.stop
          />
        </template>
        <v-list density="comfortable" min-width="200">
          <v-list-item
            prepend-icon="fas fa-arrow-up-right-from-square"
            title="在网页控制台打开"
            @click="emit('openWeb')"
          />
          <v-divider class="my-2" />
          <v-list-item prepend-icon="fas fa-trash" title="删除隧道" base-color="error" @click="emit('delete')" />
        </v-list>
      </v-menu>
      <v-btn
        :icon="running ? 'fas fa-terminal' : 'fas fa-play'"
        variant="flat"
        size="small"
        :color="running ? undefined : 'primary'"
        :class="{ 'md-btn-tonal': running }"
        :loading="starting"
        :disabled="!running && startDisabled"
        :title="running ? '查看运行' : '启动'"
        @click.stop="emit('start')"
      />
    </div>
  </div>

  <div
    v-else
    class="md-card"
    :class="{ 'md-card--running': running }"
    role="button"
    tabindex="0"
    @click="emit('detail')"
    @keydown.enter.self="emit('detail')"
  >
    <!-- 头部：类型图标 + 标题 + 状态 -->
    <div class="d-flex align-center ga-4">
      <div class="md-icon-container">
        <v-icon size="18">{{ typeMeta.icon }}</v-icon>
      </div>
      <div class="min-w-0 flex-grow-1">
        <div class="text-title-medium font-weight-medium text-truncate" :title="title">
          {{ title }}
        </div>
        <div class="d-flex align-center ga-2 text-body-medium md-on-surface-variant min-w-0">
          <span class="flex-shrink-0">{{ typeMeta.label }}</span>
          <span class="md-sep" />
          <span class="text-truncate">{{ tunnel.node_name || `节点 #${tunnel.node_id}` }}</span>
        </div>
      </div>
      <div class="md-status text-label-medium" :class="`text-${statusMeta.color}`">
        <span class="md-status-dot" :class="{ 'md-status-dot--pulse': running }" />
        {{ statusMeta.text }}
      </div>
    </div>

    <!-- 映射：本地服务 → 访问地址 -->
    <div class="md-mapping">
      <div class="md-mapping-side md-mapping-side--local">
        <div class="text-label-small md-on-surface-variant">本地服务</div>
        <div class="text-body-medium mono text-truncate">{{ getLocalAddress(tunnel) }}</div>
      </div>
      <v-icon size="14" class="md-mapping-arrow md-on-surface-variant flex-shrink-0">fas fa-arrow-right-long</v-icon>
      <div class="md-mapping-side flex-grow-1">
        <div class="text-label-small md-on-surface-variant">访问地址</div>
        <div class="text-body-medium mono text-truncate" :title="accessAddress || '-'">
          {{ accessAddress || "-" }}
        </div>
      </div>
      <v-btn
        v-if="accessAddress"
        icon="fas fa-copy"
        variant="text"
        size="small"
        color="on-surface"
        class="flex-shrink-0"
        title="复制访问地址"
        @click.stop="emit('copy', accessAddress)"
      />
    </div>

    <!-- 流量 -->
    <div
      class="d-flex align-center ga-4 text-body-small md-on-surface-variant"
      :title="`近 ${trafficDays} 天流量`"
    >
      <span class="d-flex align-center ga-1">
        <v-icon size="10">fas fa-arrow-down</v-icon>
        {{ formatBytes(tunnel.total_in) }}
      </span>
      <span class="d-flex align-center ga-1">
        <v-icon size="10">fas fa-arrow-up</v-icon>
        {{ formatBytes(tunnel.total_out) }}
      </span>
      <v-spacer />
      <span>近 {{ trafficDays }} 天</span>
    </div>

    <!-- 操作 -->
    <div class="d-flex align-center ga-1 mt-auto">
      <v-btn
        variant="flat"
        :color="running ? undefined : 'primary'"
        :class="{ 'md-btn-tonal': running }"
        :prepend-icon="running ? 'fas fa-terminal' : 'fas fa-play'"
        :loading="starting"
        :disabled="!running && startDisabled"
        @click.stop="emit('start')"
      >
        {{ running ? "查看运行" : "启动" }}
      </v-btn>
      <v-spacer />
      <v-btn
        icon="fas fa-pen"
        variant="text"
        color="on-surface"
        title="编辑"
        @click.stop="emit('edit')"
      />
      <v-menu location="bottom end">
        <template #activator="{ props: menuProps }">
          <v-btn
            v-bind="menuProps"
            icon="fas fa-ellipsis-vertical"
            variant="text"
            color="on-surface"
            title="更多"
            @click.stop
          />
        </template>
        <v-list density="comfortable" min-width="200">
          <v-list-item prepend-icon="fas fa-circle-info" title="详情" @click="emit('detail')" />
          <v-list-item
            prepend-icon="fas fa-arrow-up-right-from-square"
            title="在网页控制台打开"
            @click="emit('openWeb')"
          />
          <v-divider class="my-2" />
          <v-list-item
            prepend-icon="fas fa-trash"
            title="删除隧道"
            base-color="error"
            @click="emit('delete')"
          />
        </v-list>
      </v-menu>
    </div>
  </div>
</template>

<style scoped>
/* MD3 outlined card：surface + outline-variant 描边，12dp 圆角，hover / focus 叠加 state layer */
.md-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
  padding: 16px;
  border-radius: 12px;
  background: var(--md-surface);
  border: 1px solid var(--md-outline-variant);
  cursor: pointer;
  outline: none;
  overflow: hidden;
  transition: border-color 0.2s ease;
}

.md-card::before {
  content: "";
  position: absolute;
  inset: 0;
  background: rgb(var(--v-theme-on-surface));
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
}

.md-card:hover::before {
  opacity: 0.04;
}

.md-card:focus-visible::before {
  opacity: 0.1;
}

.md-card--compact {
  gap: 12px;
  padding: 12px 12px 8px 16px;
}

.md-icon-container.md-icon-container--small {
  width: 36px;
  height: 36px;
  border-radius: 10px;
}

.md-address {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 40px;
  padding: 0 4px 0 12px;
  border-radius: 8px;
  background: var(--md-surface-container-high);
}

.md-card--running {
  border-color: rgb(var(--v-theme-primary));
}

.md-icon-container {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 12px;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-sep {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
  flex-shrink: 0;
}

.md-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.md-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.md-status-dot--pulse {
  animation: pulse 1.6s ease-in-out infinite;
}

.md-mapping {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  padding: 8px 4px 8px 12px;
  border-radius: 8px;
  background: var(--md-surface-container-high);
}

.md-mapping-side {
  min-width: 0;
}

.md-mapping-side--local {
  flex: 0 1 auto;
  max-width: 45%;
}

.md-on-surface-variant {
  color: var(--md-on-surface-variant);
}

/* 窄屏（手机）：本地服务与访问地址分两行显示，避免两侧都被截断 */
@media (max-width: 600px) {
  .md-mapping {
    flex-wrap: wrap;
    row-gap: 4px;
  }

  .md-mapping-side--local {
    flex-basis: 100%;
    max-width: none;
  }

  .md-mapping-arrow {
    display: none;
  }
}

.mono {
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
}

.min-w-0 {
  min-width: 0;
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
</style>
