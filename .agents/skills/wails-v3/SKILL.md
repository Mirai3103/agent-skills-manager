---
name: wails-v3
description: Build desktop apps with Wails v3 (Go backend + web frontend in the OS WebView) — scaffolding with wails3 init, services and generated JS/TS bindings, events, windows, menus, system tray, dialogs, @wailsio/runtime, wails3 dev/build/package, signing and cross-platform distribution for Windows, macOS and Linux. Use this skill whenever the project's go.mod requires github.com/wailsapp/wails/v3, the user mentions Wails v3, wails3, @wailsio/runtime, application.New / app.Window / app.Event, or asks to build a Go desktop app with a web UI — even if they just say "Wails" and the project turns out to be v3. Not for Wails v2 projects (github.com/wailsapp/wails/v2, wails.Run, wailsjs/).
---

# Wails v3 app development

Wails v3 compiles a Go program and a web frontend into one native binary. The UI runs in the OS WebView (WebView2 on Windows, WKWebView on macOS, WebKitGTK on Linux). Frontend code calls Go through generated bindings, and Go pushes data to the frontend with events.

This skill was written against **v3.1.0** (stable since 2026-11-01). Wails v3 ships a new minor version every week, so the installed version may be newer. When the skill and the installed source disagree, the source wins.

## Why this needs care

Most Wails code online, and much of what you remember, is **v2** or **v3-alpha**. Both look plausible and both are wrong here: `wails.Run(&options.App{...})`, `runtime.EventsEmit(ctx, …)`, `Bind: []interface{}`, `OnStartup`, `wailsjs/go/main/App`, `app.NewWebviewWindow()`, `app.On(...)`. Before you write a Wails call you are not sure of, check it against the real API (next section). Checking takes seconds. A wrong guess compiles badly or misbehaves at runtime.

## Step 1: Pin the version and find the source of truth

```bash
grep wailsapp/wails go.mod                      # v3? which version?
go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3   # module dir in GOMODCACHE
wails3 version                                  # CLI version (should match go.mod)
```

- If go.mod has `github.com/wailsapp/wails/v2`, this skill does not apply. Say so and follow v2 conventions.
- If the CLI and go.mod versions differ a lot, mention it. Generated templates and Taskfiles follow the CLI version. Upgrade with `go install github.com/wailsapp/wails/v3/cmd/wails3@<version>` (or `wails3 update cli`).

Where to look, in priority order:

1. **The module source**, `$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)`. It is exact for this project and works offline.
   - `go doc github.com/wailsapp/wails/v3/pkg/application <Symbol>` gives signatures (`Options`, `WebviewWindowOptions`, `Window`, `EventManager`, …).
   - `examples/` holds about 60 runnable apps (systray-menu, frameless, dialogs, events, notifications, single-instance, streams, updater, …). Grep these first when you need a working pattern.
   - The JS runtime source is in `internal/runtime/desktop/@wailsio/runtime/src/`.
2. **Docs at the matching tag**, fetched raw: `https://raw.githubusercontent.com/wailsapp/wails/v<version>/docs/mpress/content/<path>.md`. Fall back to `master` if the tag 404s (for example on nightlies). Page paths are in `references/docs-index.md`. A few doc examples lag the code, so if a doc snippet does not compile, trust `go doc`.
3. Your own memory, last, and only for things that are not Wails-specific.

## Step 2: Know the project layout

`wails3 init -n myapp -t <template> -mod github.com/me/myapp` produces:

```
main.go            application.New(...), windows, app.Run()
greetservice.go    example service (exported methods become JS functions)
go.mod             module name; DEFAULT IS "changeme" unless -mod/-git is given
Taskfile.yml       build/package/run/dev → dispatches to build/<os>/Taskfile.yml
build/config.yml   app metadata (name, identifier, version) + dev_mode watcher config
build/{windows,darwin,linux,ios,android,docker}/   per-platform assets & Taskfiles
frontend/          Vite project; package.json pins @wailsio/runtime to the CLI version
frontend/bindings/ GENERATED — never hand-edit; regenerate instead
```

