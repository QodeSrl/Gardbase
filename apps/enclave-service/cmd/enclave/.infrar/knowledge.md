---
schema_version: 1
id: ef7a89bb-d366-4b97-8659-1e50deae804e
name: enclave-service
node: apps/enclave-service/cmd/enclave
category: app
---
## Purpose

`enclave-service` is the trusted core of Gardbase. It runs inside an AWS Nitro Enclave — a hardware-isolated VM on the same EC2 host as the `api` process, with no network, no persistent storage, and no operator shell access — and it is the only place where plaintext key material ever exists.

Everything it does serves one goal: make the surrounding infrastructure untrusted. It proves its own identity and code measurements via NSM-signed attestation documents, establishes end-to-end encrypted sessions directly with *clients* (the parent API is only a relay), unwraps KMS key material that was encrypted specifically to it, re-seals that material under the session key, and zeroes the plaintext before returning. A compromised API host, a malicious operator, or a leaked database dump yields nothing usable.

The entrypoint is `apps/enclave-service/cmd/enclave/main.go`; handlers and helpers live under `apps/enclave-service/internal/`.

## Structure

```
apps/enclave-service/
  cmd/enclave/main.go            # NSM init, attestation refresher, vsock listener, request dispatch
  internal/handlers/             # one file per request type
    health.go                    #   "health"           → status + uptime
    get_attestation.go           #   "get_attestation"  → cached NSM document
    session_init.go              #   "session_init"     → X25519 handshake + bound attestation
    session_unwrap.go            #   "session_unwrap"   → KMS blobs → session-sealed DEKs
    session_prepare_dek.go       #   "session_prepare_dek" → new DEKs sealed 2 ways + IEK
    session_prepare_iek.go       #   "session_prepare_iek" → table index key, session-sealed
    session_generate_table_hash.go # "session_generate_table_hash" → salted hash of table name
    prepare_kek.go               #   "prepare_kek"      → registered but currently unused
  internal/session/
    store.go                     # in-memory session map (id → key, expiry)
    cleanup.go                   # periodic expiry sweeper
  internal/utils/
    attestation.go               # NSM attestation request + guarded cached document
    deriveSessionKey.go          # X25519 → HKDF-SHA256 → 32-byte ChaCha20-Poly1305 key
    openssl.go                   # CMS decrypt of KMS CiphertextForRecipient via openssl CLI
    connection.go, hash.go, zero.go
  project.json                   # Nx targets: build/serve/test/lint/format, docker-*
  go.mod                         # module github.com/qodesrl/gardbase/apps/enclave-service
```

Built by `Dockerfile.enclave` at the repo root; the image is converted to an EIF on the host by `nitro-cli build-enclave` (see `infrastructure/main/user_data.sh`). The image installs `openssl` because DEK unwrapping shells out to it.

## Behavior

**Boot.** Opens an NSM session, generates a 2048-bit RSA keypair *using the NSM device as the entropy source*, marshals the public key to DER, and requests an initial attestation document that binds that public key. This RSA key is the recipient key for every KMS `Recipient` call the parent makes — it is what makes `CiphertextForRecipient` decryptable here and nowhere else. A background ticker refreshes the attestation document every 4 minutes under a `sync.RWMutex`, so the parent always has a fresh one to hand to KMS (which rejects stale documents).

**Serving.** Listens on vsock port `ENCLAVE_PORT` (default 5000; the deployed host sets 5000 and CID 16) and handles each accepted connection in its own goroutine. The wire protocol is newline-delimited JSON: one `enclaveproto.Request{type, payload}` per line in, one `enclaveproto.Response[T]{success, message, data, error}` per line out. Connections are long-lived (the parent pools them) with a 5-minute read deadline reset after every request. Unknown types return a structured error rather than dropping the connection.

**Session handshake (`session_init`).** Validates the client's 32-byte X25519 public key, generates its own ephemeral keypair, derives a shared secret with `curve25519.X25519`, and runs it through HKDF-SHA256 with info `"gardbase-enclave-session-v1"` to produce a 32-byte key. The session gets a random 16-byte base64 id and a 60-minute TTL, stored in an in-process map. It then requests a *new* attestation document whose `public_key` field is the session's ephemeral public key and whose `nonce` is the client's nonce — this is what lets the client prove the key it's about to encrypt to belongs to a genuine, measured enclave and isn't a replay. The ephemeral private key is zeroed on return.

