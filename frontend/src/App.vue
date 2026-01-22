<script lang="ts" setup>
import HelloWorld from "./components/HelloWorld.vue";
import AppHeader from "./components/AppHeader.vue";
</script>

<template>
  <div class="h-screen flex flex-col overflow-hidden">
    <AppHeader style="--wails-draggable: drag" />
    <div class="flex-1 overflow-auto">
      <HelloWorld />
    </div>
  </div>
</template>
