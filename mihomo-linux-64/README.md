# mihomo (Linux amd64)

This directory holds the bundled mihomo (Clash.Meta) core used by the Linux
build and the fnOS package.

The binary is **not** committed to git. Fetch it with:

```bash
bash scripts/fetch-core.sh linux
# specific version / mirror:
MIHOMO_VERSION=v1.19.31 MIHOMO_MIRROR=https://gh-proxy.com/ bash scripts/fetch-core.sh linux
```

The script downloads the **`-compatible`** asset by default (GOAMD64=v1). This
is required on many NAS CPUs (Celeron J4125/N5105, older Atom/AMD, etc.) that
do not support x86-64-v3/AVX2. Using the normal asset on such a CPU fails with:

```
This program can only be run on AMD64 processors with v3 microarchitecture support.
```

Use the upstream v3 build only when you know the CPU supports AVX2:

```bash
MIHOMO_VARIANT=default bash scripts/fetch-core.sh linux
```

Expected file: `mihomo` (executable, Linux amd64).
