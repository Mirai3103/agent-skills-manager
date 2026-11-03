# Native APIs: dialogs, clipboard, screens, environment, browser, notifications, shortcuts, single instance, file drop, built-in services

Docs: `features/dialogs/*.md`, `features/clipboard/basics.md`, `features/screens/info.md`, `features/environment/info.md`, `features/browser/integration.md`, `features/notifications/overview.md`, `features/keyboard/global-shortcuts.md`, `features/autostart/basics.md`, `features/platform/dock.md`, `features/drag-and-drop/files.md`, `guides/single-instance.md`.
Examples: `examples/dialogs`, `examples/clipboard`, `examples/screen`, `examples/environment`, `examples/notifications`, `examples/global-shortcuts`, `examples/autostart`, `examples/dock`, `examples/badge`, `examples/drag-n-drop`, `examples/single-instance`.

## Dialogs (Go)

Native dialogs only show once the app is running. A dialog created before `app.Run()` is dropped with a warning. Trigger dialogs from bound methods, menu clicks or event handlers.

```go
// Message dialogs: Info / Warning / Error / Question
app.Dialog.Info().SetTitle("Saved").SetMessage("All changes saved.").Show()

d := app.Dialog.Question().SetTitle("Delete?").SetMessage("This cannot be undone.")
del := d.AddButton("Delete"); del.OnClick(func() { doDelete() })
cancel := d.AddButton("Cancel")
d.SetDefaultButton(cancel).SetCancelButton(cancel)
d.AttachToWindow(win).Show()   // AttachToWindow → sheet on macOS / owned dialog elsewhere
```

`Show()` returns no result. Handle the user's choice in the button `OnClick` callbacks. If you need a result returned to JS, call the JS `Dialogs.Question()` from the frontend instead (it resolves with the button label), or bridge a callback to a channel inside a bound method.

```go
// Open file(s) / folder
path, err := app.Dialog.OpenFile().
	SetTitle("Open image").
	AddFilter("Images", "*.png;*.jpg;*.jpeg").
	AddFilter("All files", "*.*").
	AttachToWindow(win).
	PromptForSingleSelection()           // "" + nil error when the user cancels
paths, err := app.Dialog.OpenFile().PromptForMultipleSelection()
dir, err := app.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).PromptForSingleSelection()

// Save
dest, err := app.Dialog.SaveFile().SetFilename("report.csv").AddFilter("CSV", "*.csv").PromptForSingleSelection()
```

There is also `OpenFileWithOptions(&application.OpenFileDialogOptions{...})` / `SaveFileWithOptions(...)` for struct-style configuration. `SetDirectory`, `ShowHiddenFiles` and `CanCreateDirectories` are available too. **Always handle the cancel case** (empty string / empty slice).

Dialogs from JS are covered in `frontend-runtime.md`. They are often simpler when the UI needs the answer.

## Clipboard

```go
ok := app.Clipboard.SetText("hello")
text, ok := app.Clipboard.Text()
```
JS: `Clipboard.SetText(s)` and `await Clipboard.Text()`. Only text is supported.

## Screens

```go
for _, s := range app.Screen.GetAll() { fmt.Println(s.Name, s.Size, s.ScaleFactor, s.IsPrimary, s.WorkArea) }
primary := app.Screen.GetPrimary()
scr, _ := win.GetScreen()
```
Coordinates are DIPs (logical pixels). Helpers such as `app.Screen.DipToPhysicalRect` convert to physical pixels. JS: `Screens.GetAll()`, `GetPrimary()`, `GetCurrent()`.

## Environment & browser

```go
info := app.Env.Info()          // OS, Arch, Debug, OSInfo, PlatformInfo
dark := app.Env.IsDarkMode()
accent := app.Env.GetAccentColor()
_ = app.Env.OpenFileManager("/path/to/file", true) // reveal in Finder/Explorer/Files
_ = app.Browser.OpenURL("https://example.com")     // default browser
_ = app.Browser.OpenFile("/path/doc.pdf")          // default app for the file
```
Plain `<a href="https://…">` links inside the WebView do not open the system browser by themselves. Use `Browser.OpenURL` (JS) or a WML attribute (`data-wml-openURL`).

