# internal/app architecture

Wails v3 domain layer for Junimo Hut. Go business logic lives here; [`main.go`](../../main.go) only boots Wails and registers services.

## Components

```text
main.go
  └── app.NewServices() → Core + 8 Wails services
        ├── Core (shared state, not a Wails service)
        │     ├── config.Store, profiles, categories, nexus, …
        │     └── ModCatalog (mod cache + assembly)
        ├── SystemService      → ServiceStartup → Core.Startup
        ├── SettingsService
        ├── ModsService
        ├── ProfilesService
        ├── CategoriesService
        ├── SMAPIService
        ├── NexusService
        └── ConfigEditorService
```

## Core

`Core` is constructed once in `main` and injected into every service via `NewX(core *Core)`. It holds:

- Internal domain services (`*config.Store`, `*profiles.Service`, …)
- `*ModCatalog` for the scanned mod library
- `EventPublisher` for frontend events (`mods-changed`, `nxm-url`, …)
- `*application.App` set via `SetApplication` before `Run()` (dialogs + events)

Initialization runs in `SystemService.ServiceStartup`, which calls `Core.Startup(ctx)`.

## ModCatalog

Owns `mods`, `duplicates`, and `unmanaged` caches plus refresh/assembly logic.

**Lock order (always acquire in this order):**

1. `refreshMu` — serializes full rescans
2. `mu` — protects cache slices
3. `assembleMu` — serializes profile assembly / unmanaged scan

`Refresh` returns errors from scanning; assembly failures are logged with `slog.Warn` (no silent swallow).

## Frontend bindings

Each service generates bindings under `frontend/bindings/junimohut/internal/app/*service.js`.

[`frontend/src/lib/api/index.ts`](../../frontend/src/lib/api/index.ts) re-exports all RPC methods so UI code keeps `import * as API from "$lib/api"`.

## Events

| Event | Emitter | Purpose |
|-------|---------|---------|
| `mods-changed` | `EventBridge` | Mod list mutated |
| `nxm-url` | `EventBridge` | Nexus NXM link received |
| `nexus-download-ready` | `EventBridge` | Download finished |
| `config-editor-open-mod` | Config editor window | Open/switch mod in editor |
| `config-editor-reload` | Config editor window | Reload editor content |

## Adding RPC methods

1. Add the method to the appropriate `*_service.go` file.
2. Run `wails3 generate bindings`.
3. Confirm the function is exported via `api/index.ts` (same name as before).
