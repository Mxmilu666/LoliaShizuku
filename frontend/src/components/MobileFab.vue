<script lang="ts" setup>
import { onBeforeUnmount, onMounted } from "vue";
import { mobileFabCount } from "@/utils/platform";

defineProps<{
  icon: string;
  label: string;
  loading?: boolean;
  disabled?: boolean;
  danger?: boolean;
}>();

defineEmits<{ click: [] }>();

onMounted(() => {
  mobileFabCount.value += 1;
});
onBeforeUnmount(() => {
  mobileFabCount.value -= 1;
});
</script>

<template>
  <!-- MD3 extended FAB（移动端）：位于底部导航栏上方右侧 -->
  <v-btn
    class="md-mobile-fab"
    :class="{ 'md-mobile-fab--danger': danger }"
    :prepend-icon="icon"
    :loading="loading"
    :disabled="disabled"
    variant="flat"
    @click="$emit('click')"
  >
    {{ label }}
  </v-btn>
</template>

<style scoped>
.md-mobile-fab {
  position: fixed;
  right: 16px;
  bottom: calc(80px + 16px + env(safe-area-inset-bottom, 0px));
  z-index: 1003;
  height: 56px !important;
  padding: 0 20px !important;
  border-radius: 16px !important;
  background: var(--md-primary-container) !important;
  color: var(--md-on-primary-container) !important;
  box-shadow:
    0 1px 3px rgba(0, 0, 0, 0.3),
    0 4px 8px 3px rgba(0, 0, 0, 0.15) !important;
}

.md-mobile-fab--danger {
  background: rgb(var(--v-theme-error-container)) !important;
  color: rgb(var(--v-theme-on-error-container)) !important;
}
</style>
