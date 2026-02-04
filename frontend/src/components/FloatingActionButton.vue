<script lang="ts" setup>
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

// 判断是否显示悬浮按钮（OAuth 页面不显示）
const showFab = computed(() => route.path !== '/oauth' && route.path !== '/runner');

// 点击事件
const handleClick = () => {
  // TODO: 根据当前页面执行不同的操作
  console.log('FAB clicked on route:', route.path);
};
</script>

<template>
  <v-fab
    v-if="showFab"
    class="fab-global"
    color="primary"
    size="large"
    location="bottom end"
    @click="handleClick"
    prepend-icon="fas fa-terminal"
    to="/runner"
  >
    回到运行
  </v-fab>
</template>

<style scoped>
.fab-global {
  position: fixed !important;
  bottom: 24px !important;
  right: 24px !important;
  z-index: 1000;
}
</style>
