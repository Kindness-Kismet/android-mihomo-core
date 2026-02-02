# Mihomo Android .so Builder

This repository builds Android shared libraries (`.so`) from upstream `MetaCubeX/mihomo` and publishes them via GitHub Releases.

## Artifacts

Each upstream `mihomo` release tag (for example `v1.19.19`) maps to a same-tag GitHub Release in this repo, with 2 assets:

- `libclash_arm64.so` (arm64-v8a)
- `libclash_x86_64.so` (x86_64)

Local builds output into `dist/` (gitignored). cgo headers are generated next to the `.so`:

- `dist/libclash_arm64.so` + `dist/libclash_arm64.h`
- `dist/libclash_x86_64.so` + `dist/libclash_x86_64.h`

## Exported C ABI

Built via `go build -buildmode=c-shared`. The authoritative signatures are in the generated `dist/libclash_*.h`. Current exported symbols:

- `invokeAction`: perform an action via JSON (request/response are JSON)
- `setEventListener`: set async event listener callback pointer
- `suspend`: notify suspend/resume
- `forceGC`: trigger a GC cycle
- `updateDns`: request system DNS update (cmfa)
- `getTraffic`, `getTotalTraffic`: return JSON strings (caller must free)
- `startTUN`, `stopTun`: start/stop Android TUN listener

Host callback bridge declarations live in `android-wrapper/bridge.h` (JNI integration: release handle, free strings, protect socket, deliver result).

## invokeAction Protocol (JSON)

`invokeAction` expects JSON like:

```json
{"id":"1","method":"initClash","data":{"home-dir":"/data/user/0/app"}}
```

Notes:

- This project intentionally does not provide legacy/compat decoding. `data` must match the expected JSON type for each `method`.
- `invokeAction` is asynchronous: it returns immediately and the response is delivered via the `callback`.
- Response shape is always: `{id, method, data, code}`. `code=0` means success, `code=-1` means error.
- For the full list of supported `method` values and `data` shapes, check `android-wrapper/contract/contract.go`.

### Method Reference

#### setupConfig

Load and apply configuration. Supports two modes:

**File mode** - load config from file path:
```json
{"id":"1","method":"setupConfig","data":{"config-path":"/path/to/config.yaml"}}
```

**Payload mode** - load config from memory (content string):
```json
{"id":"1","method":"setupConfig","data":{"payload":"mixed-port: 7890\nallow-lan: true\n..."}}
```

Full `data` schema:
```json
{
  "config-path": "string (optional, file path)",
  "payload": "string (optional, config content)",
  "selected-map": {"group-name": "proxy-name", ...},
  "test-url": "string (optional)"
}
```

- If `payload` is provided, it takes precedence over `config-path`
- If neither is provided, reloads the current config file
- `selected-map` applies proxy selections after config load

## Threading and Ownership

- Threading: async events (for example logs) call host `result_func` from a background goroutine; callbacks must be thread-safe.
- Callback pointer ownership:
  - `invokeAction(callback, ...)` treats `callback` as a one-shot handle; it is released after the response is sent.
  - `setEventListener(listener)` holds `listener` and may call it multiple times; when replaced, the old listener is released after in-flight calls complete.
- String ownership:
  - Input `char*` is freed by the library via `free_string()` (host must set `free_string_func`).
  - Output `char*` (from `getTraffic/getTotalTraffic`) must be freed by the host (for example `free()`).

## How it works

- GitHub Actions runs on a schedule (12:00 US Eastern) or via manual dispatch, and checks the latest upstream release tag.
- If this repo does not have a Release with the same tag, it builds and uploads the 2 `.so` files.
- If a Release already exists, it skips (unless `force_build=true`).

## Local build

Requirements:

- Python 3
- Go
- Android NDK (set `ANDROID_NDK_HOME`/`ANDROID_NDK`/`ANDROID_NDK_ROOT`, or provide `ANDROID_HOME`/`ANDROID_SDK_ROOT`)

Steps:

```bash
# 1) Download/update upstream source into mihomo-source/
python3 prebuild.py --tag v1.19.19

# 2) Build shared libs into dist/
python3 build.py --arch all --tag v1.19.19
# Build release libs (strip symbols to reduce size)
python3 build.py --arch all --tag v1.19.19 --mode release

# 3) Verify existing outputs only (no build)
python3 build.py --verify-only --arch all
```

More options:

- `python3 prebuild.py --help`
- `python3 build.py --help`

## Export to Android projects

Some Android projects expect `libclash/android/<abi>/libclash.so` and `libclash/android/includes/<abi>/*.h`. After building, you can export this layout:

```bash
# Export to ./libclash/android (default path)
python3 build.py --verify-only --arch all --export-android

# Or export directly into another project (example)
python3 build.py --verify-only --arch all --export-android "E:/path/to/your/project/libclash/android"
```

The export step copies `android-wrapper/bridge.h` as `bridge.h`.

Notes:

- Upstream source is cloned into `mihomo-source/` (gitignored)
- Build entry is `android-wrapper/` (minimal Android-oriented C ABI wrapper)
- Default output directory is `dist/`
- The build script verifies `.so` (ELF arch + required exported symbols); use `--no-verify` to disable
- Build tags are fixed to `with_gvisor,cmfa` (not configurable).
