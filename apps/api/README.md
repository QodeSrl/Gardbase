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
