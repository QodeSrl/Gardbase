---
schema_version: 1
id: 30d1db76-770f-49f5-9d67-b96ea1bcc10f
name: api
node: apps/api/cmd/server
category: app
---
## Purpose

`api` is the Gardbase **parent application**: the public HTTP API of a zero-trust, fully encrypted NoSQL DBaaS. It is the only component clients talk to over the network, and it is deliberately built so that it never holds plaintext data or unwrapped key material.

Its job is orchestration, not cryptography:

- terminate client HTTP requests, authenticate them per tenant, and enforce coarse permissions;
- persist already-encrypted blobs and encrypted index tokens in DynamoDB and S3;
- call AWS KMS with a Nitro Enclave attestation document so key material comes back only as `CiphertextForRecipient` — a blob only the enclave's NSM-held RSA key can open;
- proxy those KMS ciphertexts to the enclave service over vsock and hand the enclave's session-sealed results back to the client.

The intended security property is that compromising this process yields ciphertext and KMS-wrapped keys but no usable plaintext: the plaintext key path runs client → enclave, with the API acting as an untrusted courier.

## Structure

The node entrypoint is `apps/api/cmd/server` (`main.go`); the implementation lives under `apps/api/internal`. The app is its own Go module (`github.com/qodesrl/gardbase/apps/api`) with `replace` directives pointing at the sibling `pkg/*` modules.

- `cmd/server/main.go` — process wiring. Defines `Server` (gin engine + zap logger + `Config`) and `AWSConfig`, loads all configuration from environment variables, constructs the S3 / DynamoDB / KMS clients, registers every route in `setupRoutes`, and runs `start()` with a 10s graceful-shutdown window on SIGINT/SIGTERM. The env helpers at the bottom (`getEnv`, `getEnvOrPanic`, `getEnvAsInt`, `getEnvAsBool`, `getEnvUint32`) are what make the required AWS resource names hard failures at boot.
- `internal/handlers/` — one file per route group:
  - `healthCheck.go` — API, enclave (vsock round-trip), storage (S3 + DynamoDB), and KMS (`DescribeKey`) probes.
  - `tenants.go` — tenant creation (generates the KMS-wrapped master key and table salt, persists them, mints the first API key). Also carries not-yet-routed API-key management handlers (`CreateNewAPIKey`, `DeleteAPIKey`, `ListAPIKeys`).
  - `objects.go` — the data plane: `GetTableHash`, `Put`, `RequestPutLarge`/`ConfirmPutLarge`, `Get`, `Scan`, `Query`, `Delete`, `Recover`, plus the `generateS3Key` helper.
  - `encryption.go` — the key plane: session init, session unwrap, DEK generation, per-table index-key (IEK) retrieval, and one-shot `decrypt`.
- `internal/middleware/` — `zapLogger.go` (structured request logging + panic recovery), `cors.go`, `ratelimit.go`, `tenant.go` (header-based tenant + API-key authentication), `permission.go` (permission-set check).
- `internal/services/` — `kms.go` (thin KMS wrapper that always attaches a `RecipientInfo` attestation document and a `tenant_id`/`purpose` encryption context) and `enclaveVSock.go` (`Vsock` client plus a small idle-connection pool).
- `internal/storage/` — `dynamo.go` (the bulk of the persistence logic across five tables), `s3.go` (presigned URLs, existence checks, deletion tagging), `errors.go` (sentinel errors used to map storage failures onto HTTP status codes).

Build and packaging live outside the node directory: `apps/api/project.json` (Nx targets) and the repo-root `Dockerfile.Api`.

## Behavior

**Routing.** Everything is mounted under `/api`. Global middleware order is zap logger → zap recovery → CORS → rate limit.

- `/api/health`, `/api/health/enclave`, `/api/health/storage`, `/api/health/kms` — unauthenticated.
- `/api/tenants/` (POST) — unauthenticated tenant bootstrap.
- `/api/objects/*` — `TenantMiddleware` then `PermissionMiddleware`. `get-table-hash` accepts read *or* write; `get`/`scan`/`query` require `read`; `put`, `request-put-large`, `confirm-put-large`, `delete`, `recover` require `write`.
- `/api/encryption/*` — `TenantMiddleware` plus the `crypto` permission.

**Authentication.** `TenantMiddleware` reads `X-Tenant-ID` (must match `^[a-z0-9-]{3,64}$`) and `X-API-Key`, queries every API key row for the tenant, checks the `gdb_live_` prefix and a bcrypt comparison, and rejects expired keys. On success it puts `tenantId` and `permissions` into both the gin context and the request context.

**Object writes.** A blob under ~100 KB is sent inline and stored in the DynamoDB objects table; anything above 100 KB goes through `request-put-large` (returns a 15-minute presigned S3 PUT URL and the expected version) followed by `confirm-put-large` (verifies the object exists in S3, then writes the metadata row). Concurrency is controlled by optimistic locking: the client sends `version`, and updates use a conditional `PutItem` on `version = :current AND status <> deleted`, returning `409` on mismatch. Object + index rows are written in a single `TransactWriteItems` when there are ≤25 index items, otherwise the object is written first and the indexes are batched 25 at a time.

