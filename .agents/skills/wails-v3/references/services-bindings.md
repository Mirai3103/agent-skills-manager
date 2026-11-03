# Services & bindings

Docs: `features/bindings/{services,methods,models,enums,advanced,best-practices}.md`, `concepts/bridge.md`.
Examples in the module: `examples/binding`, `examples/services`, `examples/cancel-async`, `examples/cancel-chaining`.

## Contents
1. Defining and registering services
2. Service lifecycle
3. Generating bindings and what you get
4. Type mapping (verified against the v3.1.0 generator)
5. Errors
6. Context, calling window, cancellation
7. Directives and generator flags
8. Service HTTP routes

## 1. Defining and registering services

A service is any Go struct pointer. Its **exported methods** become callable from JS.

```go
type NotesService struct {
	mu    sync.Mutex
	notes []Note
}

func (s *NotesService) List() []Note            { ... }
func (s *NotesService) Save(n Note) (Note, error) { ... }
```

You can register a service in three ways:

```go
// 1. Up-front (most common)
app := application.New(application.Options{
	Services: []application.Service{
		application.NewService(&NotesService{}),
		application.NewServiceWithOptions(&FilesService{}, application.ServiceOptions{Name: "files"}),
	},
})

// 2. After New() — e.g. when the service needs the *App
app.RegisterService(application.NewService(NewWindowService(app)))

// 3. Services that need the app can also call application.Get() lazily inside methods.
```

Things to know:
- Register **at most one service per Go type**. If you need two configured instances of `*Store`, wrap them in distinct types (`type PrimaryStore struct{ *Store }`).
- `ServiceOptions.Name` only affects logs. The JS namespace is the Go type name (`NotesService`).
- Unexported methods, and methods on types you never registered, are not callable. `//wails:internal` and `//wails:ignore` only hide the generated wrapper. They are **not** access control, so enforce authorisation inside exported methods.
- Services are plain structs, so unit-test them directly. They need no Wails context.

## 2. Service lifecycle

Both hooks are optional interfaces, picked up automatically:

```go
func (s *NotesService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error {
	// open DB, load config, start goroutines bound to ctx
	return nil // non-nil aborts startup; app.Run() returns the wrapped error
}

func (s *NotesService) ServiceShutdown() error {
	// close resources; error goes to Options.ErrorHandler (logged by default)
	return nil
}

func (s *NotesService) ServiceName() string { return "notes" } // optional, logging only
```

- Startup runs in registration order: `Options.Services` first, then `RegisterService` calls. Shutdown runs in reverse order, **after** `Options.OnShutdown` / `app.OnShutdown(...)` hooks.
- `ctx` lives as long as the app and is cancelled right before shutdown. Background goroutines should `select` on `ctx.Done()`.
- If a startup fails, the services that already started still receive `ServiceShutdown`.

## 3. Generating bindings and what you get

```bash
wails3 generate bindings -clean=true -ts -i   # what the template's Taskfile runs
```

`wails3 dev` and `wails3 build` regenerate bindings for you (task `common:generate:bindings`). Run it manually after changing method signatures or models if you are not inside `wails3 dev`.

The output goes to `frontend/bindings/`, mirroring Go import paths:

```
frontend/bindings/
  <module path>/            e.g. changeme/  or  github.com/me/app/
    index.ts                export * as NotesService …; re-exports models/enums
    notesservice.ts         one file per service
    models.ts               structs + enums used by services
  events/                   typed custom events (from application.RegisterEvent)
  <other packages>/         models from imported packages get their own dirs
```

```ts
import { NotesService, Note } from "../bindings/github.com/me/app"; // via index
import { Save } from "../bindings/github.com/me/app/notesservice";   // or direct
const saved = await NotesService.Save(note);
```

Each wrapper returns `CancellablePromise<T>` from `@wailsio/runtime`. Calls go through `$Call.ByID(<hash>)`, or `$Call.ByName("pkg.Type.Method")` with `-names`.

