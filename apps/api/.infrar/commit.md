# Changes

Added a new API endpoint to list all Go module dependencies of the repository and flag which ones are updatable.

## What changed
- `internal/services/dependencies.go` (new): `Dependencies` service that parses `go.mod` with `golang.org/x/mod/modfile`, queries the Go module proxy `@latest` endpoint concurrently for each module, and compares versions using `semver` to set an `updatable` flag.
- `internal/handlers/dependencies.go` (new): `DependenciesHandler.HandleListDependencies` returning total/updatable counts plus the per-dependency report.
- `cmd/server/main.go`: registered `GET /api/dependencies/` and wired the handler; the go.mod path is configurable via the `GO_MOD_PATH` env var (defaults to `go.mod`).
- `go.mod`: promoted `golang.org/x/mod` from indirect to a direct dependency (hashes already present in go.sum).
- `README.md`: documented the new endpoint and its response shape.

## Notes
- The Go toolchain is not available in the sandbox, so the build could not be compiled/verified here. The new code uses only already-vendored dependencies (`golang.org/x/mod`, gin) and follows existing handler/service patterns.
- The proxy is taken from `GOPROXY` when set to a single direct URL, otherwise defaults to `https://proxy.golang.org`. Per-dependency lookup errors are surfaced in the response rather than failing the whole request.
