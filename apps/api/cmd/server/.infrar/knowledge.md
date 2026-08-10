---
schema_version: 1
id: 1b63a8f9-2ef7-47ed-b075-377e7d37712e
name: api
node: apps/api/cmd/server
category: app
---
## Purpose

`api` is the Gardbase parent application: the public-facing HTTP API of a zero-trust, fully encrypted NoSQL DBaaS. It is the only component clients talk to directly, and it is deliberately built to be incapable of reading customer data.

Its role is that of a *trusted-with-nothing* broker. It authenticates tenants, enforces per-API-key permissions, stores ciphertext in DynamoDB and S3, calls AWS KMS to wrap and unwrap key material, and relays everything key-related over a vsock channel to the Nitro Enclave sibling process (`enclave-service`). Plaintext objects, plaintext DEKs, and the tenant master key never exist in this process's memory: every KMS call carries an enclave attestation document as `Recipient`, so the useful output (`CiphertextForRecipient`) is encrypted to the enclave's RSA key and is opaque to the API.

The entrypoint is `apps/api/cmd/server/main.go`; the implementation lives under `apps/api/internal/`.

## Structure

```
apps/api/
  cmd/server/main.go        # entrypoint: config, AWS clients, route table, graceful shutdown
  internal/handlers/
    healthCheck.go          #   /api/health/, /health/enclave, /health/storage, /health/kms
    tenants.go              #   tenant provisioning + API-key CRUD helpers
    objects.go              #   object CRUD, presigned large-object flow, scan/query
    encryption.go           #   secure-session endpoints + single-shot decrypt
  internal/middleware/      # zapLogger (+recovery), cors, ratelimit, tenant, permission
  internal/services/
    enclaveVSock.go         # pooled newline-delimited JSON client over vsock
    kms.go                  # KMS wrapper; every call carries an attestation document
  internal/storage/
    dynamo.go               # access layer for objects/indexes/table+tenant configs/api keys
    s3.go                   # presign PUT/GET, existence check, deletion tagging
    errors.go               # sentinel errors (ErrNotFound, ErrVersionMismatch, …)
  project.json              # Nx targets: build/serve/dev/test/lint/format/tidy, docker-*
  go.mod                    # module github.com/qodesrl/gardbase/apps/api
```

Built by `Dockerfile.Api` at the repo root: multi-stage `golang:1.25-alpine` → `alpine`, `CGO_ENABLED=1` (vsock needs cgo), `GOWORK=off` so the module resolves through its local `replace` directives, runs as non-root `appuser`, exposes 80 and 443.

Route table, all under `/api`:

| Group | Endpoints | Guard |
| --- | --- | --- |
| health | `GET /health/`, `/health/enclave`, `/health/storage`, `/health/kms` | none |
| tenants | `POST /tenants/` | none (bootstrap path) |
| objects | `POST /objects/get-table-hash` | tenant + (read or write) |
| objects | `POST /objects/get`, `/scan`, `/query` | tenant + read |
| objects | `POST /objects/put`, `/request-put-large`, `/confirm-put-large`, `/delete`, `/recover` | tenant + write |
| encryption | `POST /encryption/secure-session/{init,unwrap,generate-deks,get-table-iek}`, `/encryption/decrypt` | tenant + crypto |

## Behavior

**Startup.** `main` builds a zap production logger, loads `Config` (`PORT`, `ENVIRONMENT`) and `AWSConfig` from the environment. Storage and key identifiers — `S3_BUCKET`, the five `DYNAMO_*_TABLE` variables, and `KMS_KEY_ID` — are mandatory and panic through `getEnvOrPanic` if absent. `USE_LOCALSTACK=true` substitutes static test credentials and a `BaseEndpoint` override on each AWS client for local development. S3 and DynamoDB connectivity is probed at boot, but failures are only logged, not fatal. The process then registers routes and blocks on SIGINT/SIGTERM, draining with a 10-second graceful shutdown.

**Request pipeline.** Global middleware runs in order: zap request logging → zap panic recovery → CORS → in-memory per-IP rate limiting (1000 requests/minute). Tenant-scoped groups add `TenantMiddleware`, which validates `X-Tenant-ID` against `^[a-z0-9-]{3,64}$`, resolves `X-API-Key` against the API-keys table (bcrypt comparison against the stored hash, prefix check, expiry check), and stores `tenantId` and `permissions` in both the gin context and the request context. `PermissionMiddleware` then gates on `read`, `write`, or `crypto`.