**`-i` (interfaces) vs. classes.** The default template config (`useInterfaces: true`) passes `-i`, so models are TypeScript `interface`s and you build plain object literals. Without `-i`, models are classes with `new Note({...})` and `Note.createFrom(json)`. Check the generated `models.ts` before writing `new X()`.

## 4. Type mapping (v3.1.0)

| Go | TS |
|---|---|
| `string`, `bool`, all ints/floats | `string`, `boolean`, `number` |
| `[]T` | `T[] \| null`. **A nil slice arrives as `null`**, so return `[]T{}` if the UI expects an array |
| `[]byte` | `string` (base64) |
| `map[string]V` | `{ [_: string]: V }`. Non-string keys become a mapped type, not a JS `Map` |
| `*T` | `T \| null` (with `omitempty`: optional `field?: T \| null`) |
| struct | `interface` (with `-i`) or `class` |
| `time.Time` | `string` (RFC3339) by default; `-time-type Date` changes it |
| named string/int type with a `const` block | TS `enum` (includes a `$zero` member) |
| `any` | `any` |
| `error` as last return | not part of the TS signature; failures reject the promise |
| `context.Context` as first param | **not** part of the TS signature (injected) |
| `chan`, `func`, non-empty interfaces | unsupported. Pass IDs/handles instead |

JSON tags control field names (`json:"name"`). Unexported fields are not sent. Generic types work. Enums are not generated for generic types or types with custom marshalers.

## 5. Errors

```go
func (s *UserService) Get(id int) (*User, error) {
	if id <= 0 { return nil, fmt.Errorf("invalid id %d", id) }
	...
}
```

```ts
import { Call } from "@wailsio/runtime";
try {
	await UserService.Get(0);
} catch (err) {
	if (err instanceof Call.RuntimeError) { /* Go returned an error or panicked */ }
	// err.name: "RuntimeError" | "TypeError" (bad args) | "Error" (other)
	// err.message: Go error text; err.cause: JSON of the Go error if it marshals
}
```

To give the frontend structured errors, return a custom error type with JSON tags and read `err.cause.field`. `Options.MarshalError` or `ServiceOptions.MarshalError` customise the serialisation.

## 6. Context, calling window, cancellation

```go
func (s *JobService) Run(ctx context.Context, n int) (int, error) {
	if w, ok := ctx.Value(application.WindowKey).(application.Window); ok {
		w.SetTitle("Working…") // the window that made the call
	}
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			return i, ctx.Err() // JS called promise.cancel()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return n, nil
}
```

```ts
const p = JobService.Run(50);
cancelButton.onclick = () => p.cancel();   // cancels ctx on the Go side
```

Bound methods run on their own goroutine, so blocking work in them is fine. Protect shared state with a mutex. For long jobs, emit progress events (see `events-lifecycle.md`) or use Streams (`advanced.md`) rather than returning huge payloads.

## 7. Directives & generator flags

| Directive (Go comment) | Effect |
|---|---|
| `//wails:inject <code>` on a service type (`j*:` / `t*:` prefixes limit it to JS/TS) | inject code into that service's generated file |
| `//wails:include js/*.js` on the package doc comment | copy extra files into the bindings |
| `//wails:internal` | omit a type/method from the generated SDK (still callable) |
| `//wails:ignore` | skip a method in generation (still callable) |
| `//wails:id 42` | fixed method ID (keeps IDs stable across renames) |

Useful flags: `-ts`, `-i`, `-names`, `-b` (bundle the runtime, no npm package), `-noindex`, `-noevents`, `-time-type Date`, `-d <dir>`, `-dry`, `-v`, `-obfuscated` (for garble builds). Run `wails3 generate bindings -help` for the full list.

## 8. Service HTTP routes

A service that implements `http.Handler` can be mounted on the asset server:

```go
application.NewServiceWithOptions(&MediaService{}, application.ServiceOptions{Route: "/media"})
// frontend: <img src="/media/cover.jpg">  or fetch("/media/…")
```

This is handy for streaming large files or images into the WebView without base64. `application.AssetOptions.Middleware` wraps the whole asset server for cross-cutting concerns.
