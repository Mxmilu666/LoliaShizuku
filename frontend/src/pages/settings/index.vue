<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useTheme } from "vuetify";
import { storeToRefs } from "pinia";
import { openExternalURL } from "@/utils/browser";
import { App } from "../../../bindings/github.com/Mxmilu666/LoliaShizuku/backend";
import { CenterService, TokenService } from "../../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";
import type { Info as VersionInfo } from "../../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/version";
import AppLogo from "@/components/AppLogo.vue";
import {
  DYNAMIC_ACCENT_ID,
  accentPresets,
  applyAccentColors,
  defaultAccentId,
  readCachedDynamicSeed,
  readSavedAccentId,
  saveAccentId,
  type DynamicSeed,
} from "@/plugins/theme";
import { refreshDynamicColors } from "@/services/dynamicColor";
import { isMobile } from "@/utils/platform";
import { formatDateTime } from "@/utils/format";
import { findSettingsPanel, settingsPanels, type SettingsPanel } from "@/utils/settingsPanels";
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

type MirrorMode = "official" | "builtin" | "custom";
type CustomMirrorMode = "base" | "template";
type ThemeMode = "system" | "lightTheme" | "darkTheme";

const router = useRouter();
const route = useRoute();
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
const showMirrorSwitchHint = ref(false);
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

const navItems = settingsPanels;

// 移动端：设置首页为分类列表，点击进入二级页面（?panel=xxx），可用系统返回手势返回
const mobilePanel = computed(() => findSettingsPanel(route.query.panel)?.value ?? null);
watch(
  mobilePanel,
  (panel) => {
    if (isMobile && panel) {
      activePanel.value = panel;
    }
  },
  { immediate: true },
);
const openMobilePanel = (panel: SettingsPanel) => {
  void router.push({ path: "/settings", query: { panel } });
};

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
      return "frpc 管理";
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
const mirrorConfigDirty = computed(() => {
  const saved = status.value?.mirror_config;
  if (!saved) {
    return false;
  }

  const savedMode = saved.mode || "official";
  if (mirrorMode.value !== savedMode) {
    return true;
  }
  if (mirrorMode.value === "builtin") {
    return builtinMirrorPresetID.value !== (saved.preset_id || "");
  }
  if (mirrorMode.value === "custom") {
    if (customMirrorMode.value === "template") {
      return (
        customMirrorURLTemplate.value.trim() !== (saved.custom_url_template || "")
      );
    }
    return customMirrorBaseURL.value.trim() !== (saved.custom_base_url || "");
  }
  return false;
});
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
    label: "可更新",
    value: updateAvailable.value ? "是" : "否",
    icon: "fas fa-arrow-up-from-bracket",
  },
]);

const formatTime = (value?: string) => formatDateTime(value);

