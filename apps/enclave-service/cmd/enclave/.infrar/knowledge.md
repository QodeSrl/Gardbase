---
schema_version: 1
id: 0355739a-68c9-4398-8c58-4b8e3a26319e
name: enclave-service
node: apps/enclave-service/cmd/enclave
category: app
---
## Purpose

`enclave-service` is the trusted core of Gardbase. It runs inside an AWS Nitro Enclave — a hardware-isolated VM on the same EC2 host as the `api` process, with no network, no persistent storage, and no operator access — and it is the only place where plaintext key material ever exists.

Everything it does serves one goal: making the surrounding infrastructure untrusted. It proves its own identity and code measurements through NSM-signed attestation documents, establishes end-to-end encrypted sessions directly with *clients* (the parent API is only a relay), unwraps KMS key material that was encrypted specifically to it, re-seals that material under the session key, and zeroes the plaintext before responding. A compromised API host, a malicious operator, or a leaked database dump yields nothing usable.

The entrypoint is `apps/enclave-service/cmd/enclave/main.go`; handlers and helpers live under `apps/enclave-service/internal/`.

## Structure

```
apps/enclave-service/
  cmd/enclave/main.go              # NSM init, attestation refresher, vsock listener, dispatch
  internal/handlers/               # one file per request type
    health.go                      #   "health"                      → status + uptime
    get_attestation.go             #   "get_attestation"             → cached NSM document
    session_init.go                #   "session_init"                → X25519 handshake + bound attestation
    session_unwrap.go              #   "session_unwrap"              → KMS blobs → session-sealed DEKs
    session_prepare_dek.go         #   "session_prepare_dek"         → new DEKs sealed two ways + IEK
    session_prepare_iek.go         #   "session_prepare_iek"         → table index key, session-sealed
    session_generate_table_hash.go #   "session_generate_table_hash" → salted hash of the table name
    decrypt.go                     #   "decrypt"                     → sessionless DEK unwrap via NaCl box
    prepare_kek.go                 #   "prepare_kek"                 → dispatched but currently unused
  internal/session/
    store.go                       # in-memory session map (id → key, expiry)
    cleanup.go                     # periodic expiry sweeper
  internal/utils/
    attestation.go                 # NSM attestation request + mutex-guarded cached document
    deriveSessionKey.go            # X25519 → HKDF-SHA256 → 32-byte XChaCha20-Poly1305 key
    openssl.go                     # CMS decrypt of KMS CiphertextForRecipient via the openssl CLI
    connection.go, hash.go, zero.go
  project.json                     # Nx targets: build/serve/dev/test/lint/format/tidy, docker-*
  go.mod                           # module github.com/qodesrl/gardbase/apps/enclave-service
```

Built by `Dockerfile.enclave` at the repo root (`CGO_ENABLED=1`, `GOWORK=off`, non-root `appuser`); the resulting image is converted into an EIF on the host by `nitro-cli build-enclave`, driven by `infrastructure/main/user_data.sh`. The runtime image installs `openssl` because DEK unwrapping shells out to it.

## Behavior

**Boot.** Opens an NSM session, generates a 2048-bit RSA keypair *using the NSM device as the entropy source*, marshals the public key to DER, logs its SHA-256 fingerprint, and requests an initial attestation document binding that public key. This RSA key is the recipient key for every KMS `Recipient` call the parent makes — it is what makes `CiphertextForRecipient` decryptable here and nowhere else. A background ticker refreshes the attestation document every 4 minutes under a `sync.RWMutex`, so the parent always has a fresh one to hand to KMS, which rejects stale documents.

**Serving.** Listens on vsock port `ENCLAVE_PORT` (default 5000; the deployment uses port 5000 at CID 16) and handles each accepted connection in its own goroutine. The wire protocol is newline-delimited JSON: one `enclaveproto.Request{type, payload}` per line in, one `enclaveproto.Response[T]{success, message, data, error}` per line out. Connections are long-lived because the parent pools them, with a 5-minute read deadline reset after every request. Unknown request types return a structured error instead of dropping the connection.

**Session handshake (`session_init`).** Validates the client's 32-byte X25519 public key, generates its own ephemeral keypair, derives a shared secret with `curve25519.X25519`, and runs it through HKDF-SHA256 with info `"gardbase-enclave-session-v1"` to produce a 32-byte key. The session gets a random 16-byte base64 id and a 60-minute TTL in an in-process map. It then requests a *new* attestation document whose `public_key` field is the session's ephemeral public key and whose `nonce` is the client's nonce. That binding is what lets the client prove the key it is about to encrypt to belongs to a genuine, measured enclave and is not a replay. The ephemeral private key is zeroed on return.