Built-in templates are `vanilla`, `react`, `vue` and `svelte`, each with a `-js` variant (TypeScript is the default), plus `ios`. List them with `wails3 init -l`. There are two init gotchas:

- `-t vanilla` is the default value, so it is **silently replaced** by `defaultTemplate` in `~/.config/wails/defaults.yaml` if one is set. The `Framework` line printed by init tells you what you actually got.
- Bindings live under `frontend/bindings/<module path>/`. A module named `changeme` gives `frontend/bindings/changeme/`. Renaming the module later moves the bindings, so pass `-mod` at init.

## Step 3: Use the development loop

| Goal | Command |
|---|---|
| Dev mode (watch Go + Vite HMR, auto-regenerates bindings) | `wails3 dev` (frontend dev server on port 9245 by default) |
| Regenerate bindings after changing service signatures/models | `wails3 generate bindings -clean=true -ts` (the template task also adds `-i` for TS interfaces) |
| Production build for host OS → `bin/` | `wails3 build` |
| Cross-build | `wails3 build GOOS=windows GOARCH=arm64` (see build reference for Docker needs) |
| Installers/packages | `wails3 package [GOOS=…]` |
| Any Taskfile task (no separate `task` install needed) | `wails3 task --list`, `wails3 task linux:create:deb` |
| Environment check | `wails3 doctor` |

To verify your work without launching a GUI:

- `go vet ./...` and `go build ./...`. The `//go:embed all:frontend/dist` line fails to compile if `frontend/dist` does not exist yet. Run `wails3 task common:build:frontend` first, or create the directory with a placeholder `index.html`.
- `wails3 generate bindings -clean=true -ts`. Its summary line (`N Services, M Methods, …`) confirms the service was picked up.
- In `frontend/`, run `npx tsc --noEmit` (TS templates) to check that the frontend matches the generated bindings.

## Golden rules (v3 vs. v2/alpha)

| Don't write (v2 / alpha) | Write (v3) |
|---|---|
| `wails.Run(&options.App{...})` | `app := application.New(application.Options{...})`, then create windows, then `app.Run()` |
| `Bind: []interface{}{&App{}}` | `Services: []application.Service{application.NewService(&MyService{})}` |
| `OnStartup: func(ctx)` / storing `ctx` for runtime calls | No `OnStartup` exists. Implement `ServiceStartup(ctx context.Context, opts application.ServiceOptions) error` on a service, or listen for `events.Common.ApplicationStarted` |
| `runtime.EventsEmit(ctx, "x", d)` / `runtime.EventsOn` | `app.Event.Emit("x", d)` / `app.Event.On("x", func(e *application.CustomEvent){ e.Data })` |
| `runtime.WindowSetTitle(ctx, …)` | `window.SetTitle(…)` on a `*WebviewWindow` / `application.Window` |
| `app.NewWebviewWindow()`, `app.NewWebviewWindowWithOptions()` | `app.Window.New()`, `app.Window.NewWithOptions(application.WebviewWindowOptions{...})` |
| `app.NewMenu()`, `app.SetMenu()` | `app.Menu.New()`, `app.Menu.Set(menu)` (+ `UseApplicationMenu: true` on windows for Win/Linux) |
| `app.NewSystemTray()` | `app.SystemTray.New()` |
| `application.OpenFileDialog()`, `app.InfoDialog()` | `app.Dialog.OpenFile()…PromptForSingleSelection()`, `app.Dialog.Info()…Show()` |
| `import … from '../wailsjs/go/main/App'` | `import { MyService } from '../bindings/<module>'` then `MyService.Method()` |
| `window.runtime.*` / `wailsjs/runtime` | `import { Events, Window, Dialogs, … } from '@wailsio/runtime'` |
| JS listener reads data directly: `Events.On('x', d => d)` | The callback gets a `WailsEvent`, so read `ev.data` (also `ev.name`, `ev.sender`) |

