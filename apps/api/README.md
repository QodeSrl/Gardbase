<div align="center">

<br>

# Gardbase API

</div>

## Endpoints

### Dependencies

`GET /api/dependencies/`

Lists every Go module dependency declared in the repository's `go.mod`, reporting
the version currently in use and whether a newer version is available on the Go
module proxy.

The go.mod location can be overridden via the `GO_MOD_PATH` environment variable
(defaults to `go.mod`). The module proxy is taken from `GOPROXY` when set to a
single direct URL, otherwise it defaults to `https://proxy.golang.org`.

Example response:

```json
{
  "total": 2,
  "updatable": 1,
  "dependencies": [
    {
      "path": "github.com/gin-gonic/gin",
      "current_version": "v1.11.0",
      "latest_version": "v1.11.0",
      "updatable": false,
      "indirect": false
    },
    {
      "path": "go.uber.org/zap",
      "current_version": "v1.27.0",
      "latest_version": "v1.27.1",
      "updatable": true,
      "indirect": false
    }
  ]
}
```

`POST /api/dependencies/update`

Upgrades a single dependency listed by `GET /api/dependencies/`. Provide the
module `path` and, optionally, the target `version`. When `version` is omitted
the dependency is upgraded to its latest available version on the module proxy.

The update runs `go get <module>@<version>` followed by `go mod tidy` in the
directory containing the configured `go.mod` (see `GO_MOD_PATH`), so the Go
toolchain must be available in the runtime environment.

Example request:

```json
{
  "path": "go.uber.org/zap",
  "version": "v1.27.1"
}
```

Example response:

```json
{
  "updated": {
    "path": "go.uber.org/zap",
    "previous_version": "v1.27.0",
    "new_version": "v1.27.1"
  }
}
```