**Key operations.** Every handler that touches key material follows the same shape: resolve and validate the session, build an XChaCha20-Poly1305 AEAD from the session key, CMS-decrypt the incoming `CiphertextForRecipient` with the NSM RSA private key, re-seal the plaintext under the session key with an NSM-sourced 24-byte nonce, and `utils.Zero` the plaintext before responding.

- `session_unwrap` — batch operation: for each `{object_id, ciphertext}`, unwrap and re-seal using the object id as AEAD associated data, binding a sealed DEK to its object. Per-item failures are reported inside the result list rather than failing the whole batch.
- `session_prepare_dek` — unwraps the tenant master key, then for each freshly generated DEK produces both a session-sealed copy (for the client to use now) and a master-key-sealed copy (durable, stored alongside the object), plus the pass-through KMS ciphertext blob. It also seals the table's IEK. Master key, IEK, and every DEK are zeroed.
- `session_prepare_iek` — the same, for the index encryption key alone.
- `session_generate_table_hash` — decrypts the session-encrypted table name and returns `base64url(SHA-256(name ‖ tableSalt))`. The table name never leaves the enclave in the clear.
- `decrypt` — the sessionless path: unwraps a single DEK and returns it in a NaCl box sealed to the client's ephemeral public key, together with the request nonce so the client can match it against its attestation check.

**Nonces and randomness.** Session, master-key, DEK, and IEK nonces are read from the NSM device (`nsmSession.Read`) rather than the OS RNG — inside an enclave the NSM is the authoritative entropy source. `session_init` and `decrypt` use `crypto/rand` for ephemeral keypairs and box nonces.

**Shutdown.** SIGINT/SIGTERM cancels the refresher context, closes the NSM session and the listener, and exits. Nothing is persisted, by design.

## Dependencies

- `github.com/hf/nsm` — Nitro Secure Module: attestation requests, entropy, and the device-backed RNG used for RSA key generation.
- `github.com/mdlayher/vsock` — the only transport. Requires cgo (`CGO_ENABLED=1`) and `/dev/vsock`.
- `golang.org/x/crypto` — `curve25519`, `hkdf`, `chacha20poly1305` (XChaCha20-Poly1305), `nacl/box`.
- `pkg/enclaveproto`, wired through a local `replace` directive — the request/response types shared with the parent API. Any change there is a wire-protocol change affecting both sides.
- **External binary:** `openssl`, invoked as a subprocess to CMS-decrypt KMS `CiphertextForRecipient` blobs. It must be present in the image.
- **Peers:** the `api` parent process is the sole vsock client; `infrastructure-main` builds and launches the EIF and pins the KMS key policy to this image's PCR0.

## Notes

- This is the security boundary. Anything that lets plaintext key material leave here unsealed — a log line, an error message echoing a buffer, an extra response field — defeats the design. Handlers deliberately return only sealed blobs and nonces.
- `utils.DecryptWithOpenSSL` writes the NSM RSA private key and the ciphertext to temp files and shells out to `openssl cms -decrypt`. It is a stopgap for the lack of a pure-Go CMS/PKCS#7 decryptor: it costs a process spawn per DEK (a 100-DEK batch spawns 100 processes) and briefly materializes the private key on the enclave's ephemeral filesystem. Replacing it with an in-process implementation is the obvious hardening and performance win.
- Sessions live only in this process's memory. Restarting the enclave invalidates every session, and the design assumes one enclave per host — sessions do not replicate.
- `session.StartSessionCleanup` exists but is never called from `main`, so expired entries are rejected on read by `GetSession` yet are never evicted from the map.
- `prepare_kek` is dispatched but unreachable in practice: the parent's call site is commented out. It is the placeholder for the planned self-managed / client-held master key mode.
- `main.go` defines an unused `getEnv` helper.
- Changing anything that alters the enclave image changes its PCR0, which invalidates the `enclave_pcr0_sha384` pinned in the KMS key policy. Redeploys must re-extract PCRs (`/opt/gardbase/extract-pcrs.sh` publishes them to SSM) and update the Terraform variable, or KMS will refuse every call.
