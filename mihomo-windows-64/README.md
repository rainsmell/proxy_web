# mihomo (Windows amd64)

This directory holds the bundled mihomo (Clash.Meta) core used by the Windows
build.

The binary is **not** committed to git. Fetch it with:

```powershell
.\scripts\fetch-core.ps1 -Target windows
# specific version / mirror:
.\scripts\fetch-core.ps1 -Version v1.19.31 -Target windows -Mirror https://gh-proxy.com/
```

The script downloads the **`-compatible`** asset by default (GOAMD64=v1), which
also runs on older x86-64 CPUs without AVX2. Pass `-Variant default` for the
upstream v3 build.

Expected file: `mihomo.exe` (Windows amd64).