**Indexes and queries.** Index rows live in a separate table keyed by `TENANT#<id>#TABLE#<hash>#IDX#<index_name>` with a binary sort key: the deterministic hash token (32 bytes), optionally an order-preserving range token (8 bytes), then the object UUID (16 bytes) so equal values stay distinct. `QueryIndexes` translates `eq`/`lt`/`lte`/`gt`/`gte`/`between` into DynamoDB `begins_with` or `BETWEEN` key conditions by padding the range segment with min/max sentinels, then fans out `BatchGetItem` calls (100 keys per goroutine, with backoff on unprocessed keys) to hydrate the matching objects while preserving index order.

**Deletes.** `Delete` is a soft delete: the object row is flipped to `deleted`, its version bumped, and a 30-day `ttl` set; index rows are removed and the S3 object is tagged `status=deleted` (a bucket lifecycle rule expires it). `Recover` reverses all of that.

**Key operations.** Every enclave-bound call follows the same shape: fetch a fresh attestation document from the enclave over vsock, call KMS (`GenerateDataKey` or `Decrypt`) with that document as `RecipientInfo` and a `tenant_id`/`purpose` encryption context, forward the resulting `CiphertextForRecipient` to the enclave, and return whatever the enclave sealed to the caller's session key. Purposes are `MASTER_KEY`, `DATA_KEY`, `INDEX_KEY`, and `TABLE_SALT`. Per-table index keys are generated lazily on first use and their KMS-wrapped form is persisted in the table-configs table.

**Enclave transport.** `Vsock.SendToEnclave` writes newline-delimited JSON over a pooled `vsock` connection (up to 10 idle), sets a per-call deadline (5s attestation/health, 10s session init and table hash, 15s decrypt, 30s unwrap and DEK generation), reads one line back, and returns the connection to the pool.

**Configuration.** `PORT` (default `80`), `ENVIRONMENT` (default `development`; `production` switches gin to release mode), `AWS_REGION` (default `eu-central-1`), `BASE_URL` (default `https://api.gardbase.com`), `ENCLAVE_CID` (default `16`), `ENCLAVE_PORT` (default `8080`), `AWS_MAX_RETRIES`, `AWS_REQUEST_TIMEOUT`, `USE_LOCALSTACK`, `LOCALSTACK_URL`. Required (the process panics without them): `S3_BUCKET`, `DYNAMO_OBJECTS_TABLE`, `DYNAMO_INDEXES_TABLE`, `DYNAMO_TABLE_CONFIGS_TABLE`, `DYNAMO_TENANT_CONFIGS_TABLE`, `DYNAMO_API_KEYS_TABLE`, `KMS_KEY_ID`.

## Dependencies

**Internal (workspace modules, wired via `replace`):**
- `pkg/models` — DynamoDB item structs and key builders (`Object`, `Index`, `TableConfig`, `TenantConfig`, `APIKey`), status/permission/sensitivity constants, and the index-token length constants the query builder relies on.
- `pkg/api` — the wire contract with clients (`objects`, `encryption`, `tenants` request/response types).
- `pkg/enclaveproto` — the wire contract with the enclave (`Request`, generic `Response[T]`, and one type per message).

**Runtime peers:**
- The `enclave-service` node, reachable only over vsock at `ENCLAVE_CID`/`ENCLAVE_PORT`. Every crypto route fails if the enclave is down.
- AWS: DynamoDB (five tables), S3 (one bucket), KMS (one key). All provisioned by `infrastructure/main`, whose `user_data.sh` injects the resource names as environment variables into the container.

**Third party:** `gin-gonic/gin`, `go.uber.org/zap`, `aws-sdk-go-v2` (config, credentials, dynamodb + attributevalue, s3, kms), `mdlayher/vsock`, `google/uuid`, `golang.org/x/crypto/bcrypt`.

**Tooling:** Nx targets in `apps/api/project.json` (`build`, `serve`, `dev` via air, `test`, `lint` via golangci-lint, `format`, `tidy`, and the `docker-*` / `build-and-push` chain). `Dockerfile.Api` builds with `GOWORK=off` and `CGO_ENABLED=1` (required for vsock), then ships a non-root Alpine image exposing 80 and 443.

## Notes

- The default `ENCLAVE_PORT` in code is `8080`, but the enclave service defaults to `5000`; the deployed unit file in `infrastructure/main/user_data.sh` sets `ENCLAVE_PORT=5000` and `ENCLAVE_CID=16` explicitly, so the defaults never line up on their own. Set both sides when running locally.
- `pkg/crypto` is *not* a dependency of this module. The API only moves opaque ciphertext around; all client-side encryption lives in the SDK.
- CORS is wide open (`Access-Control-Allow-Origin: *`) and carries a `todo: restrict this in prod` comment.
- The rate limiter is per-process and in-memory: 1000 requests/minute per client IP, keyed in a map that is never pruned, so IP entries accumulate for the life of the process. It also does not survive a restart or coordinate across instances.
- `testAWSConnectivity` at boot logs S3/DynamoDB failures but always returns `nil`, so the server starts even when storage is unreachable.
- `Scan` and `Query` populate `GetURL` with the raw S3 key rather than a presigned URL; only `Get` presigns.
- `SoftDeleteObjectAndIndexes` launches index cleanup in a goroutine using the request context, which is cancelled as soon as the HTTP response is written.
- Tenant creation still contains the commented-out `prepare_kek` flow for self-managed keys, matching the enclave handler that is registered but unused.
- `FindAPIKey` returns `nil` (rejecting the request) as soon as it sees a stored key whose prefix does not match the presented one, so it does not survey all of a tenant's keys with mixed prefixes.
