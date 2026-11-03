# Build, package, sign, distribute (Windows · macOS · Linux)

Docs: `getting-started/installation.md`, `guides/build/{building,cross-platform,customization,windows,macos,linux,msix,signing,obfuscation}.md`, `guides/installers.md`, `guides/cli.md`, `concepts/build-system.md`.

## Contents
1. Prerequisites per OS
2. How the build works (Taskfile pipeline, production tag)
3. Commands
4. Cross-compilation matrix
5. Project metadata, icons, build assets
6. Packaging per platform
7. Code signing and notarization
8. Customising the build
9. CI hints

## 1. Prerequisites

All platforms need Go (the version in go.mod; v3.1.0 requires Go ≥ 1.26), Node/npm (or set `PACKAGE_MANAGER=pnpm|yarn|bun` for the Taskfile), and the CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@<version>`. Run `wails3 doctor` to diagnose your setup.

| OS | Dev libraries |
|---|---|
| Windows | WebView2 runtime (preinstalled on Win10/11). No C compiler needed (CGO off by default). NSIS (`winget install NSIS.NSIS`) for installers |
| macOS | Xcode Command Line Tools (`xcode-select --install`) |
| Linux (default GTK4) | Debian/Ubuntu: `build-essential pkg-config libgtk-4-dev libwebkitgtk-6.0-dev`; Fedora: `gcc pkg-config gtk4-devel webkitgtk6.0-devel`; Arch: `base-devel gtk4 webkitgtk-6.0` |
| Linux (`-tags gtk3`, legacy) | `libgtk-3-dev libwebkit2gtk-4.1-dev` (Debian/Ubuntu); `gtk3-devel webkit2gtk4.1-devel` (Fedora); `gtk3 webkit2gtk-4.1` (Arch) |

End users on Linux need the matching runtime libraries: `libgtk-4-1 libwebkitgtk-6.0-4`, or the GTK3 set for gtk3 builds. A blank or white window on NVIDIA usually needs `WEBKIT_DISABLE_DMABUF_RENDERER=1`. Wails sets this automatically when it detects `/sys/module/nvidia`.

## 2. How the build works

`wails3 build` / `wails3 package` are wrappers around the project's **Taskfile** (Task is embedded in the CLI, so no separate install is needed). `Taskfile.yml` dispatches to `build/<GOOS>/Taskfile.yml`. A production build runs:

`go mod tidy` → install frontend deps → `wails3 generate bindings` → frontend `npm run build` (→ `frontend/dist`) → generate icons/syso → `go build -tags production -trimpath -buildvcs=false -ldflags="-w -s"` (on Windows also `-H windowsgui`) → `bin/<APP_NAME>`

Consequences:
- **A plain `go build` produces a development/debug build** (no `production` tag: DevTools and debug logging enabled). For a manual release build, replicate the flags above, or use `wails3 build`.
- The frontend must be built before Go compiles, because of `//go:embed all:frontend/dist`.
- Edit the Taskfiles freely. They are yours (generated once by init, not regenerated). Variables such as `APP_NAME`, `BIN_DIR`, `PACKAGE_MANAGER` and `VITE_PORT` sit at the top of `Taskfile.yml`.

## 3. Commands

```bash
wails3 dev                         # watcher + Vite; config in build/config.yml (dev_mode)
wails3 dev -port 3000              # custom frontend dev port
wails3 build                       # production build for host → bin/
wails3 build DEV=true              # debug build (what dev mode uses)
wails3 build -tags gtk3            # extra Go tags (forwarded as EXTRA_TAGS)
wails3 build GOOS=windows GOARCH=arm64
wails3 package [GOOS=…] [GOARCH=…] # installers/packages → bin/
wails3 task --list                 # all tasks (platform ones: linux:create:deb, darwin:build:universal, …)
wails3 task build -f               # force (ignore Task's up-to-date checks)
wails3 run                         # build and run
```

## 4. Cross-compilation

| Host → Target | Windows | macOS | Linux |
|---|---|---|---|
| Windows | native | Docker | Docker |
| macOS | native Go (CGO off) | native | Docker |
| Linux | native Go (CGO off) | Docker | native (needs gcc/clang, else Docker) |

- To set up Docker cross-builds once, run `wails3 task setup:docker` (the `wails-cross` image with Zig and the macOS SDK, about 800 MB). The Taskfiles pick Docker automatically when needed.
- Experimental: `WAILS_PUREGO=1 wails3 build` (or `-tags purego`) builds macOS/Linux **without CGO**. It cross-compiles with plain Go and needs no Docker, but target machines still need the native GTK/WebKit libraries. Mention that it's experimental.
- macOS universal binary: `wails3 task darwin:build:universal` / `darwin:package:universal`.

