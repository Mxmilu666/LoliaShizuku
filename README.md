![LoliaShizuku](https://socialify.git.ci/Mxmilu666/LoliaShizuku/image?description=1&forks=1&issues=1&language=1&name=1&owner=1&pulls=1&stargazers=1&theme=Auto)

# LoliaShizuku

「ロリア・雫」由 Wails 驱动的 Lolia FRP 第三方客户端

## 功能概览

- OAuth 登录
- 控制台数据看板（用户信息、流量、隧道、版本）
- 隧道列表与流量概览
- 本地 Runner 启停与日志查看
- 内置 frpc 安装/更新/移除
- Material Design 3 界面，支持莫奈取色（桌面端从壁纸自动生成主题色，Android 跟随系统动态配色）

## 技术栈

- 后端：Go 1.25、Wails v3、OAuth2、系统 Keyring
- 前端：Vue 3、TypeScript、Vuetify 4（Material Design 3）、Pinia、Vite

## 环境要求

- Go `>= 1.25`
- Bun（用于前端依赖与构建）
- Wails v3 CLI（`wails3`）
- Linux 额外需要 `libgtk-3-dev`、`libwebkit2gtk-4.1-dev`（构建时需带 `EXTRA_TAGS=gtk3`）

安装 Wails v3 CLI（版本需与 `go.mod` 中的 `github.com/wailsapp/wails/v3` 保持一致）：

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.28
wails3 doctor
```

## 本地开发

在仓库根目录运行：

```bash
wails3 dev
```

这会自动执行 `bun install`、生成前端绑定（`frontend/bindings`）并启动前后端开发环境（配置见 `Taskfile.yml` 与 `build/config.yml`）。

如果只调试前端：

```bash
cd frontend
bun install
bun run dev
```

## 构建

在仓库根目录运行：

```bash
wails3 build      # 输出到 bin/
wails3 package    # 打包：macOS .app / Windows NSIS 安装包 / Linux 包
```

Linux 下请使用 `wails3 build EXTRA_TAGS=gtk3`（项目使用 GTK3 + WebKitGTK 4.1）。
修改导出的 Go 服务方法后，运行 `wails3 generate bindings -clean=true -ts -i` 重新生成前端绑定。

## Android（实验性）

Android 端与桌面端共用同一套运行方式：在应用内下载 frpc 并以子进程运行，同时使用前台服务保持后台存活。
为了能够执行应用私有目录中下载的二进制，应用与 Termux 一样将 `targetSdkVersion` 设为 28（API 29+ 禁止执行），
因此只能通过 APK 分发，无法上架 Google Play。

构建需要 Android SDK、NDK 与 JDK 17–25（可通过 `wails3 task android:install:deps` 检查和安装）。
Gradle 9.2 无法运行在 JDK 26 及以上版本；未设置 `JAVA_HOME` 时构建任务会自动使用 Android Studio 自带的 JBR，
未设置 `ANDROID_HOME` 时使用默认 SDK 路径。release APK 默认使用 debug 证书签名，正式签名可通过
`ANDROID_KEYSTORE_FILE`、`ANDROID_KEYSTORE_PASSWORD`、`ANDROID_KEY_ALIAS`、`ANDROID_KEY_PASSWORD` 环境变量提供：

```bash
wails3 task android:build            # 调试构建（Go 共享库 + 前端）
wails3 task android:package          # 生成 release APK
wails3 task android:run              # 安装到模拟器 / 设备并启动
```

移动端会自动切换为 MD3 底部导航栏、全屏对话框等布局；莫奈取色在 Android 12+ 跟随系统动态配色（Material You），Android 8.1–11 取壁纸主色。

## OAuth 与认证说明

Token 存储在系统 Keyring（service: `LoliaShizuku`, key: `oauth_token`）

默认 OAuth 回调地址为 `http://localhost:1145`。
桌面端登录固定使用 Authorization Code + PKCE。

## 配置项（环境变量）

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `LOLIA_CENTER_API_BASE_URL` | 中心 API 基地址 | `https://api.lolia.link/api/v1` |
| `LOLIA_HTTP_USER_AGENT` | 自定义请求 UA | — |
| `LOLIA_OAUTH_CLIENT_ID` | OAuth Client ID | — |
| `LOLIA_OAUTH_AUTHORIZE_URL` | OAuth 授权地址 | `https://dash.lolia.link/oauth/authorize` |
| `LOLIA_OAUTH_TOKEN_URL` | OAuth Token 地址 | `https://api.lolia.link/api/v1/oauth2/token` |
| `LOLIA_OAUTH_REDIRECT_URL` | OAuth 回调地址 | `http://localhost:1145` |
| `LOLIA_FRPC_REPO_OWNER` | frpc Release 仓库 Owner | `Lolia-FRP` |
| `LOLIA_FRPC_REPO_NAME` | frpc Release 仓库名 | `lolia-frp` |

## frpc 本地目录

frpc 安装在 `os.UserConfigDir()/LoliaShizuku/userdata/frpc/` 下，主要包括：

- `bin/`：frpc 可执行文件
- `downloads/`：下载缓存
- `installed.json`：安装状态
- `settings.json`：下载镜像设置

## 项目状态

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/4d343b8cfbec4da4ac31da906bd41b3f)](https://app.codacy.com/gh/Mxmilu666/LoliaShizuku/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/Mxmilu666/LoliaShizuku)](https://goreportcard.com/report/github.com/Mxmilu666/LoliaShizuku)
![Build](https://github.com/Mxmilu666/LoliaShizuku/actions/workflows/release.yml/badge.svg)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/Mxmilu666/LoliaShizuku)

![Alt](https://repobeats.axiom.co/api/embed/d79f920147af98c01983db8a421018c63bcddc57.svg "Repobeats analytics image")

## 许可证

本项目使用 `MIT` 许可证开源

## 感谢
[LoliaFRP-CLI](https://github.com/Lolia-FRP/lolia-frp)

[FRP](https://github.com/fatedier/frp)
