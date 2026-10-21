import { createApp } from "vue";
import { createPinia } from "pinia";
import "vuetify/styles";
import { createVuetify } from "vuetify";
import { aliases, fa } from "vuetify/iconsets/fa";
import { md3 } from "vuetify/blueprints";
import {
  lightTheme,
  darkTheme,
  applyAccentColors,
  readSavedAccentId,
  DYNAMIC_ACCENT_ID,
} from "./plugins/theme";
import { refreshDynamicColors } from "./services/dynamicColor";
import { isMobile } from "./utils/platform";

// 移动端在根元素上标记，供全局样式切换为移动布局
if (isMobile) {
  document.documentElement.classList.add("is-mobile");
}
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";

import App from "./App.vue";
import router from "./router";
import "./assets/css/style.css";
import "./assets/css/comfortaa-fonts.css";
import "@fortawesome/fontawesome-free/css/all.css";
import "unfonts.css";

const themeStorageKey = "lolia.theme";
type ThemeMode = "system" | "lightTheme" | "darkTheme";

const getSystemThemeName = (): "lightTheme" | "darkTheme" =>
  window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "darkTheme"
    : "lightTheme";

const readSavedThemeMode = (): ThemeMode => {
  try {
    const savedTheme = localStorage.getItem(themeStorageKey);
    if (
      savedTheme === "system" ||
      savedTheme === "lightTheme" ||
      savedTheme === "darkTheme"
    ) {
      return savedTheme;
    }
  } catch {
    // ignore localStorage errors
  }
  return "system";
};

const resolveThemeName = (mode: ThemeMode): "lightTheme" | "darkTheme" => {
  if (mode === "system") {
    return getSystemThemeName();
  }
  return mode;
};

// Apply the saved accent before Vuetify reads the theme definitions so the
// first paint already uses the chosen color.
applyAccentColors({ lightTheme, darkTheme }, readSavedAccentId());

const vuetify = createVuetify({
  components,
  directives,
  blueprint: md3,
  // 保持 v3 的断点，避免 Vuetify 4 缩小断点后现有页面的栅格布局变化
  display: {
    thresholds: {
      md: 960,
      lg: 1280,
      xl: 1920,
      xxl: 2560,
    },
  },
  defaults: {
    // MD3 chip 形状为 8dp 圆角（blueprint 默认为 2px）
    VChip: {
      rounded: "lg",
    },
  },
  theme: {
    defaultTheme: resolveThemeName(readSavedThemeMode()),
    themes: {
      lightTheme,
      darkTheme,
    },
  },
  icons: {
    defaultSet: "fa",
    aliases,
    sets: {
      fa,
    },
  },
});

const syncSystemTheme = () => {
  if (readSavedThemeMode() === "system") {
    vuetify.theme.global.name.value = getSystemThemeName();
  }
};

const prefersDarkMedia = window.matchMedia("(prefers-color-scheme: dark)");
if (typeof prefersDarkMedia.addEventListener === "function") {
  prefersDarkMedia.addEventListener("change", syncSystemTheme);
}

syncSystemTheme();

const pinia = createPinia();

createApp(App).use(pinia).use(router).use(vuetify).mount("#app");

// 莫奈取色：启动时先使用缓存的种子色，再在后台按当前壁纸重新取色（壁纸可能已更换）
if (readSavedAccentId() === DYNAMIC_ACCENT_ID) {
  void refreshDynamicColors(vuetify.theme.themes.value).catch(() => {
    // 取色失败时保留缓存配色
  });
}

// Android：从后台切回时重新取色（期间可能更换了壁纸 / 系统配色）
if (isMobile) {
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible" && readSavedAccentId() === DYNAMIC_ACCENT_ID) {
      void refreshDynamicColors(vuetify.theme.themes.value).catch(() => {});
    }
  });
}