**Enclave transport.** `services.Vsock` maintains a small pool (max 10 idle connections) to `ENCLAVE_CID` (default 16) and `ENCLAVE_PORT` (default 8080 in code; the deployment sets 5000, matching the enclave's listener). Each call writes one line of JSON — `{type, payload}` — under a per-call deadline, reads one line back with a `bufio.Scanner`, clears the deadline, and returns the connection to the pool. Any I/O error closes the connection instead of recycling it.

**Secure-session flow.** A client opens a session at `/secure-session/init` by sending its X25519 public key and a random nonce. The API forwards the payload verbatim and returns the enclave's ephemeral public key, a session id, an expiry, and a fresh attestation document bound to both. Every subsequent key operation follows the same shape:

1. Ask the enclave for a current attestation document.
2. Call KMS (`GenerateDataKey` or `Decrypt`) passing that document as `Recipient` with `RSAES_OAEP_SHA_256`, and an `EncryptionContext` of `{tenant_id, purpose}` where purpose is one of `MASTER_KEY`, `DATA_KEY`, `INDEX_KEY`, `TABLE_SALT`.
3. Forward the resulting `CiphertextForRecipient` — readable only inside the enclave — to the enclave, which unwraps it and re-seals it under the session key.
4. Return the session-sealed blob to the client.

`generate-deks` additionally fetches the tenant's KMS-wrapped master key and the table's index encryption key (creating and persisting a new IEK on first use for that table), so the enclave can return each DEK sealed three ways: under the session key for immediate client use, under the tenant master key for durable storage, and as the raw KMS ciphertext blob. `count` is validated to 1–100.

**Table hashing.** Table names are never stored in the clear. The client sends the table name encrypted under the session key; the enclave decrypts it and returns `base64url(SHA-256(name ‖ tableSalt))`. That `table_hash` becomes the partition-key component of every subsequent object and index operation, so the API sees only an opaque identifier.

**Object storage.** Payloads under 100 KB are stored inline in DynamoDB as `encrypted_blob`. Larger payloads use a two-step presigned flow: `request-put-large` returns a presigned PUT URL for `tenant-<id>/<tableHash>/<objectId>/v<version>` with a 15-minute TTL, and `confirm-put-large` verifies via `HeadObject` that the upload landed before writing the DynamoDB record. Writes are optimistically locked — the client sends the intended new `version`, and the conditional update requires the previous one, returning 409 on mismatch. Object creation writes the object and its index entries in a single `TransactWriteItems` when there are 25 items or fewer, otherwise a plain put plus batched index writes of 25.

**Deletes.** Deletion is soft: status flips to `deleted`, a 30-day TTL attribute is set, index entries are removed (in a background goroutine), and any S3 object is tagged `status=deleted` so the bucket lifecycle rule expires it after 30 days. `/recover` reverses all of it.

**Queries.** Encrypted index entries live in a separate table keyed `TENANT#<id>#TABLE#<hash>#IDX#<name>` with the index token as a *binary* sort key, plus a GSI keyed by object id used to find an object's index entries for update and cleanup. The object id is appended to every token to keep entries unique across objects with equal values. Equality queries use `begins_with` on the deterministic token; range queries (`gt`, `gte`, `lt`, `lte`, `between`) construct lower/upper bounds from the deterministic hash prefix, an order-preserving range segment, and min/max object-id sentinels, then run a `BETWEEN` key condition. Matching object ids are then fetched in parallel batches of 100 with `BatchGetItem`, retrying unprocessed keys with exponential backoff, and re-ordered to match the index order.

**Tenant creation.** `POST /api/tenants/` is unauthenticated by design — it is how a tenant comes into existence. It mints a UUID, generates a KMS master key and a table salt, persists both as KMS ciphertext blobs, and returns a first API key carrying all three permissions plus the attestation document. The API key is bcrypt-hashed at rest and returned only once.

## Dependencies

- **AWS:** `aws-sdk-go-v2` for KMS, DynamoDB (with `attributevalue`), and S3 including presigning. All KMS calls use `RecipientInfo` with `RSAES_OAEP_SHA_256`.
- **Transport:** `github.com/mdlayher/vsock` — requires cgo and `/dev/vsock` inside the container.
- **HTTP and utilities:** `gin-gonic/gin`, `go.uber.org/zap`, `google/uuid`, `golang.org/x/crypto/bcrypt`.
- **Workspace modules** (wired via local `replace` directives): `pkg/enclaveproto` for the enclave wire format, `pkg/api` for client-facing request/response types, `pkg/models` for DynamoDB record shapes and key builders.
- **Peers:** `enclave-service` over vsock; `infrastructure-main` provisions the EC2 host, IAM role, KMS key, S3 bucket, and DynamoDB tables, and injects every environment variable through `user_data.sh`.

## Notes

- The API is intentionally *unable* to decrypt. A change that lets this process observe a plaintext DEK or object body breaks the core security property; it is not an optimization.
- KMS access is gated in the key policy by `kms:RecipientAttestation:ImageSha384` matching the enclave's PCR0 — unless `enable_debug_mode` is set, in which case the condition degrades to `*`. Debug mode is development-only.
- `CorsMiddleware` sends `Access-Control-Allow-Origin: *` and carries an explicit `todo: restrict this in prod`.
- Rate limiting is a per-process in-memory map with no eviction of idle client entries. It does not survive restarts and does not coordinate across instances.
- The deployment is a single EC2 instance, so in-memory state (rate limiter, vsock pool) and the enclave's session store are naturally co-located. Horizontal scaling would require revisiting all three.
- `TenantHandler.CreateNewAPIKey`, `DeleteAPIKey`, and `ListAPIKeys` are implemented but not registered on any route.
- The commented-out `prepare_kek` block in `tenants.go` is the placeholder for the planned self-managed / client-held master key mode; master-key recovery is an open TODO.
- `Delete` binds its JSON body without returning after a bind error, so a malformed delete request writes a 400 and then continues into the delete path.
- `Scan` and `Query` return the raw `s3_key` in the `GetURL` field rather than a presigned URL, unlike `Get`, which presigns.
- `handlers/encryption.go` uses stdlib `log.Printf` for DEK diagnostics, unlike the rest of the app, which uses zap.
