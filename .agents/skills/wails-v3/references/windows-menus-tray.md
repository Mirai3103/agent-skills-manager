# Windows, menus, context menus, system tray, key bindings

Docs: `features/windows/{basics,options,multiple,frameless,events}.md`, `guides/customising-windows.md`, `features/menus/{application,context,systray,reference}.md`, `features/keyboard/shortcuts.md`.
Examples: `examples/window`, `examples/window-api`, `examples/frameless`, `examples/menu`, `examples/window-menu`, `examples/contextmenus`, `examples/systray-basic`, `examples/systray-menu`, `examples/systray-custom`, `examples/keybindings`.

## Contents
1. Creating windows and key options
2. The Window API
3. Multiple windows
4. Frameless windows and custom title bars
5. Application menu
6. Context menus
7. System tray
8. Key bindings (window-focused shortcuts)

## 1. Creating windows

```go
win := app.Window.NewWithOptions(application.WebviewWindowOptions{
	Name:   "main",          // lookup key for app.Window.GetByName and JS Window.Get
	Title:  "My App",
	Width:  1200, Height: 800,
	MinWidth: 800, MinHeight: 500,
	URL:    "/",             // path served by the asset server; or "https://…", or use HTML:
	BackgroundColour: application.NewRGB(27, 38, 54),
	InitialPosition:  application.WindowCentered, // or WindowXY with X/Y
	StartState:       application.WindowStateNormal, // Minimised | Maximised | Fullscreen
	Hidden:           false,  // create hidden, Show() later (avoids white flash while loading)
	Frameless:        false,
	AlwaysOnTop:      false,
	DisableResize:    false,
	EnableFileDrop:   false,  // OS file drops → WindowFilesDropped event
	DevToolsEnabled:  true,   // inspector (always available in dev builds)
	UseApplicationMenu: true, // Windows/Linux: show app.Menu menu in this window
	HideOnFocusLost:  false, HideOnEscape: false, // handy for popovers / tray windows
	KeyBindings: map[string]func(application.Window){"CmdOrCtrl+R": func(w application.Window){ w.Reload() }},
	Mac: application.MacWindow{
		TitleBar: application.MacTitleBarHiddenInset, // presets: MacTitleBarDefault, MacTitleBarHidden, MacTitleBarHiddenInset, MacTitleBarHiddenInsetUnified
		Backdrop: application.MacBackdropTranslucent,
		InvisibleTitleBarHeight: 50,
	},
	// BackdropType (Mica/Acrylic/Tabbed, Win11 22621+) only applies with BackgroundType: application.BackgroundTypeTranslucent
	Windows: application.WindowsWindow{ BackdropType: application.Mica, HiddenOnTaskbar: false },
	Linux:   application.LinuxWindow{ WebviewGpuPolicy: application.WebviewGpuPolicyOnDemand },
})
```

- `app.Window.New()` creates a window with default options. You can then chain setters: `app.Window.New().SetTitle("x").SetSize(800, 600)`.
- Windows can be created before `app.Run()` (they appear at startup) or at any time afterwards.
- Run `go doc github.com/wailsapp/wails/v3/pkg/application WebviewWindowOptions` for every field. Platform-only fields live in `Mac`, `Windows` and `Linux`.

## 2. The Window API

`app.Window.*` returns `application.Window` (an interface) or `*application.WebviewWindow`. Most setters return `Window`, so calls chain.

