---
schema_version: 1
id: 645bbf4e-c309-4832-ba62-ed70fa1e2d2e
name: enclave-service
node: apps/enclave-service/cmd/enclave
category: app
---
## Purpose

`enclave-service` is the trusted half of Gardbase. It runs inside an AWS Nitro Enclave on the same EC2 host as the `api` node and is the only place in the system where key material exists in plaintext.

It exists to give clients something they can verify rather than trust:

- produce NSM-signed attestation documents that bind the enclave's public key (and a client-supplied nonce) to the measured code running inside it;
- establish per-client sessions over X25519 + HKDF so the enclave and the client share a key the API server never sees;
- decrypt KMS `CiphertextForRecipient` blobs with the NSM-held RSA private key, then immediately re-seal the plaintext to the client's session key before returning it;
- derive tenant table hashes and index encryption keys without exposing the tenant master key or table salt.

It has no network stack, no persistent storage, and no AWS credentials. Its only interface is a vsock socket to the parent instance, and it treats the parent as untrusted.

## Structure

The node entrypoint is `apps/enclave-service/cmd/enclave` (`main.go`); the rest lives under `apps/enclave-service/internal`. It is a separate Go module (`github.com/qodesrl/gardbase/apps/enclave-service`) that depends only on `pkg/enclaveproto` plus NSM, vsock, and `golang.org/x/crypto`.

- `cmd/enclave/main.go` — process lifecycle. Opens the NSM session, generates the long-lived RSA-2048 keypair used as the KMS recipient key, starts the attestation refresher, listens on vsock, and dispatches each request type to a handler. Also holds the graceful-shutdown goroutine and the env helpers.
- `internal/handlers/` — one file per request type:
  - `health.go` — status and uptime.
  - `get_attestation.go` — returns the cached attestation document.
  - `session_init.go` — ephemeral X25519 handshake, session key derivation, session storage, and a fresh attestation bound to the session key and client nonce.
  - `session_unwrap.go` — batch DEK unwrap, re-sealed per object.
  - `session_prepare_dek.go` — unwraps freshly generated DEKs, seals each to both the session key and the tenant master key, and seals the table IEK.
  - `session_prepare_iek.go` — the IEK-only variant of the above.
  - `session_generate_table_hash.go` — decrypts a session-encrypted table name and hashes it with the tenant table salt.
  - `decrypt.go` — one-shot DEK unwrap sealed with NaCl box instead of a session key.
  - `prepare_kek.go` — the self-managed-key path; wired into the dispatcher but not currently reachable from the API.
- `internal/session/` — `store.go` (in-memory `map[string]*SessionEntry` guarded by an RWMutex) and `cleanup.go` (a background expiry sweeper).
- `internal/utils/` — `attestation.go` (NSM attestation request + the shared `Attestation` cache type), `deriveSessionKey.go` (X25519 → HKDF-SHA256), `openssl.go` (CMS decryption of KMS recipient ciphertext), `hash.go`, `zero.go`, `connection.go` (response/error encoding).

Packaging is external: `apps/enclave-service/project.json` for Nx targets, the repo-root `Dockerfile.enclave` for the container image, and `infrastructure/main/user_data.sh` for the `nitro-cli build-enclave` step that turns that image into an EIF.

## Behavior

**Startup.** `initiateNSM` opens the default NSM session, uses it as the entropy source for a 2048-bit RSA keypair, marshals the public key to DER, logs its SHA-256 fingerprint, and requests a first attestation document embedding that public key. That document is cached in a mutex-guarded `utils.Attestation` and refreshed every 4 minutes by a ticker goroutine, so `get_attestation` never blocks on NSM.

**Transport.** The service listens on vsock port `ENCLAVE_PORT` (default `5000`). Each accepted connection gets its own goroutine that reads newline-delimited JSON `enclaveproto.Request` values with a 5-minute read deadline reset after every request, and writes one `enclaveproto.Response[T]` per request. Unknown types and unmarshal failures produce an error response but keep the connection open.

**Session handshake (`session_init`).** The client's 32-byte X25519 public key arrives via the API. The enclave generates its own ephemeral keypair, derives a 32-byte session key with `X25519 → HKDF-SHA256(info: "gardbase-enclave-session-v1")`, stores it under a random 16-byte base64 session ID with a 60-minute TTL, and requests a *new* attestation document whose `public_key` is the ephemeral session public key and whose `nonce` is the client's nonce. The ephemeral private key is zeroed before returning. The client can therefore verify that the key it just did a handshake with is the one the enclave attested to.

