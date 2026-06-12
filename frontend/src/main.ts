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
} from "./plugins/theme";
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
