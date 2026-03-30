<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useTheme } from "vuetify";
import { storeToRefs } from "pinia";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { useGlobalLoadingStore } from "@/stores/globalLoading";
import { useFrpcInstallStore } from "@/stores/frpcInstall";
import {
  getFrpcStatus,
  removeFrpc,
  setMirrorConfig,
  type FrpcMirrorConfig,
  type FrpcStatus,
} from "@/services/frpc";

defineOptions({
  name: "SettingsPage",
});

type SettingsPanel = "appearance" | "frpc" | "about" | "account";
type MirrorMode = "official" | "builtin" | "custom";
type CustomMirrorMode = "base" | "template";
type ThemeMode = "system" | "lightTheme" | "darkTheme";

const router = useRouter();
const theme = useTheme();
const prefersDarkMedia =
  typeof window !== "undefined" && typeof window.matchMedia === "function"
    ? window.matchMedia("(prefers-color-scheme: dark)")
    : null;

const status = ref<FrpcStatus | null>(null);
const activePanel = ref<SettingsPanel>("frpc");
const snackbar = ref(false);
const snackbarText = ref("");
const snackbarColor = ref<"success" | "error" | "info">("info");
const mirrorMode = ref<MirrorMode>("official");
const builtinMirrorPresetID = ref("");
const customMirrorMode = ref<CustomMirrorMode>("base");
const customMirrorBaseURL = ref("");
const customMirrorURLTemplate = ref("");
const themeMode = ref<ThemeMode>("system");
const logoutLoading = ref(false);

const themeStorageKey = "lolia.theme";

const mirrorModeItems = [
  { title: "github.com", value: "official" as const },
  { title: "内置镜像", value: "builtin" as const },
  { title: "自定义网址", value: "custom" as const },
];

const customMirrorModeItems = [
  { title: "基础地址", value: "base" as const },
  { title: "URL 模板", value: "template" as const },
];

const themeModeItems = [
  { title: "跟随系统", value: "system" as const },
  { title: "浅色模式", value: "lightTheme" as const },
  { title: "深色模式", value: "darkTheme" as const },
];

const globalLoadingStore = useGlobalLoadingStore();
const withGlobalLoading = <T>(task: () => Promise<T>) =>
  globalLoadingStore.withGlobalLoading(task);

const frpcInstallStore = useFrpcInstallStore();
const { installing, canceling } = storeToRefs(frpcInstallStore);
const { startInstall, cancelInstall } = frpcInstallStore;

const showMessage = (
  text: string,
  color: "success" | "error" | "info" = "info",
) => {
  snackbarText.value = text;
  snackbarColor.value = color;
  snackbar.value = true;
};

const panelTitle = computed(() => {
  switch (activePanel.value) {
    case "appearance":
      return "外观设置";
    case "frpc":
      return "frps 管理";
    case "about":
      return "关于";
    case "account":
      return "账号";
    default:
      return "设置";
  }
});

const installedVersion = computed(
  () => status.value?.installed?.version || "未安装",
);
const latestVersion = computed(() => status.value?.latest?.tag_name || "-");
const builtinMirrorItems = computed(() =>
  (status.value?.builtin_mirrors ?? []).map((preset) => ({
    title: preset.name || preset.id,
    value: preset.id,
    props: {
      subtitle:
        preset.description ||
        preset.base_url ||
        preset.url_template ||
        "未提供描述",
    },
  })),
);
const actionText = computed(() => {
  if (!status.value?.installed?.binary_exists) {
    return "安装 frpc";
  }
  if (status.value.update_available) {
    return "更新 frpc";
  }
  return "重装 frpc";
});

const formatTime = (value?: string) => {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
};

const openURL = (url: string) => {
  BrowserOpenURL(url);
};

const getSystemThemeName = (): "lightTheme" | "darkTheme" =>
  prefersDarkMedia?.matches ? "darkTheme" : "lightTheme";

const resolveThemeName = (mode: ThemeMode): "lightTheme" | "darkTheme" => {
  if (mode === "system") {
    return getSystemThemeName();
  }
  return mode;
};

