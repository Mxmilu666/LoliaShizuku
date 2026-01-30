<script lang="ts" setup>
import AppHeader from "./components/AppHeader.vue";
</script>

<template>
  <v-app>
    <v-main>
      <AppHeader style="--wails-draggable: drag" />
      <div style="height: calc(100vh - 64px); overflow-y: auto;">
        <router-view v-slot="{ Component, route }">
          <v-fade-transition :leave-absolute="true">
            <div :key="route.name">
              <component :is="Component" />
            </div>
          </v-fade-transition>
        </router-view>
      </div>
    </v-main>
  </v-app>
</template>