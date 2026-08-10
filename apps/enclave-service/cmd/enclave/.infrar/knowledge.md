---
schema_version: 1
id: fc77e954-536d-4ab8-8324-621b02d75c49
name: enclave-service
node: apps/enclave-service/cmd/enclave
category: app
---
## Purpose

`enclave-service` is the trusted core of Gardbase. It runs inside an AWS Nitro Enclave — a hardware-isolated VM on the same EC2 host as the API, with no network interfaces, no persistent storage, and no operator shell — and it is the only place in the system where key material exists in plaintext.

It exists to make the "the backend cannot read your data" claim mechanically true rather than a policy promise. Three properties do that work:

1. **Attestation.** The Nitro Secure Module (NSM) signs a document containing the enclave's PCR measurements (a hash of the exact image running) and a public key of the enclave's choosing. AWS KMS accepts that document as a recipient credential, and clients verify it independently before trusting anything the enclave says.
2. **Recipient encryption.** Because KMS is given the attestation document, it returns key material encrypted to the enclave's NSM-generated RSA key. The parent API relays that ciphertext but cannot open it.
3. **Session sealing.** Before returning anything, the enclave re-seals key material under a session key derived from an X25519 exchange with the *client*. The API sees only sealed blobs on the way back out too.

`cmd/enclave` is the entrypoint: NSM setup, attestation lifecycle, the vsock listener, and the request-type dispatch table. Handler logic lives in `apps/enclave-service/internal/`.

## Structure

The node directory holds a single file, `main.go`:

- Package-level state — `startTime`, the `*nsm.Session`, the enclave's RSA private key and marshalled public key, and an `utils.Attestation` (a `[]byte` document behind an `RWMutex`).
- `main()` — initializes NSM, starts the attestation refresher, opens a vsock listener on `ENCLAVE_PORT` (default `5000`), installs a signal handler, and accepts connections in a loop, handling each in its own goroutine.
- `initiateNSM()` — opens the default NSM session, generates a 2048-bit RSA key *using the NSM session as the entropy source*, marshals the public key to DER, logs its SHA-256 fingerprint, and fetches the first attestation document.
- `refreshAttestation()` / `startAttestationRefresher()` — re-request the attestation document every 4 minutes and swap it under the mutex.
- `handleConnection()` — the per-connection loop: newline-delimited JSON in, dispatch on `req.Type`, JSON out.
- `getEnvUint32()` — the port lookup helper. (A sibling `getEnv` is defined but unused.)

Supporting packages:

- `internal/handlers/` — one file per request type: `health.go`, `get_attestation.go`, `session_init.go`, `session_unwrap.go`, `session_prepare_dek.go`, `session_prepare_iek.go`, `session_generate_table_hash.go`, `decrypt.go`, `prepare_kek.go`.
- `internal/session/` — `store.go` (a package-level `map[string]*SessionEntry` of session key + expiry, behind an `RWMutex`) and `cleanup.go` (a background sweeper for expired entries).
- `internal/utils/` — `attestation.go` (NSM attestation request), `deriveSessionKey.go` (X25519 + HKDF-SHA256), `openssl.go` (CMS decryption of KMS recipient blobs), `hash.go` (salted SHA-256, base64url), `connection.go` (`SendResponse` / `SendError`), `zero.go` (memory wipe).

Wire types come from `pkg/enclaveproto`, resolved by a `replace` directive to `../../pkg/enclaveproto`.

## Behavior

**Startup.** Record start time → open the NSM session → generate the RSA-2048 keypair → fetch and cache an attestation document bound to that public key → start the 4-minute refresher → listen on vsock. Any failure in NSM initialization is fatal.

**Transport.** vsock only — there is no TCP listener and the enclave has no network stack. Each accepted connection gets a 5-minute read deadline, reset after every request, and is served by a goroutine that reads newline-delimited JSON `enclaveproto.Request` values (`{type, payload}`) and writes JSON responses. Multiple requests are multiplexed over the same connection, which is what lets the parent pool connections.

