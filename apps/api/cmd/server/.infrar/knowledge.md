---
schema_version: 1
id: 39fd4ea4-cc63-4e70-b294-fac91456bc97
name: api
node: apps/api/cmd/server
category: app
---
## Purpose

`api` is the Gardbase parent application: the public-facing HTTP API of a zero-trust, fully encrypted NoSQL DBaaS. It is the only component clients talk to directly, and it deliberately holds no ability to read customer data.

Its job is to be a *trusted-with-nothing* broker. It authenticates tenants, enforces per-API-key permissions, stores ciphertext in DynamoDB and S3, calls AWS KMS to wrap/unwrap key material, and relays everything key-related over a vsock channel to the Nitro Enclave sibling process (`enclave-service`). Plaintext objects, plaintext DEKs, and the tenant master key never exist in this process's memory — KMS is always called with an enclave attestation document as `Recipient`, so the useful output (`CiphertextForRecipient`) is encrypted to the enclave's RSA key and is opaque to the API.

The entrypoint lives at `apps/api/cmd/server/main.go`; the implementation lives under `apps/api/internal/`.

## Structure

```
apps/api/
  cmd/server/main.go        # entrypoint: config, AWS clients, route table, graceful shutdown
  internal/handlers/        # HTTP handlers (gin)
    healthCheck.go          #   /api/health, /health/enclave, /health/storage, /health/kms
    tenants.go              #   tenant provisioning + API-key CRUD helpers
    objects.go              #   object CRUD, presigned large-object flow, scan/query
    encryption.go           #   secure-session endpoints + single-shot decrypt
  internal/middleware/      # cors, ratelimit, zapLogger (+recovery), tenant, permission
  internal/services/
    enclaveVSock.go         # pooled newline-delimited JSON client over vsock
    kms.go                  # KMS wrapper; all calls carry an attestation document
  internal/storage/
    dynamo.go               # single-table-ish access for objects/indexes/configs/keys
    s3.go                   # presign PUT/GET, existence check, deletion tagging
    errors.go               # sentinel errors (ErrNotFound, ErrVersionMismatch, …)
  project.json              # Nx targets: build/serve/dev/test/lint/format, docker-*
  go.mod                    # module github.com/qodesrl/gardbase/apps/api
```

Built by `Dockerfile.Api` at the repo root (multi-stage golang:1.25-alpine → alpine, `CGO_ENABLED=1` because vsock needs cgo, runs as non-root `appuser`, exposes 80/443).

Route table (all under `/api`):

| Group | Endpoint | Guard |
| --- | --- | --- |
| health | `GET /health/`, `/health/enclave`, `/health/storage`, `/health/kms` | none |
| tenants | `POST /tenants/` | none (bootstrap path) |
| objects | `POST /objects/get-table-hash` | tenant + read\|write |
| objects | `POST /objects/get`, `/scan`, `/query` | tenant + read |
| objects | `POST /objects/put`, `/request-put-large`, `/confirm-put-large`, `/delete`, `/recover` | tenant + write |
| encryption | `POST /encryption/secure-session/{init,unwrap,generate-deks,get-table-iek}`, `/encryption/decrypt` | tenant + crypto |

## Behavior

**Startup.** `main` builds a zap production logger, loads `Config` (PORT, ENVIRONMENT) and `AWSConfig` from env. Storage/KMS identifiers (`S3_BUCKET`, the five `DYNAMO_*_TABLE` vars, `KMS_KEY_ID`) are required — missing ones panic via `getEnvOrPanic`. `USE_LOCALSTACK` swaps in static test credentials and a `BaseEndpoint` override for local development. S3 and DynamoDB connectivity is probed at boot but failures are only *logged*, not fatal. The server then registers routes and blocks on SIGINT/SIGTERM with a 10s graceful shutdown.

**Request pipeline.** Global middleware: zap request logging → zap panic recovery → CORS → in-memory per-IP rate limit. Tenant-scoped groups add `TenantMiddleware` (validates `X-Tenant-ID` against `^[a-z0-9-]{3,64}$`, looks up `X-API-Key` in DynamoDB, puts `tenantId` and `permissions` into both gin context and request context) and then `PermissionMiddleware` for `read` / `write` / `crypto`.

**Enclave transport.** `services.Vsock` keeps a small pool (max 10 idle) of vsock connections to `ENCLAVE_CID` (default 16) : `ENCLAVE_PORT` (default 8080; the deployed enclave listens on 5000 and user_data sets `ENCLAVE_PORT=5000`). Each request is one line of JSON — `{type, payload}` — with a per-call deadline; the response is read with a `bufio.Scanner` and the connection returned to the pool. Any I/O error closes the connection rather than recycling it.

