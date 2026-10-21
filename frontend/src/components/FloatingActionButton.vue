<script lang="ts" setup>
import { onBeforeUnmount, onMounted, computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { getRunnerRuntimeStatus } from "@/services/center";

const route = useRoute();
const runnerRunning = ref(false);
let timer: ReturnType<typeof setInterval> | null = null;

const refreshRunnerStatus = async () => {
  try {
    const status = await getRunnerRuntimeStatus();
    runnerRunning.value = !!status.running;
  } catch {
    runnerRunning.value = false;
  }
};

const showFab = computed(
  () => runnerRunning.value && route.path !== "/oauth" && route.path !== "/runner",
);

onMounted(() => {
  void refreshRunnerStatus();
  timer = setInterval(() => {
    void refreshRunnerStatus();
  }, 3000);
});

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
});

watch(
  () => route.path,
  () => {
    void refreshRunnerStatus();
  },
);
</script>

<template>
  <!-- MD3 extended FAB：primary container、16dp 圆角 -->
  <v-fab
    v-if="showFab"
    class="md-fab"
    size="large"
    location="bottom end"
    prepend-icon="fas fa-terminal"
    to="/runner"
  >
    回到运行
  </v-fab>
</template>

<style scoped>
.md-fab {
  position: fixed !important;
  bottom: 24px !important;
  right: 24px !important;
  z-index: 1000;
}

.md-fab :deep(.v-btn) {
  height: 56px;
  padding: 0 20px;
  border-radius: 16px;
  background: var(--md-primary-container);
  color: var(--md-on-primary-container);
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.3),
    0 4px 8px 3px rgba(0, 0, 0, 0.15);
}
</style>
