# Events & lifecycle

Docs: `features/events/system.md`, `guides/events-reference.md`, `reference/events.md`, `concepts/lifecycle.md`.
Examples: `examples/events`, `examples/hide-window`, `examples/window`.

## Contents
1. Custom events (Go ⇄ JS, window ⇄ window)
2. Typed events with RegisterEvent
3. Application events (system/OS)
4. Window events and cancellable hooks
5. App lifecycle: startup, quit, shutdown
6. Patterns: confirm-before-close, hide-instead-of-close, background worker

## 1. Custom events

Go side (`app.Event` is `*application.EventManager`):

```go
app.Event.Emit("progress", 42)                 // data = 42
app.Event.Emit("pair", "a", "b")               // data = ["a","b"]  (variadic >1 → slice)
off := app.Event.On("ui:save", func(e *application.CustomEvent) {
	log.Println(e.Name, e.Data, e.Sender)        // Data is decoded JSON (map[string]any, []any, float64…)
})
defer off()                                    // On returns an unsubscribe func
app.Event.OnMultiple("x", cb, 3)               // fire at most 3 times
app.Event.Off("ui:save"); app.Event.Reset()
win.EmitEvent("only-this-window", data)        // target one window
```

JS side:

```ts
import { Events } from "@wailsio/runtime";
const off = Events.On("progress", (ev) => bar.value = ev.data); // ev: WailsEvent {name, data, sender}
Events.Once("ready", (ev) => …);
await Events.Emit("ui:save", { id: 1 });   // reaches Go listeners AND other windows' JS listeners
Events.Off("progress"); Events.OffAll(); off();
```

- Data crosses the bridge as JSON. On the Go side, an **unregistered** event's `e.Data` is generic decoded JSON (`map[string]any`, `float64`, …), so either use typed events (below) or re-marshal it into a struct.
- An event emitted from Go reaches every window. To communicate between windows, emit from one window's JS and listen in the other's.
- Prefer events over polling for pushing state from Go. Don't stream huge payloads through events; use Streams or a service HTTP route instead.

## 2. Typed events

```go
type ProgressData struct { Done, Total int }

func init() {
	application.RegisterEvent[ProgressData]("job:progress") // constant name, called from init()
	application.RegisterEvent[string]("time")
}
```

- The binding generator emits typed declarations under `frontend/bindings/events`, and the template's Vite plugin (`wails("./bindings")`) wires them up, so `Events.On("job:progress", ev => ev.data.Done)` is typed in TS.
- When you emit a registered event from Go, Wails checks the payload type exactly. A mismatch is reported to the error handler and the event is cancelled.
- Registering the same name twice panics.

## 3. Application events

These come from the platform. Use `events.Common.*` for cross-platform code. `events.Mac.*`, `events.Windows.*` and `events.Linux.*` give platform-specific ones.

```go
import "github.com/wailsapp/wails/v3/pkg/events"

app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(e *application.ApplicationEvent) {
	// app is up: safe to show dialogs, start work that needs windows
})
app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(e *application.ApplicationEvent) {
	dark := e.Context().IsDarkMode()
	_ = dark
})
app.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(e *application.ApplicationEvent) {
	path := e.Context().Filename()
	_ = path
})
// Synchronous, cancellable variant (same signature): app.Event.RegisterApplicationEventHook(eventType, cb)
```

To veto quitting, use `Options.ShouldQuit` (section 5), not an event hook.

`events.Common` application events in v3.1.0: `ApplicationStarted`, `ApplicationOpenedWithFile`, `ApplicationLaunchedWithUrl`, `ThemeChanged`, `SystemDidWake`, `SystemWillSleep`, `BatteryChanged`, `NetworkChanged`, `ScreenLocked`, `ScreenUnlocked`, `LowMemory`.

`ApplicationEventContext` offers `Filename()`, `OpenedFiles()`, `URL()`, `IsDarkMode()`, `HasVisibleWindows()` and `Data()`.