const openURL = (url: string) => {
  openExternalURL(url);
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

const versionInfo = ref<VersionInfo | null>(null);

const loadVersionInfo = async () => {
  try {
    versionInfo.value = await App.GetVersionInfo();
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
  if (id !== DYNAMIC_ACCENT_ID) {
    lastPresetId.value = id;
  }
  // Mutating the theme color maps triggers Vuetify to regenerate its CSS vars.
  applyAccentColors(theme.themes.value, id);
  saveAccentId(id);
  showMessage("强调色已更新", "success");
};

// 莫奈取色
const isDynamicAccent = computed(() => accentId.value === DYNAMIC_ACCENT_ID);
const dynamicSeed = ref<DynamicSeed | null>(readCachedDynamicSeed());
const dynamicLoading = ref(false);
// 关闭动态取色时恢复到最近一次选择的预设色
const lastPresetId = ref(isDynamicAccent.value ? defaultAccentId : accentId.value);
const dynamicSourceText = computed(() => {
  if (!dynamicSeed.value) return "尚未取色";
  switch (dynamicSeed.value.source) {
    case "system":
      return "取自系统动态配色";
    case "wallpaper":
      return isMobile ? "取自当前壁纸" : "取自当前桌面壁纸";
    default:
      return "取自系统强调色（未能读取壁纸）";
  }
});

const refreshDynamicAccent = async () => {
  if (dynamicLoading.value) return;
  dynamicLoading.value = true;
  try {
    dynamicSeed.value = await refreshDynamicColors(theme.themes.value);
    showMessage("已重新取色", "success");
  } catch (error) {
    showMessage(error instanceof Error ? error.message : "取色失败", "error");
  } finally {
    dynamicLoading.value = false;
  }
};

const handleDynamicToggle = async (enabled: boolean | null) => {
  if (enabled) {
    accentId.value = DYNAMIC_ACCENT_ID;
    saveAccentId(DYNAMIC_ACCENT_ID);
    // 先套用缓存的取色结果，再重新取色
    applyAccentColors(theme.themes.value, DYNAMIC_ACCENT_ID);
    await refreshDynamicAccent();
    return;
  }
  handleAccentChange(lastPresetId.value);
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
  showMirrorSwitchHint.value = false;
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
    if (message.includes("下载失败") || message.includes("下载超时")) {
      showMirrorSwitchHint.value = true;
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
      showMirrorSwitchHint.value = false;
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
    await CenterService.StopRunner();
    await TokenService.ClearOAuthToken();
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
  <div class="md-page settings-page">
    <v-snackbar v-model="snackbar" location="bottom" :timeout="2600">
      {{ snackbarText }}
    </v-snackbar>

    <header class="md-page-header">
      <div>
        <h1 class="text-headline-small">设置</h1>
        <p class="text-body-medium md-on-surface-variant mt-1">管理 frpc、外观与账号</p>
      </div>
    </header>

    <div class="settings-layout">
      <!-- 移动端：MD3 列表形式的设置分类 -->
      <section v-if="isMobile && !mobilePanel" class="md-settings-list">
        <button
          v-for="item in navItems"
          :key="item.value"
          type="button"
          class="md-settings-list-item"
          @click="openMobilePanel(item.value)"
        >
          <div class="md-icon-container">
            <v-icon size="16">{{ item.icon }}</v-icon>
          </div>
          <div class="flex-grow-1 min-w-0 text-left">
            <div class="text-body-large">{{ item.title }}</div>
            <div class="text-body-medium md-on-surface-variant">{{ item.description }}</div>
          </div>
          <v-icon size="14" class="md-on-surface-variant">fas fa-chevron-right</v-icon>
        </button>
      </section>

      <!-- MD3 navigation drawer 风格的侧边导航 -->
      <nav v-if="!isMobile" class="md-nav-drawer">
        <button
          v-for="item in navItems"
          :key="item.value"
          type="button"
          class="md-nav-drawer-item text-label-large"
          :class="{ 'md-nav-drawer-item--active': activePanel === item.value }"
          @click="activePanel = item.value"
        >
          <v-icon size="16">{{ item.icon }}</v-icon>
          {{ item.title }}
        </button>
      </nav>

      <main v-if="!isMobile || mobilePanel" class="d-flex flex-column ga-4 min-w-0">
        <div v-if="!isMobile" class="d-flex align-center">
          <h2 class="text-title-large">{{ panelTitle }}</h2>
          <v-spacer />
          <span v-if="activePanel === 'frpc' && status" class="text-label-large md-on-surface-variant">
            {{ status.goos }}/{{ status.goarch }}
          </span>
        </div>

        <!-- 外观 -->
        <template v-if="activePanel === 'appearance'">
          <section class="md-card-outlined">
            <h3 class="text-title-small md-section-title">主题模式</h3>
            <!-- MD3 segmented button -->
            <v-btn-toggle
              :model-value="themeMode"
              mandatory
              divided
              variant="outlined"
              rounded="xl"
              class="md-segmented"
              @update:model-value="handleThemeChange"
            >
              <v-btn
                v-for="item in themeModeItems"
                :key="item.value"
                :value="item.value"
                :prepend-icon="themeMode === item.value ? 'fas fa-check' : undefined"
              >
                {{ item.title }}
              </v-btn>
            </v-btn-toggle>
            <p class="text-body-small md-on-surface-variant mt-3">
              支持跟随系统、浅色、深色模式，设置会自动保存到本地。
            </p>
          </section>

          <section class="md-card-outlined">
            <h3 class="text-title-small md-section-title">强调色</h3>

            <!-- 莫奈取色：桌面端读取桌面壁纸，Android 使用系统动态配色 -->
            <div class="md-switch-row">
              <div class="md-icon-container">
                <v-icon size="16">fas fa-wand-magic-sparkles</v-icon>
              </div>
              <div class="flex-grow-1 min-w-0">
                <div class="text-body-large">莫奈取色</div>
                <div class="text-body-medium md-on-surface-variant">
                  {{ isMobile ? "跟随系统壁纸配色（Material You）" : "像 Android 一样，从桌面壁纸自动提取主题色" }}
                </div>
              </div>
              <v-switch
                :model-value="isDynamicAccent"
                color="primary"
                inset
                hide-details
                :loading="dynamicLoading"
                @update:model-value="handleDynamicToggle"
              />
            </div>

            <div v-if="isDynamicAccent" class="dynamic-seed">
              <span class="dynamic-seed-dot" :style="{ background: dynamicSeed?.seed }" />
              <div class="flex-grow-1 min-w-0">
                <div class="text-body-medium md-mono">{{ dynamicSeed?.seed?.toUpperCase() || "-" }}</div>
                <div class="text-body-small md-on-surface-variant">{{ dynamicSourceText }}</div>
              </div>
              <v-btn
                class="md-btn-tonal"
                variant="flat"
                prepend-icon="fas fa-rotate"
                :loading="dynamicLoading"
                @click="refreshDynamicAccent"
              >
                重新取色
              </v-btn>
            </div>

            <v-divider class="my-4 md-divider" />

            <div class="text-body-medium md-on-surface-variant mb-3">
              {{ isDynamicAccent ? "关闭莫奈取色后可选择预设颜色" : "预设颜色" }}
            </div>
            <div class="d-flex flex-wrap ga-3" :class="{ 'accent-swatches--disabled': isDynamicAccent }">
              <button
                v-for="preset in accentPresets"
                :key="preset.id"
                type="button"
                class="accent-swatch"
                :class="{ 'accent-swatch--active': accentId === preset.id }"
                :style="{ background: preset.seed }"
                :title="preset.name"
                :aria-label="preset.name"
                :disabled="isDynamicAccent"
                @click="handleAccentChange(preset.id)"
              >
                <v-icon v-if="accentId === preset.id" size="14" color="white">fas fa-check</v-icon>
              </button>
            </div>
            <p class="text-body-small md-on-surface-variant mt-3">
              主题色按 MD3 规范生成按钮、容器与各级表面色调，选择即时生效并保存到本地。
            </p>
          </section>
        </template>

        <!-- frpc 管理 -->
        <template v-else-if="activePanel === 'frpc'">
          <section class="md-card-primary d-flex align-center flex-wrap ga-4">
            <div class="md-icon-container frpc-icon">
              <v-icon size="20">fas fa-cube</v-icon>
            </div>
            <div class="flex-grow-1" style="min-width: 180px">
              <div class="d-flex align-center flex-wrap ga-2">
                <span class="text-title-large">frpc</span>
                <v-chip v-if="frpcInstalled" size="small" variant="tonal" color="primary">
                  {{ installedVersion }}
                </v-chip>
                <v-chip size="small" variant="flat" :color="frpcStatusChip.color">
                  {{ frpcStatusChip.text }}
                </v-chip>
              </div>
              <div class="text-body-medium md-on-primary-container mt-1">最新版本 {{ latestVersion }}</div>
            </div>
            <v-btn
              color="primary"
              variant="flat"
              size="large"
              prepend-icon="fas fa-download"
              :loading="installing"
              :disabled="installing"
              @click="handleInstallOrUpdate"
            >
              {{ actionText }}
            </v-btn>
          </section>

          <div class="d-flex flex-wrap ga-2">
            <v-btn class="md-btn-tonal"
              v-if="installing"
              variant="flat"
              prepend-icon="fas fa-stop"
              :loading="canceling"
              :disabled="canceling"
              @click="handleCancelInstall"
            >
              终止下载
            </v-btn>
            <v-btn class="md-btn-tonal"
              variant="flat"
              prepend-icon="fas fa-rotate"
              :disabled="installing"
              @click="loadStatus"
            >
              检查更新
            </v-btn>
            <v-btn
              variant="text"
              color="error"
              prepend-icon="fas fa-trash"
              :disabled="installing"
              @click="handleRemove"
            >
              删除本地 frpc
            </v-btn>
          </div>

          <section v-if="installing" class="md-card-filled">
            <div class="d-flex align-center justify-space-between mb-2">
              <span class="text-body-medium">{{ phaseLabel }}</span>
              <span class="text-body-small md-on-surface-variant">
                {{ progressDetail }}
                <template v-if="phase === 'downloading' && total > 0"> · {{ Math.floor(percent) }}%</template>
              </span>
            </div>
            <v-progress-linear
              :model-value="percent"
              :indeterminate="indeterminate"
              color="primary"
              bg-color="secondary-container"
              bg-opacity="1"
              height="4"
              rounded
            />
          </section>

          <v-alert
            v-model="showMirrorSwitchHint"
            type="warning"
            variant="tonal"
            rounded="lg"
            density="compact"
            closable
          >
            frpc 下载失败或超时，可能是当前下载源网络不佳，建议在下方「GitHub 下载源」中切换镜像后重试。
          </v-alert>

          <v-alert
            v-if="status?.latest_error"
            type="warning"
            variant="tonal"
            rounded="lg"
            density="compact"
          >
            获取最新版本失败：{{ status.latest_error }}
          </v-alert>

          <div class="two-col">
            <section class="md-card-filled">
              <h3 class="text-title-small md-section-title">安装状态</h3>
              <div v-for="detail in installDetails" :key="detail.label" class="md-kv-row">
                <v-icon :icon="detail.icon" size="13" class="md-on-surface-variant" />
                <span class="text-body-medium md-on-surface-variant">{{ detail.label }}</span>
                <span class="text-body-medium ms-auto text-end text-truncate">{{ detail.value }}</span>
              </div>
            </section>
            <section class="md-card-filled">
              <h3 class="text-title-small md-section-title">最新版本</h3>
              <div v-for="detail in latestDetails" :key="detail.label" class="md-kv-row">
                <v-icon :icon="detail.icon" size="13" class="md-on-surface-variant" />
                <span class="text-body-medium md-on-surface-variant">{{ detail.label }}</span>
                <span class="text-body-medium ms-auto text-end text-truncate">{{ detail.value }}</span>
              </div>
            </section>
          </div>

          <section class="md-card-outlined d-flex flex-column ga-3">
            <div>
              <h3 class="text-title-small">GitHub 下载源</h3>
              <p class="text-body-small md-on-surface-variant mt-1">下载 frpc 时使用的地址，网络不佳时可切换到镜像</p>
            </div>
            <v-select
              v-model="mirrorMode"
              :items="mirrorModeItems"
              item-title="title"
              item-value="value"
              label="下载源"
              hide-details="auto"
              :disabled="installing"
            />
            <v-select
              v-if="mirrorMode === 'builtin'"
              v-model="builtinMirrorPresetID"
              :items="builtinMirrorItems"
              item-title="title"
              item-value="value"
              label="内置镜像"
              hide-details="auto"
              :disabled="installing"
              no-data-text="当前没有可用的内置镜像"
            />
            <v-alert
              v-if="mirrorMode === 'builtin' && !builtinMirrorItems.length"
              type="info"
              variant="tonal"
              rounded="lg"
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
              label="自定义方式"
              hide-details="auto"
              :disabled="installing"
            />
            <v-text-field
              v-if="mirrorMode === 'custom' && customMirrorMode === 'base'"
              v-model="customMirrorBaseURL"
              label="基础地址"
              hide-details="auto"
              placeholder="https://example.com/github.com"
              :disabled="installing"
            />
            <v-text-field
              v-if="mirrorMode === 'custom' && customMirrorMode === 'template'"
              v-model="customMirrorURLTemplate"
              label="URL 模板"
              placeholder="https://mirrors.114514.com/{owner}/{repo}/{tag}/{asset}"
              hint="可用占位符：{owner}、{repo}、{tag}、{asset}"
              persistent-hint
              :disabled="installing"
            />
            <v-alert
              v-if="mirrorConfigDirty"
              type="warning"
              variant="tonal"
              rounded="lg"
              density="compact"
            >
              下载源修改尚未保存，点击「保存设置」后生效。
            </v-alert>
            <div class="d-flex flex-wrap ga-2">
              <v-btn color="primary" variant="flat" :disabled="installing" @click="handleSaveMirrorConfig">
                保存设置
              </v-btn>
              <v-btn color="primary" variant="text" :disabled="installing" @click="handleUseOfficialMirror">
                使用 github.com
              </v-btn>
            </div>
          </section>

          <section class="md-card-outlined">
            <h3 class="text-title-small md-section-title">本地目录</h3>
            <div class="d-flex flex-column ga-2">
              <div v-for="path in pathItems" :key="path.label" class="frpc-path-row">
                <span class="frpc-path-label text-body-medium md-on-surface-variant">{{ path.label }}</span>
                <code class="frpc-path-value md-mono" :title="path.value">{{ path.value || "-" }}</code>
                <v-btn
                  icon="fas fa-copy"
                  size="small"
                  variant="text"
                  color="on-surface"
                  :title="`复制 ${path.label}`"
                  :disabled="!path.value"
                  @click="copyPath(path.value)"
                />
              </div>
            </div>
          </section>
        </template>

        <!-- 关于 -->
        <template v-else-if="activePanel === 'about'">
          <section class="md-card-primary d-flex align-center ga-5">
            <div class="about-logo">
              <AppLogo :size="40" />
            </div>
            <div class="flex-grow-1">
              <div class="d-flex align-center flex-wrap ga-2">
                <span class="text-headline-small">LoliaShizuku</span>
                <v-chip size="small" variant="flat" color="primary">v{{ appVersion }}</v-chip>
              </div>
              <div class="text-body-medium md-on-primary-container mt-1">
                「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端
              </div>
            </div>
          </section>

          <section class="md-card-outlined">
            <h3 class="text-title-small md-section-title">构建信息</h3>
            <div v-if="aboutDetails.length" class="about-grid">
              <div v-for="detail in aboutDetails" :key="detail.label" class="about-item">
                <div class="d-flex align-center ga-2 text-body-small md-on-surface-variant">
                  <v-icon :icon="detail.icon" size="12" />
                  {{ detail.label }}
                </div>
                <div class="text-body-large mt-1 about-detail-value">{{ detail.value }}</div>
              </div>
            </div>
            <div v-else class="text-body-small md-on-surface-variant">无法获取构建信息。</div>
          </section>

          <section class="md-card-outlined">
            <h3 class="text-title-small md-section-title">相关链接</h3>
            <div class="d-flex flex-wrap ga-2">
              <v-btn class="md-btn-tonal"
                v-for="link in aboutLinks"
                :key="link.url"
                variant="flat"
                :prepend-icon="link.icon"
                @click="openURL(link.url)"
              >
                {{ link.label }}
              </v-btn>
            </div>
          </section>

          <p class="text-body-small md-on-surface-variant text-center">
            以 MIT 许可证开源 · Made with ♥ by Mxmilu666
          </p>
        </template>

        <!-- 账号 -->
        <template v-else>
          <section class="md-card-outlined d-flex align-center flex-wrap ga-4">
            <div class="md-icon-container">
              <v-icon size="16">fas fa-right-from-bracket</v-icon>
            </div>
            <div class="flex-grow-1" style="min-width: 200px">
              <h3 class="text-title-medium">退出登录</h3>
              <p class="text-body-medium md-on-surface-variant mt-1">
                将清除本地 OAuth 凭据，并停止当前本地运行的 frpc。
              </p>
            </div>
            <v-btn
              color="error"
              variant="flat"
              :loading="logoutLoading"
              :disabled="logoutLoading"
              @click="handleLogout"
            >
              退出登录
            </v-btn>
          </section>
        </template>
      </main>
    </div>
  </div>
</template>

<style scoped>
.min-w-0 {
  min-width: 0;
}

.md-on-primary-container {
  color: var(--md-on-primary-container);
}

.settings-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 24px;
  align-items: start;
}

/* MD3 navigation drawer：56dp 高的全圆角条目，选中项为 secondary container */
.md-nav-drawer {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px;
  border-radius: 16px;
  background: var(--md-surface-container-low);
}

.md-nav-drawer-item {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 56px;
  padding: 0 24px 0 16px;
  border: none;
  border-radius: 28px;
  background: transparent;
  color: var(--md-on-surface-variant);
  cursor: pointer;
  text-align: left;
  transition: background-color 0.2s ease;
}

.md-nav-drawer-item:hover {
  background: rgba(var(--v-theme-on-surface), 0.06);
}

.md-nav-drawer-item--active,
.md-nav-drawer-item--active:hover {
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.is-mobile .settings-layout {
  display: block;
}

.md-settings-list {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 16px;
  background: var(--md-surface-container-low);
}

.md-settings-list-item {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 72px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: rgb(var(--v-theme-on-surface));
  cursor: pointer;
}

.md-settings-list-item:active {
  background: rgba(var(--v-theme-on-surface), 0.1);
}

.md-settings-list-item + .md-settings-list-item {
  border-top: 1px solid var(--md-outline-variant);
}

.md-segmented {
  height: 40px !important;
  border-color: var(--md-outline) !important;
}

.md-segmented :deep(.v-btn) {
  min-width: 112px;
}

.md-segmented :deep(.v-btn--active) {
  background: var(--md-secondary-container);
  color: var(--md-on-secondary-container);
}

.md-segmented :deep(.v-btn--active .v-btn__overlay) {
  opacity: 0;
}

/* 位于 primary container 卡片上，使用最浅的 surface 以区分 */
.frpc-icon {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  background: var(--md-surface-container-lowest);
  color: rgb(var(--v-theme-primary));
}

.two-col {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.accent-swatch {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: none;
  outline: 2px solid transparent;
  outline-offset: 3px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: outline-color 0.15s ease;
}

.accent-swatch:hover {
  outline-color: var(--md-outline-variant);
}

.accent-swatch--active,
.accent-swatch--active:hover {
  outline-color: rgb(var(--v-theme-on-surface));
}

.accent-swatches--disabled {
  opacity: 0.38;
}

.accent-swatch:disabled {
  cursor: default;
}

.md-switch-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.md-switch-row .v-switch {
  flex: 0 0 auto;
}

.dynamic-seed {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
  padding: 12px 12px 12px 16px;
  border-radius: 12px;
  background: var(--md-surface-container-high);
}

.dynamic-seed-dot {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 50%;
  outline: 1px solid var(--md-outline-variant);
}

.md-divider {
  border-color: var(--md-outline-variant);
  opacity: 1;
}

.frpc-path-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.frpc-path-label {
  flex-shrink: 0;
  width: 76px;
}

.frpc-path-value {
  flex: 1;
  min-width: 0;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 0.8rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: rgb(var(--v-theme-on-surface));
  background: var(--md-surface-container-highest);
}

.about-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  flex-shrink: 0;
  border-radius: 20px;
  color: rgb(var(--v-theme-primary));
  background: var(--md-surface-container-lowest);
}

.about-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.about-detail-value {
  word-break: break-word;
}

@media (max-width: 760px) {
  .settings-layout,
  .two-col {
    grid-template-columns: 1fr;
  }

  /* 窄屏（手机）：侧边导航改为可横向滚动的单行 */
  .md-nav-drawer {
    position: static;
    flex-direction: row;
    flex-wrap: nowrap;
    overflow-x: auto;
    padding: 8px;
    scrollbar-width: none;
  }

  .md-nav-drawer-item {
    flex-shrink: 0;
    height: 40px;
    padding: 0 16px;
    border-radius: 20px;
  }

  .md-segmented {
    display: flex !important;
    width: 100%;
  }

  .md-segmented :deep(.v-btn) {
    flex: 1;
    min-width: 0;
    padding: 0 8px;
  }

  .about-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
