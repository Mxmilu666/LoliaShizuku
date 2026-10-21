import { BrowserService } from "../../bindings/github.com/Mxmilu666/LoliaShizuku/backend/services";

/**
 * 在系统浏览器中打开外部链接。
 * 不使用 @wailsio/runtime 的 Browser.OpenURL：它在 Android 上不可用（Wails 内部调用 xdg-open），
 * 改由 Go 侧 BrowserService 处理（Android 使用 Intent）。
 */
export function openExternalURL(url: string): void {
  void BrowserService.OpenURL(url).catch((error: unknown) => {
    console.error("打开链接失败", error);
  });
}
