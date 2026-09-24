# proxy-web

基于本地 [mihomo](https://github.com/MetaCubeX/mihomo)（Clash.Meta）内核的轻量 Web 管理程序。

> 早期版本使用 V2Ray Core，现已替换为 mihomo，以获得更活跃的维护和更全面的协议支持。

## 功能

- 添加订阅链接并拉取节点列表
  - 支持 base64 订阅（分享链接）以及 **Clash / mihomo YAML** 订阅
  - Clash YAML 中的任意 mihomo 协议都会被原样透传，无需逐协议适配
- 列表展示节点，支持协议筛选：
  VMess / VLESS（含 REALITY）/ Trojan / Shadowsocks / ShadowsocksR /
  Hysteria / Hysteria2 / TUIC / Snell 等
- 节点 TCP 并发测速与进度显示
- 选择节点后启动本地 SOCKS5 / HTTP 代理
- 配置 SOCKS5 / HTTP 入站端口
- 通过 mihomo REST API 查看实时活动连接，以及核心运行日志

## 内核二进制

仓库不再提交内核二进制。首次构建/运行前先下载 mihomo：

```bash
# Linux
bash scripts/fetch-core.sh linux

# Windows（PowerShell）
.\scripts\fetch-core.ps1 -Target windows

# 同时下载两个平台
bash scripts/fetch-core.sh all
```

默认版本 `v1.19.31`，可用环境变量覆盖：

```bash
MIHOMO_VERSION=v1.19.31 MIHOMO_MIRROR=https://gh-proxy.com/ bash scripts/fetch-core.sh all
```

下载脚本默认取官方 **`-compatible`**（GOAMD64=v1）内核。飞牛等 NAS 的低功耗 CPU（赛扬 J4125/N5105、老 Atom/AMD 等）普遍不支持 x86-64-v3/AVX2，用普通内核会报 `This program can only be run on AMD64 processors with v3 microarchitecture support.`，此时务必用 compatible 版本。若确定 CPU 支持 AVX2，可 `MIHOMO_VARIANT=default` 取官方普通版。

下载脚本会按顺序尝试直连、`gh-proxy.com`、`ghfast.top`，也可用 `MIHOMO_MIRROR` 指定加速前缀。

下载后目录结构：

- Windows：`mihomo-windows-64/mihomo.exe`
- Linux：`mihomo-linux-64/mihomo`

程序按以下顺序查找内核：

1. 环境变量 `MIHOMO_CORE_PATH`（兼容旧变量 `V2RAY_CORE_PATH`）
2. `mihomo-windows-64/mihomo.exe` 或 `mihomo-linux-64/mihomo`
3. 旧版 `v2ray-windows-64/v2ray.exe` 或 `v2ray-linux-64/v2ray`（兼容保留）
4. `PATH` 中的 `mihomo`

## 本地运行

```bash
go run .
```

默认访问：`http://127.0.0.1:8080/`

可用环境变量：

- `V2RAY_WEB_ADDR`：Web 服务监听地址，例如 `:8080`
- `V2RAY_WEB_DATA_DIR`：配置、runtime 配置与日志目录，默认 `data`
- `MIHOMO_CORE_PATH`：指定 mihomo 可执行文件路径

运行时会在 `data/runtime/` 下生成：

- `config.yaml`：mihomo 的 Clash 配置（由订阅节点自动生成）
- `process.log`：mihomo 的 stdout/stderr

## 普通发布包

```bash
./build.sh          # Linux 构建机
.\build.ps1         # Windows 构建机
```

生成：

- `release/windows`
- `release/linux`

目标机器不需要安装 Go。

## fnOS / 飞牛 NAS 原生 FPK 打包

当前分支包含 fnOS 原生应用配置，不使用 Docker。

目录：

- `packaging/fnos/manifest`：应用元数据，平台为 `x86`，服务端口默认 `8080`
- `packaging/fnos/config/privilege`：使用 package 用户运行
- `packaging/fnos/config/resource`：资源配置
- `packaging/fnos/cmd/main`：`start|stop|status` 生命周期脚本
- `packaging/fnos/app/ui/config`：桌面入口配置
- `scripts/build-fnos.ps1` / `scripts/build-fnos.sh`：构建 Linux/amd64 FPK

准备：

1. 安装 Go。
2. 准备 Linux mihomo 内核：`bash scripts/fetch-core.sh linux` 或 `.\scripts\fetch-core.ps1 -Target linux`。
3. 将 fnpack 放到 `fnos-utils/`，例如：
   - `fnos-utils/fnpack.exe`
   - 或 `fnos-utils/fnpack-1.2.3-windows-amd64`

构建：

```powershell
.\scripts\build-fnos.ps1
```

输出位于：

```text
dist/fnos/*.fpk
```

fnOS 运行时使用的环境变量：

- `TRIM_APPDEST`：安装后的应用文件目录
- `TRIM_PKGVAR`：持久化数据目录，程序的配置、运行日志、mihomo runtime config 都写在这里
- `TRIM_SERVICE_PORT`：fnOS 分配/配置的 Web 服务端口

生命周期脚本会：

- 启动 `${TRIM_APPDEST}/v2ray-web`
- 设置 `V2RAY_WEB_DATA_DIR=${TRIM_PKGVAR}`
- 设置 `V2RAY_WEB_ADDR=:${TRIM_SERVICE_PORT:-8080}`
- 设置 `MIHOMO_CORE_PATH=${TRIM_APPDEST}/mihomo-linux-64/mihomo`
- 停止 Web 进程时清理由它启动的 mihomo 子进程

注意：当前 Web 页面没有用户认证，应只暴露在可信局域网或由 fnOS 权限入口访问。

## 实现说明

- 运行配置为 Clash YAML，由 `mihomo.go` 生成；订阅节点若来自 Clash YAML，
  则直接透传其 `proxies` 映射。
- 启动前用 `mihomo -t` 校验配置合法性。
- 为读取活动连接，会在 `127.0.0.1` 上随机分配一个仅本机可访问的
  `external-controller` 端口，并使用随机 `secret`；Web 端通过该 REST API 拉取
  `/connections`。
