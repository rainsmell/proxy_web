---
name: fnos-app-packager
description: Create, package, version, and debug native fnOS/飞牛 NAS FPK applications. Use when working on fnOS app manifests, app/ui/config desktop or gateway entries, cmd/main lifecycle scripts, fnpack builds, Linux/amd64 packaging from Windows, TRIM_* environment variables, FPK install/upgrade issues, 502 gateway errors, mojibake in app descriptions, or native non-Docker fnOS application delivery.
---

# fnOS App Packager

Use this skill for native fnOS `.fpk` app development and packaging. Prefer official fnOS docs when behavior is uncertain because platform schemas can change.

## Start here

1. Confirm whether the target is native FPK or Docker. Do not assume Docker; fnOS native apps use `manifest`, `cmd/`, `config/`, `wizard/`, `app/`, and icons.
2. If exact schema details are needed, read `references/fnos-native-fpk.md`.
3. If a Windows build script is needed, copy/adapt `scripts/build-fnos-template.ps1` into the repo.
4. Always validate by actually running app tests/build, `fnpack build -d <staging-dir>`, package content inspection, and lifecycle script syntax checks when Bash is available.

## Native FPK layout

Use this staging layout:

```text
<appname>/
  manifest
  ICON.PNG
  ICON_256.PNG
  config/privilege
  config/resource
  cmd/main
  cmd/install_init
  cmd/install_callback
  cmd/upgrade_init
  cmd/upgrade_callback
  cmd/uninstall_init
  cmd/uninstall_callback
  cmd/config_init
  cmd/config_callback
  wizard/
  app/
    <linux-amd64 binaries and assets>
    ui/config
    ui/images/icon_64.png
    ui/images/icon_256.png
```

Do not put placeholder files or subdirectories inside `wizard/` unless they are valid wizard JSON definitions. `fnpack` parses entries under `wizard/`; bogus `.gitkeep` or named directories can fail packaging.

## Manifest rules

Use `platform = x86` for Linux/amd64 fnOS packages. Bump `version` for every installable update; fnOS may not upgrade same-version packages.

Prefer ASCII for `desc` and `changelog` unless verified on the target fnOS version. Chinese UTF-8 in `manifest` has caused mojibake in fnOS UI. Ensure no UTF-8 BOM.

Minimal service-style manifest pattern:

```text
appname               = myapp
version               = 0.1.0
display_name          = My App
desc                  = Native fnOS application.
platform              = x86
source                = thirdparty
maintainer            = local
distributor           = local
os_min_version        = 1.2.0401
desktop_uidir         = ui
desktop_applaunchname = myapp.main
ctl_stop              = true
install_type          =
service_port          = 8080
checkport             = true
changelog             = Initial release.
```

## app/ui/config patterns

For LAN/direct port apps, the template style is:

```json
{
  ".url": {
    "myapp.main": {
      "title": "My App",
      "icon": "images/icon_{0}.png",
      "type": "url",
      "protocol": "http",
      "port": "{port}",
      "url": "/",
      "allUsers": false
    }
  }
}
```

For fnOS desktop gateway/external access, prefer Unix socket gateway:

```json
{
  ".url": {
    "myapp.main": {
      "title": "My App",
      "icon": "images/icon_{0}.png",
      "type": "iframe",
      "protocol": "",
      "port": "",
      "gatewayPrefix": "/app/myapp",
      "gatewaySocket": "app.sock",
      "url": "/app/myapp",
      "allUsers": true
    }
  }
}
```

If using gateway prefix, make the web app base-path aware. It must serve both `/` locally and `/app/<appname>/...` through the gateway; frontend API calls must prepend the base path.

## Lifecycle script requirements

`cmd/main` must support:

- `start`: launch the app and write pidfile
- `stop`: terminate app and owned child processes
- `status`: exit `0` when running, `3` when not running

Use fnOS env vars:

- `TRIM_APPDEST`: installed app directory
- `TRIM_PKGVAR`: persistent app data directory
- `TRIM_PKGETC`: config directory
- `TRIM_SERVICE_PORT`: service port from manifest/config
- `TRIM_TEMP_LOGFILE`: optional temporary error output target

Robust scripts should handle both `${TRIM_APPDEST}/binary` and `${TRIM_APPDEST}/app/binary`.

For gateway apps, create `${TRIM_APPDEST}/app.sock`; export an app-specific socket env var; remove stale sockets before start and after stop.

## Windows packaging pitfalls

- PowerShell `Set-Content -Encoding UTF8` on Windows PowerShell 5 can add BOM. Use Python or .NET `UTF8Encoding(false)` for JSON/manifest/scripts that fnpack parses.
- Bash scripts must be LF, not CRLF.
- PowerShell execution policy may block `.ps1`; test with `powershell -NoProfile -ExecutionPolicy Bypass -File ...`.
- Do not commit `fnos-utils/`, `dist/`, `release/`, or generated `.fpk` unless explicitly requested.
- Use safe deletion: verify paths are inside the workspace before recursive cleanup.

## Debug checklist

For 502 in fnOS desktop:

1. Check app process: `ps -ef | grep <app>`.
2. Check socket/port:
   - gateway: `ls -l /vol*/@appcenter/<app>/app.sock`
   - port: `ss -lntp | grep <port>`
3. Health check:
   - gateway: `curl --unix-socket /volX/@appcenter/<app>/app.sock http://<app>/app/<app>/healthz`
   - port: `curl -v http://127.0.0.1:<port>/healthz`
4. Read logs under `/volX/@appdata/<app>/`.
5. Confirm `app/ui/config` matches the serving mode.

If process and health checks pass but desktop still 502, suspect `app/ui/config` gateway fields or fnOS version compatibility, not the backend.

## Validation checklist

Before telling the user a package is ready:

- Run project tests.
- Build Linux/amd64 artifact with `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` unless the app requires cgo.
- Run `fnpack build -d <staging-dir>` and fail on text such as `Packing failed` even if exit code is `0`.
- Inspect the `.fpk` as tar: top level must contain `app.tgz`, `cmd`, `config`, `manifest`, icons, `wizard`.
- Inspect `app.tgz` for app binaries, UI config, and required assets.
- Confirm text files used by fnOS are no-BOM and Bash scripts are LF.
