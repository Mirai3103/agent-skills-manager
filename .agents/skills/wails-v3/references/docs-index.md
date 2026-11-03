# Wails v3 docs index (English pages, v3.1.0)

Fetch a page raw (Markdown, front-matter included):

```
https://raw.githubusercontent.com/wailsapp/wails/v<VERSION>/docs/mpress/content/<path>
```

- `<VERSION>` comes from the project's go.mod, e.g. `v3.1.0`. If that tag 404s (nightlies, or pages added later), use `master`.
- Rendered site: `https://v3.wails.io/<path without .md>/`.
- Pages use M-Press directives (`@note{…} … @end`, `@tabs`, `@details`, `@filetree`); read through them as plain Markdown.
- Examples are not in the docs tree. They ship in the Go module: `$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v3)/examples/<name>`, or on GitHub at `https://github.com/wailsapp/wails/tree/v<VERSION>/v3/examples/<name>`.


## concepts

- `concepts/architecture.md`: Understanding the Wails architecture and how it achieves native performance
- `concepts/bridge.md`: Deep dive into how Wails enables direct communication between Go and JavaScript
- `concepts/build-system.md`: Understanding how Wails builds and packages your application
- `concepts/index.md`: section index
- `concepts/lifecycle.md`: Understanding the Wails application lifecycle from startup to shutdown
- `concepts/manager-api.md`: Organized API structure with focused manager interfaces

## experimental

- `experimental/index.md`: A home for in-progress experiments in Wails v3 — what they are, why they exist, and how to give feedback.
- `experimental/wake.md`: Create your first Wake project, follow the project creation wizard, build and run it, and migrate an existing Taskfile project.
- `experimental/yaml-builds.md`: Configure and run the experimental Wails v3 native build system with one wails.yaml file.

## (root)

- `faq.md`: Answers to the most common questions about building applications with Wails v3

## features/autostart

- `features/autostart/basics.md`: Register your application to launch at user login on macOS, Windows, and Linux

## features/bindings

- `features/bindings/advanced.md`: Advanced binding techniques including directives, code injection, and custom IDs
- `features/bindings/best-practices.md`: Design patterns and best practices for Go-JavaScript bindings
- `features/bindings/enums.md`: Automatic enum generation from Go constants
- `features/bindings/methods.md`: Call Go methods from JavaScript with type safety
- `features/bindings/models.md`: Bind complex data structures between Go and JavaScript
- `features/bindings/services.md`: Build modular, reusable application components with services

## features/browser

- `features/browser/integration.md`: Open URLs and files in the user's default web browser

## features/clipboard

- `features/clipboard/basics.md`: Copy and paste text with the system clipboard

## features/dialogs

- `features/dialogs/custom.md`: Create custom dialogs in your Wails applications
- `features/dialogs/file.md`: Open, save, and folder selection dialogs
- `features/dialogs/message.md`: Display information, warnings, errors, and questions
- `features/dialogs/overview.md`: Display native system dialogs in your application

## features/drag-and-drop

- `features/drag-and-drop/files.md`: Accept files dragged from the operating system into your application
- `features/drag-and-drop/html.md`: Drag and drop elements within your application

## features/environment

- `features/environment/info.md`: Access system environment information in your Wails application

## features/events

- `features/events/application.md`: Application-level events emitted by Wails v3
- `features/events/custom.md`: Emitting and listening to your own events in Wails v3
- `features/events/system.md`: Communicate between components with the event system
- `features/events/window.md`: Window-level events emitted by Wails v3

## features

- `features/index.md`: section index

## features/keyboard

- `features/keyboard/global-shortcuts.md`: Register system-wide keyboard shortcuts that fire even when your application is not focused
- `features/keyboard/shortcuts.md`: Register global keyboard shortcuts for quick access to functionality

## features/menus

- `features/menus/application.md`: Create native menu bars for your desktop application
- `features/menus/context.md`: Create right-click context menus for your application
- `features/menus/reference.md`: Complete reference for menu item types, properties, and methods
- `features/menus/systray.md`: Add system tray (notification area) integration to your application

## features/notifications

- `features/notifications/overview.md`: Display native system notifications with action buttons and text input

## features/platform

- `features/platform/dock.md`: Manage dock icon visibility and display badges on macOS and Windows

## features/screens

- `features/screens/info.md`: Get information about displays and monitors

## features/windows