## 5. Metadata, icons, build assets

- `build/config.yml` → `info:` holds companyName, productName, productIdentifier (reverse-DNS), version, copyright and so on. After editing it, run `wails3 task common:update:build-assets` to regenerate Info.plist, the Windows manifest/`info.json`, the .desktop file and the nfpm config. **This overwrites manual edits to those generated assets.**
- Icons: put a 1024×1024 PNG at `build/appicon.png`. `wails3 task common:generate:icons` (part of the build) produces `.ico`/`.icns`. To run it manually: `wails3 generate icons -input build/appicon.png`.
- File associations: the `fileAssociations:` section in `build/config.yml`, then update build assets. Handle opened files through `events.Common.ApplicationOpenedWithFile` (see `advanced.md`).
- Version injection: add `-ldflags "-X main.version=…"` to the `BUILD_FLAGS` in the platform Taskfiles, or read `info.version` at build time.

## 6. Packaging

| Platform | Command | Output |
|---|---|---|
| Windows | `wails3 package GOOS=windows` | NSIS installer `bin/<App>-<arch>-installer.exe` (includes the WebView2 bootstrapper); script in `build/windows/nsis/` |
| Windows | `wails3 package GOOS=windows FORMAT=msix` | `.msix` (needs `makeappx`/MSIX tools: `wails3 task install:msix:tools`) |
| macOS | `wails3 package GOOS=darwin` | `bin/<App>.app` (ad-hoc signed) |
| macOS | `wails3 task darwin:package:dmg` | styled DMG (background/icon assets in `build/darwin/`) |
| Linux | `wails3 package GOOS=linux` | AppImage + .deb + .rpm + Arch package |
| Linux | `wails3 task linux:create:{appimage,deb,rpm,aur}` | a single format; metadata in `build/linux/nfpm/nfpm.yaml`, desktop entry in `build/linux/desktop` |

## 7. Signing

Start with `wails3 setup`, a wizard that stores signing config in `~/.config/wails/defaults.yaml` and passwords in the OS keychain. `wails3 signing` (credentials) and `wails3 entitlements` (macOS) are narrower wizards. Then:

```bash
wails3 sign GOOS=windows     # = wails3 task windows:sign / windows:sign:installer
wails3 sign GOOS=darwin      # = wails3 task darwin:sign
wails3 task darwin:sign:notarize
wails3 task linux:sign:packages   # GPG-sign deb/rpm
```

- Per-project vars live in the platform Taskfiles: Windows `SIGN_CERTIFICATE` (.pfx) or `SIGN_THUMBPRINT`, plus `TIMESTAMP_SERVER`. macOS `SIGN_IDENTITY: "Developer ID Application: Name (TEAMID)"`, `ENTITLEMENTS`, `KEYCHAIN_PROFILE` (from `xcrun notarytool store-credentials`).
- Cross-platform signing is built in. Exported Windows PFX files and exported Apple identities (.p12) plus App Store Connect API keys can sign and notarize from **any host**. Keychain identities, notary profiles, MSI and cert-store identities require the native OS.
- Unsigned Windows binaries trigger SmartScreen. Unsigned or un-notarized macOS apps get "damaged / can't be opened". macOS notifications and some entitlements need a properly signed bundle.

## 8. Customising the build

- Add a step by editing the platform Taskfile's `build`/`package` tasks, for example running codegen before `common:build:frontend`.
- Obfuscation: `wails3 build --obfuscated [--garbleargs "-literals -tiny"]` uses garble (install garble first). Bindings must be generated with `-obfuscated` (the Taskfile handles this). See `guides/build/obfuscation.md`.
- Server mode (no GUI, HTTP only, CGO-free): `wails3 task build:server` / `build:docker`. See `advanced.md`.
- Private macOS APIs (some window effects) are opt-in; `guides/build/private-macos-apis.md` lists them. Avoid them for App Store builds.
- Experimental **Wake** (`WAILS_EXP_USE_WAKE=1`) replaces Taskfiles with a single `wails.yaml` (`wails3 init -ui`, `wails3 build setup`, `wails3 migrate --dry-run`). It's opt-in. See `experimental/wake.md` and `experimental/yaml-builds.md`.

## 9. CI hints

- The usual pattern is a matrix of native runners (windows-latest, macos-latest, ubuntu-latest), each running `go install …/wails3@<version>`, `npm ci` and `wails3 package`. On Linux, install the GTK4/WebKitGTK 6 `-dev` packages first.
- Alternatively, run one Linux runner with `wails3 task setup:docker`, cross-build everything, and sign with exported credentials.
- `guides/build/signing.md` has complete GitHub Actions workflows for macOS, Windows and the cross-platform Linux runner.