| Area | Methods |
|---|---|
| Visibility | `Show()`, `Hide()`, `Close()`, `Focus()`, `IsVisible()`, `IsFocused()` |
| Geometry | `SetSize(w,h)`, `Size()`, `SetPosition(x,y)`, `Position()`, `SetRelativePosition`, `Center()`, `Bounds()`, `SetBounds(rect)`, `SetMinSize`, `SetMaxSize`, `SetResizable` |
| State | `Maximise()`, `UnMaximise()`, `ToggleMaximise()`, `Minimise()`, `UnMinimise()`, `Restore()`, `Fullscreen()`, `UnFullscreen()`, `ToggleFullscreen()`, `IsMaximised()`, `IsMinimised()`, `IsFullscreen()` |
| Content | `SetURL(s)`, `SetHTML(s)`, `Reload()`, `ForceReload()`, `ExecJS(js)`, `OpenDevTools()`, `Print()`, `SetZoom/ZoomIn/ZoomOut/ZoomReset` |
| Look | `SetTitle`, `SetBackgroundColour`, `SetFrameless`, `SetAlwaysOnTop`, `SetMenu(menu)`, `ShowMenuBar/HideMenuBar/ToggleMenuBar`, `SetContentProtection`, `SetIgnoreMouseEvents` |
| Other | `Name()`, `ID()`, `GetScreen()`, `SetScreen()`, `EmitEvent(name, data...)`, `OnWindowEvent`, `RegisterHook`, `AttachModal(w)`, `SetEnabled`, `Flash`, `NativeWindow()` |

Manager: `app.Window.GetByName("main")` returns `(Window, bool)`. Also `GetByID(id)`, `GetAll()`, `Current()` (the focused window), `OnCreate(cb)`, `Remove(id)` and `RemoveByName(name)`.

All of these are safe to call from any goroutine, including bound methods and event handlers. Wails marshals them onto the UI thread. Only raw native calls through `NativeWindow()` need `application.InvokeSync(func(){…})`.

## 3. Multiple windows

```go
func (s *WindowService) OpenSettings() {
	app := application.Get()
	if w, ok := app.Window.GetByName("settings"); ok { w.Show().Focus(); return }
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "settings", Title: "Settings", Width: 500, Height: 400, URL: "/#/settings",
	})
}
```

- Every window loads from the same asset server, so route by URL path or hash (`/settings.html`, `/#/settings`). With Vite multi-page builds, add each HTML entry to `build.rollupOptions.input`.
- Use events for cross-window state. An event emitted from Go reaches all windows. `win.EmitEvent` targets one window.
- In JS, `Window` (default import) is the current window. `Window.Get("settings")` returns a handle to another window by name.
- Modal child window: `parent.AttachModal(child)`.

## 4. Frameless windows

```go
app.Window.NewWithOptions(application.WebviewWindowOptions{Frameless: true, URL: "/"})
```

```css
.titlebar { --wails-draggable: drag; }          /* drag the window by this area */
.titlebar button { --wails-draggable: no-drag; } /* keep buttons clickable */
```

```ts
import { Window } from "@wailsio/runtime";
minBtn.onclick = () => Window.Minimise();
maxBtn.onclick = () => Window.ToggleMaximise();
closeBtn.onclick = () => Window.Close();
```

