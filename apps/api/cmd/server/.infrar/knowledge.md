---
schema_version: 1
id: 06f55ae4-7859-4edb-9ec4-b945cf80f916
name: api
node: apps/api/cmd/server
category: app
---
## Purpose

`api` is the Gardbase parent application: the public-facing HTTP server that fronts a zero-trust encrypted NoSQL database. It is the only component clients talk to over the network, and it is deliberately built so that it never holds plaintext data or unwrapped encryption keys.

Its job is threefold:

1. **Authenticate and authorize tenants** — every data request carries a tenant ID plus an API key, which is resolved against DynamoDB and checked against a permission set.
2. **Own the AWS resources** — it reads and writes encrypted objects and encrypted index tokens in DynamoDB, stores large encrypted blobs in S3 (via presigned URLs so payloads never transit the API), and calls KMS to generate and unwrap data keys.
3. **Proxy every cryptographic operation into the Nitro Enclave** — it forwards key material to the enclave over a vsock channel, always attaching an enclave attestation document to the KMS call so KMS releases key material *encrypted to the enclave*, not to the API.

The zero-trust property comes from that third responsibility: the API handles KMS `CiphertextForRecipient` blobs, which only the enclave's NSM-held RSA private key can open. A compromised API host yields ciphertext and nothing else.

`cmd/server` specifically is the entrypoint — wiring, configuration, route registration, and lifecycle. The behavior lives in `apps/api/internal/`.

## Structure

The node directory holds a single file, `main.go`, which is the composition root:

- `Config` / `AWSConfig` — plain structs holding the server port, environment, AWS region, table and bucket names, KMS key ID, retry and timeout settings, and Localstack toggles.
- `Server` — wraps a `*gin.Engine`, a `*zap.Logger`, and the config.
- `main()` — builds the logger, loads config, constructs the server, initializes AWS clients, registers routes, and blocks in `start()`.
- `NewServer()` — creates a bare `gin.New()` engine (no default middleware) and installs the middleware stack; switches Gin to release mode when `ENVIRONMENT == "production"`.
- `setupRoutes()` — the interesting part; see *Behavior*.
- `start()` — runs `http.Server.ListenAndServe` in a goroutine and waits on SIGINT/SIGTERM for a 10-second graceful shutdown.
- `initAWSServices()` / `loadAWSSDKConfig()` / `testAWSConnectivity()` — AWS client construction and a best-effort startup reachability probe.
- `getEnv*` helpers — typed environment lookups with defaults; `getEnvOrPanic` for the values with no sane default.

The rest of the application, referenced from here:

- `internal/handlers/` — `healthCheck.go`, `tenants.go`, `objects.go` (the largest, ~530 lines: put/get/scan/query/delete/recover plus the large-object presigned flow), `encryption.go` (secure-session lifecycle and single-shot decrypt).
- `internal/middleware/` — `zapLogger.go` (structured request logging + panic recovery), `cors.go`, `ratelimit.go`, `tenant.go` (auth), `permission.go` (authorization).
- `internal/storage/` — `dynamo.go` (~1000 lines: objects, index rows, tenant configs, API keys, optimistic-locked updates, soft delete/recover, paginated scan and index queries), `s3.go` (presign put/get, existence check, deletion tagging), `errors.go`.
- `internal/services/` — `enclaveVSock.go` (vsock connection pool and newline-delimited JSON request/response), `kms.go` (attestation-bound `GenerateDataKey` and `Decrypt`).

Shared modules come from `pkg/api` (HTTP DTOs), `pkg/enclaveproto` (parent↔enclave wire types), and `pkg/models` (persisted shapes, permission constants), all wired via `replace` directives in `apps/api/go.mod`.

## Behavior

**Startup.** Build a production zap logger → load `Config` from `PORT`/`ENVIRONMENT` → construct the Gin engine with middleware → load `AWSConfig` (panicking if `S3_BUCKET`, the five `DYNAMO_*_TABLE` names, or `KMS_KEY_ID` are missing) → load the AWS SDK config (static `test`/`test` credentials under Localstack, otherwise the default provider chain, which on EC2 means the instance profile) → construct the S3, DynamoDB, and KMS clients → probe S3 and DynamoDB connectivity → register routes → listen.

The connectivity probe is deliberately non-fatal: failures are logged at error level and `testAWSConnectivity` still returns `nil`, so the server starts even if AWS is temporarily unreachable.

