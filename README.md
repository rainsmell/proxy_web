# v2ray-web

基于目录内 V2Ray 核心程序的局域网 Web 管理服务。

## 运行

需要安装 Go 1.22 或更高版本：

```bash
go run .
```

浏览器访问 `http://<本机IP>:8080`。可通过 `V2RAY_WEB_ADDR=:8080` 修改 Web 监听地址。

## 生成免安装 Go 的发布包

Go 只需要出现在构建机上，发布给用户的是已经包含 Go 运行时的单个可执行文件。Windows 和 Linux 发布包可使用：

```powershell
.\build.ps1
```

或在 Linux/macOS 上：

```bash
./build.sh
```

脚本会生成 `release/windows` 和 `release/linux`，其中包含 Web 程序、对应的 V2Ray 核心及资源文件。目标机器无需安装 Go。

程序会自动选择：

- Windows：`v2ray-windows-64/v2ray.exe`
- Linux：`v2ray-linux-64/v2ray`

配置保存到 `data/config.json`，运行时配置保存到 `data/runtime/config.json`。

支持 VMess、VLESS、Trojan、Shadowsocks 的 URI 或 Base64 订阅。测速按钮当前执行节点 TCP 连通性测试；代理延迟字段会在后续接入临时 V2Ray 测试实例时使用。

> 当前 Web 服务无认证，请仅在可信局域网使用，不要直接暴露到公网。