For branching on the platform in shared code, use `application.System.IsDesktop()`, `IsMobile()`, `IsServer()` or `IsPlatform(application.PlatformLinux)`, and use build tags only for code that cannot compile elsewhere.

## Notifications (built-in service)

```go
import "github.com/wailsapp/wails/v3/pkg/services/notifications"

notifier := notifications.New()
app := application.New(application.Options{Services: []application.Service{application.NewService(notifier)}})

// later (app running):
if ok, _ := notifier.CheckNotificationAuthorization(); !ok {   // macOS asks the user; Win/Linux always true
	_, _ = notifier.RequestNotificationAuthorization()
}
_ = notifier.SendNotification(notifications.NotificationOptions{ID: "sync-1", Title: "Sync done", Body: "42 files"})
notifier.OnNotificationResponse(func(r notifications.NotificationResult) { /* r.Response.ActionIdentifier … */ })
```
Action buttons are added by calling `RegisterNotificationCategory(...)` and then `SendNotificationWithActions` with a `CategoryID`. On macOS, notifications require a **signed .app bundle** with a bundle identifier, so they will not work from a plain `go run` binary.

## Global shortcuts (system-wide)

```go
if err := app.GlobalShortcut.Register("CmdOrCtrl+Shift+Space", func() { win.Show().Focus() }); err != nil {
	app.Logger.Error("shortcut", "err", err)  // e.g. already grabbed by another app (X11)
}
app.GlobalShortcut.Unregister("CmdOrCtrl+Shift+Space")
```
You can register shortcuts before `Run` (binding is deferred). The callback runs on its own goroutine. On Linux/Wayland the compositor (via the XDG portal) decides the final keys and may prompt the user, so the actual combination can differ. For shortcuts that fire only while a window is focused, use `app.KeyBinding`.

## Autostart, dock/badge

```go
_ = app.Autostart.Enable()               // takes effect at next login
on, _ := app.Autostart.IsEnabled()
_ = app.Autostart.Disable()
```
Dock icon visibility and badges come from the `pkg/services/dock` service (macOS dock, Windows taskbar badge). See `features/platform/dock.md`.

## OS file drop

```go
win := app.Window.NewWithOptions(application.WebviewWindowOptions{EnableFileDrop: true, URL: "/"})
win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
	files := e.Context().DroppedFiles()
	details := e.Context().DropTargetDetails() // which element (id/classes) received the drop, if any
	_, _ = files, details
})
```
To mark drop zones in HTML, add the `data-file-drop-target` attribute. The runtime adds the class `file-drop-target-active` while files hover over the zone, and `DropTargetDetails()` tells you which zone received the drop. The WebView never receives real file paths through HTML5 `drop` events. Paths come only through this Go event. For in-page drag and drop between elements, use ordinary HTML5 DnD.

## Single instance

```go
SingleInstance: &application.SingleInstanceOptions{
	UniqueID: "com.example.myapp",
	OnSecondInstanceLaunch: func(d application.SecondInstanceData) {
		if w, ok := app.Window.GetByName("main"); ok { w.Show().Focus() }
		handleArgs(d.Args, d.WorkingDir) // treat as untrusted input
	},
	// EncryptionKey: [32]byte{…}  // AES-256-GCM between instances; recommended if args matter
},
```
The second process exits (with `ExitCode`) after it forwards its args. Combine this with file associations and custom URL schemes (`advanced.md`) so that "open with" requests reach the running instance.

## Other built-in services (`pkg/services/...`)

| Package | Use |
|---|---|
| `kvstore` | `kvstore.NewWithConfig(&kvstore.Config{Filename: "settings.json", AutoSave: true})`: tiny persistent key/value store with `Get/Set/Delete/Save/Load` |
| `sqlite` | `sqlite.NewWithConfig(&sqlite.Config{...})`: `Open/Execute/Query/Prepare`, callable from JS through bindings |
| `fileserver` | serve a directory on a service route |
| `log` | logging service exposed to the frontend |
| `dock` | dock/taskbar visibility and badges |
| `notifications` | see above |

Register any of them with `application.NewService(x)`, or `NewServiceWithOptions` with a `Route` for fileserver. Then regenerate bindings so the frontend can call them. For app data paths, `application.Path(application.PathConfigHome)` / `Paths(...)` return per-OS standard dirs (check `go doc … PathType` for the selectors).
