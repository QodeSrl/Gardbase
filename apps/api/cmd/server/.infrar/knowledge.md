---
schema_version: 1
id: 24de065a-7aaa-43d2-ba85-2e8ec8d1ee7b
name: api
node: apps/api/cmd/server
category: app
---
## Purpose

`api` is the Gardbase parent application: the public HTTP API for a zero-trust, fully encrypted NoSQL DBaaS. It is the only component clients talk to over the network, and it deliberately never holds plaintext data or unwrapped keys.

Its job is orchestration, not cryptography:

- terminate client HTTP requests, authenticate them per tenant, and enforce coarse permissions;
- persist already-encrypted blobs and encrypted index tokens in DynamoDB and S3;
- call AWS KMS with a Nitro Enclave attestation document so that key material is returned *only* in a form the enclave can decrypt (`CiphertextForRecipient`);
- proxy those KMS ciphertexts to the enclave service over vsock, and hand the enclave's session-sealed results back to the client.

The design intent is that compromising this process yields ciphertext and KMS-wrapped keys, but no usable plaintext — the plaintext key path runs client → enclave, with the API as an untrusted courier.

## Structure

The node entrypoint is `apps/api/cmd/server` (`main.go`); the implementation lives under `apps/api/internal`.

- `cmd/server/main.go` — process wiring. Defines `Server` (gin engine + zap logger + `Config`) and `AWSConfig`, loads configuration purely from environment variables, constructs the S3 / DynamoDB / KMS clients, registers every route in `setupRoutes`, and runs `start()` with a 10s graceful-shutdown window on SIGINT/SIGTERM. Env helpers at the bottom (`getEnv`, `getEnvOrPanic`, `getEnvAsInt`, `getEnvAsBool`, `getEnvUint32`) — `getEnvOrPanic` is what makes the required AWS resource names hard failures at boot.
- `internal/handlers/` — one file per route group:
  - `healthCheck.go` — `/api/health`, `/health/enclave`, `/health/storage`, `/health/kms`.
  - `tenants.go` — tenant creation (generates tenant ID, master key, table salt, first API key).
  - `objects.go` — the data plane: `get-table-hash`, `get`, `scan`, `query`, `put`, `request-put-large`, `confirm-put-large`, `delete`, `recover`.
  - `encryption.go` — the key plane: secure-session init/unwrap/generate-deks/get-table-iek and one-shot `decrypt`.
- `internal/middleware/` — `zapLogger.go` (request logging + panic recovery), `cors.go`, `ratelimit.go`, `tenant.go` (API-key auth, populates `tenantId`/`permissions`), `permission.go` (checks required permissions against that context).
- `internal/services/` — `enclaveVSock.go` (`Vsock` + `VsockPool`, newline-delimited JSON over vsock) and `kms.go` (`KMS` wrapper that always passes a `RecipientInfo` attestation document).
- `internal/storage/` — `dynamo.go` (the bulk of the logic: objects, indexes, tenants, API keys, table configs, scan/query/pagination), `s3.go` (presigned PUT/GET, existence check, deletion tagging), `errors.go` (sentinel errors like `ErrVersionMismatch`, `ErrNotFoundOrDeleted`).

Wire types are not defined here — they come from the shared modules `pkg/api` (HTTP request/response structs), `pkg/enclaveproto` (vsock protocol), and `pkg/models` (DynamoDB item shapes, permission constants).

Build and packaging: Nx project `@gardbase/api` (`apps/api/project.json`) with `build`, `serve`, `dev` (air), `test`, `lint`, `format`, `tidy`, plus the ECR docker targets. The container is built from the repo-root `Dockerfile.Api`, which builds with `GOWORK=off` and `CGO_ENABLED=1` (required for vsock) and ships an Alpine image running as UID 1000.

## Behavior

**Routing.** Everything is under `/api`. Health endpoints are unauthenticated. `/api/tenants` is currently unauthenticated too — creating a tenant is the bootstrap that mints the first API key. `/api/objects/*` and `/api/encryption/*` sit behind `TenantMiddleware`, then `PermissionMiddleware`: reads need `read`, writes need `write`, `get-table-hash` accepts either, and the whole encryption group needs `crypto`.

**Authentication.** `TenantMiddleware` requires `X-Tenant-ID` (validated against `^[a-z0-9-]{3,64}$`) and `X-API-Key`. It resolves the key via `DynamoClient.FindAPIKey`, which lists the tenant's keys and bcrypt-compares — so auth cost grows with the number of keys a tenant holds. On success it stores `tenantId` and `permissions` in both the gin context and the request context.

