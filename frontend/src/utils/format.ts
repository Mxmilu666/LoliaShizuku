// 应用界面为中文，日期时间固定按 zh-CN 格式化（不跟随系统语言，Android WebView 默认可能是 en-US）
const dateTimeFormatter = new Intl.DateTimeFormat("zh-CN", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hour12: false,
});

export const formatDateTime = (value?: string | number | Date | null): string => {
  if (value === null || value === undefined || value === "") return "-";
  const date = value instanceof Date ? value : new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : dateTimeFormatter.format(date);
};