**Dispatch table** — `req.Type` selects a handler:

| Type | What it does |
| --- | --- |
| `health` | Returns status and uptime. |
| `get_attestation` | Returns a copy of the cached attestation document. |
| `session_init` | Establishes a client session (below). |
| `session_unwrap` | Re-seals a batch of DEKs from KMS recipient blobs to the session key. |
| `session_prepare_dek` | Seals freshly generated DEKs under both the session key and the tenant master key, plus the table IEK. |
| `session_prepare_iek` | Seals an existing table index key under the session key. |
| `session_generate_table_hash` | Decrypts a session-encrypted table name and returns a salted hash. |
| `decrypt` | Single-shot DEK unwrap sealed with NaCl box to a client ephemeral key. |
| `prepare_kek` | Placeholder for self-managed keys; currently unused. |

Unknown types get a JSON error response and the connection stays open.

**Session establishment (`session_init`).** The client sends a 32-byte X25519 public key and a nonce. The enclave generates its own ephemeral X25519 keypair, computes the shared secret, and derives a 32-byte ChaCha20-Poly1305 key via HKDF-SHA256 with info `"gardbase-enclave-session-v1"`. It stores that key in memory against a random 16-byte base64 session ID with a **60-minute TTL**, then requests a *fresh* attestation document from NSM bound to the session's ephemeral public key and the client's nonce. The response carries the session ID, the enclave's ephemeral public key, the expiry, and that attestation document. The ephemeral private key and the shared secret are zeroed.

That per-session attestation is what makes the channel trustworthy end-to-end: the client can check that the public key it just did a key exchange with is the one a genuine, correctly-measured enclave attested to, with its own fresh nonce proving the document is not a replay.

**Unwrapping (`session_unwrap`).** For each item: look up the session (rejecting expired or unknown IDs), CMS-decrypt the KMS `CiphertextForRecipient` blob with the NSM RSA private key to recover the plaintext DEK, draw a 24-byte XChaCha20 nonce from NSM, seal the DEK under the session key **with the object ID as associated data**, zero the plaintext, and append the result. Per-item failures are reported inline rather than failing the batch.

**DEK preparation (`session_prepare_dek`).** Same unwrap step, but each DEK is sealed twice: once under the session key (for the client to use now) and once under the tenant master key (for durable storage). The master key itself arrives as a KMS recipient blob and is CMS-decrypted inside the enclave. The table IEK is sealed under the session key as well. Master key, IEK, and every plaintext DEK are zeroed before returning.

**Table hashing (`session_generate_table_hash`).** Opens the session-encrypted table name with the session AEAD, then returns `base64url(SHA256(tableName || tableSalt))`. The salt arrives from KMS as a recipient blob decrypted by the caller's flow. This is how storage-level table identifiers are derived without the server ever learning table names.

**Single-shot decrypt (`decrypt`).** For clients not holding a session: CMS-decrypt the KMS recipient blob, generate a fresh NaCl box keypair, and seal the DEK to the client's ephemeral public key. The response includes the enclave's box public key, the sealed ciphertext (nonce-prefixed), and the request nonce echoed back.

**Randomness.** Nonces and the RSA key use the NSM session as the entropy source (`nsmSession.Read`, `rand.GenerateKey(nsmSession, …)`), which draws from the Nitro hardware RNG rather than the guest's `/dev/urandom`. `decrypt.go` and `session_init.go` use `crypto/rand` for the NaCl keypair and session ID.

**Hygiene.** Every handler calls `utils.Zero` on plaintext key material before returning. Sessions live only in enclave RAM and vanish on restart.

**Shutdown.** SIGINT/SIGTERM cancels the refresher context, closes the NSM session and the listener, and calls `os.Exit(0)`.

## Dependencies

**Go module** — `github.com/qodesrl/gardbase/apps/enclave-service`, Go 1.24.4, part of the root `go.work` workspace. A deliberately small dependency set, since everything here is inside the trust boundary.

