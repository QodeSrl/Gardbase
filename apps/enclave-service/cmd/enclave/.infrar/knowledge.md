---
schema_version: 1
id: 656ca69f-3981-4c49-9e54-7901b7277cd9
name: enclave-service
node: apps/enclave-service/cmd/enclave
category: app
---
## Purpose

`enclave-service` is the trusted core of Gardbase. It runs inside an AWS Nitro Enclave — no network, no persistent storage, memory isolated from the parent EC2 host — and is the only place in the system where key material exists in plaintext.

It exists so that the API server can be untrusted. The enclave:

- produces AWS-signed attestation documents proving which code image is running (PCR measurements), which the parent then presents to KMS and clients verify independently;
- establishes end-to-end encrypted sessions directly with *clients*, using ephemeral X25519 keypairs, so the parent cannot read the session traffic it relays;
- decrypts KMS `CiphertextForRecipient` blobs with its NSM-bound RSA key, then immediately re-seals the plaintext under the client's session key before returning it;
- derives table hashes from a tenant salt without ever revealing the salt or the table name to the parent;
- zeroes plaintext key material as soon as it is done with it.

The parent process sees only ciphertext going in and ciphertext coming out.

## Structure

The node entrypoint is `apps/enclave-service/cmd/enclave` (`main.go`); logic lives under `apps/enclave-service/internal`.

- `cmd/enclave/main.go` — the whole server loop. Holds package-level state: `startTime`, the NSM session, a 2048-bit RSA keypair generated *from* the NSM as entropy source, its DER-marshalled public key, and a mutex-guarded cached `Attestation`. Opens the NSM session, starts an attestation refresher, listens on vsock, and dispatches each request by `type` in a switch.
- `internal/handlers/` — one file per message type: `health.go`, `get_attestation.go`, `session_init.go`, `session_unwrap.go`, `session_prepare_dek.go`, `session_prepare_iek.go`, `session_generate_table_hash.go`, `decrypt.go`, `prepare_kek.go`.
- `internal/session/` — `store.go` (in-memory `map[string]*SessionEntry` of session key + expiry, RWMutex-guarded) and `cleanup.go` (`StartSessionCleanup`, a goroutine that periodically evicts expired entries).
- `internal/utils/` — `attestation.go` (`RequestAttestation`, `Attestation` struct), `deriveSessionKey.go` (X25519 + HKDF-SHA256), `openssl.go` (`DecryptWithOpenSSL`, KMS CMS blob decryption by shelling out), `hash.go` (SHA-256 of value‖salt, base64url), `connection.go` (`SendResponse`/`SendError` JSON encoders), `zero.go` (`Zero`).

Wire types come from the shared `pkg/enclaveproto` module. The container image is built from the repo-root `Dockerfile.enclave` (Alpine, includes the `openssl` binary, `CGO_ENABLED=1` for vsock, runs as UID 1000); Nx project `@gardbase/enclave-service` provides build/serve/test/lint and the ECR docker targets. On the host, `nitro-cli build-enclave` converts that Docker image into an EIF whose PCR0 is the measurement clients pin.

## Behavior

**Startup.** `initiateNSM` opens the default NSM session, generates an RSA-2048 keypair using the NSM as the `crypto/rand.Reader`, marshals the public key to PKIX DER, logs its SHA-256 fingerprint, and requests an initial attestation document binding that public key. A ticker refreshes the cached attestation every 4 minutes (KMS rejects stale documents), guarded by an RWMutex so readers never block each other.

**Transport.** `vsock.Listen` on `ENCLAVE_PORT` (default 5000; the deployed systemd unit and the parent both use 5000). Each accepted connection is handled in its own goroutine with a `bufio.Scanner` reading newline-delimited JSON requests and a `json.Encoder` writing responses. The read deadline is 5 minutes and is reset after every request, so idle connections eventually drop while the parent's pooled connections stay alive under load. Unknown request types and malformed JSON produce an error response but do not close the connection. Shutdown on SIGINT/SIGTERM cancels the refresher context, closes the NSM session and listener, and exits.

**Session establishment (`session_init`).** The client sends its X25519 public key and a random nonce (relayed verbatim by the parent). The enclave generates an ephemeral X25519 keypair, computes the shared secret with `curve25519.X25519`, and derives a 32-byte key via HKDF-SHA256 with info `"gardbase-enclave-session-v1"` — the identical derivation lives in `pkg/crypto`, which is what makes the channel truly end-to-end. It stores the session under a random 16-byte base64 ID with a 60-minute TTL, then requests a *fresh* attestation document that binds the ephemeral public key and echoes the client's nonce. The client verifies that document, and the nonce + public-key binding is what defeats replay and man-in-the-middle by the parent. The ephemeral private key is zeroed on return.

