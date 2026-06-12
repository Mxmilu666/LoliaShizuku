<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useTheme } from "vuetify";
import { storeToRefs } from "pinia";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { GetVersionInfo } from "../../../wailsjs/go/backend/App";
import type { version } from "../../../wailsjs/go/models";
import AppLogo from "@/components/AppLogo.vue";
import {
  accentPresets,
  applyAccentColors,
  readSavedAccentId,
  saveAccentId,
} from "@/plugins/theme";
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
const accentId = ref(readSavedAccentId());
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
const { installing, canceling, phase, downloaded, total, percent, indeterminate } =
  storeToRefs(frpcInstallStore);
const { startInstall, cancelInstall } = frpcInstallStore;

const formatBytes = (bytes: number): string => {
  if (!bytes || bytes <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${value.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
};

const phaseLabel = computed(() => {
  switch (phase.value) {
    case "resolving":
      return "正在获取最新版本…";
    case "downloading":
      return "正在下载…";
    case "verifying":
      return "正在校验文件…";
    case "extracting":
      return "正在解压安装…";
    case "done":
      return "安装完成";
    default:
      return "准备中…";
  }
});

const progressDetail = computed(() => {
  if (phase.value !== "downloading" || total.value <= 0) {
    return "";
  }
  return `${formatBytes(downloaded.value)} / ${formatBytes(total.value)}`;
});

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

const frpcInstalled = computed(() => !!status.value?.installed?.binary_exists);
const updateAvailable = computed(() => !!status.value?.update_available);

const frpcStatusChip = computed(() => {
  if (!frpcInstalled.value) {
    return { text: "未安装", color: "grey" };
  }
  if (status.value?.latest_error) {
    return { text: "已安装", color: "success" };
  }
  if (updateAvailable.value) {
    return { text: "可更新", color: "warning" };
  }
  return { text: "已是最新", color: "success" };
});

const installDetails = computed(() => [
  { label: "当前版本", value: installedVersion.value, icon: "fas fa-tag" },
  {
    label: "二进制",
    value: frpcInstalled.value ? "已安装" : "未安装",
    icon: "fas fa-microchip",
  },
  {
    label: "安装时间",
    value: formatTime(status.value?.installed?.installed_at),
    icon: "fas fa-clock",
  },
]);

const pathItems = computed(() => {
  const paths = status.value?.paths;
  return [
    { label: "userdata", value: paths?.userdata_dir },
    { label: "frpc", value: paths?.frpc_dir },
    { label: "bin", value: paths?.bin_dir },
    { label: "binary", value: paths?.binary_path },
    { label: "downloads", value: paths?.download_dir },
    { label: "state", value: paths?.state_path },
    { label: "settings", value: paths?.settings_path },
  ];
});

const latestDetails = computed(() => [
  { label: "最新标签", value: latestVersion.value, icon: "fas fa-tags" },
  {
    label: "发布时间",
    value: formatTime(status.value?.latest?.published_at),
    icon: "fas fa-calendar-day",
  },
  {
    label: "可更新",
    value: updateAvailable.value ? "是" : "否",
    icon: "fas fa-arrow-up-from-bracket",
  },
]);

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

const copyPath = async (value?: string) => {
  if (!value) {
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    showMessage("已复制路径", "success");
  } catch {
    showMessage("复制失败", "error");
  }
};

const versionInfo = ref<version.Info | null>(null);

const loadVersionInfo = async () => {
  try {
    versionInfo.value = await GetVersionInfo();
  } catch {
    versionInfo.value = null;
  }
};

const appVersion = computed(() => versionInfo.value?.version || "-");

const aboutDetails = computed(() => {
  const info = versionInfo.value;
  if (!info) {
    return [] as Array<{ label: string; value: string; icon: string }>;
  }
  const shortCommit =
    info.git_commit && info.git_commit.length > 7
      ? info.git_commit.slice(0, 7)
      : info.git_commit || "-";
  return [
    { label: "版本", value: info.version || "-", icon: "fas fa-tag" },
    { label: "提交", value: shortCommit, icon: "fas fa-code-commit" },
    { label: "分支", value: info.git_branch || "-", icon: "fas fa-code-branch" },
    { label: "构建时间", value: info.build_time || "-", icon: "fas fa-clock" },
    { label: "平台", value: info.platform || "-", icon: "fas fa-desktop" },
    { label: "Go", value: info.go_version || "-", icon: "fab fa-golang" },
  ];
});

const aboutLinks = [
  {
    label: "项目仓库",
    icon: "fab fa-github",
    color: "primary",
    url: "https://github.com/Mxmilu666/LoliaShizuku",
  },
  {
    label: "Lolia 控制台",
    icon: "fas fa-gauge-high",
    color: "primary",
    url: "https://dash.lolia.link",
  },
  {
    label: "Lolia 官网",
    icon: "fas fa-globe",
    color: "secondary",
    url: "https://lolia.link",
  },
  {
    label: "Wails",
    icon: "fas fa-book",
    color: "secondary",
    url: "https://wails.io",
  },
];

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

const handleAccentChange = (id: string) => {
  if (accentId.value === id) {
    return;
  }
  accentId.value = id;
  // Mutating the theme color maps triggers Vuetify to regenerate its CSS vars.
  applyAccentColors(theme.themes.value, id);
  saveAccentId(id);
  showMessage("强调色已更新", "success");
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
  void loadVersionInfo();
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

            <v-sheet border rounded="lg" class="pa-3 d-flex flex-column ga-3 soft-card">
              <div class="text-subtitle-2">强调色</div>
              <div class="d-flex flex-wrap ga-3">
                <button
                  v-for="preset in accentPresets"
                  :key="preset.id"
                  type="button"
                  class="accent-swatch"
                  :class="{ 'accent-swatch--active': accentId === preset.id }"
                  :style="{ background: preset.light }"
                  :title="preset.name"
                  :aria-label="preset.name"
                  @click="handleAccentChange(preset.id)"
                >
                  <v-icon v-if="accentId === preset.id" size="16" color="white">
                    fas fa-check
                  </v-icon>
                </button>
              </div>
              <div class="text-caption text-medium-emphasis">
                强调色会应用到按钮、链接等主色元素，选择即时生效并保存到本地。
              </div>
            </v-sheet>
          </v-card-text>

          <v-card-text v-else-if="activePanel === 'frpc'" class="d-flex flex-column ga-4">
            <v-sheet rounded="xl" class="frpc-hero pa-5 d-flex align-center flex-wrap ga-4">
              <div class="flex-grow-1" style="min-width: 180px">
                <div class="d-flex align-center flex-wrap ga-2">
                  <span class="text-h6 font-weight-bold">frpc</span>
                  <v-chip
                    v-if="frpcInstalled"
                    size="small"
                    color="primary"
                    variant="tonal"
                    rounded="pill"
                    class="font-weight-medium"
                  >
                    {{ installedVersion }}
                  </v-chip>
                  <v-chip
                    size="small"
                    :color="frpcStatusChip.color"
                    variant="flat"
                    rounded="pill"
                    class="font-weight-medium"
                  >
                    {{ frpcStatusChip.text }}
                  </v-chip>
                </div>
                <div class="text-caption text-medium-emphasis mt-1">
                  最新 {{ latestVersion }} · 发布 {{ formatTime(status?.latest?.published_at) }}
                </div>
              </div>
              <v-btn
                color="primary"
                :loading="installing"
                :disabled="installing"
                @click="handleInstallOrUpdate"
              >
                <v-icon start>fas fa-download</v-icon>
                {{ actionText }}
              </v-btn>
            </v-sheet>

            <div class="d-flex flex-wrap ga-2">
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

            <v-sheet v-if="installing" border rounded="lg" class="pa-3 soft-card">
              <div class="d-flex align-center justify-space-between mb-2">
                <span class="text-body-2">{{ phaseLabel }}</span>
                <span
                  v-if="progressDetail"
                  class="text-caption text-medium-emphasis"
                >
                  {{ progressDetail }}
                </span>
              </div>
              <v-progress-linear
                :model-value="percent"
                :indeterminate="indeterminate"
                color="primary"
                height="8"
                rounded
              />
              <div
                v-if="phase === 'downloading' && total > 0"
                class="text-caption text-medium-emphasis mt-1 text-end"
              >
                {{ Math.floor(percent) }}%
              </div>
            </v-sheet>

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
                <v-sheet border rounded="lg" class="pa-4 soft-card h-100">
                  <div class="text-subtitle-2 mb-3">安装状态</div>
                  <div class="d-flex flex-column ga-3">
                    <div
                      v-for="detail in installDetails"
                      :key="detail.label"
                      class="d-flex align-center ga-3"
                    >
                      <v-icon
                        :icon="detail.icon"
                        size="14"
                        color="medium-emphasis"
                        class="frpc-detail-icon"
                      />
                      <span class="text-caption text-medium-emphasis">{{ detail.label }}</span>
                      <span class="text-body-2 font-weight-medium ms-auto text-end">
                        {{ detail.value }}
                      </span>
                    </div>
                  </div>
                </v-sheet>
              </v-col>
              <v-col cols="12" md="6">
                <v-sheet border rounded="lg" class="pa-4 soft-card h-100">
                  <div class="text-subtitle-2 mb-3">最新版本</div>
                  <div class="d-flex flex-column ga-3">
                    <div
                      v-for="detail in latestDetails"
                      :key="detail.label"
                      class="d-flex align-center ga-3"
                    >
                      <v-icon
                        :icon="detail.icon"
                        size="14"
                        color="medium-emphasis"
                        class="frpc-detail-icon"
                      />
                      <span class="text-caption text-medium-emphasis">{{ detail.label }}</span>
                      <span class="text-body-2 font-weight-medium ms-auto text-end">
                        {{ detail.value }}
                      </span>
                    </div>
                  </div>
                </v-sheet>
              </v-col>
            </v-row>

            <v-sheet border rounded="lg" class="pa-4 d-flex flex-column ga-3 soft-card">
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

            <v-sheet border rounded="lg" class="pa-4 d-flex flex-column ga-3 soft-card">
              <div class="text-subtitle-2">本地目录</div>
              <div class="d-flex flex-column ga-1">
                <div
                  v-for="path in pathItems"
                  :key="path.label"
                  class="frpc-path-row"
                >
                  <span class="frpc-path-label text-caption text-medium-emphasis">
                    {{ path.label }}
                  </span>
                  <code class="frpc-path-value" :title="path.value">
                    {{ path.value || "-" }}
                  </code>
                  <v-btn
                    icon="fas fa-copy"
                    size="x-small"
                    variant="text"
                    density="comfortable"
                    :disabled="!path.value"
                    @click="copyPath(path.value)"
                  />
                </div>
              </div>
            </v-sheet>
          </v-card-text>

          <v-card-text v-else-if="activePanel === 'about'" class="d-flex flex-column ga-4">
            <v-sheet rounded="xl" class="about-hero pa-6 d-flex align-center ga-5">
              <div class="about-logo d-flex align-center justify-center">
                <AppLogo :size="44" />
              </div>
              <div class="flex-grow-1">
                <div class="d-flex align-center flex-wrap ga-2">
                  <span class="text-h5 font-weight-bold">LoliaShizuku</span>
                  <v-chip
                    size="small"
                    color="primary"
                    variant="flat"
                    rounded="pill"
                    class="font-weight-medium"
                  >
                    v{{ appVersion }}
                  </v-chip>
                </div>
                <div class="text-body-2 text-medium-emphasis mt-1">
                  「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端
                </div>
              </div>
            </v-sheet>

            <v-sheet border rounded="lg" class="pa-4 soft-card">
              <div class="text-subtitle-2 mb-3">构建信息</div>
              <v-row dense>
                <v-col
                  v-for="detail in aboutDetails"
                  :key="detail.label"
                  cols="6"
                  sm="4"
                >
                  <div class="d-flex align-center ga-2">
                    <v-icon :icon="detail.icon" size="14" color="medium-emphasis" />
                    <span class="text-caption text-medium-emphasis">{{ detail.label }}</span>
                  </div>
                  <div class="text-body-2 font-weight-medium mt-1 about-detail-value">
                    {{ detail.value }}
                  </div>
                </v-col>
              </v-row>
              <div
                v-if="aboutDetails.length === 0"
                class="text-caption text-medium-emphasis"
              >
                无法获取构建信息。
              </div>
            </v-sheet>

            <v-sheet border rounded="lg" class="pa-4 d-flex flex-column ga-3 soft-card">
              <div class="text-subtitle-2">相关链接</div>
              <div class="d-flex flex-wrap ga-2">
                <v-btn
                  v-for="link in aboutLinks"
                  :key="link.url"
                  variant="tonal"
                  :color="link.color"
                  :prepend-icon="link.icon"
                  @click="openURL(link.url)"
                >
                  {{ link.label }}
                </v-btn>
              </div>
            </v-sheet>

            <div class="text-caption text-medium-emphasis text-center">
              以 MIT 许可证开源 · Made with ♥ by Mxmilu666
            </div>
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

.soft-card {
  border-color: rgba(var(--v-theme-on-surface), 0.08) !important;
}

.about-logo {
  width: 76px;
  height: 76px;
  flex-shrink: 0;
  border-radius: 20px;
  color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.1);
}

.about-detail-value {
  word-break: break-word;
}

.accent-swatch {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: transform 0.1s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.accent-swatch:hover {
  transform: scale(1.1);
}

.accent-swatch--active {
  border-color: rgb(var(--v-theme-on-surface));
}

.frpc-path-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.frpc-path-label {
  flex-shrink: 0;
  width: 76px;
}

.frpc-path-value {
  flex: 1;
  min-width: 0;
  padding: 5px 10px;
  border-radius: 8px;
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
    "Courier New", monospace;
  font-size: 0.75rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: rgb(var(--v-theme-on-surface));
  background: rgba(var(--v-theme-on-surface), 0.04);
}
</style>
