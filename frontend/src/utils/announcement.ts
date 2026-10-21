import DOMPurify from "dompurify";
import MarkdownIt from "markdown-it";

// 与网页控制台一致：Markdown（允许内嵌 HTML）渲染后再经 DOMPurify 清洗
const markdown = new MarkdownIt({ html: true, breaks: true, linkify: true });

const sanitize = (html: string): string =>
  DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      "p", "br", "hr", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6",
      "strong", "b", "em", "i", "u", "s", "del", "sup", "sub", "mark", "small",
      "ul", "ol", "li", "blockquote", "pre", "code", "a", "img",
      "table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
      "details", "summary",
    ],
    ALLOWED_ATTR: ["href", "src", "alt", "title", "colspan", "rowspan", "start"],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  });

export const renderAnnouncementContent = (content: string): string =>
  sanitize(markdown.render(content ?? ""));

export interface AnnouncementLevelMeta {
  // Vuetify 主题色名；normal 不使用语义色
  color?: "info" | "warning" | "error" | "success";
  icon: string;
}

export const getAnnouncementLevelMeta = (level: string): AnnouncementLevelMeta => {
  switch (level) {
    case "warning":
      return { color: "warning", icon: "fas fa-triangle-exclamation" };
    case "error":
      return { color: "error", icon: "fas fa-circle-exclamation" };
    case "success":
      return { color: "success", icon: "fas fa-circle-check" };
    case "info":
      return { color: "info", icon: "fas fa-circle-info" };
    default:
      return { icon: "fas fa-bullhorn" };
  }
};

// 仅允许 http / https 链接，避免 javascript: 等协议
export const safeExternalURL = (value?: string | null): string | null => {
  if (!value) return null;
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:" ? url.href : null;
  } catch {
    return null;
  }
};