**Key operations.** Every handler that touches key material follows the same shape: look up and validate the session, build an XChaCha20-Poly1305 AEAD from the session key, CMS-decrypt the incoming `CiphertextForRecipient` with the NSM RSA private key, re-seal the plaintext under the session key with an NSM-sourced 24-byte nonce, and `utils.Zero` the plaintext before responding.

- `session_unwrap` — batch: for each `{object_id, ciphertext}`, unwrap and re-seal using the object id as AEAD associated data (binding a sealed DEK to its object). Per-item failures are reported inside the result list rather than failing the batch.
- `session_prepare_dek` — unwraps the tenant master key, then for each freshly generated DEK produces *both* a session-sealed copy (for the client to use now) and a master-key-sealed copy (durable, stored alongside the object), plus the passthrough KMS ciphertext blob. Also seals the table's IEK. Master key, IEK, and every DEK are zeroed.
- `session_prepare_iek` — same, for the index encryption key alone.
- `session_generate_table_hash` — decrypts the session-encrypted table name and returns `base64url(SHA-256(name || tableSalt))`. The table name never leaves the enclave in the clear.
- `decrypt` — the sessionless path: unwraps a single DEK and returns it in a NaCl box sealed to the client's ephemeral public key, along with the request nonce so the client can match it to its attestation check.

**Nonces and randomness.** Session and master-key nonces are read from the NSM device (`nsmSession.Read`), not the OS RNG — inside an enclave the NSM is the authoritative entropy source. `session_init` and `decrypt` use `crypto/rand` for ephemeral keypairs.

**Shutdown.** SIGINT/SIGTERM cancels the refresher context, closes the NSM session and listener, and exits. Nothing is persisted, by design.

## Dependencies

- `github.com/hf/nsm` — Nitro Secure Module: attestation requests, entropy, and the device-backed RNG used for RSA generation.
- `github.com/mdlayher/vsock` — the only transport. Requires cgo (`CGO_ENABLED=1`) and `/dev/vsock`.
- `golang.org/x/crypto` — `curve25519`, `hkdf`, `chacha20poly1305` (XChaCha20-Poly1305), `nacl/box`.
- `pkg/enclaveproto` (via a local `replace` directive) — the request/response types shared with the parent API. Any change there is a wire-protocol change affecting both sides.
- **External binary:** `openssl`, invoked as a subprocess to CMS-decrypt KMS `CiphertextForRecipient` blobs. It must be present in the image.
- **Peers:** the `api` parent process is the sole vsock client; `infrastructure-main` builds and launches the EIF and pins the KMS key policy to this image's PCR0.

## Notes

- This is the security boundary. Anything that would let plaintext key material leave here unsealed — a log line, an error message echoing a buffer, a response field — defeats the whole design. Handlers deliberately return only sealed blobs and nonces.
- `utils.DecryptWithOpenSSL` writes the NSM RSA private key and the ciphertext to temp files and shells out to `openssl cms -decrypt`. It's a stopgap for the lack of a pure-Go CMS/PKCS#7 decryptor; it costs a process spawn per DEK (so a 100-DEK batch spawns 100 processes) and briefly materializes the private key on the enclave's ephemeral filesystem. Replacing it with an in-process implementation is the obvious hardening/perf win.
- Sessions live only in this process's memory. Restarting the enclave invalidates every session, and the design assumes a single enclave per host — sessions do not replicate.
- `session.StartSessionCleanup` exists but is never called from `main`, so expired entries are filtered on read (`GetSession` rejects them) but are never evicted from the map.
- `prepare_kek` is dispatched but unreachable in practice: the parent's call site is commented out. It is the placeholder for the planned self-managed / client-held master key mode.
- `main.go` defines an unused `getEnv` helper.
- Changing anything that alters the enclave image changes its PCR0, which invalidates the `enclave_pcr0_sha384` pinned in the KMS key policy — redeploys must re-extract PCRs (`/opt/gardbase/extract-pcrs.sh` writes them to SSM) and update the Terraform variable, or KMS will refuse every call.
