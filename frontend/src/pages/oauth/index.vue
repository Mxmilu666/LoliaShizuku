<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { TokenService } from "../../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";

defineOptions({
  name: "OAuthPage",
});

const router = useRouter();

const isLoading = ref(false);
const errorMessage = ref("");
const successMessage = ref("");

function parseError(error: unknown): string {
  if (typeof error === "string" && error.trim()) {
    return error;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (
    typeof error === "object" &&
    error !== null &&
    "message" in error &&
    typeof (error as { message?: unknown }).message === "string"
  ) {
    return (error as { message: string }).message;
  }

  return "OAuth 登录失败，请稍后重试。";
}

async function handleLogin() {
  errorMessage.value = "";
  successMessage.value = "";

  isLoading.value = true;
  try {
    const ok = await TokenService.BeginOAuthLogin();

    if (!ok) {
      throw new Error("OAuth 授权失败，请重试。");
    }

    successMessage.value = "登录成功，正在跳转...";
    await router.replace("/");
  } catch (error) {
    errorMessage.value = parseError(error);
  } finally {
    isLoading.value = false;
  }
}
</script>

<template>
  <div class="oauth-container">
    <!-- MD3 卡片：28dp 圆角、surface container -->
    <section class="oauth-card">
      <v-img
        src="./imgs/yuzu_happy.png"
        alt="Lolia Shizuku"
        width="132"
        class="mx-auto"
      />

      <div class="text-center">
        <h1 class="text-headline-medium">Lolia Shizuku</h1>
        <p class="text-body-medium md-on-surface-variant mt-2">
          「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端
        </p>
      </div>

      <v-alert v-if="errorMessage" type="error" variant="tonal" rounded="lg" density="compact">
        {{ errorMessage }}
      </v-alert>

      <v-alert v-if="successMessage" type="success" variant="tonal" rounded="lg" density="compact">
        {{ successMessage }}
      </v-alert>

      <v-btn
        :loading="isLoading"
        color="primary"
        variant="flat"
        size="large"
        block
        append-icon="fas fa-arrow-right-long"
        @click="handleLogin"
      >
        使用 Lolia FRP 账号登录
      </v-btn>

      <p class="text-body-small md-on-surface-variant text-center">
        将在浏览器中打开授权页面，完成后自动返回
      </p>
    </section>
  </div>
</template>

<style scoped>
.oauth-container {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.oauth-card {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: 100%;
  max-width: 400px;
  padding: 32px 24px 24px;
  border-radius: 28px;
  background: var(--md-surface-container);
}
</style>