- `features/windows/basics.md`: Creating and managing application windows in Wails
- `features/windows/events.md`: Handle window lifecycle and state change events
- `features/windows/frameless.md`: Create custom window chrome with frameless windows
- `features/windows/multiple.md`: Patterns and best practices for multi-window applications
- `features/windows/notch-windows.md`: Create native macOS windows attached to the camera housing.
- `features/windows/options.md`: Window options, defaults and platform constraints
- `features/windows/permissions.md`: Control camera, microphone, geolocation and other capability requests from web content

## getting-started

- `getting-started/index.md`: section index
- `getting-started/installation.md`: Install Wails and set up your development environment
- `getting-started/setup.md`: section index
- `getting-started/your-first-app.md`: Create your first Wails desktop application step by step

## guides/advanced

- `guides/advanced/custom-templates.md`: How to generate, customise, and host your own Wails v3 project templates
- `guides/advanced/streams-internals.md`: How the stream transport works, why it is built this way, what the buffer constants mean, and what is not finished
- `guides/advanced/wml.md`: A markup language for describing Wails v3 user interfaces

## guides

- `guides/architecture.md`: Design patterns for Wails applications

## guides/build

- `guides/build/building.md`: Build and package your Wails application
- `guides/build/cross-platform.md`: Build for multiple platforms from a single machine
- `guides/build/customization.md`: Customize your build process using Task and Taskfile.yml
- `guides/build/linux.md`: Package your Wails application for Linux distribution
- `guides/build/macos.md`: Package your Wails application for macOS distribution
- `guides/build/msix.md`: Packaging your Wails v3 application as an MSIX package
- `guides/build/obfuscation.md`: Build your Wails application with Garble to protect your source code from reverse engineering
- `guides/build/private-macos-apis.md`: Every Wails feature and option that depends on private macOS APIs, with opt-in commands and public-build fallbacks.
- `guides/build/signing.md`: Guide for signing your Wails applications on all platforms
- `guides/build/windows.md`: Package your Wails application for Windows distribution

## guides

- `guides/cli.md`: Look up flags and usage examples for individual Wails CLI commands
- `guides/customising-windows.md`: Customize window appearance and behavior in your Wails applications
- `guides/custom-transport.md`: Learn how to create and customize your own Wails v3 custom IPC transport layer

## guides/dev

- `guides/dev/debugging.md`: Investigating issues and profiling app performance
- `guides/dev/frontend-frameworks.md`: How to use a framework that doesn't have a built-in template by dropping your own Vite project into the frontend directory
- `guides/dev/project-structure.md`: Understand the file layout for a typical Wails project
- `guides/dev/workflow.md`: A recommended workflow for developing Wails v3 applications

## guides/distribution

- `guides/distribution/custom-protocols.md`: Register custom URL schemes to launch your application from links

## guides

- `guides/e2e-testing.md`: Test complete user workflows
- `guides/events-reference.md`: A practical guide to using events in Wails v3 for application communication and lifecycle management
- `guides/file-associations.md`: Configure file associations for your Wails application
- `guides/gin-routing.md`: A comprehensive guide to integrating Gin web framework with Wails v3 applications
- `guides/gin-services.md`: A guide to integrating the Gin web framework with Wails v3 Services
- `guides/index.md`: section index
- `guides/installers.md`: Package your application for distribution
- `guides/mcp-service.md`: Let LLM agents test and control your app via the Model Context Protocol
- `guides/menus.md`: A guide to creating and customising menus in Wails v3

## guides/mobile

- `guides/mobile/android.md`: Build and run Wails applications on Android — toolchain setup, emulator, APK signing, Play Store packaging, and API reference
- `guides/mobile/first-mobile-app.md`: Run your Wails app on iOS Simulator or Android Emulator in minutes
- `guides/mobile/index.md`: Build iOS and Android apps from the same Go codebase as your desktop app
- `guides/mobile/ios.md`: Build and run Wails applications on iOS — setup, simulator, device builds, configuration, and native features
- `guides/mobile/mobile-api.md`: The cross-platform application.Mobile manager — one build-guarded entry point for the native mobile capabilities shared by iOS and Android

## guides

- `guides/panic-handling.md`: How to handle panics in your Wails application

## guides/patterns

- `guides/patterns/database.md`: Integrating SQLite, PostgreSQL, and other databases with Wails v3
- `guides/patterns/rest-api.md`: Exposing RESTful APIs from your Wails v3 application

## guides

