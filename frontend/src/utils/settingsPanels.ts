export type SettingsPanel = "frpc" | "appearance" | "about" | "account";

export interface SettingsPanelItem {
  value: SettingsPanel;
  title: string;
  description: string;
  icon: string;
}

export const settingsPanels: SettingsPanelItem[] = [
  { value: "frpc", title: "frpc 管理", description: "安装、更新与下载源", icon: "fas fa-cloud-arrow-down" },
  { value: "appearance", title: "外观", description: "主题模式与主题色", icon: "fas fa-palette" },
  { value: "about", title: "关于", description: "版本与构建信息", icon: "fas fa-circle-info" },
  { value: "account", title: "账号", description: "退出登录", icon: "fas fa-user" },
];

export const findSettingsPanel = (value: unknown): SettingsPanelItem | undefined =>
  settingsPanels.find((item) => item.value === value);
