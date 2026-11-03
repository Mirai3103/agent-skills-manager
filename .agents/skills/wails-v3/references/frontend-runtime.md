# Frontend: `@wailsio/runtime` and frontend patterns

Docs: `reference/frontend-runtime.md`, `guides/dev/frontend-frameworks.md`, `guides/routing.md`, `guides/advanced/wml.md`.
Source of truth: `<module dir>/internal/runtime/desktop/@wailsio/runtime/src/*.ts`.

## Setup facts

- `frontend/package.json` pins `@wailsio/runtime` to the **same version as the CLI/Go module** (for example `"3.1.0"`). After upgrading the Go module, bump this too (`npm i @wailsio/runtime@<version>`). A mismatched runtime can silently break calls or events.
- The template's `vite.config.ts` contains `wails("./bindings")` from `@wailsio/runtime/plugins/vite` (typed events plus HMR of bindings) and a dev server on `127.0.0.1:${WAILS_VITE_PORT || 9245}` with `strictPort`. Keep these when you replace the template's framework or config.
- The runtime must be imported somewhere for drag regions, context menus, file-drop targets and WML to work. `import "@wailsio/runtime"` is enough.
- Without a bundler, run `wails3 generate runtime` and load `<script type="module" src="./runtime.js">`. That exposes `window.wails.*`. Alternatively, bindings generated with `-b` import the bundled runtime.
- Any framework works (React/Vue/Svelte templates exist; Solid, Angular, Lit and others work through a plain Vite project, see `guides/dev/frontend-frameworks.md`). Only two things matter: the build must output `frontend/dist` (embedded by `//go:embed all:frontend/dist`), and the dev server must run on the port that `wails3 dev` passes in.

## API (named exports)

```ts
import { Events, Window, Dialogs, Clipboard, Application, Browser, Screens, System, Call, Flags, WML } from "@wailsio/runtime";
```

| Export | Key functions |
|---|---|
| `Events` | `On(name, cb) → off()`, `Once`, `OnMultiple(name, cb, n)`, `Off(...names)`, `OffAll()`, `Emit(name, data) → Promise<boolean>`; `Events.Types.Common.*` / `.Mac` / `.Windows` / `.Linux` for built-in event names. Callback arg is `WailsEvent {name, data, sender}` |
| `Window` | default = **current window**; `Window.Get("name")` → another window. Methods return Promises: `SetTitle`, `SetSize`, `Size`, `SetPosition`, `Position`, `Center`, `Show`, `Hide`, `Close`, `Minimise`, `Maximise`, `ToggleMaximise`, `UnMaximise`, `Fullscreen`, `UnFullscreen`, `ToggleFullscreen`, `IsMaximised`, `IsFullscreen`, `SetAlwaysOnTop`, `SetResizable`, `SetMinSize`, `SetMaxSize`, `SetBackgroundColour(r,g,b,a)`, `SetZoom`, `ZoomIn/Out/Reset`, `Reload`, `ForceReload`, `OpenDevTools`, `Print`, `Name`, `GetScreen`, `SnapAssist` |
| `Dialogs` | `Info/Warning/Error/Question({Title, Message, Buttons:[{Label, IsDefault, IsCancel}], Detached})` → `Promise<string>` (clicked label); `OpenFile({Title, Filters:[{DisplayName, Pattern}], CanChooseDirectories, CanChooseFiles, AllowsMultipleSelection, Directory, ButtonText, ...})` → `string` or `string[]` when `AllowsMultipleSelection: true`; `SaveFile({Filename, Filters, ...})` → `string` (`""` on cancel) |
| `Clipboard` | `SetText(s)`, `Text()` |
| `Application` | `Show()`, `Hide()`, `Quit()` |
| `Browser` | `OpenURL(url)`: opens the system browser |
| `Screens` | `GetAll()`, `GetPrimary()`, `GetCurrent()` |
| `System` | `IsDarkMode()`, `Environment()`, `IsWindows/IsMac/IsLinux/IsDesktop/IsMobile/IsIOS/IsAndroid()`, `IsDebug()`, `Capabilities()` |
| `Call` | `ByID(id, ...args)`, `ByName("pkg.Service.Method", ...args)`; error classes `Call.RuntimeError`; returns `CancellablePromise` |
| `Flags` | `GetFlag(key)`: reads values passed in Go `Options.Flags` |
| `WML` | `Enable()` / `Reload()`: activates `wml-*` / `data-wml-*` attributes (call `Reload()` after inserting new WML elements dynamically) |
| `Stream`, `JSONStream`, `WailsSocket` | byte/JSON streams, see `advanced.md` |
| `Updater` | front end of the self-updater, see `advanced.md` |

## Calling Go

```ts
import { NotesService } from "../bindings/github.com/me/app";
import type { Note } from "../bindings/github.com/me/app";   // interfaces mode (-i)

const notes = (await NotesService.List()) ?? [];   // slices can be null
const p = NotesService.Search("q"); p.cancel();     // CancellablePromise
```

- Never hand-edit `frontend/bindings/`. If a call signature is wrong, fix the Go code and regenerate.
- Bindings are ESM. Import from the package index (`…/<module>`), or the service file (`…/<module>/notesservice`) for tree-shaking.
- Do not `fetch()` Go functions over HTTP. Use bindings. Use `fetch` only for service `Route`s or asset URLs.

## CSS hooks the runtime reads

| Property / attribute | Effect |
|---|---|
| `--wails-draggable: drag` / `no-drag` | drag window by this element (frameless / hidden title bar) |
| `--custom-contextmenu: <id>` + `--custom-contextmenu-data: <string>` | show the Go context menu registered under `<id>` |
| `--default-contextmenu: show` / `hide` | force or suppress the WebView's own context menu (default: shown for editable/selected text) |
| `data-file-drop-target` attribute; class `file-drop-target-active` | OS file-drop zones (needs `EnableFileDrop`) |

## WML (declarative, no JS)

```html
<button wml-event="refresh">Refresh</button>                  <!-- emits custom event "refresh" -->
<button wml-window="Close" wml-confirm="Close window?">×</button>  <!-- calls Window method -->
<a wml-openurl="https://wails.io">Docs</a>                      <!-- opens system browser -->
<div wml-event="hovered" wml-trigger="mouseover">…</div>
```
Every attribute also has a `data-` prefixed form. Call `WML.Enable()` once at startup.

## Routing and multiple pages

- Client-side routers work. Prefer **hash routing** (`/#/settings`) or make sure unknown paths fall back to `index.html`. The asset server serves files from `frontend/dist`, and a history-mode deep link like `/settings` on a fresh window needs that fallback (see `guides/routing.md`).
- A second window can load a different HTML entry (`URL: "/settings.html"`). Add it to Vite's `build.rollupOptions.input`.

## Dev vs production

- In `wails3 dev`, the WebView loads from the Vite dev server (HMR). Go changes trigger a rebuild and restart, and bindings regenerate.
- In production, assets are embedded and served from memory under the internal `wails://` / `http://wails.localhost` origin. Use relative or absolute-path URLs (`/assets/x.png`), never `localhost:5173`.