**Secure session flow.** A client opens a session (`/secure-session/init`) by sending its X25519 public key and a nonce; the API forwards it verbatim and returns the enclave's ephemeral public key, session id, expiry, and a fresh attestation document bound to both. Subsequent operations follow one shape:

1. API asks the enclave for a current attestation document.
2. API calls KMS (`GenerateDataKey` or `Decrypt`) passing that document as `Recipient` and an `EncryptionContext` of `{tenant_id, purpose}` where purpose ∈ `MASTER_KEY | DATA_KEY | INDEX_KEY | TABLE_SALT`.
3. API forwards the resulting `CiphertextForRecipient` (readable only inside the enclave) to the enclave, which unwraps it and re-seals it under the *session* key.
4. API returns the session-sealed blob to the client.

`generate-deks` additionally fetches the tenant's KMS-wrapped master key and the table's IEK (creating and persisting a new IEK on first use), so the enclave can return each DEK sealed three ways: under the session key (for the client now), under the tenant master key (durable), and as the raw KMS ciphertext blob. `count` is clamped to 1–100.

**Table hashing.** Table names are never stored in the clear. The client sends the table name encrypted under the session key; the enclave decrypts it and returns `base64url(SHA-256(name || tableSalt))`. That `table_hash` becomes the partition-key component for all subsequent object operations.

**Object storage.** Small payloads go inline in DynamoDB (`encrypted_blob`); payloads over 100 KB use a two-step presigned flow — `request-put-large` returns a presigned PUT URL under `tenant-<id>/<tableHash>/<objectId>/v<version>` (15 min TTL) and `confirm-put-large` verifies the object landed in S3 before writing the DynamoDB record. Writes are optimistically locked: the client sends the intended new `version` and the update is conditioned on the previous one, returning 409 on mismatch. Deletes are soft (status flipped, indexes removed, S3 object tagged `status=deleted` for the 30-day lifecycle rule) and reversible via `/recover`.

**Queries.** Encrypted indexes are stored in a separate table keyed by `TENANT#…#TABLE#…#IDX#<name>` with the index token as a binary sort key, plus a GSI keyed by object id for index cleanup. Equality queries match a deterministic token prefix; range queries append an order-preserving segment and use DynamoDB range conditions.

**Tenant creation.** `POST /api/tenants/` is unauthenticated by design (it's how a tenant comes into existence). It mints a UUID, generates a KMS master key and table salt, persists both as KMS ciphertext blobs, and returns a first API key carrying all three permissions plus the attestation document. The API key is bcrypt-hashed at rest and shown only once.

## Dependencies

- **Runtime/AWS:** `aws-sdk-go-v2` for KMS, DynamoDB (+ attributevalue), S3 and presigning. All KMS calls use `RecipientInfo` with `RSAES_OAEP_SHA_256`.
- **Transport:** `github.com/mdlayher/vsock` (requires cgo and `/dev/vsock` in the container).
- **HTTP:** `gin-gonic/gin`; logging via `go.uber.org/zap`; ids via `google/uuid`; bcrypt via `golang.org/x/crypto`.
- **Workspace packages** (wired through `replace` directives to local paths): `pkg/enclaveproto` (wire format with the enclave), `pkg/api` (client-facing request/response types), `pkg/models` (DynamoDB record shapes and key builders).
- **Peers:** `enclave-service` over vsock; `infrastructure-main` provisions the EC2 host, IAM role, KMS key, S3 bucket, DynamoDB tables and injects every env var through `user_data.sh`.

## Notes

- The API is intentionally *unable* to decrypt. If a change makes it possible for this process to observe a plaintext DEK or object, that's a break of the core security property, not an optimization.
- KMS access is gated in the key policy by `kms:RecipientAttestation:ImageSha384` matching the enclave's PCR0 — unless `enable_debug_mode` is set, in which case the condition degrades to `*`. Debug mode is for development only.
- `CorsMiddleware` sends `Access-Control-Allow-Origin: *` and carries an explicit `todo: restrict this in prod`.
- Rate limiting is a per-process in-memory map (1000 req/min/IP) with no eviction of idle client entries — it does not survive restarts and does not coordinate across instances.
- The deployment is a single EC2 instance, so in-memory state (rate limiter, vsock pool) and the enclave's session store are naturally co-located; horizontal scaling would need all three revisited.
- `TenantHandler.CreateNewAPIKey`, `DeleteAPIKey`, and `ListAPIKeys` exist but are not registered on any route yet.
- The commented-out `prepare_kek` block in `tenants.go` is the placeholder for the planned self-managed / client-held master key mode; master-key recovery is an open TODO.
- `handlers/encryption.go` still uses stdlib `log.Printf` for DEK generation diagnostics, unlike the rest of the app which uses zap.
