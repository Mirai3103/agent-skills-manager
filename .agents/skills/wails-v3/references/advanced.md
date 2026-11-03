# Advanced & experimental topics

Each section gives the essentials and points to the doc page (paths are relative to `docs/mpress/content/`; see `docs-index.md` for raw URLs) and the module example. Read the doc page before implementing any of these. They have more options than fit here.

## Streams: bidirectional bytes, WebSocket-style, no TCP port
Doc: `guides/streams.md`; migrating from WebSockets: `guides/streams-from-websockets.md`; internals: `guides/advanced/streams-internals.md`. Example: `examples/streams`.

```go
app.HandleStream("telemetry", func(c *application.StreamConn) {   // one goroutine per connection
	defer c.Close()
	for {
		frame, err := c.Receive()          // []byte frame; err on reload/close/shutdown
		if err != nil { return }
		_ = c.Send(process(frame))
	}
})
// JSON helpers: c.ReceiveJSON(&v), c.SendJSON(v)
```
```ts
import { Stream, JSONStream } from "@wailsio/runtime";
const s = Stream("telemetry");                 // sync, WebSocket interface (onopen/onmessage/send/close)
s.onmessage = (ev) => use(new Uint8Array(ev.data));
```
Use streams instead of a localhost WebSocket server, and instead of high-frequency events or huge return values.

## Self-updater
Doc: `guides/updater.md`, `guides/updater-github-release-assets.md`, `reference/update-manifest.md`; tutorial `tutorials/04-self-update-a-wails-app.md`. Example: `examples/updater`.

```go
import ("github.com/wailsapp/wails/v3/pkg/updater"; "github.com/wailsapp/wails/v3/pkg/updater/providers/github")
gh, _ := github.New(github.Config{Repository: "me/myapp"})
_ = app.Updater.Init(updater.Config{CurrentVersion: version, Providers: []updater.Provider{gh} /* + public key for verification */})
go app.Updater.CheckAndInstall(ctx)   // default themable window; headless mode available
```
Providers: `github`, `endpoint` (Wails update manifest), `appcast` (Sparkle) and `keygen`. Artifacts are signature-verified, so generate keys with `wails3 updater …`.

## Server mode (no GUI, HTTP only)
Doc: `guides/server-build.md`. Build with `wails3 task build:server` / `run:server` / `build:docker`, or `go build -tags server`. Configure `Options.Server: application.ServerOptions{Host, Port}`. The same code runs as desktop or server. Window, dialog and tray calls are no-ops or stubs in server mode, so guard them with `application.System.IsServer()`.

## Custom URL schemes & file associations
Docs: `guides/distribution/custom-protocols.md`, `guides/file-associations.md`. Examples: `examples/custom-protocol-example`, `examples/file-association`, `examples/single-instance-url-scheme`.
- Declare both in `build/config.yml` (`protocols:` / `fileAssociations:`), then run `wails3 task common:update:build-assets` and repackage. They are registered **by the installer/bundle**, not at runtime. There is no `Protocols` field in `application.Options`.
- Handle them with `events.Common.ApplicationLaunchedWithUrl` (`e.Context().URL()`) and `events.Common.ApplicationOpenedWithFile` (`e.Context().Filename()`). On Windows/Linux a second launch carries the URL or file in its args, so combine with `SingleInstance` and parse `SecondInstanceData.Args`.

## Raw messages & custom transport
Docs: `guides/raw-messages.md`, `guides/custom-transport.md`. `Options.RawMessageHandler func(window Window, message string, origin *OriginInfo)` receives strings sent with `System.invoke(msg)` from JS. It's a low-level escape hatch, so prefer bindings. `Options.Transport` replaces the IPC transport (an advanced case, rarely needed).

## Asset server tricks
- `AssetOptions.Middleware` wraps every request (auth headers, custom routes, logging).
- `application.ServiceOptions{Route: "/api"}` mounts an `http.Handler` service. Gin and other routers fit here (`guides/gin-routing.md`, `guides/gin-services.md`).
- `application.BundledAssetFileServer(fs)` is like `AssetFileServerFS` but also serves the compiled runtime at `/wails/runtime.js`. Use it for no-bundler frontends that load `<script type="module" src="/wails/runtime.js">`.

## Security
Doc: `guides/security.md`. Treat the frontend as untrusted. Every exported service method is callable (generator directives don't hide it), so validate arguments, sanitise paths and authorise inside methods. Loading remote URLs into a window gives that site access to the runtime and your bindings. Avoid it, or open such content in the system browser with `Browser.OpenURL`. Per-window web permissions (camera, mic, geolocation) are set with `WebviewWindowOptions.Permissions` (`features/windows/permissions.md`).

## Panics, logging, errors
- `Options.PanicHandler`, `Options.ErrorHandler`, `Options.WarningHandler`, and `Options.Logger` (`*slog.Logger`) with `LogLevel`. `app.Logger.Info(...)` is available everywhere. Doc: `guides/panic-handling.md`.

## Testing
Docs: `guides/testing.md`, `guides/e2e-testing.md`.
- Unit-test services as plain Go structs. They need no app.
- For end-to-end tests, or to let an agent drive the UI, use the experimental in-app MCP server: `WAILS_MCP=1 wails3 dev`, or build with `-tags mcp`. It serves at `http://127.0.0.1:9099/mcp` and offers tools to inspect windows/DOM, click, type, call bound methods and watch events (`guides/mcp-service.md`). **Never ship the `mcp` tag.**
- The project-level MCP server `wails3 mcp --stdio --root <abs path>` exposes project inspect/doctor/build/dev/generate-bindings tools to MCP clients. Builds return job IDs, so poll `wails_job_status` (`guides/cli.md#mcp`).

## Mobile (experimental)
Docs: `guides/mobile/{index,first-mobile-app,ios,android,mobile-api}.md`. Example: `examples/ios-poc` (kitchen sink linked from docs).
- The same project builds for mobile: `wails3 task ios:run` (macOS + Xcode) or `wails3 task android:run` (Android SDK/NDK). The scaffolding lives in `build/ios`, `build/android`.
- Build tags: `ios` implies `darwin`, and `android` implies `linux`. Use `//go:build darwin && !ios` and `linux && !android` for desktop-only files. At runtime, branch with `application.System.IsMobile()` / `System.IsMobile()` in JS. `application.Mobile` is the native mobile API.
- This is outside the stable compatibility contract. Tell the user so.

## Experimental build features
- **Wake** (`WAILS_EXP_USE_WAKE=1`): a native YAML build system (`wails.yaml`) with incremental builds, parallel packaging and portable signing. Start with `wails3 init -ui`, then `wails3 build setup`. To migrate, run `wails3 migrate --dry-run`, then `wails3 migrate`, then `--activate`. Docs: `experimental/wake.md`, `experimental/yaml-builds.md`, `reference/wails-yaml-fields.md`.
- **purego / CGO-free** (`WAILS_PUREGO=1` or `-tags purego`): macOS/Linux builds without a C toolchain (`CGO_ENABLED=0 go build -tags purego,production .` once the frontend is built). Doc: `experimental/index.md`.
- Opt-in only. Never switch a user's project to these without asking.

## Misc pages worth knowing
`guides/single-instance.md`, `guides/windows-uac.md` (require admin), `features/windows/notch-windows.md` (macOS notch UI), `guides/performance.md`, `guides/patterns/{database,rest-api}.md`, `guides/advanced/custom-templates.md` (your own `wails3 init -t` templates), `migration/v2-to-v3.md`, `migration/from-electron.md`.