- `github.com/hf/nsm` — Nitro Secure Module driver: attestation requests and hardware entropy.
- `github.com/mdlayher/vsock` — the AF_VSOCK listener. **Requires CGO**, hence `CGO_ENABLED=1` in the Dockerfile.
- `golang.org/x/crypto` — `chacha20poly1305` (XChaCha20-Poly1305 sealing), `curve25519` (X25519), `hkdf`, `nacl/box`.
- `github.com/qodesrl/gardbase/pkg/enclaveproto` via `replace ../../pkg/enclaveproto`.
- Transitively, `fxamacker/cbor` and `x448/float16` through the NSM library.

**Runtime dependencies:**

- A Nitro Enclave with `/dev/nsm` available; NSM initialization is fatal on failure.
- The **`openssl` binary on `PATH`** — `utils.DecryptWithOpenSSL` shells out to `openssl cms -decrypt` to open KMS recipient blobs. `Dockerfile.enclave` installs it explicitly.
- A vsock peer (the parent API) that can dial the enclave CID and port.
- `ENCLAVE_PORT` — the only environment variable read; defaults to `5000`.

**Build and packaging.** Nx targets in `apps/enclave-service/project.json` mirror the API's: `build`, `serve`, `dev`, `test`, `lint`, `format`, `tidy`, and the Docker chain pushing to the `latest-enclave` ECR tag. The image comes from the repo-root `Dockerfile.enclave` (Go 1.25 Alpine builder, `GOWORK=off`, `CGO_ENABLED=1`; Alpine runtime with `openssl`; non-root `appuser`).

On the host, `infrastructure/main/user_data.sh` pulls that image and runs `nitro-cli build-enclave` to turn it into an EIF, then `nitro-cli run-enclave --enclave-cid 16` with the CPU and memory allocation from Terraform. The PCR measurements printed by `build-enclave` are extracted to `/opt/gardbase/pcr-values.json` and pushed to SSM Parameter Store, where clients pick them up for attestation verification.

**Counterparties:** the parent API (`apps/api`) on one side, and `pkg/crypto` on the other — the SDK that verifies these attestation documents and unseals what the enclave returns.

## Notes

- **`session.StartSessionCleanup` is never called.** The sweeper exists in `internal/session/cleanup.go` but no code invokes it. `GetSession` still rejects expired entries, so no expired session key is usable, but the map grows unboundedly with dead entries for the lifetime of the enclave — a memory leak that also keeps expired session keys resident in enclave RAM.
- **Shelling out to `openssl` for CMS decryption writes key material to temp files.** `DecryptWithOpenSSL` marshals the NSM RSA private key to PEM and writes both it and the ciphertext to `os.CreateTemp` files, then execs `openssl cms -decrypt`. Files are removed via `defer` but not overwritten, and the plaintext DEK arrives back through `CombinedOutput` rather than being wiped. This is the weakest link in an otherwise careful memory-hygiene story; it exists because Go's standard library has no CMS/PKCS#7 implementation. It is also a per-call process spawn on the hot path.
- **A crash or restart invalidates every session.** Session state is process-local and in-memory only, which is the intended security property but means clients must handle re-initialization.
- **The attestation refresh interval (4 minutes) is tuned against `max_attestation_age_minutes` (default 5) in Terraform.** They are configured independently; if the max age is lowered below the refresh interval, the cached document served by `get_attestation` will intermittently be rejected by clients.
- **Errors are returned as free-text strings.** `utils.SendError` produces `{success: false, error: "..."}` with the underlying message, which crosses the trust boundary into the parent's logs and, in most API handlers, into HTTP responses.
- `prepare_kek` is dispatched but marked in-source as unimplemented, reserved for advanced self-managed keys.
- `main.go` defines `getEnv` but never uses it.
- A per-connection read deadline of 5 minutes with no cap on concurrent connections means the enclave's connection goroutine count is bounded only by what the parent opens; the parent's pool caps idle connections at 10, but that is a client-side convention, not an enclave-side limit.