**Key handling.** Everything that arrives from the API is a KMS `CiphertextForRecipient` blob, i.e. a CMS envelope encrypted to the NSM RSA public key. `utils.DecryptWithOpenSSL` opens it by writing the ciphertext and a PEM copy of the private key to temp files and shelling out to `openssl cms -decrypt -inform DER`. The recovered plaintext is then re-protected before it leaves:

- `session_unwrap` seals each DEK with XChaCha20-Poly1305 under the session key, using the object ID as associated data, with a 24-byte nonce drawn from NSM.
- `session_prepare_dek` seals each DEK twice — once to the session key (so the client can use it) and once to the tenant master key (so it can be stored server-side as `master_wrapped_dek`) — and also seals the table IEK to the session key.
- `session_prepare_iek` does the IEK half only.
- `decrypt` generates a throwaway NaCl box keypair and seals the DEK to the client's ephemeral box public key instead of a session key.

Every plaintext buffer (`plainDEK`, `iek`, `masterKey`, shared secrets, ephemeral private keys) is overwritten with `utils.Zero` as soon as it is no longer needed.

**Table hashes.** `session_generate_table_hash` opens the session-encrypted table name with the session AEAD, then returns `base64url(SHA-256(tableName || tableSalt))`. The plaintext table name never leaves the enclave and the API only ever sees the resulting opaque hash.

**Failure model.** Handlers respond with `{"success": false, "error": ...}` rather than closing the connection. `session_unwrap` degrades per item: a failed unwrap marks only that object's result as unsuccessful and the batch continues.

## Dependencies

**Internal:** `pkg/enclaveproto` only (via a `replace` directive) — the shared `Request`/`Response[T]` envelope and one struct pair per message type. Notably it does *not* depend on `pkg/models` or `pkg/api`, which keeps the trusted computing base small.

**Third party:**
- `github.com/hf/nsm` — Nitro Secure Module driver. Used both for attestation requests and, crucially, as the `io.Reader` supplying entropy for RSA key generation and every AEAD nonce.
- `github.com/mdlayher/vsock` — the listener.
- `golang.org/x/crypto` — `curve25519`, `hkdf`, `chacha20poly1305` (XChaCha20-Poly1305), `nacl/box`.
- The `openssl` binary, installed in the runtime image, is a hard runtime dependency of every key-unwrapping path.

**Platform:** an EC2 instance with `enclave_options.enabled`, the Nitro Enclaves allocator configured with the CPU/memory reserved for the enclave, and an EIF built from the container image. `infrastructure/main` provisions all of this; the KMS key policy gates `kms:Decrypt`/`kms:GenerateDataKey` on `kms:RecipientAttestation:ImageSha384` matching this enclave's PCR0.

**Peer:** the `api` node is the only client. `pkg/crypto` implements the verification side of the attestation and session protocol this service speaks.

## Notes

- `session.StartSessionCleanup` is defined but never called from `main.go`. Sessions are still rejected once expired (`GetSession` checks `ExpiresAt`), but expired entries are never evicted, so the session map grows for the life of the enclave process.
- Sessions live only in enclave memory: an enclave restart invalidates every in-flight session, and there is no way to recover one.
- Unwrapping shells out to `openssl` and writes both the ciphertext and the RSA private key to temporary files on every call. That is a per-request process spawn plus disk I/O inside the enclave, and the Go standard library cannot replace it because KMS returns a CMS envelope rather than a bare RSA ciphertext.
- `prepare_kek` is dispatched but unreachable — the corresponding API handler code is commented out pending the "advanced self-managed keys" feature.
- In `session_prepare_dek` and `session_prepare_iek`, `sessNonce`/`masterKeyNonce` are read from NSM at the top; `session_prepare_dek` then draws separate per-DEK nonces and leaves those two unused.
- `main.go` defines `getEnv` but only `getEnvUint32` is used; `ENCLAVE_PORT` is the sole configuration knob.
- The service logs request types and payload sizes to stdout, which is only visible via `nitro-cli console` when the enclave is started in debug mode (`enable_debug_mode`). Debug mode also disables the PCR-based KMS condition, so it must not be used in production.