More rules that prevent subtle bugs:

- **Only exported methods on registered services** become callable, and only after you regenerate bindings. A `context.Context` first parameter is injected by Wails and hidden from JS. It is cancelled when the JS promise is `.cancel()`ed. Return `(T, error)`, and a non-nil error rejects the JS promise.
- **Services hold the app via `application.Get()`** or constructor injection (`NewMyService(app)` followed by `app.RegisterService(...)`). Don't stash a context for runtime calls.
- **Events crossing the bridge are JSON.** `Emit(name, a)` sends `a`. `Emit(name, a, b)` sends `[a, b]`. For typed events, call `application.RegisterEvent[T]("name")` in an `init()` func. Bindings then include a typed event API, and a mismatched payload type is rejected.
- **Windows/Linux quit when the last window closes** unless `Options.Windows/Linux.DisableQuitOnLastWindowClosed` is set. macOS does not quit unless `Mac.ApplicationShouldTerminateAfterLastWindowClosed: true`. For tray apps, keep the process alive and hide windows instead of closing them.
- **Cancel a close** with `win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent){ e.Cancel() })`. Hooks run synchronously and can cancel. `OnWindowEvent` listeners cannot.
- **Before `app.Run()`** you may create windows, menus, the tray and global shortcuts. Native dialogs need the app running, so trigger them from events, menu clicks or bound methods.
- **The frontend must load the runtime.** Import `@wailsio/runtime` somewhere (even just `import "@wailsio/runtime"`), or drag regions (`--wails-draggable: drag`), context menus and WML won't work.
- **Linux defaults to GTK4 + WebKitGTK 6.0.** `-tags gtk3` selects the legacy GTK3/WebKit2GTK-4.1 build. Android/iOS, Wake (`WAILS_EXP_USE_WAKE=1`), CGO-free `purego` builds and the in-app MCP server are **experimental**. Say so when you use them.

## Minimal correct skeleton

```go
package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

type TodoService struct{ items []string }

func (s *TodoService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error { return nil }
func (s *TodoService) Add(item string) ([]string, error) {
	s.items = append(s.items, item)
	application.Get().Event.Emit("todos:changed", s.items)
	return s.items, nil
}

func main() {
	app := application.New(application.Options{
		Name:     "Todo",
		Services: []application.Service{application.NewService(&TodoService{})},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "main", Title: "Todo", Width: 1000, Height: 700, URL: "/",
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { /* e.Cancel() to veto */ })
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

```ts
import { Events } from "@wailsio/runtime";
import { TodoService } from "../bindings/github.com/me/todo"; // = frontend/bindings/<module path>

const items = await TodoService.Add("milk");          // CancellablePromise<string[]>
Events.On("todos:changed", (ev) => render(ev.data));   // payload is ev.data
```

## Reference files (read the one you need)

| Topic | File |
|---|---|
| Services, lifecycle, bindings generation, type mapping, models, enums, errors, context/cancel, directives | `references/services-bindings.md` |
| Events (custom, typed, application, window), hooks, app lifecycle/quit behaviour | `references/events-lifecycle.md` |
| Windows (options, multi-window, frameless/drag), menus, context menus, system tray, key bindings | `references/windows-menus-tray.md` |
| Dialogs, clipboard, screens, environment, browser, notifications, global shortcuts, single instance, file drop, built-in services | `references/native-apis.md` |
| `@wailsio/runtime` API, Vite plugin, frontend patterns, no-bundler setup | `references/frontend-runtime.md` |
| Build/package/sign/cross-compile for Windows, macOS, Linux; prerequisites; Taskfile customisation | `references/build-distribute.md` |
| Streams, updater, server mode, custom protocols, file associations, raw messages, security, testing, mobile, experimental features | `references/advanced.md` |
| Map of every v3 docs page → raw URL path | `references/docs-index.md` |

When a task touches a topic, open its reference before you write code. For anything the reference does not cover, go to the module's `examples/` and then the docs page.
