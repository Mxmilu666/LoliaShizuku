<script lang="ts" setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Application, Events, Window } from "@wailsio/runtime";
import AppLogo from "./AppLogo.vue";
import { isMobile } from "@/utils/platform";
import { findSettingsPanel } from "@/utils/settingsPanels";

const route = useRoute();
const router = useRouter();

// 移动端顶栏：显示当前页面标题；设置二级页显示返回按钮
const settingsSubPanel = computed(() =>
  route.path === "/settings" ? findSettingsPanel(route.query.panel) : undefined,
);
const mobileTitle = computed(
  () => settingsSubPanel.value?.title ?? (route.meta.title as string | undefined) ?? "LoliaShizuku",
);
const handleBack = () => {
  if (window.history.length > 1) {
    router.back();
  } else {
    void router.replace("/settings");
  }
};

const navItems = [
  { title: "首页", to: "/", icon: "fas fa-house" },
  { title: "隧道", to: "/tunnels", icon: "fas fa-server" },
  { title: "设置", to: "/settings", icon: "fas fa-gear" },
];
const maximised = ref(false);
let offWindowChanged: (() => void) | null = null;

// 判断是否在 OAuth 页面
const isOAuthPage = computed(() => route.path === '/oauth');

const onToggleMaximize = (isMaximised: boolean) => {
  maximised.value = isMaximised;
};

onMounted(async () => {
  // 移动端没有窗口最大化 / 最小化等概念
  if (isMobile) return;
  const isMax = await Window.IsMaximised();
  onToggleMaximize(isMax);

  offWindowChanged = Events.On("window_changed", (event) => {
    const info = event.data as { fullscreen?: boolean; maximised?: boolean };
    const isMaximised = info?.maximised;
    if (isMaximised !== undefined) {
      onToggleMaximize(isMaximised);
    }
  });
});

onUnmounted(() => {
  offWindowChanged?.();
  offWindowChanged = null;
});

async function handleMinimize() {
  Window.Minimise();
}

async function handleMaximize() {
  Window.ToggleMaximise();
}

function handleClose() {
  Application.Quit();
}
</script>

<template>
  <!-- 移动端：MD3 small top app bar（页面标题 + 右侧页面操作） -->
  <v-app-bar v-if="isMobile && !isOAuthPage" flat height="64" class="md-top-app-bar">
    <template #prepend>
      <v-btn
        v-if="settingsSubPanel"
        icon="fas fa-arrow-left"
        variant="text"
        color="on-surface"
        class="ml-1"
        title="返回"
        @click="handleBack"
      />
      <span class="text-title-large" :class="settingsSubPanel ? 'ml-1' : 'ml-4'">{{ mobileTitle }}</span>
    </template>
    <template #append>
      <!-- 页面通过 <Teleport to="#app-bar-actions"> 放置操作按钮 -->
      <div id="app-bar-actions" class="d-flex align-center mr-1" />
    </template>
  </v-app-bar>

  <!-- 桌面端：MD3 top app bar：surface container 背景，无阴影 -->
  <v-app-bar v-else flat height="64" class="md-top-app-bar">
    <template #prepend>
      <div class="d-flex align-center ga-2 ml-4">
        <AppLogo :size="20" />
        <span class="font-comfortaa text-title-medium font-weight-bold">LoliaShizuku</span>
      </div>
    </template>

    <v-spacer />

    <!-- 导航：选中项使用 secondary container 指示器 -->
    <nav v-if="!isOAuthPage && !isMobile" class="md-nav">
      <v-btn
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        variant="text"
        :prepend-icon="item.icon"
        class="md-nav-item"
      >
        {{ item.title }}
      </v-btn>
    </nav>

    <v-spacer />

    <template v-if="!isMobile" #append>
      <div class="d-flex align-center mr-2">
        <v-btn
          icon="fas fa-minus"
          variant="text"
          size="small"
          class="md-window-btn"
          title="最小化"
          @click="handleMinimize"
        />
        <v-btn
          :icon="maximised ? 'fas fa-window-restore' : 'fas fa-window-maximize'"
          variant="text"
          size="small"
          class="md-window-btn"
          :title="maximised ? '还原' : '最大化'"
          @click="handleMaximize"
        />
        <v-btn
          icon="fas fa-xmark"
          variant="text"
          size="small"
          class="md-window-btn md-window-btn--close"
          title="关闭"
          @click="handleClose"
        />
      </div>
    </template>
  </v-app-bar>
</template>

<style scoped>
.md-top-app-bar {
  background: var(--md-surface-container) !important;
  color: rgb(var(--v-theme-on-surface)) !important;
}

.md-nav {
  display: flex;
  gap: 4px;
}

.md-nav-item {
  height: 40px !important;
  padding: 0 16px !important;
  border-radius: 20px;
  color: var(--md-on-surface-variant) !important;
}

.md-nav-item :deep(.v-icon) {
  font-size: 14px;
}

/* 选中态：去掉默认 overlay，改用 secondary container 填充 */
.md-nav-item.v-btn--active {
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container) !important;
}

.md-nav-item.v-btn--active :deep(.v-btn__overlay) {
  opacity: 0;
}

.md-window-btn {
  color: var(--md-on-surface-variant) !important;
}

.md-window-btn--close:hover {
  color: rgb(var(--v-theme-error)) !important;
}
</style>
