<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import { useRouter } from "vue-router";
import { openExternalURL } from "@/utils/browser";
import { TokenService } from "../../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";
import QRCode from "qrcode";
import { copyText } from "@/utils/tunnel";
import { isMobile } from "@/utils/platform";

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

// 设备码登录（RFC 8628）：在任意设备的浏览器中输入用户码完成授权，无需本机接收回调
type DeviceLogin = {
  userCode: string;
  verificationURI: string;
  verificationURIComplete: string;
  expiresAt: number;
};
const device = ref<DeviceLogin | null>(null);
// 桌面端显示二维码，可用手机扫码在手机上完成授权（手机端扫自己的屏幕没有意义，不显示）
const qrDataURL = ref("");
const deviceStarting = ref(false);
const codeCopied = ref(false);
const now = ref(Date.now());
let countdownTimer: number | undefined;

const remainingText = computed(() => {
  if (!device.value) return "";
  const seconds = Math.max(0, Math.floor((device.value.expiresAt - now.value) / 1000));
  const minutes = Math.floor(seconds / 60);
  return `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
});

const stopCountdown = () => {
  if (countdownTimer !== undefined) {
    window.clearInterval(countdownTimer);
    countdownTimer = undefined;
  }
};

async function handleDeviceLogin() {
  errorMessage.value = "";
  successMessage.value = "";
  deviceStarting.value = true;
  try {
    const info = await TokenService.StartDeviceLogin();
    if (!info?.user_code) {
      throw new Error("获取设备码失败，请重试。");
    }
    device.value = {
      userCode: info.user_code,
      verificationURI: info.verification_uri,
      verificationURIComplete: info.verification_uri_complete || info.verification_uri,
      expiresAt: Date.now() + info.expires_in * 1000,
    };
    codeCopied.value = false;
    qrDataURL.value = "";
    if (!isMobile) {
      // 二维码需深色码点 + 浅色底才能被稳定识别，不跟随主题色
      qrDataURL.value = await QRCode.toDataURL(device.value.verificationURIComplete, {
        margin: 1,
        width: 320,
        errorCorrectionLevel: "M",
      }).catch(() => "");
    }
    now.value = Date.now();
    stopCountdown();
    countdownTimer = window.setInterval(() => {
      now.value = Date.now();
    }, 1000);
  } catch (error) {
    errorMessage.value = parseError(error);
    return;
  } finally {
    deviceStarting.value = false;
  }

  const current = device.value;
  try {
    await TokenService.WaitDeviceLogin();
    successMessage.value = "登录成功，正在跳转...";
    stopCountdown();
    await router.replace("/");
  } catch (error) {
    // 用户已取消或重新获取了设备码时忽略旧的结果
    if (device.value === current) {
      errorMessage.value = parseError(error);
      device.value = null;
      stopCountdown();
    }
  }
}

function cancelDeviceLogin() {
  device.value = null;
  stopCountdown();
  void TokenService.CancelDeviceLogin();
}

function openVerificationPage() {
  if (device.value) {
    openExternalURL(device.value.verificationURIComplete);
  }
}

async function copyUserCode() {
  if (!device.value) return;
  try {
    await copyText(device.value.userCode);
    codeCopied.value = true;
  } catch {
    // 复制失败时用户仍可手动输入
  }
}

onBeforeUnmount(() => {
  stopCountdown();
  if (device.value) {
    void TokenService.CancelDeviceLogin();
  }
});

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
        v-if="!device"
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

      <!-- 设备码登录：展示用户码，等待用户在浏览器中确认 -->
      <template v-if="device">
        <div class="device-panel">
          <p class="text-body-medium md-on-surface-variant text-center">
            {{ qrDataURL ? "用手机扫码，或在任意设备的浏览器中打开验证页面，输入设备码并确认授权" : "在任意设备的浏览器中打开验证页面，输入下方设备码并确认授权" }}
          </p>
          <img v-if="qrDataURL" :src="qrDataURL" alt="扫码打开验证页面" class="device-qr" />
          <button type="button" class="device-code md-mono" title="复制设备码" @click="copyUserCode">
            <span>{{ device.userCode }}</span>
            <v-icon size="16">{{ codeCopied ? "fas fa-check" : "fas fa-copy" }}</v-icon>
          </button>
          <p class="text-body-small md-on-surface-variant text-center">
            <span class="md-mono">{{ device.verificationURI }}</span>
            <br />
            等待授权中 · 剩余 {{ remainingText }}
          </p>
          <v-progress-linear indeterminate color="primary" rounded height="4" />
        </div>

        <v-btn
          color="primary"
          variant="flat"
          size="large"
          block
          append-icon="fas fa-arrow-up-right-from-square"
          @click="openVerificationPage"
        >
          打开验证页面
        </v-btn>
        <v-btn variant="text" block @click="cancelDeviceLogin">取消</v-btn>
      </template>

      <template v-else>
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

        <v-btn
          class="md-btn-tonal"
          variant="flat"
          size="large"
          block
          prepend-icon="fas fa-qrcode"
          :loading="deviceStarting"
          :disabled="isLoading"
          @click="handleDeviceLogin"
        >
          使用设备码登录
        </v-btn>

        <p class="text-body-small md-on-surface-variant text-center">
          账号登录会在本机浏览器中授权后自动返回；设备码登录可在其他设备的浏览器中完成授权
        </p>
      </template>
    </section>
  </div>
</template>

<style scoped>
.oauth-container {
  position: fixed;
  inset: 0;
  display: flex;
  /* 顶部留出 64px 标题栏 */
  padding: calc(64px + 24px) 24px 24px;
  /* 内容高于窗口时可滚动（居中用 margin: auto，避免 align-items: center 裁掉顶部） */
  overflow-y: auto;
}

.device-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.device-qr {
  width: 160px;
  height: 160px;
  border-radius: 12px;
  image-rendering: pixelated;
}

/* MD3 filled tonal 容器承载用户码，点击复制 */
.device-code {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
  border-radius: 16px;
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
  font-size: 1.75rem;
  font-weight: 500;
  letter-spacing: 0.12em;
  border: 0;
  outline: none;
  cursor: pointer;
}

.device-code:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}

.oauth-card {
  margin: auto;
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
