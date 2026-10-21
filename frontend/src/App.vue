<script lang="ts" setup>
import AppHeader from "./components/AppHeader.vue";
import { computed, ref, watch, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { useTheme } from "vuetify";
import FloatingActionButton from "./components/FloatingActionButton.vue";
import MobileNavBar from "./components/MobileNavBar.vue";
import { useGlobalLoadingStore } from "@/stores/globalLoading";
import { fullscreenDialogCount, isMobile, mobileFabCount, setSystemBars } from "@/utils/platform";

const globalLoadingStore = useGlobalLoadingStore();
const route = useRoute();
const showMobileNav = computed(() => isMobile && route.path !== "/oauth");

// 切换页面时回到顶部（内容区是独立的滚动容器，不会随路由自动复位）
const contentScroll = ref<HTMLElement | null>(null);
watch(
  () => route.fullPath,
  () => {
    contentScroll.value?.scrollTo({ top: 0 });
  },
);

// Android：状态栏与顶栏同色（surface container），导航栏与底部导航栏同色；登录页无底部导航栏时用 surface
const theme = useTheme();
if (isMobile) {
  watchEffect(() => {
    const colors = theme.current.value.colors as Record<string, string | undefined>;
    const container = colors["surface-container"] ?? colors.surface ?? "#ffffff";
    const surface = colors.surface ?? container;
    // 全屏对话框打开时，系统栏与对话框的 surface 背景一致
    const inDialog = fullscreenDialogCount.value > 0;
    setSystemBars({
      statusBar: inDialog ? surface : container,
      navigationBar: !inDialog && showMobileNav.value ? container : surface,
      dark: theme.current.value.dark,
    });
  });

  // Snackbar 需避开底部导航栏与 FAB（见 style.css）；全屏对话框会盖住二者，此时无需避让
  watchEffect(() => {
    const root = document.documentElement.classList;
    const inDialog = fullscreenDialogCount.value > 0;
    root.toggle("has-mobile-nav", showMobileNav.value && !inDialog);
    root.toggle("has-mobile-fab", mobileFabCount.value > 0 && !inDialog);
  });
}
</script>

<template>
  <v-app>
    <v-main>
      <AppHeader :style="isMobile ? undefined : '--wails-draggable: drag'" />
      <div
        ref="contentScroll"
        class="app-content-scroll"
        :class="{ 'app-content-scroll--mobile-nav': showMobileNav }"
      >
        <v-progress-linear
          :active="globalLoadingStore.isLoading"
          indeterminate
          color="primary"
          class="app-global-loading-bar"
        />
        <router-view v-slot="{ Component, route }">
          <transition name="fade" mode="out-in">
            <div :key="route.name" class="app-page-wrap">
              <v-container>
                <component :is="Component" />
              </v-container>
            </div>
          </transition>
        </router-view>
      </div>

      <!-- 桌面端：全局悬浮按钮；移动端：底部导航栏（含「运行」入口） -->
      <MobileNavBar v-if="showMobileNav" />
      <FloatingActionButton v-else-if="!isMobile" />
    </v-main>
  </v-app>
</template>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.1s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.app-content-scroll {
  height: calc(100vh - 64px);
  overflow-y: auto;
}

/* 移动端为底部导航栏让出空间 */
.app-content-scroll--mobile-nav {
  height: calc(100vh - 64px - 80px - env(safe-area-inset-bottom, 0px));
}

.app-page-wrap {
  width: 100%;
  height: 100%;
}

.app-global-loading-bar {
  position: fixed;
  left: 0;
  right: 0;
  top: 0;
  transform: translateY(64px);
  z-index: 2000;
  pointer-events: none;
}
</style>