**Key operations.** All follow the same shape: look up the session (reject if missing or expired), build an XChaCha20-Poly1305 AEAD from the session key, decrypt the parent-supplied KMS recipient blob with the NSM RSA key, re-seal under the session key with a fresh 24-byte nonce read from the NSM, and zero the plaintext.

- `session_prepare_dek` — decrypts the tenant master key and a batch of DEKs. Each DEK is sealed twice: once under the session key (for the client) and once under the tenant master key (stored as `master_wrapped_dek`, giving a second recovery path independent of KMS). The table's index encryption key (IEK) is sealed under the session key in the same round trip.
- `session_prepare_iek` — the IEK-only variant, for clients that already have DEKs.
- `session_unwrap` — bulk DEK unwrapping for reads. Per item, and notably, the object ID is used as AEAD *associated data*, binding each sealed DEK to its object so a compromised parent cannot swap DEKs between objects. Per-item failures are reported inside the result array rather than failing the whole batch.
- `session_generate_table_hash` — decrypts the client's session-encrypted table name, decrypts the tenant table salt, and returns `base64url(SHA-256(tableName ‖ salt))`. The parent learns only the opaque hash it uses as a partition key.
- `decrypt` — the one-shot path with no prior session: decrypts a DEK and seals it with NaCl `box` to the client's ephemeral public key using a freshly generated enclave keypair.
- `health` — status, timestamp, uptime.
- `get_attestation` — returns a copy of the cached document; this is what the parent attaches to every KMS call.
- `prepare_kek` — implemented but not reachable in practice; `main.go` marks it as reserved for future self-managed keys, and the parent's call site is commented out.

**Decrypting KMS recipient blobs.** KMS returns `CiphertextForRecipient` as a CMS/PKCS#7 enveloped structure, which Go's standard library does not parse. `DecryptWithOpenSSL` therefore writes the ciphertext and a PEM copy of the RSA private key to temp files and shells out to `openssl cms -decrypt`, removing both files afterward. This is why the enclave image installs the `openssl` binary.

## Dependencies

- **Internal**: `pkg/enclaveproto` only (via a local `replace`). Notably it does *not* depend on `pkg/models`, `pkg/api`, or the AWS SDK — the enclave makes no AWS API calls itself.
- **Libraries**: `github.com/hf/nsm` (Nitro Secure Module — attestation and hardware entropy), `github.com/mdlayher/vsock`, `golang.org/x/crypto` (curve25519, hkdf, chacha20poly1305, nacl/box), and `github.com/fxamacker/cbor/v2` transitively.
- **External binary**: `openssl` must be on `PATH` inside the image.
- **Platform**: an AWS Nitro Enclave with `/dev/nsm` available, launched by `nitro-cli run-enclave` with CID 16 and the CPU/memory allocated by the host's `nitro-enclaves-allocator` (defaults 2 vCPU / 2048 MiB from `infrastructure-main`).
- **Peers**: `api` is the only vsock client. `pkg/crypto` is the counterpart implementation on the client side — session key derivation, sealing formats, and attestation verification must stay in lockstep with it.
- **Configuration**: `ENCLAVE_PORT` only (default 5000). There is no other configuration surface by design.

## Notes

- No plaintext key ever leaves this process unsealed, and `utils.Zero` is called on master keys, DEKs, IEKs, and the ephemeral private key after use. Go's GC can still leave copies behind, so this is best-effort rather than a guarantee.
- Sessions live only in enclave memory: restarting the enclave invalidates every session, and there is no cross-instance session sharing.
- `session.StartSessionCleanup` is defined but never called from `main.go`, so expired sessions are rejected on lookup (`GetSession` checks expiry) but their entries are not evicted — the map grows for the lifetime of the process.
- `HandleDecrypt` uses `crypto/rand` for its keypair and nonce, while the session handlers read entropy from the NSM; both are sound, but the inconsistency is worth knowing when reasoning about entropy sources.
- `getEnv` in `main.go` is unused.
- `DecryptWithOpenSSL` writes the RSA private key to a temp file on every call. Inside an enclave the filesystem is ephemeral tmpfs and unreachable from the host, but it is a per-call fork+exec cost on the hot path.
- Attestation refresh (4 min) is tuned against KMS's tolerance for document age; `infrastructure-main` separately exposes `max_attestation_age_minutes` (default 5) for the client-side check in `pkg/crypto`.
- The enclave trusts the parent to supply the correct tenant's wrapped keys. Tenant isolation is enforced by the KMS encryption context (`tenant_id` + `purpose`) set by the parent, not by the enclave itself.
