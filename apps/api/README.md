# Gardbase API

Go HTTP API for tenant provisioning, encrypted objects and indexes, S3 upload coordination, and enclave sessions. Runs on the EC2 parent instance and communicates with the Nitro Enclave over vsock.

[Project overview](../../README.md) · [Deployment](../../infrastructure/README.md) · [Enclave service](../enclave-service/README.md) · [Request types](../../pkg/api)

## Build and run

Use the Go toolchain declared in [`../../go.work`](../../go.work), currently 1.27.1. From the repository root:

```bash
pnpm exec nx build @gardbase/api
# Or build without Node.js / Nx:
(cd apps/api && go build -o ../../dist/apps/api/main ./cmd/server)
```

To run on a configured Linux parent host, set the environment below and use `go run ./cmd/server` from `apps/api`, or run the built binary. The API initializes its vsock connection at startup; an enclave must be available. Localstack configuration changes AWS client endpoints but does not replace the enclave or Nitro Secure Module.

## Configuration

Values are read in [`cmd/server/main.go`](cmd/server/main.go).

| Variable                      | Required / default         | Purpose                                                                  |
| ----------------------------- | -------------------------- | ------------------------------------------------------------------------ |
| `S3_BUCKET`                   | Required                   | Encrypted large-object bucket                                            |
| `DYNAMO_OBJECTS_TABLE`        | Required                   | Object metadata and inline ciphertext                                    |
| `DYNAMO_INDEXES_TABLE`        | Required                   | Query indexes                                                            |
| `DYNAMO_TABLE_CONFIGS_TABLE`  | Required                   | Table configuration                                                      |
| `DYNAMO_TENANT_CONFIGS_TABLE` | Required                   | Tenant configuration                                                     |
| `DYNAMO_API_KEYS_TABLE`       | Required                   | API-key records                                                          |
| `KMS_KEY_ID`                  | Required                   | KMS key identifier                                                       |
| `AWS_REGION`                  | `eu-central-1`             | AWS SDK region                                                           |
| `PORT`                        | `80`                       | HTTP listener port                                                       |
| `ENVIRONMENT`                 | `development`              | Gin release mode when set to `production`                                |
| `BASE_URL`                    | `https://api.gardbase.com` | Public origin; the server appends `/api`                                 |
| `ENCLAVE_CID`                 | `16`                       | Enclave context ID                                                       |
| `ENCLAVE_PORT`                | `8080` in API code         | Enclave vsock port; set explicitly to match the enclave                  |
| `AWS_MAX_RETRIES`             | `3`                        | AWS SDK retry limit                                                      |
| `AWS_REQUEST_TIMEOUT`         | `5` seconds                | Loaded configuration field; not currently applied to AWS client timeouts |
| `USE_LOCALSTACK`              | `false`                    | Use local AWS service endpoints and test credentials                     |
| `LOCALSTACK_URL`              | `http://localhost:4566`    | Local AWS endpoint                                                       |

The enclave defaults to port `5000`; the deployment script explicitly configures the API to use `5000`. Match both sides when running manually. AWS credentials come from the default SDK credential chain outside Localstack mode. On EC2, use the configured instance role.

## Authentication and permissions

Object and encryption routes require:

```http
X-Tenant-ID: <tenant-id>
X-API-Key: <api-key>
Content-Type: application/json
```

The tenant middleware checks both headers and looks up the API key. Permission checks require **every** listed permission, so `get-table-hash` needs both `read` and `write`.

Health routes and `POST /api/tenants/` currently have no tenant middleware. Limit access to tenant provisioning and diagnostic routes at the deployment boundary. API-key management handler methods exist, but creation/deletion/listing routes for them are not registered in the current router.

## Routes

The registered routes are defined in [`setupRoutes`](cmd/server/main.go). All paths below include the `/api` prefix.