**Middleware order** (global, applied to every request): structured request logging → panic recovery → CORS → rate limiting. The rate limiter is a per-process, per-client-IP sliding window allowing 1000 requests per minute, backed by an in-memory map guarded by a mutex.

**Route groups** — everything lives under `/api`:

- `/api/health` — four unauthenticated probes: overall, enclave (round-trips a vsock `health` request), storage, and KMS.
- `/api/tenants` — `POST /` creates a tenant. Unauthenticated by design, since it is what bootstraps a tenant's wrapped master key, wrapped table salt, and first API key.
- `/api/objects` — guarded by `TenantMiddleware`, then split by permission. `POST /get-table-hash` needs read *or* write. The read group covers `/get`, `/scan`, `/query`; the write group covers `/put`, `/request-put-large`, `/confirm-put-large`, `/delete`, `/recover`.
- `/api/encryption` — guarded by `TenantMiddleware` plus the `crypto` permission. Holds the secure-session lifecycle (`/secure-session/init`, `/unwrap`, `/generate-deks`, `/get-table-iek`) and the single-shot `/decrypt`.

**Authentication.** `TenantMiddleware` requires `X-Tenant-ID` (matching `^[a-z0-9-]{3,64}$`) and `X-API-Key`. It looks the key up in the API-keys table and, on success, stashes the tenant ID and permission list on both the Gin context and the request context. `PermissionMiddleware` then requires *all* listed permissions to be present, returning 403 otherwise.

**The cryptographic round-trip.** The recurring pattern in `handlers/encryption.go` and `handlers/objects.go` is:

1. Ask the enclave for a current attestation document over vsock (`get_attestation`).
2. Call KMS `GenerateDataKey` or `Decrypt` with that document as `Recipient.AttestationDocument` and an encryption context of `{tenant_id, purpose}` where purpose is one of `MASTER_KEY`, `DATA_KEY`, `INDEX_KEY`, `TABLE_SALT`. KMS then returns `CiphertextForRecipient` — the key material sealed to the enclave's RSA public key.
3. Forward that blob to the enclave over vsock (`session_prepare_dek`, `session_prepare_iek`, `session_unwrap`, `decrypt`, …).
4. Return the enclave's response — DEKs re-sealed under the client's session key — straight to the caller.

The API therefore never observes a plaintext DEK, master key, or index key: it moves blobs between KMS and the enclave.

**Table IEK lifecycle.** `generate-deks` and `get-table-iek` both lazily provision a per-table index encryption key: look for a KMS-wrapped IEK in DynamoDB; if absent, `GenerateDataKey` a new one, persist the `CiphertextBlob`, and hand the `CiphertextForRecipient` to the enclave; if present, KMS-decrypt it to a recipient blob first. `generate-deks` caps `count` at 1–100.

**Table names are never stored in the clear.** `/get-table-hash` decrypts the tenant's table salt via KMS, sends it plus the session-encrypted table name to the enclave, and gets back a salted SHA-256 hash used as the storage-level table identifier.

**Large objects.** `request-put-large` issues a presigned S3 PUT (15-minute TTL, `PresignTTL` on the handler) so the client uploads its own ciphertext directly to S3; `confirm-put-large` then records the metadata. Reads mirror this with presigned GETs. `BASE_URL` (default `https://api.gardbase.com`) prefixes URLs the handler returns.

**vsock transport.** `services.Vsock` maintains a pool of up to 10 idle connections to `(ENCLAVE_CID, ENCLAVE_PORT)`. Each request is one line of JSON written to the connection, with the response read back by a `bufio.Scanner`; per-call deadlines are 5s for attestation, 10s for session init and table hash, 15s for decrypt, and 30s for unwrap and DEK preparation. Connections are returned to the pool on success and closed on error.

**Shutdown.** SIGINT or SIGTERM triggers `srv.Shutdown` with a 10-second context; a failed shutdown is fatal.

## Dependencies

**Go module** — `github.com/qodesrl/gardbase/apps/api`, Go 1.24.4, part of the root `go.work` workspace.

Direct third-party dependencies:

- `github.com/gin-gonic/gin` — HTTP routing and middleware.
- `go.uber.org/zap` — structured logging.
- `github.com/aws/aws-sdk-go-v2` with `config`, `credentials`, `service/s3`, `service/dynamodb`, `feature/dynamodb/attributevalue`, `service/kms`.
- `github.com/mdlayher/vsock` — AF_VSOCK sockets to the enclave. **Requires CGO**, which is why the Dockerfile builds with `CGO_ENABLED=1`.
- `github.com/google/uuid` — object IDs.