- Dragging only works when `@wailsio/runtime` is loaded in the page.
- Edge resizing of a resizable frameless window is handled automatically by the runtime when the cursor is near the window border. (The docs also mention a `--wails-resize` CSS property. The v3.1.0 runtime source does not implement it, so don't rely on it.)
- macOS: you usually want `Mac.TitleBar: application.MacTitleBarHiddenInset` (traffic lights stay, content goes under the title bar) rather than full `Frameless`. Use `InvisibleTitleBarHeight` to make the top strip draggable.
- Windows: `Windows.DisableFramelessWindowDecorations` removes the shadow and rounded corners. Snap Layouts and native caption hit-testing are opt-in via `NonClientRegionSupport` / `WebView2CompositionHosting`. See `features/windows/frameless.md`.

## 5. Application menu

```go
menu := app.Menu.New()
if runtime.GOOS == "darwin" {
	menu.AddRole(application.AppMenu) // macOS: first menu is always the app menu
}
file := menu.AddSubmenu("File")
file.Add("Open…").SetAccelerator("CmdOrCtrl+O").OnClick(func(ctx *application.Context) { openFile() })
file.AddSeparator()
file.AddRole(application.Quit)
menu.AddRole(application.EditMenu)   // Undo/Redo/Cut/Copy/Paste — needed on macOS for ⌘C/⌘V in inputs
menu.AddRole(application.WindowMenu)

view := menu.AddSubmenu("View")
dark := view.AddCheckbox("Dark mode", false)
dark.OnClick(func(ctx *application.Context) { setDark(ctx.ClickedMenuItem().Checked()) })
view.AddRadio("Small", true); view.AddRadio("Large", false) // adjacent radios form a group

app.Menu.Set(menu)
// On Windows/Linux the menu shows in windows created with UseApplicationMenu: true,
// or per-window: win.SetMenu(menu) / WebviewWindowOptions.Windows.Menu / .Linux.Menu
```

- Roles: `AppMenu`, `FileMenu`, `EditMenu`, `ViewMenu`, `WindowMenu`, `HelpMenu`, `ServicesMenu`, `SpeechMenu`, plus item roles such as `Quit`, `About`, `CloseWindow`, `Reload`, `ToggleFullscreen`, `Minimise`, `Undo`, `Copy`, `Paste`, `SelectAll`, `OpenDevTools`, `ZoomIn` and `ZoomOut`.
- Item API: `SetLabel`, `SetEnabled`, `SetChecked`, `SetHidden`, `SetTooltip`, `SetAccelerator`, `SetBitmap`, `OnClick(func(*application.Context))`. **Call `menu.Update()`** after changing items so the native menu refreshes.
- Accelerators use the forms `"CmdOrCtrl+S"`, `"Shift+CmdOrCtrl+Z"`, `"Alt+F4"` and `"F11"`.
- Inside a click handler, `application.Get().Window.Current()` is the window the menu was used from.

## 6. Context menus

```go
cm := app.ContextMenu.New()
cm.Add("Rename").OnClick(func(ctx *application.Context) { rename(ctx.ContextMenuData()) })
cm.Add("Delete").OnClick(func(ctx *application.Context) { del(ctx.ContextMenuData()) })
app.ContextMenu.Add("file-item", cm)
```

```html
<li style="--custom-contextmenu: file-item; --custom-contextmenu-data: 42">report.pdf</li>
<!-- --default-contextmenu: auto | show | hide controls the WebView's native menu -->
```

`ctx.ContextMenuData()` returns the `--custom-contextmenu-data` string from the clicked element. Set it per element (inline style or a CSS variable) to identify what was clicked.

## 7. System tray

```go
tray := app.SystemTray.New()
if runtime.GOOS == "darwin" {
	tray.SetTemplateIcon(icons.SystrayMacTemplate) // github.com/wailsapp/wails/v3/pkg/icons
} else {
	tray.SetIcon(icons.SystrayLight); tray.SetDarkModeIcon(icons.SystrayDark)
}
tray.SetTooltip("My App") // SetLabel shows text next to the icon (macOS/Linux)

m := app.Menu.New()
m.Add("Show").OnClick(func(*application.Context) { win.Show().Focus() })
m.Add("Quit").OnClick(func(*application.Context) { app.Quit() })
tray.SetMenu(m)

// Popover-style window anchored to the icon; toggles on click, hides on focus loss
tray.AttachWindow(win).WindowOffset(5)
tray.OnClick(func() { … }); tray.OnRightClick(func() { tray.OpenMenu() })
```

A typical tray-only app is set up like this:
- `Mac: application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory}` (no Dock icon).
- `Windows: application.WindowsOptions{DisableQuitOnLastWindowClosed: true}`, and the same for `Linux`.
- The window is created with `Hidden: true, Frameless: true, AlwaysOnTop: true` (and `Windows.HiddenOnTaskbar: true`).
- A `WindowClosing` hook that hides instead of closing.

Linux tray support depends on the desktop environment (StatusNotifierItem / AppIndicator). GNOME needs the AppIndicator extension.

## 8. Key bindings

`app.KeyBinding.Add("CmdOrCtrl+K", func(w application.Window){…})`, `WebviewWindowOptions.KeyBindings` and `win.RegisterKeyBinding(...)` fire **only while a Wails window is focused**. For system-wide hotkeys that work while the app is unfocused, use `app.GlobalShortcut.Register` (see `native-apis.md`).