- `guides/performance.md`: Optimise your Wails application for maximum performance
- `guides/raw-messages.md`: Implement custom frontend-to-backend communication for performance-critical applications
- `guides/routing.md`: Using frontend routing in your Wails application
- `guides/security.md`: Secure your Wails application
- `guides/server-build.md`: Run Wails applications as HTTP servers without a native GUI window
- `guides/single-instance.md`: Limiting your app to a single running instance
- `guides/streams-from-websockets.md`: Step-by-step conversion of an existing WebSocket implementation to Wails streams, including the differences that break silently
- `guides/streams.md`: Bidirectional byte streams between Go and JavaScript, with the WebSocket programming model and no listening socket
- `guides/testing.md`: Test your Wails application
- `guides/updater-github-release-assets.md`: How the Wails updater selects application artifacts from GitHub Releases and avoids installer packages.
- `guides/updater.md`: In-app self-updates for Wails v3 — pluggable providers, cryptographic verification, atomic swap, and a default UI you can theme or replace.
- `guides/windows-uac.md`: Configure User Account Control (UAC) for your Windows Wails application

## (root)

- `index.md`: Native desktop applications using Go and web technologies

## migration

- `migration/from-electron.md`: Migrating an Electron application to Wails v3
- `migration/index.md`: section index
- `migration/v2-to-v3.md`: Complete guide to migrating your Wails v2 application to v3

## quick-start

- `quick-start/first-app.md`: Build a working Wails application in 10 minutes
- `quick-start/index.md`: section index
- `quick-start/installation.md`: Get Wails installed and ready to build applications
- `quick-start/next-steps.md`: Where to go after building your first app
- `quick-start/why-wails.md`: Understand why Wails is the right choice for your desktop application

## reference

- `reference/application.md`: Complete reference for the Application API
- `reference/cli.md`: Find Wails CLI commands by purpose: projects, builds, signing, tools and services
- `reference/dialogs.md`: Complete reference for native dialog APIs
- `reference/events.md`: Complete reference for the Events API
- `reference/frontend-runtime.md`: The Wails JavaScript runtime package for frontend integration
- `reference/index.md`: section index
- `reference/menu.md`: Complete reference for the Menu API
- `reference/overview.md`: Look up Wails types, methods, options and CLI commands
- `reference/update-manifest.md`: The open JSON protocol Wails apps use to discover and verify self-updates, servable from any static file host or dynamic update server.
- `reference/wails-yaml-fields.md`: Generated field, type, default and applicability reference for the Wails v3 YAML manifest.
- `reference/wails-yaml-run.md`: Local run defaults, platform overrides and launch behaviour for wails3 run.
- `reference/window.md`: Complete reference for the Window API

## (root)

- `status.md`: Wails v3 project status, planned features, and how to contribute

## tutorials

- `tutorials/01-creating-a-service.md`: Create a QR code service to learn about Wails services
- `tutorials/02-todo-vanilla.md`: Build a complete TODO list application with CRUD operations
- `tutorials/03-notes-vanilla.md`: Build a notes application with file operations and native dialogs
- `tutorials/04-self-update-a-wails-app.md`: Build a Wails v3 application that updates itself from GitHub Releases — from `wails3 init` through signed release verification and the helper-mode swap.
- `tutorials/index.md`: section index
- `tutorials/overview.md`: Learn Wails by building applications

## (root)

- `whats-new.md`: Discover the major improvements and new features in Wails v3

## Known doc/code mismatches at v3.1.0 (trust the code)

- `features/windows/frameless.md` documents a `--wails-resize` CSS property. The v3.1.0 runtime does not read it; frameless edge resizing is automatic.
- `concepts/manager-api.md` lists 12 managers. `*App` also has `GlobalShortcut` and `Updater`.
- `contributing/codebase-layout.md` lists templates (lit, preact, qwik, solid, sveltekit, react-swc) that are not built in at v3.1.0. `wails3 init -l` is authoritative: vanilla, react, vue, svelte (+ `-js` variants) and ios.
- Some docs use `//go:embed frontend/dist`, but templates use `//go:embed all:frontend/dist`. `all:` is needed so files starting with `.` or `_` are embedded.
- The alpha→manager rename list in `changelog.md` ("Manager API Refactoring") is partly stale. `app.Dialog.Message()`, `app.SystemTray.Get()`, `app.KeyBinding.Register()` and `app.Env.GetAll()` do not exist. The real methods are `app.Dialog.Info/Warning/Error/Question()`, `app.KeyBinding.Add()` and `app.Env.Info()`. Check with `go doc`.
