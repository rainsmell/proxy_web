# fnOS native FPK reference

## Official docs to check first

Use official docs when internet access is allowed:

- https://developer.fnnas.com/docs/quick-started/create-application/
- https://developer.fnnas.com/docs/core-concepts/framework/
- https://developer.fnnas.com/docs/core-concepts/manifest/
- https://developer.fnnas.com/docs/core-concepts/environment-variables/
- https://developer.fnnas.com/docs/core-concepts/privilege/
- https://developer.fnnas.com/docs/core-concepts/resource/
- https://developer.fnnas.com/docs/core-concepts/app-entry/
- https://developer.fnnas.com/docs/core-concepts/gateway-registration/
- https://developer.fnnas.com/docs/cli/fnpack/

## fnpack behavior observed

- `fnpack create <appname> -t native` creates a usable skeleton.
- `fnpack build -d <dir>` can print failure text while returning success-like process behavior. Parse output, not only exit code.
- JSON files with UTF-8 BOM fail with errors like `invalid character '茂' looking for beginning of value`.
- Invalid entries under `wizard/` fail as wizard JSON validation errors.

## Good lifecycle script properties

- Log every resolved path at start: `TRIM_APPDEST`, `TRIM_PKGVAR`, app binary, socket/port, data dir.
- For app daemons that spawn children, install signal handlers in the app where possible and also best-effort cleanup child processes in `cmd/main stop`.
- Keep pidfiles in `TRIM_PKGVAR`, not app install directory.
- Use health checks after launch; return nonzero on failure so fnOS reports start failure instead of silent 502.

## Gateway mode notes

- `gatewaySocket` path is relative to `TRIM_APPDEST` in the UI config, e.g. `app.sock`.
- The app should listen on `${TRIM_APPDEST}/app.sock`.
- The app must understand requests under `gatewayPrefix`, e.g. `/app/myapp/healthz`.
- Frontend code must not hardcode `/api/...` when gateway mode is used; inject or compute a base path.

## Versioning

- Increment package version for every user-installable FPK fix.
- Suggested sequence for iterative debugging: `0.1.0`, `0.1.1`, ...
- Keep changelog ASCII if fnOS UI shows mojibake.

## Common failure signatures

- Desktop 502 but local `curl http://127.0.0.1:<port>/healthz` works: wrong `app/ui/config` mode or gateway/port mismatch.
- `invalid character '茂'`: BOM in JSON/manifest.
- `File "uninstall" is not valid due to JSON format`: invalid `wizard/` entry.
- Service starts but app cannot open through external NAS URL: port entry may not work through fnOS gateway; switch to Unix socket gateway.
