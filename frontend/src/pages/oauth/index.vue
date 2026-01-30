<script setup lang="ts">
import { ref } from "vue";

defineOptions({
  name: "OAuthPage",
});

const isLoading = ref(false);

function handleLogin() {
  isLoading.value = true;
  // TODO: 调用 Wails 后端进行 OAuth 认证
  setTimeout(() => {
    isLoading.value = false;
  }, 1000);
}
</script>

<template>
  <v-container class="oauth-container">
    <div class="oauth-content">
      <!-- Logo Image -->
      <div class="text-center mb-2">
        <v-img
          src="/src/assets/imgs/yuzu_happy.png"
          alt="Lolia Shizuku"
          max-width="150"
          class="mx-auto"
        />
      </div>

      <!-- Logo / Title -->
      <div class="text-center mb-7">
        <h1 class="text-h4 font-weight-bold mb-2">Lolia Shizuku</h1>
        <p class="text-subtitle-1">
          「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端
        </p>
      </div>

      <!-- Login Button -->
      <v-btn
        :loading="isLoading"
        color="primary"
        size="large"
        block
        @click="handleLogin"
      >
        <v-icon v-if="!isLoading" start>fas fa-arrow-right-long</v-icon>
        使用 Lolia FRP 账号登录
      </v-btn>
    </div>
  </v-container>
</template>

<style scoped>
.oauth-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.oauth-content {
  width: 100%;
  max-width: 400px;
}
</style>
