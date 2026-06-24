# Changes

Added a new API endpoint that, reusing the existing dependencies service, lets a caller actually apply an upgrade to any updatable Go module dependency.

## What changed
- `internal/services/dependencies.go`: added `Dependencies.Update(ctx, modulePath, targetVersion)` plus the `UpdateResult` type. It validates the module path, ensures the dependency is declared in `go.mod`, resolves the latest version via the proxy when no target is given, rejects downgrades, then runs `go get <module>@<version>` and `go mod tidy` (with `GOFLAGS=-mod=mod`) in the go.mod directory and re-reads go.mod to report the pinned version. Added `requirement` and `runGo` helpers.
- `internal/handlers/dependencies.go`: added `DependenciesHandler.HandleUpdateDependency`, accepting a JSON body `{ "path": "...", "version": "..." }` (version optional → latest) and returning the applied change.
- `cmd/server/main.go`: registered `POST /api/dependencies/update` on the existing `/api/dependencies` group, reusing the already-wired `DependenciesHandler`.
- `README.md`: documented the new endpoint, its request/response shape, and the requirement that the Go toolchain be available at runtime.

## Notes
- The Go toolchain is not available in the sandbox, so the build could not be compiled/verified here. The new code only adds stdlib imports (`os/exec`, `strings`) on top of packages already used by the service.
- The update mutates `go.mod`/`go.sum` on disk by shelling out to `go`, which must be present in the runtime environment; `GO_MOD_PATH` still controls which module is targeted.
- Downgrades and undeclared modules are rejected; per-step failures surface the `go` command output in the error response.