Internal modules, resolved by `replace` to sibling directories: `pkg/api`, `pkg/enclaveproto`, `pkg/models`.

**Runtime dependencies:**

- A running Nitro Enclave reachable at the configured CID/port — required for every crypto path, and for the `/health/enclave` probe.
- AWS: one S3 bucket, five DynamoDB tables (objects, indexes, table-configs, tenant-configs, api-keys), and one KMS key that permits attestation-conditioned `GenerateDataKey`/`Decrypt`.
- Credentials from the EC2 instance profile in deployment; static `test` credentials against Localstack.

**Environment variables:**

| Variable | Required | Default |
| --- | --- | --- |
| `S3_BUCKET`, `DYNAMO_OBJECTS_TABLE`, `DYNAMO_INDEXES_TABLE`, `DYNAMO_TABLE_CONFIGS_TABLE`, `DYNAMO_TENANT_CONFIGS_TABLE`, `DYNAMO_API_KEYS_TABLE`, `KMS_KEY_ID` | yes — panics if unset | — |
| `PORT` | no | `80` |
| `ENVIRONMENT` | no | `development` |
| `AWS_REGION` | no | `eu-central-1` |
| `AWS_MAX_RETRIES` | no | `3` |
| `AWS_REQUEST_TIMEOUT` | no | `5` (seconds) |
| `BASE_URL` | no | `https://api.gardbase.com` |
| `ENCLAVE_CID` | no | `16` |
| `ENCLAVE_PORT` | no | `8080` |
| `USE_LOCALSTACK`, `LOCALSTACK_URL` | no | `false`, `http://localhost:4566` |

**Build and run** (Nx targets in `apps/api/project.json`): `build`, `serve` (`go run ./cmd/server`), `dev` (`air` hot reload), `test`, `lint` (golangci-lint), `format`, `tidy`, and the Docker chain `docker-build` / `docker-tag` / `docker-login` / `docker-push` / `build-and-push`. The image is built from the repo-root `Dockerfile.Api` (Go 1.25 Alpine builder with `GOWORK=off` and `CGO_ENABLED=1`, Alpine runtime, non-root `appuser`, exposes 80 and 443).

Deployed by `infrastructure/main`: pushed to ECR, pulled on an enclave-enabled EC2 instance, and run by the `gardbase-parent` systemd unit with `--device=/dev/vsock` and all the environment variables above injected from Terraform outputs.

## Notes

- **`ENCLAVE_PORT` defaults disagree with deployment.** `main.go` defaults to `8080`, but the enclave service defaults to `5000` and `infrastructure/main/user_data.sh` explicitly sets `ENCLAVE_PORT=5000` on the parent container. Local runs that rely on the default will fail to reach the enclave.
- **CORS is wide open.** `cors.go` sends `Access-Control-Allow-Origin: *` together with `Allow-Credentials: true` and carries a `// todo: restrict this in prod` comment. The combination is rejected by browsers and is unsuitable for production as written.
- **The rate limiter is per-process and unbounded in memory.** `clients` is a `map[string][]time.Time` that is never pruned of idle IPs, so it grows with the number of distinct client IPs seen. It also resets on restart and does not coordinate across instances.
- **Tenant creation is unauthenticated.** `POST /api/tenants/` sits outside `TenantMiddleware`, so anyone who can reach the API can create tenants.
- **Startup does not fail closed on AWS problems.** `testAWSConnectivity` logs errors and returns `nil`; KMS is not probed at all at startup (only via `/health/kms`).
- **The `Config.Environment` check is inconsistent with the DynamoDB Terraform.** `main.go` treats `"production"` as the production marker (for Gin release mode), while `infrastructure/main` is applied with `environment = "prod"` and its DynamoDB capacity conditionals compare against `"production"` — so the higher provisioned capacity branch never fires under the shipped tfvars.
- `vsockConn` carries an `encoder` field that is constructed but never used; requests are written directly to the connection instead.
- Handlers mostly return errors as bare `500` with the raw underlying message in the JSON body, which leaks AWS and internal error detail to callers.
- `handlers/encryption.go` has one place in `HandleSessionUnwrap` where an attestation-fetch failure writes a 500 response but does not `return`, so execution continues with an empty attestation document.
