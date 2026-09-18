# v2ray-web

基于本地 V2Ray Core 的轻量 Web 管理程序。

## 功能

- 添加订阅链接并拉取节点列表
- 列表展示 VMess / VLESS / Trojan / Shadowsocks 节点
- 节点 TCP 并发测速与进度显示
- 选择节点后启动本地 SOCKS5 / HTTP 代理
- 配置 SOCKS5 / HTTP 入站端口
- 查看 V2Ray access log 中的连接记录和 error log

## 本地运行

```powershell
& 'C:\Program Files\Go\bin\go.exe' run .
```

默认访问：`http://127.0.0.1:8080/`

可用环境变量：

- `V2RAY_WEB_ADDR`：Web 服务监听地址，例如 `:8080`
- `V2RAY_WEB_DATA_DIR`：配置和运行日志目录，默认 `data`
- `V2RAY_CORE_PATH`：指定 V2Ray core 可执行文件路径

程序会自动选择当前目录下的 core：

- Windows：`v2ray-windows-64/v2ray.exe`
- Linux：`v2ray-linux-64/v2ray`

## 普通发布包

```powershell
.\build.ps1
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
- `scripts/build-fnos.ps1`：Windows 下构建 Linux/amd64 FPK

准备：

1. 安装 Go。
2. 将 fnpack 放到 `fnos-utils/`，例如：
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
- `TRIM_PKGVAR`：持久化数据目录，程序的配置、运行日志、V2Ray runtime config 都写在这里
- `TRIM_SERVICE_PORT`：fnOS 分配/配置的 Web 服务端口

生命周期脚本会：

- 启动 `${TRIM_APPDEST}/v2ray-web`
- 设置 `V2RAY_WEB_DATA_DIR=${TRIM_PKGVAR}`
- 设置 `V2RAY_WEB_ADDR=:${TRIM_SERVICE_PORT:-8080}`
- 设置 `V2RAY_CORE_PATH=${TRIM_APPDEST}/v2ray-linux-64/v2ray`
- 停止 Web 进程时清理由它启动的 V2Ray 子进程

注意：当前 Web 页面没有用户认证，应只暴露在可信局域网或由 fnOS 权限入口访问。