| Method | Path                                           | Permission           | Purpose                                                |
| ------ | ---------------------------------------------- | -------------------- | ------------------------------------------------------ |
| GET    | `/api/health/`                                 | No tenant middleware | API liveness                                           |
| GET    | `/api/health/enclave`                          | No tenant middleware | Enclave health                                         |
| GET    | `/api/health/storage`                          | No tenant middleware | S3 and DynamoDB health                                 |
| GET    | `/api/health/kms`                              | No tenant middleware | KMS health                                             |
| POST   | `/api/tenants/`                                | No tenant middleware | Create tenant and initial API key                      |
| POST   | `/api/objects/get-table-hash`                  | `read` + `write`     | Resolve an encrypted table name                        |
| POST   | `/api/objects/get`                             | `read`               | Fetch an object                                        |
| POST   | `/api/objects/scan`                            | `read`               | Paginated table scan                                   |
| POST   | `/api/objects/query`                           | `read`               | Query encrypted index tokens                           |
| POST   | `/api/objects/put`                             | `write`              | Create or update an inline encrypted object            |
| POST   | `/api/objects/request-put-large`               | `write`              | Request a presigned S3 upload                          |
| POST   | `/api/objects/confirm-put-large`               | `write`              | Confirm uploaded ciphertext and metadata               |
| POST   | `/api/objects/delete`                          | `write`              | Soft-delete an object                                  |
| POST   | `/api/objects/recover`                         | `write`              | Recover an object                                      |
| POST   | `/api/encryption/secure-session/init`          | `crypto`             | Start an enclave session                               |
| POST   | `/api/encryption/secure-session/unwrap`        | `crypto`             | Unwrap through an established session                  |
| POST   | `/api/encryption/secure-session/generate-deks` | `crypto`             | Generate session-sealed DEKs                           |
| POST   | `/api/encryption/secure-session/get-table-iek` | `crypto`             | Obtain a table index key through a session             |
| POST   | `/api/encryption/decrypt`                      | `crypto`             | Unwrap KMS material and seal it to a client public key |

## Request examples

Check an existing deployment (API health does not verify every dependency):

```bash
curl --fail-with-body "$GARDBASE_ORIGIN/api/health/"
```

After resolving a table hash, scan encrypted records with an API key that has `read` permission:

```bash
curl --fail-with-body "$GARDBASE_ORIGIN/api/objects/scan" \
  -H 'Content-Type: application/json' \
  -H "X-Tenant-ID: $GARDBASE_TENANT_ID" \
  -H "X-API-Key: $GARDBASE_API_KEY" \
  --data '{"table_hash":"<resolved-table-hash>","limit":20}'
```

Set `GARDBASE_ORIGIN` to your deployment origin, and supply your tenant/key securely. Replace the table-hash placeholder before sending. A scan returns encrypted records; it does not decrypt them. Treat API keys as credentials and avoid recording them in logs.

See [`pkg/api/objects/requests.go`](../../pkg/api/objects/requests.go) for object and query bodies. Binary `[]byte` fields are base64 strings in JSON. Query operators are numeric: `0` equality, `1` less-than, `2` less-than-or-equal, `3` greater-than, `4` greater-than-or-equal, and `5` between. Range queries require the matching token encoding and index structure; they do not accept raw plaintext comparison values.

## Object and crypto workflows

For small objects, send the encrypted payload, wrapped DEKs, nonce, and optional indexes to `put`. For larger objects, request a presigned URL, upload ciphertext to S3, and confirm the object record. The API currently issues presigned URLs with a 15-minute TTL. Consult [`objects.go`](internal/handlers/objects.go) for version and size validation.

The session client in [`pkg/crypto`](../../pkg/crypto/README.md) expects an endpoint ending in `/api/encryption`. The caller configures attestation verification and expected PCRs. Object encryption and index-token generation are client responsibilities.

## Validation

```bash
# From the repository root, with the matching Go toolchain:
pnpm exec nx test @gardbase/api
```

Build checks do not exercise AWS permissions, the enclave, or live key exchange. Validate those separately against a configured deployment. The complete trust-boundary discussion is in the [root security model](../../README.md#security-model).
