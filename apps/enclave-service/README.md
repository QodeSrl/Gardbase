# Gardbase enclave service

Go service that runs inside an AWS Nitro Enclave. It uses the Nitro Secure Module (NSM) for attestation and exposes key/session operations to the parent API over vsock. It has no public HTTP listener.

[Architecture](../../README.md#architecture) · [API service](../api/README.md) · [Deployment](../../infrastructure/README.md) · [Protocol types](../../pkg/enclaveproto)

## Responsibilities

- Generate attestation documents binding enclave keys and request nonces.
- Open recipient-encrypted key material supplied by the API's KMS calls.
- Establish client sessions and return session-sealed DEKs and table index keys.
- Derive table hashes and support key-unwrapping operations.
- Keep session state in enclave memory and clean up expired sessions.

The parent API makes AWS network requests. KMS returns material encrypted to the attested enclave recipient; the parent relays that material over vsock. Object content encryption normally happens in the integrating application.

## Build

Use the Go toolchain in [`go.work`](../../go.work), currently 1.27.1. From the repository root:

```bash
pnpm exec nx build @gardbase/enclave-service
# Or without Node.js / Nx:
(cd apps/enclave-service && go build -o ../../dist/apps/enclave-service/main ./cmd/enclave)

# Linux container image used to build an enclave image file (EIF):
pnpm exec nx run @gardbase/enclave-service:docker-build
```

A local binary build does not create an enclave. At startup the service opens the NSM device, generates an RSA recipient key, creates an attestation, and listens on vsock. Running it as an ordinary laptop process is not a substitute for the Nitro environment. The Docker build is defined in [`Dockerfile.enclave`](../../Dockerfile.enclave); host startup and EIF creation are in [`user_data.sh`](../../infrastructure/main/user_data.sh).

## Configuration

`ENCLAVE_PORT` defaults to **5000**. The API code defaults to **8080**, so set the parent API's `ENCLAVE_PORT` explicitly to `5000` or configure both processes to use another matching port. The supplied host startup script sets the API's port to `5000` and its enclave CID to `16`.

Attestation refresh and session behavior are implemented in [`cmd/enclave/main.go`](cmd/enclave/main.go) and [`internal/session`](internal/session). Configure client attestation-age and PCR verification in the integrating application's [`SessionConfig`](../../pkg/crypto/kms.go).

## Protocol

The service dispatches the following message types from the parent:

| Type                          | Purpose                                                              |
| ----------------------------- | -------------------------------------------------------------------- |
| `health`                      | Enclave health information                                           |
| `get_attestation`             | Recipient attestation document                                       |
| `prepare_kek`                 | Prepare key material sealed to a client public key                   |
| `session_init`                | Establish a client session                                           |
| `session_generate_table_hash` | Resolve a table hash through a session                               |
| `session_unwrap`              | Unwrap and reseal key material through a session                     |
| `session_prepare_dek`         | Prepare object DEKs and index-key material                           |
| `session_prepare_iek`         | Prepare a table index key                                            |
| `decrypt`                     | Open recipient-encrypted material and seal it to a client public key |

See [`pkg/enclaveproto`](../../pkg/enclaveproto) for payload definitions. The parent transport handles framing, request correlation, and timeouts in [`enclaveVSock.go`](../api/internal/services/enclaveVSock.go).

## Deployment checks

Build the intended enclave image, obtain its PCR measurements with `nitro-cli describe-eif`, and configure both the KMS policy and client verification with those measurements. The checked-in Terraform PCR value is a placeholder. Keep debug mode disabled for a deployment that relies on verified code identity.

After deployment, the parent API exposes `/api/health/enclave` for connectivity checks. That check is separate from a client verifying the enclave's identity and exercising a key session. See the [deployment guide](../../infrastructure/README.md) for the full sequence.
