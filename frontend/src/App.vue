<script lang="ts" setup>
import AppHeader from "./components/AppHeader.vue";
import FloatingActionButton from "./components/FloatingActionButton.vue";
</script>

<template>
  <v-app>
    <v-main>
      <AppHeader style="--wails-draggable: drag" />
      <div style="height: calc(100vh - 64px); overflow-y: auto">
        <router-view v-slot="{ Component, route }">
          <transition name="fade" mode="out-in">
            <div :key="route.name" style="width: 100%; height: 100%">
              <v-container>
                <component :is="Component" />
              </v-container>
            </div>
          </transition>
        </router-view>
      </div>

      <!-- 全局悬浮按钮 -->
      <FloatingActionButton />
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
</style>
