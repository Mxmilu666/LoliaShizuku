type FrpcServiceBinding = {
  GetFrpcStatus: () => Promise<any>;
  GetGitHubMirrorURL: () => Promise<string>;
  InstallOrUpdateFrpc: () => Promise<any>;
  CancelInstallOrUpdateFrpc: () => Promise<void>;
  RemoveFrpc: () => Promise<void>;
  SetGitHubMirrorURL: (url: string) => Promise<void>;
  SetMirrorConfig: (config: FrpcMirrorConfig) => Promise<void>;
};

function getFrpcServiceBinding(): FrpcServiceBinding {
  const svc = (window as any).go?.services?.FrpcService;
  if (!svc) {
    throw new Error("FrpcService 未绑定，请重启应用。");
  }
  return svc as FrpcServiceBinding;
}

function parseError(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }
  if (typeof error === "string") {
    return new Error(error);
  }
  if (typeof error === "object" && error !== null && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string") {
      return new Error(message);
    }
  }
  return new Error("请求失败");
}

export interface FrpcPaths {
  userdata_dir: string;
  frpc_dir: string;
  bin_dir: string;
  binary_path: string;
  download_dir: string;
  state_path: string;
  settings_path: string;
}

export interface FrpcInstalledInfo {
  version: string;
  asset_name: string;
  sha256: string;
  installed_at: string;
  binary_path: string;
  binary_exists: boolean;
}

export interface FrpcReleaseAsset {
  name: string;
  download_url: string;
  content_type: string;
  size: number;
  digest: string;
  sha256: string;
  os: string;
  arch: string;
  archive_format: string;
}

export interface FrpcReleaseInfo {
  tag_name: string;
  name: string;
  html_url: string;
  asset: FrpcReleaseAsset;
}

export interface FrpcMirrorPreset {
  id: string;
  name: string;
  description?: string;
  base_url?: string;
  url_template?: string;
}

export interface FrpcMirrorConfig {
  mode: "official" | "builtin" | "custom";
  preset_id?: string;
  custom_base_url?: string;
  custom_url_template?: string;
}

export interface FrpcStatus {
  goos: string;
  goarch: string;
  paths: FrpcPaths;
  github_mirror_url: string;
  mirror_config: FrpcMirrorConfig;
  builtin_mirrors: FrpcMirrorPreset[];
  installed?: FrpcInstalledInfo;
  latest?: FrpcReleaseInfo;
  update_available: boolean;
  latest_error?: string;
}

export interface FrpcInstallResult {
  release: FrpcReleaseInfo;
  status: FrpcStatus;
}

export async function getFrpcStatus(): Promise<FrpcStatus> {
  try {
    const svc = getFrpcServiceBinding();
    return (await svc.GetFrpcStatus()) as FrpcStatus;
  } catch (error) {
    throw parseError(error);
  }
}

export async function getGitHubMirrorURL(): Promise<string> {
  try {
    const svc = getFrpcServiceBinding();
    return (await svc.GetGitHubMirrorURL()) as string;
  } catch (error) {
    throw parseError(error);
  }
}

export async function installOrUpdateFrpc(): Promise<FrpcInstallResult> {
  try {
    const svc = getFrpcServiceBinding();
    return (await svc.InstallOrUpdateFrpc()) as FrpcInstallResult;
  } catch (error) {
    throw parseError(error);
  }
}

export async function cancelInstallOrUpdateFrpc(): Promise<void> {
  try {
    const svc = getFrpcServiceBinding();
    await svc.CancelInstallOrUpdateFrpc();
  } catch (error) {
    throw parseError(error);
  }
}

export async function removeFrpc(): Promise<void> {
  try {
    const svc = getFrpcServiceBinding();
    await svc.RemoveFrpc();
  } catch (error) {
    throw parseError(error);
  }
}

export async function setGitHubMirrorURL(url: string): Promise<void> {
  try {
    const svc = getFrpcServiceBinding();
    await svc.SetGitHubMirrorURL(url);
  } catch (error) {
    throw parseError(error);
  }
}

export async function setMirrorConfig(config: FrpcMirrorConfig): Promise<void> {
  try {
    const svc = getFrpcServiceBinding();
    await svc.SetMirrorConfig(config);
  } catch (error) {
    throw parseError(error);
  }
}
