import {
  argbFromRgb,
  hexFromArgb,
  sourceColorFromImageBytes,
} from "@material/material-color-utilities";
import { SystemColorService } from "../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";
import {
  DYNAMIC_ACCENT_ID,
  applySeedColors,
  readSavedAccentId,
  saveCachedDynamicSeed,
  type DynamicSeed,
  type ThemeColorMaps,
} from "@/plugins/theme";
import { getAndroidDynamicSeed } from "@/utils/platform";

const decodeBase64 = (value: string): Uint8ClampedArray => {
  const binary = atob(value);
  return Uint8ClampedArray.from(binary, (char) => char.charCodeAt(0));
};

// 解析 "rgb(r,g,b)" / "#rrggbb" 形式的系统强调色
const parseCssColor = (value: string): string | null => {
  const text = value.trim();
  if (/^#[0-9a-f]{6}$/i.test(text)) {
    return text.toLowerCase();
  }
  const match = text.match(/^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
  if (!match) {
    return null;
  }
  const [r, g, b] = match.slice(1, 4).map((part) => Math.min(255, Number(part)));
  return hexFromArgb(argbFromRgb(r, g, b));
};

/**
 * 莫奈取色：优先从桌面壁纸提取种子色（与 Android 动态取色相同的 Celebi 量化 + Score 评分算法），
 * 读取壁纸失败时回退到系统强调色。Android 直接使用系统提供的莫奈种子色。
 */
export async function extractDynamicSeed(): Promise<DynamicSeed> {
  const android = getAndroidDynamicSeed();
  if (android) {
    return android;
  }

  try {
    const pixels = await SystemColorService.GetWallpaperPixels();
    if (pixels?.rgba) {
      const seed = sourceColorFromImageBytes(decodeBase64(pixels.rgba));
      return { seed: hexFromArgb(seed), source: "wallpaper" };
    }
  } catch {
    // 回退到系统强调色
  }

  const accent = parseCssColor(await SystemColorService.GetSystemAccentColor());
  if (!accent) {
    throw new Error("无法获取系统颜色");
  }
  return { seed: accent, source: "accent" };
}

/**
 * 重新提取动态种子色并缓存；当前强调色为「动态取色」时立即应用到主题。
 */
export async function refreshDynamicColors(themes: ThemeColorMaps): Promise<DynamicSeed> {
  const result = await extractDynamicSeed();
  saveCachedDynamicSeed(result);
  if (readSavedAccentId() === DYNAMIC_ACCENT_ID) {
    applySeedColors(themes, result.seed);
  }
  return result;
}