const handleSystemThemePreferenceChange = () => {
  if (themeMode.value === "system") {
    theme.global.name.value = getSystemThemeName();
  }
};

const applyTheme = (mode: ThemeMode) => {
  theme.global.name.value = resolveThemeName(mode);
  try {
    localStorage.setItem(themeStorageKey, mode);
  } catch {
    // ignore localStorage errors
  }
};

const initTheme = () => {
  let resolvedTheme: ThemeMode = "system";
  try {
    const savedTheme = localStorage.getItem(themeStorageKey);
    if (
      savedTheme === "system" ||
      savedTheme === "lightTheme" ||
      savedTheme === "darkTheme"
    ) {
      resolvedTheme = savedTheme;
    }
  } catch {
    // ignore localStorage errors
  }

  themeMode.value = resolvedTheme;
  theme.global.name.value = resolveThemeName(resolvedTheme);
};

const handleThemeChange = (value: string | null) => {
  let nextTheme: ThemeMode = "lightTheme";
  if (value === "system") {
    nextTheme = "system";
  } else if (value === "darkTheme") {
    nextTheme = "darkTheme";
  }

  if (
    themeMode.value === nextTheme &&
    theme.global.name.value === resolveThemeName(nextTheme)
  ) {
    return;
  }

  themeMode.value = nextTheme;
  applyTheme(nextTheme);
  showMessage("主题已切换", "success");
};

const syncMirrorForm = (nextStatus: FrpcStatus) => {
  const config = nextStatus.mirror_config;
  mirrorMode.value = config?.mode || "official";
  builtinMirrorPresetID.value = config?.preset_id || "";
  customMirrorBaseURL.value = config?.custom_base_url || "";
  customMirrorURLTemplate.value = config?.custom_url_template || "";
  customMirrorMode.value = customMirrorURLTemplate.value ? "template" : "base";

  if (!builtinMirrorPresetID.value && nextStatus.builtin_mirrors?.length) {
    builtinMirrorPresetID.value = nextStatus.builtin_mirrors[0].id;
  }
};

const loadStatus = async () => {
  await withGlobalLoading(async () => {
    try {
      status.value = await getFrpcStatus();
      syncMirrorForm(status.value);
    } catch (error) {
      showMessage(
        error instanceof Error ? error.message : "获取 frpc 状态失败",
        "error",
      );
    }
  });
};

