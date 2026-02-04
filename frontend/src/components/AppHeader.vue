<script lang="ts" setup>
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import {
  WindowMinimise,
  WindowToggleMaximise,
  WindowIsMaximised,
  EventsOn,
  EventsOff,
  Quit,
} from "../../wailsjs/runtime/runtime";
import AppLogo from "./AppLogo.vue";
import { useTheme } from "vuetify";

const router = useRouter();
const route = useRoute();
const maximised = ref(false);
const theme = useTheme();

const currentRoute = ref("home");

// 判断是否在 OAuth 页面
const isOAuthPage = computed(() => route.path === '/oauth');

const onToggleMaximize = (isMaximised: boolean) => {
  maximised.value = isMaximised;
};

onMounted(async () => {
  const isMax = await WindowIsMaximised();
  onToggleMaximize(isMax);

  EventsOn(
    "window_changed",
    (info: { fullscreen?: boolean; maximised?: boolean }) => {
      const { maximised: isMaximised } = info;
      if (isMaximised !== undefined) {
        onToggleMaximize(isMaximised);
      }
    },
  );
});

onUnmounted(() => {
  EventsOff("window_changed");
});

async function handleMinimize() {
  WindowMinimise();
}

async function handleMaximize() {
  WindowToggleMaximise();
}

function handleClose() {
  Quit();
}
</script>

<template>
  <v-app-bar color="appbar" app>
    <template #prepend>
      <div
        style="display: flex; align-items: center; gap: 8px; margin-left: 17px"
      >
        <AppLogo :size="20" />
        <v-app-bar-title class="font-comfortaa text-lg font-semibold">
          LoliaShizuku
        </v-app-bar-title>
      </div>
    </template>

    <v-spacer />

    <!-- 中间导航按钮 -->
    <div v-if="!isOAuthPage" class="nav-buttons" style="display: flex; gap: 8px">
      <v-btn
        to="/"
      >
        <v-icon start>fas fa-home</v-icon>
        首页
      </v-btn>

      <v-btn
        to="/tunnels"
      >
        <v-icon start>fas fa-server</v-icon>
        隧道
      </v-btn>

      <v-btn
        to="/settings"
      >
        <v-icon start>fas fa-cog</v-icon>
        设置
      </v-btn>
    </div>

    <v-spacer />

    <template #append>
      <!-- 窗口控制按钮 -->
      <v-btn
        icon="fas fa-minus"
        variant="text"
        size="small"
        @click="handleMinimize"
      />

      <v-btn
        :icon="maximised ? 'fas fa-window-restore' : 'fas fa-window-maximize'"
        variant="text"
        size="small"
        @click="handleMaximize"
      />

      <v-btn
        icon="fas fa-times"
        variant="text"
        size="small"
        @click="handleClose"
        class="mr-2"
      />
    </template>
  </v-app-bar>
</template>