On the JS side, built-in event names are in `Events.Types`, e.g. `Events.On(Events.Types.Common.ThemeChanged, …)`.

## 4. Window events and hooks

```go
win.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) { … })   // listener
win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {      // hook: sync, can cancel
	if hasUnsaved { e.Cancel() }
})
win.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) {
	files := e.Context().DroppedFiles()   // needs WebviewWindowOptions.EnableFileDrop: true
	_ = files
})
```

`events.Common` window events: `WindowClosing`, `WindowFocus`, `WindowLostFocus`, `WindowShow`, `WindowHide`, `WindowMaximise`, `WindowUnMaximise`, `WindowMinimise`, `WindowUnMinimise`, `WindowRestore`, `WindowFullscreen`, `WindowUnFullscreen`, `WindowDidMove`, `WindowDidResize`, `WindowDPIChanged`, `WindowFilesDropped`, `WindowRuntimeReady`, `WindowToggleFrameless`, `WindowZoom`, `WindowZoomIn`, `WindowZoomOut`, `WindowZoomReset`.

- Hooks run before listeners and can call `e.Cancel()`. Listeners are notifications only.
- `WindowRuntimeReady` fires when the page's JS runtime has loaded. Emit initial data then, not right after `NewWithOptions`.
- `app.Window.OnCreate(func(w application.Window){…})` fires for every new window.

## 5. Lifecycle

```
application.New(opts)   → services constructed, nothing native yet
create windows/menus/tray (allowed before Run)
app.Run()               → ServiceStartup (in order) → ApplicationStarted → event loop (blocks)
quit requested          → Options.ShouldQuit() (return false to veto)
                        → Options.OnShutdown + app.OnShutdown(f) hooks
                        → ServiceShutdown (reverse order) → Options.PostShutdown → Run returns
```

- **There is no `OnStartup` option.** Do startup work in `ServiceStartup`, in an `ApplicationStarted` listener, or before `app.Run()`.
- `app.Quit()` quits programmatically. `Options.ShouldQuit func() bool` is consulted when the platform asks the app to quit (Cmd+Q / app-menu Quit on macOS, the quit path on Windows/Linux). Return false to keep running. To guard a single window, use a `WindowClosing` hook.
- Last-window behaviour: Windows and Linux quit when the last window closes unless `Windows.DisableQuitOnLastWindowClosed` / `Linux.DisableQuitOnLastWindowClosed` is set. macOS keeps running unless `Mac.ApplicationShouldTerminateAfterLastWindowClosed` is true.
- `Options.PanicHandler func(*application.PanicDetails)` catches panics in Wails-managed goroutines. A panic in a bound method rejects the JS promise instead of crashing the app.
- `Options.SingleInstance`: see `native-apis.md`.

## 6. Patterns

**Confirm before closing** (dialogs need the app running, which is the case here):

```go
win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
	if !dirty { return }
	e.Cancel()
	d := app.Dialog.Question().SetTitle("Unsaved changes").SetMessage("Quit without saving?")
	quit := d.AddButton("Quit"); quit.OnClick(func() { dirty = false; win.Close() })
	cancel := d.AddButton("Cancel")
	d.SetDefaultButton(cancel); d.SetCancelButton(cancel)
	d.AttachToWindow(win).Show()
})
```

**Hide instead of close** (tray apps):

```go
win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
	e.Cancel()
	win.Hide()
})
```

Pair this with `DisableQuitOnLastWindowClosed` on Windows/Linux and a tray "Quit" item that calls `app.Quit()`. (A hidden window does not count as closed, but setting the option avoids surprises.)

**Background worker bound to the app lifetime:**

```go
func (s *Clock) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	go func() {
		t := time.NewTicker(time.Second); defer t.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case now := <-t.C: application.Get().Event.Emit("time", now.Format(time.RFC1123))
			}
		}
	}()
	return nil
}
```
