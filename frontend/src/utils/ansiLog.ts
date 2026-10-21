import { AnsiUp } from "ansi_up";

// frpc 行首时间戳，如 "2:41PM"（ansi_up 输出前已做 HTML 转义，只匹配数字与 AM/PM）
const TIME_PREFIX_PATTERN = /^(\d{1,2}:\d{2}(?:AM|PM))(?= )/;
const RUNNER_PREFIX = "[runner]";

/**
 * 把带 ANSI 颜色的 frpc 日志渲染为 HTML。
 * ansi_up 会转义 HTML，颜色输出为 ansi-*-fg 类名，由页面样式映射到 MD3 主题色。
 */
export function renderAnsiLog(lines: string[]): string {
  const ansi = new AnsiUp();
  ansi.use_classes = true;
  // 跨行保持 ANSI 状态，与终端行为一致
  return lines
    .map((line) => {
      const html = ansi.ansi_to_html(line);
      if (line.startsWith(RUNNER_PREFIX)) {
        return `<span class="log-runner">${html}</span>`;
      }
      return html.replace(TIME_PREFIX_PATTERN, '<span class="log-time">$1</span>');
    })
    .join("\n");
}