const handleInstallOrUpdate = async () => {
  try {
    await withGlobalLoading(async () => {
      const result = await startInstall();
      status.value = result.status;
      syncMirrorForm(result.status);
      showMessage(`frpc 已安装到 ${result.status.paths.binary_path}`, "success");
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : "安装/更新 frpc 失败";
    if (message.includes("已终止")) {
      showMessage(message, "info");
      await loadStatus();
      return;
    }
    showMessage(message, "error");
  }
};

const handleCancelInstall = async () => {
  if (!installing.value || canceling.value) {
    return;
  }

  try {
    await cancelInstall();
    showMessage("已发送终止下载请求", "info");
  } catch (error) {
    showMessage(error instanceof Error ? error.message : "终止下载失败", "error");
  }
};

const handleRemove = async () => {
  await withGlobalLoading(async () => {
    try {
      await removeFrpc();
      showMessage("本地 frpc 已移除", "success");
      await loadStatus();
    } catch (error) {
      showMessage(error instanceof Error ? error.message : "移除 frpc 失败", "error");
    }
  });
};

const handleSaveMirrorConfig = async () => {
  await withGlobalLoading(async () => {
    try {
      const config: FrpcMirrorConfig = {
        mode: mirrorMode.value,
      };

      if (mirrorMode.value === "builtin") {
        if (!builtinMirrorPresetID.value) {
          showMessage("请选择一个内置镜像", "error");
          return;
        }
        config.preset_id = builtinMirrorPresetID.value;
      } else if (mirrorMode.value === "custom") {
        if (customMirrorMode.value === "template") {
          const template = customMirrorURLTemplate.value.trim();
          if (!template) {
            showMessage("请填写自定义 URL 模板", "error");
            return;
          }
          config.custom_url_template = template;
        } else {
          const baseURL = customMirrorBaseURL.value.trim();
          if (!baseURL) {
            showMessage("请填写自定义镜像基础地址", "error");
            return;
          }
          config.custom_base_url = baseURL;
        }
      }

      await setMirrorConfig(config);
      showMessage("下载源设置已保存", "success");
      await loadStatus();
    } catch (error) {
      showMessage(error instanceof Error ? error.message : "保存下载源失败", "error");
    }
  });
};

const handleUseOfficialMirror = async () => {
  mirrorMode.value = "official";
  customMirrorBaseURL.value = "";
  customMirrorURLTemplate.value = "";
  await handleSaveMirrorConfig();
};

const handleLogout = async () => {
  if (logoutLoading.value) {
    return;
  }

  logoutLoading.value = true;
  try {
    const centerService = (window as any).go?.services?.CenterService;
    if (centerService?.StopRunner) {
      await centerService.StopRunner();
    }

    const tokenService = (window as any).go?.services?.TokenService;
    if (!tokenService?.ClearOAuthToken) {
      throw new Error("后端 Token 服务未就绪，请重启应用。");
    }

    await tokenService.ClearOAuthToken();
    showMessage("已退出登录", "success");
    await router.replace("/oauth");
  } catch (error) {
    showMessage(error instanceof Error ? error.message : "退出登录失败", "error");
  } finally {
    logoutLoading.value = false;
  }
};

onMounted(() => {
  initTheme();
  if (prefersDarkMedia && typeof prefersDarkMedia.addEventListener === "function") {
    prefersDarkMedia.addEventListener("change", handleSystemThemePreferenceChange);
  }
  void loadStatus();
});

onBeforeUnmount(() => {
  if (prefersDarkMedia && typeof prefersDarkMedia.removeEventListener === "function") {
    prefersDarkMedia.removeEventListener("change", handleSystemThemePreferenceChange);
  }
});
</script>

<template>
  <div class="settings-page d-flex flex-column ga-4">
    <v-snackbar
      v-model="snackbar"
      :color="snackbarColor"
      location="bottom"
      :timeout="2600"
    >
      {{ snackbarText }}
    </v-snackbar>

    <v-row dense class="settings-layout flex-grow-1">
      <v-col cols="12" md="3">
        <v-card elevation="2" class="h-100">
          <v-list nav density="comfortable">
            <v-list-subheader>设置菜单</v-list-subheader>
            <v-list-item
              prepend-icon="fas fa-cloud-arrow-down"
              title="frps 管理"
              :active="activePanel === 'frpc'"
              @click="activePanel = 'frpc'"
            />
            <v-list-item
              prepend-icon="fas fa-palette"
              title="外观"
              :active="activePanel === 'appearance'"
              @click="activePanel = 'appearance'"
            />
            <v-list-item
              prepend-icon="fas fa-circle-info"
              title="关于"
              :active="activePanel === 'about'"
              @click="activePanel = 'about'"
            />
            <v-list-item
              prepend-icon="fas fa-user"
              title="账号"
              :active="activePanel === 'account'"
              @click="activePanel = 'account'"
            />
          </v-list>
        </v-card>
      </v-col>

      <v-col cols="12" md="9">
        <v-card elevation="2" class="h-100">
          <v-card-title class="d-flex align-center justify-space-between">
            <div class="text-h6 font-weight-bold">{{ panelTitle }}</div>
            <v-chip v-if="activePanel === 'frpc'" size="small" color="primary" variant="tonal">
              {{ status?.goos }}/{{ status?.goarch }}
            </v-chip>
          </v-card-title>
          <v-divider />

          <v-card-text v-if="activePanel === 'appearance'" class="d-flex flex-column ga-4">
            <v-sheet border rounded="lg" class="pa-3 d-flex flex-column ga-3">
              <div class="text-subtitle-2">主题模式</div>
              <v-select
                v-model="themeMode"
                :items="themeModeItems"
                item-title="title"
                item-value="value"
                hide-details="auto"
                @update:model-value="handleThemeChange"
              />
              <div class="text-caption text-medium-emphasis">
                支持跟随系统、浅色、深色模式，设置会自动保存到本地。
              </div>
            </v-sheet>
          </v-card-text>

          <v-card-text v-else-if="activePanel === 'frpc'" class="d-flex flex-column ga-4">
            <div class="d-flex flex-wrap ga-3">
              <v-btn
                color="primary"
                :loading="installing"
                :disabled="installing"
                @click="handleInstallOrUpdate"
              >
                <v-icon start>fas fa-download</v-icon>
                {{ actionText }}
              </v-btn>
              <v-btn
                color="warning"
                variant="tonal"
                :loading="canceling"
                :disabled="!installing || canceling"
                @click="handleCancelInstall"
              >
                <v-icon start>fas fa-stop</v-icon>
                终止下载
              </v-btn>
              <v-btn
                color="info"
                variant="tonal"
                :disabled="installing"
                @click="loadStatus"
              >
                <v-icon start>fas fa-rotate</v-icon>
                检查更新
              </v-btn>
              <v-btn
                color="error"
                variant="tonal"
                :disabled="installing"
                @click="handleRemove"
              >
                <v-icon start>fas fa-trash</v-icon>
                删除本地 frpc
              </v-btn>
            </div>

            <v-alert
              v-if="status?.latest_error"
              type="warning"
              variant="tonal"
              density="compact"
              class="message-alert"
            >
              获取最新版本失败：{{ status.latest_error }}
            </v-alert>

            <v-row dense>
              <v-col cols="12" md="6">
                <v-sheet border rounded="lg" class="pa-3">
                  <div class="text-subtitle-2 mb-2">安装状态</div>
                  <div class="text-body-2">当前版本：{{ installedVersion }}</div>
                  <div class="text-body-2">
                    二进制：{{ status?.installed?.binary_exists ? "已安装" : "未安装" }}
                  </div>
                  <div class="text-body-2">
                    安装时间：{{ formatTime(status?.installed?.installed_at) }}
                  </div>
                </v-sheet>
              </v-col>
              <v-col cols="12" md="6">
                <v-sheet border rounded="lg" class="pa-3">
                  <div class="text-subtitle-2 mb-2">最新版本</div>
                  <div class="text-body-2">标签：{{ latestVersion }}</div>
                  <div class="text-body-2">
                    发布：{{ formatTime(status?.latest?.published_at) }}
                  </div>
                  <div class="text-body-2">
                    可更新：{{ status?.update_available ? "是" : "否" }}
                  </div>
                </v-sheet>
              </v-col>
            </v-row>

            <v-expansion-panels variant="accordion">
              <v-expansion-panel>
                <v-expansion-panel-title class="text-subtitle-2">
                  本地目录
                </v-expansion-panel-title>
                <v-expansion-panel-text>
                  <div class="text-body-2 text-wrap">
                    userdata: {{ status?.paths.userdata_dir }}
                  </div>
                  <div class="text-body-2 text-wrap">frpc: {{ status?.paths.frpc_dir }}</div>
                  <div class="text-body-2 text-wrap">bin: {{ status?.paths.bin_dir }}</div>
                  <div class="text-body-2 text-wrap">
                    binary: {{ status?.paths.binary_path }}
                  </div>
                  <div class="text-body-2 text-wrap">
                    downloads: {{ status?.paths.download_dir }}
                  </div>
                  <div class="text-body-2 text-wrap">state: {{ status?.paths.state_path }}</div>
                  <div class="text-body-2 text-wrap">
                    settings: {{ status?.paths.settings_path }}
                  </div>
                </v-expansion-panel-text>
              </v-expansion-panel>
            </v-expansion-panels>

            <v-divider />

            <v-sheet border rounded="lg" class="pa-3 d-flex flex-column ga-3">
              <div class="text-subtitle-2">GitHub 下载源</div>
              <v-select
                v-model="mirrorMode"
                :items="mirrorModeItems"
                item-title="title"
                item-value="value"
                hide-details="auto"
                :disabled="installing"
              />
              <v-select
                v-if="mirrorMode === 'builtin'"
                v-model="builtinMirrorPresetID"
                :items="builtinMirrorItems"
                item-title="title"
                item-value="value"
                hide-details="auto"
                :disabled="installing"
                no-data-text="当前没有可用的内置镜像"
              />
              <v-alert
                v-if="mirrorMode === 'builtin' && !builtinMirrorItems.length"
                type="info"
                variant="tonal"
                density="compact"
              >
                当前未配置内置镜像，请改用官方源或自定义源。
              </v-alert>
              <v-select
                v-if="mirrorMode === 'custom'"
                v-model="customMirrorMode"
                :items="customMirrorModeItems"
                item-title="title"
                item-value="value"
                hide-details="auto"
                :disabled="installing"
              />
              <v-text-field
                v-if="mirrorMode === 'custom' && customMirrorMode === 'base'"
                v-model="customMirrorBaseURL"
                hide-details="auto"
                placeholder="https://example.com/github.com"
                :disabled="installing"
              />
              <v-text-field
                v-if="mirrorMode === 'custom' && customMirrorMode === 'template'"
                v-model="customMirrorURLTemplate"
                hide-details="auto"
                placeholder="https://mirrors.114514.com/{owner}/{repo}/{tag}/{asset}"
                :disabled="installing"
              />
              <div
                v-if="mirrorMode === 'custom' && customMirrorMode === 'template'"
                class="text-caption text-medium-emphasis"
              >
                可用占位符：{owner}、{repo}、{tag}、{asset}
              </div>
              <div class="d-flex flex-wrap ga-2">
                <v-btn
                  color="primary"
                  variant="tonal"
                  :disabled="installing"
                  @click="handleSaveMirrorConfig"
                >
                  保存设置
                </v-btn>
                <v-btn
                  color="secondary"
                  variant="text"
                  :disabled="installing"
                  @click="handleUseOfficialMirror"
                >
                  使用 github.com
                </v-btn>
              </div>
            </v-sheet>
          </v-card-text>

          <v-card-text v-else-if="activePanel === 'about'" class="d-flex flex-column ga-4">
            <v-sheet border rounded="lg" class="pa-3 d-flex flex-column ga-2">
              <div class="text-subtitle-2">LoliaShizuku</div>
              <div class="text-body-2">
                「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端
              </div>
            </v-sheet>

            <v-sheet border rounded="lg" class="pa-3 d-flex flex-column ga-2">
              <div class="text-subtitle-2">相关链接</div>
              <div class="d-flex flex-wrap ga-2">
                <v-btn
                  variant="tonal"
                  color="primary"
                  prepend-icon="fas fa-globe"
                  @click="openURL('https://dash.lolia.link')"
                >
                  控制台
                </v-btn>
                <v-btn
                  variant="tonal"
                  color="secondary"
                  prepend-icon="fas fa-book"
                  @click="openURL('https://wails.io')"
                >
                  Wails
                </v-btn>
              </div>
            </v-sheet>
          </v-card-text>

          <v-card-text v-else class="d-flex flex-column ga-4">
            <v-alert type="warning" variant="tonal">
              退出后将清除本地 OAuth 凭据，并停止当前本地 Runner。
            </v-alert>

            <div class="d-flex flex-wrap ga-3">
              <v-btn
                color="error"
                prepend-icon="fas fa-right-from-bracket"
                :loading="logoutLoading"
                :disabled="logoutLoading"
                @click="handleLogout"
              >
                退出登录
              </v-btn>
            </div>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<style scoped>
.settings-page {
  min-height: calc(100vh - 64px - 32px);
}

.settings-layout :deep(.v-col) {
  display: flex;
}

.settings-layout :deep(.v-card) {
  flex: 1;
}
</style>
