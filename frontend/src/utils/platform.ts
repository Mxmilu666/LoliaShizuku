import { onBeforeUnmount, ref, watch, type Ref } from "vue";
import { System } from "@wailsio/runtime";

type AndroidJSBridge = {
  platform?: () => string;
  setSystemBars?: (json: string) => void;
  getDynamicSeed?: () => string;
};

const androidBridge = (): AndroidJSBridge | undefined =>
  (window as unknown as { wails?: AndroidJSBridge }).wails;

/**
 * 是否运行在移动端（Android / iOS）。移动端使用底部导航栏、全屏对话框等 MD3 移动布局。
 *
 * 注意：Android 上 Wails 在页面加载后才注入 `window._wails.environment`，模块加载时
 * `System.IsMobile()` 可能仍为 false，因此同时检查 JS 桥与 User-Agent。
 */
export const isMobile: boolean = (() => {
  try {
    if (System.IsMobile()) return true;
  } catch {
    // 运行时尚未就绪
  }
  if (androidBridge()?.platform?.() === "android") return true;
  return /Android|iPhone|iPad|iPod/i.test(navigator.userAgent);
})();

/**
 * 让 Android 状态栏 / 导航栏与应用的顶栏、底部导航栏颜色保持一致（MD3）。
 * 非 Android 或模板未提供该桥接方法时为空操作。
 */
export function setSystemBars(options: { statusBar: string; navigationBar: string; dark: boolean }) {
  try {
    androidBridge()?.setSystemBars?.(JSON.stringify(options));
  } catch {
    // ignore
  }
}

/**
 * 读取 Android 系统的莫奈种子色（Android 12+ 取系统动态配色，8.1–11 取壁纸主色）。
 * 非 Android 或无法获取时返回 null。
 */
export function getAndroidDynamicSeed(): { seed: string; source: "system" | "wallpaper" } | null {
  try {
    const raw = androidBridge()?.getDynamicSeed?.();
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { seed?: unknown; source?: unknown };
    if (typeof parsed.seed !== "string" || !/^#[0-9a-f]{6}$/i.test(parsed.seed)) return null;
    return { seed: parsed.seed.toLowerCase(), source: parsed.source === "system" ? "system" : "wallpaper" };
  } catch {
    return null;
  }
}

/**
 * 当前显示的移动端 FAB 数量。大于 0 时 Snackbar 需上移到 FAB 之上（MD3）。
 */
export const mobileFabCount = ref(0);

/**
 * 当前打开的全屏对话框数量（移动端）。大于 0 时系统栏改用对话框的 surface 颜色。
 */
export const fullscreenDialogCount = ref(0);

/**
 * 移动端全屏对话框打开期间，让 Android 状态栏 / 导航栏与对话框背景一致。
 */
export function useFullscreenSystemBars(open: Ref<boolean>) {
  if (!isMobile) return;
  let counted = false;
  const sync = (value: boolean) => {
    if (value && !counted) {
      fullscreenDialogCount.value += 1;
      counted = true;
    } else if (!value && counted) {
      fullscreenDialogCount.value -= 1;
      counted = false;
    }
  };
  watch(open, sync, { immediate: true });
  onBeforeUnmount(() => sync(false));
}
