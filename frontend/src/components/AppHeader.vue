<script lang="ts" setup>
import { ref, onMounted, onUnmounted } from "vue";
import {
  WindowMinimise,
  WindowToggleMaximise,
  WindowIsMaximised,
  EventsOn,
  EventsOff,
  Quit,
} from "../../wailsjs/runtime/runtime";
import AppLogo from "./AppLogo.vue";

const maximised = ref(false);

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
  <!-- App Header - 可拖动区域 -->
  <div class="navbar bg-base-100 shadow-sm px-2">
    <div class="flex-1 flex items-center gap-2 pl-4">
      <AppLogo />
      <span class="text-lg font-semibold font-comfortaa">LoliaShizuku</span>
    </div>

    <!-- 主题切换 -->
    <div class="dropdown dropdown-end">
      <div tabindex="0" role="button" class="btn btn-ghost btn-circle btn-sm">
        <i-lucide-sun class="w-5 h-5" />
      </div>
      <ul
        tabindex="0"
        class="dropdown-content z-[1] p-2 shadow-2xl bg-base-300 rounded-box w-32"
      >
        <li>
          <input
            type="radio"
            name="theme"
            value="light"
            class="theme-controller btn btn-sm btn-block justify-start"
            aria-label="Light"
            checked
          />
        </li>
        <li>
          <input
            type="radio"
            name="theme"
            value="dark"
            class="theme-controller btn btn-sm btn-block justify-start"
            aria-label="Dark"
          />
        </li>
        <li>
          <input
            type="radio"
            name="theme"
            value="synthwave"
            class="theme-controller btn btn-sm btn-block justify-start"
            aria-label="Synthwave"
          />
        </li>
        <li>
          <input
            type="radio"
            name="theme"
            value="cyberpunk"
            class="theme-controller btn btn-sm btn-block justify-start"
            aria-label="Cyberpunk"
          />
        </li>
        <li>
          <input
            type="radio"
            name="theme"
            value="valentine"
            class="theme-controller btn btn-sm btn-block justify-start"
            aria-label="Valentine"
          />
        </li>
      </ul>
    </div>

    <!-- 窗口控制按钮 -->
    <div class="flex gap-1">
      <button
        class="btn btn-ghost btn-circle btn-sm"
        @click="handleMinimize"
        aria-label="最小化"
      >
        <i-lucide-minus class="w-5 h-5" />
      </button>

      <button
        class="btn btn-ghost btn-circle btn-sm"
        @click="handleMaximize"
        :aria-label="maximised ? '向下还原' : '最大化'"
      >
        <i-lucide-copy v-if="maximised" class="w-5 h-5" />
        <i-lucide-panel-bottom v-else class="w-5 h-5" />
      </button>

      <button
        class="btn btn-ghost btn-circle btn-sm hover:text-error"
        @click="handleClose"
        aria-label="关闭"
      >
        <i-lucide-x class="w-5 h-5" />
      </button>
    </div>
  </div>
</template>

<style scoped>
/* 窗口控制按钮样式 */
.btn-circle.btn-sm {
  width: 2rem;
  height: 2rem;
  min-height: 2rem;
  padding: 0;
}
</style>
