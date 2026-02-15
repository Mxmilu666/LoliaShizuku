# LoliaShizuku

LoliaShizuku 是一个基于 Wails + Vue 3 + TypeScript 的 Lolia FRP 第三方桌面客户端。

## 功能概览

- OAuth 登录（系统浏览器授权 + 本地回调）
- Access Token 过期后自动使用 Refresh Token 刷新
- 控制台数据看板（用户信息、流量、隧道、版本）
- 隧道列表与流量概览
- 本地 Runner 启停与日志查看
- 内置 frpc 安装/更新/移除
- 支持设置 GitHub 下载镜像地址

## 技术栈

- 后端：Go 1.24、Wails v2、OAuth2、系统 Keyring
- 前端：Vue 3、TypeScript、Vuetify、Pinia、Vite

## 环境要求

- Go `>= 1.24`
- Bun（用于前端依赖与构建）
- Wails CLI

安装 Wails CLI：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

## 本地开发

在仓库根目录运行：

```bash
wails dev
```

这会自动执行 `bun install` 并启动前后端开发环境（以 `wails.json` 为准）。

如果只调试前端：

```bash
cd frontend
bun install
bun run dev
```

## 构建

在仓库根目录运行：

```bash
wails build
```

## OAuth 与认证说明

- Token 存储在系统 Keyring（service: `LoliaShizuku`, key: `oauth_token`）
- 每次调用中心 API 前会检查 token：
  - 未过期：直接使用
  - 已过期且存在 `refresh_token`：自动刷新并回写 Keyring
  - 刷新失败或未授权：清理本地 token，并在路由守卫中回到 `/oauth`

默认 OAuth 回调地址为 `http://localhost:1145`。

## 配置项（环境变量）

### Center API

- `LOLIA_CENTER_API_BASE_URL`：中心 API 基地址  
  默认：`https://api.lolia.link/api/v1`
- `LOLIA_HTTP_USER_AGENT`：自定义请求 UA（可选）

### OAuth

- `LOLIA_OAUTH_CLIENT_ID`
- `LOLIA_OAUTH_CLIENT_SECRET`
- `LOLIA_OAUTH_AUTHORIZE_URL`  
  默认：`https://dash.lolia.link/oauth/authorize`
- `LOLIA_OAUTH_TOKEN_URL`  
  默认：`https://api.lolia.link/api/v1/oauth2/token`
- `LOLIA_OAUTH_REDIRECT_URL`  
  默认：`http://localhost:1145`
- `LOLIA_OAUTH_USE_PKCE`（默认开启；设置为 `0/false/no/off` 可关闭）

### frpc Release 源

- `LOLIA_FRPC_REPO_OWNER`（默认：`Lolia-FRP`）
- `LOLIA_FRPC_REPO_NAME`（默认：`lolia-frp`）

## frpc 本地目录

frpc 安装在 `os.UserConfigDir()/LoliaShizuku/userdata/frpc/` 下，主要包括：

- `bin/`：frpc 可执行文件
- `downloads/`：下载缓存
- `installed.json`：安装状态
- `settings.json`：下载镜像设置

## 发布流程（GitHub Actions）

- Workflow：`.github/workflows/release.yml`
- 触发条件：
  - 任意分支 push：执行多平台构建
  - 推送 `v*` tag：构建并创建 GitHub Release
- Release Notes：由 workflow 在发布时根据 commit 自动生成（基于前一个 tag 到当前 tag 的 `git log`）

## 许可证

本项目使用 `LICENSE` 中声明的许可证。
