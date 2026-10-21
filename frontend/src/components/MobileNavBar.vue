<script lang="ts" setup>
import { useRoute } from "vue-router";

const route = useRoute();

const navItems = [
  { title: "首页", to: "/", icon: "fas fa-house" },
  { title: "隧道", to: "/tunnels", icon: "fas fa-server" },
  { title: "运行", to: "/runner", icon: "fas fa-terminal" },
  { title: "设置", to: "/settings", icon: "fas fa-gear" },
];

const isActive = (to: string) => (to === "/" ? route.path === "/" : route.path.startsWith(to));
</script>

<template>
  <!-- MD3 navigation bar：80dp 高、surface container 背景、选中项 64×32 胶囊指示器 -->
  <nav class="md-navigation-bar">
    <router-link
      v-for="item in navItems"
      :key="item.to"
      :to="item.to"
      class="md-navigation-item"
      :class="{ 'md-navigation-item--active': isActive(item.to) }"
    >
      <span class="md-navigation-indicator">
        <v-icon size="18">{{ item.icon }}</v-icon>
      </span>
      <span class="text-label-medium">{{ item.title }}</span>
    </router-link>
  </nav>
</template>

<style scoped>
.md-navigation-bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1004;
  display: flex;
  height: calc(80px + env(safe-area-inset-bottom, 0px));
  padding-bottom: env(safe-area-inset-bottom, 0px);
  background: var(--md-surface-container);
}

.md-navigation-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: var(--md-on-surface-variant);
  text-decoration: none;
  -webkit-tap-highlight-color: transparent;
}

.md-navigation-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 32px;
  border-radius: 16px;
  transition: background-color 0.2s ease;
}

.md-navigation-item--active {
  color: rgb(var(--v-theme-on-surface));
}

.md-navigation-item--active .md-navigation-indicator {
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}
</style>