**Key flow (the important part).** For every operation that touches key material the API:
1. asks the enclave for a fresh attestation document over vsock (`RequestAttestationDocument`);
2. calls KMS `GenerateDataKey` or `Decrypt` with that document as `Recipient` and an encryption context of `{tenant_id, purpose}` where purpose is one of `MASTER_KEY`, `DATA_KEY`, `INDEX_KEY`, `TABLE_SALT`;
3. forwards the resulting `CiphertextForRecipient` (RSA/CMS-encrypted to the enclave's NSM key) to the enclave;
4. returns the enclave's session-sealed output to the client.

The API stores `CiphertextBlob` (the ordinary KMS-wrapped form) and never sees a plaintext key, because the recipient form is only decryptable inside the enclave.

**Objects.** Blobs under ~100KB are stored inline in DynamoDB (`encrypted_blob`); larger ones use a two-step flow — `request-put-large` returns a 15-minute presigned S3 PUT URL for key `tenant-<id>/<tableHash>/<objectId>/v<version>`, then `confirm-put-large` verifies the upload with `HeadObject` and writes the DynamoDB record. Concurrency is optimistic: the client sends the new version, and updates are conditioned on `version = current AND status <> deleted`, surfacing `ErrVersionMismatch` on conflict. Object creation writes the object and its index items in a single `TransactWriteItems` when there are ≤25 items total, otherwise a `PutItem` followed by 25-item `BatchWriteItem` batches (that fallback is not atomic). Deletes are soft: the DynamoDB record is marked `deleted`, its index rows are removed, and the S3 object is tagged `status=deleted` so a bucket lifecycle rule reaps it after 30 days; `recover` reverses both.

**Encrypted queries.** Index rows live in a separate table keyed by `pk = TENANT#<t>#TABLE#<hash>#IDX#<name>` and a binary `sk` that is the index token with the object's 16-byte UUID appended (guaranteeing uniqueness for duplicate values). Equality queries use `begins_with(sk, token)`; range queries on hash+range indexes are translated into DynamoDB `BETWEEN` expressions over synthesized lower/upper bounds built from the 32-byte deterministic hash, the 8-byte order-preserving range value, and min/max object IDs — that padding trick is what makes `>`/`>=`/`<`/`<=` exclusive or inclusive. A GSI (`gsi1`, keyed by object) supports finding all index rows for one object, which is how updates diff and reconcile the index set.

**vsock transport.** `Vsock` keeps a pool of up to 10 idle connections to the enclave (default CID 16, port 8080 from `ENCLAVE_CID`/`ENCLAVE_PORT`). Each request writes one JSON line and reads one JSON line back under a per-call deadline; on any I/O error the connection is closed rather than returned to the pool.

**Failure and observability.** Handlers return gin JSON errors, mostly `500` with the underlying message. Structured request logs go to zap (warn on 4xx, error on 5xx); panics are recovered and logged with a stack. At boot the S3 and DynamoDB connectivity probes only *log* failures — the server still starts.

## Dependencies

- **Internal modules** (via `replace` directives to local paths): `pkg/api`, `pkg/enclaveproto`, `pkg/models`. `pkg/crypto` is deliberately *not* a dependency — that is the client's SDK.
- **Sibling node**: `enclave-service`, reachable only over vsock on the same EC2 host. Every key operation and the tenant-creation path fail without it.
- **AWS**: DynamoDB (five tables: objects, indexes, table configs, tenant configs, API keys), S3 (uploads bucket, presigning), KMS (one CMK, attestation-gated).
- **Libraries**: gin, aws-sdk-go-v2 (config, credentials, dynamodb + attributevalue, s3, kms), `mdlayher/vsock`, `google/uuid`, `go.uber.org/zap`, `golang.org/x/crypto` (bcrypt, via pkg/models).
- **Configuration** — all environment variables. Required (panic if missing): `S3_BUCKET`, `DYNAMO_OBJECTS_TABLE`, `DYNAMO_INDEXES_TABLE`, `DYNAMO_TABLE_CONFIGS_TABLE`, `DYNAMO_TENANT_CONFIGS_TABLE`, `DYNAMO_API_KEYS_TABLE`, `KMS_KEY_ID`. Optional: `PORT` (80), `ENVIRONMENT` (development; `production` switches gin to release mode), `AWS_REGION` (eu-central-1), `BASE_URL`, `ENCLAVE_CID` (16), `ENCLAVE_PORT` (8080), `AWS_MAX_RETRIES`, `AWS_REQUEST_TIMEOUT`, `USE_LOCALSTACK`, `LOCALSTACK_URL`.
- **Runtime**: deployed by `infrastructure-main` as a Docker container under systemd (`gardbase-parent.service`) on a Nitro-Enclaves-enabled EC2 instance, with `/dev/vsock` passed through and ports 80/443 published. Note the deployed unit sets `ENCLAVE_PORT=5000` while the code default is 8080 — the env var is what governs.

## Notes

- Localstack support is built in (`USE_LOCALSTACK` swaps in static credentials and a base endpoint for S3/DynamoDB/KMS), but the enclave has no local equivalent, so key-plane endpoints can't be exercised offline.
- CORS is wide open (`Access-Control-Allow-Origin: *`) with a `// todo: restrict this in prod` marker.
- Rate limiting is per-process and in-memory: 1000 requests/minute per client IP, with a map that is appended to but never pruned of idle IPs — it does not survive restarts and does not coordinate across instances.
- `TenantHandler` exposes `CreateNewAPIKey`, `DeleteAPIKey`, and `ListAPIKeys` methods that are not wired to any route yet; only `POST /api/tenants/` is registered.
- Tenant creation has no authentication in front of it, and `HandleCreateTenant` carries `TODO: implement master key recovery mechanism` — the self-managed-key path (`prepare_kek`) is written but commented out on both sides.
- `HandleSessionUnwrap` writes a 500 response when the attestation request fails but does not `return`, so it continues into the KMS loop.
- `Scan` and `Query` responses put the raw `S3Key` into the `GetURL` field rather than a presigned URL; only single-object `Get` presigns.
- The `sensitivity` field on objects is accepted and persisted but marked unused; `MinRangeValue`/`OPERangeValueLength` in `pkg/models` are `TODO`-flagged as an assumed 8-byte width.
