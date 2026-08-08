---
schema_version: 1
id: 53b180f2-d38b-46de-b75b-88cd71243777
name: crypto-sdk
node: pkg/crypto
category: other
---
## Purpose

`crypto-sdk` is the client-side half of Gardbase's zero-trust model, published as the Go module `github.com/qodesrl/gardbase/pkg/crypto`. It is what an application embeds to talk to Gardbase securely without ever trusting the server.

Two responsibilities:

1. **Verified enclave sessions.** Establish an end-to-end encrypted channel with the Nitro Enclave *through* the untrusted API, and — critically — independently verify the enclave's attestation document before using any key it offers. The verification is the whole point: certificate chain to the AWS Nitro Root CA, COSE signature, freshness, nonce, public-key binding, and PCR measurements. Without it, "the enclave holds your keys" is an unverified claim.
2. **Payload encryption primitives.** Three encryption modes that trade confidentiality for queryability — probabilistic for object bodies, deterministic for equality-searchable index tokens, order-preserving for range-searchable ones.

## Structure

```
pkg/crypto/
  kms.go                       # EnclaveSecureSession, SessionConfig, all HTTP calls to the API
  attestationVerification.go   # COSE/CBOR parsing, cert chain, ES384 signature, PCR checks, embedded Root CA
  utils.go                     # X25519 keygen, HKDF session-key derivation, HMAC nonce derivation, zero, openDEK
  probabilistic.go             # AES-256-GCM, random nonce.  ct = nonce(12) || gcmct
  deterministic.go             # AES-256-GCM with HMAC-derived nonce; plus an HMAC-only "Fixed" variant
  ope.go                       # order-preserving encryption + normalizers for every numeric/time type
  kms_test.go                  # integration-style tests requiring a live -enclave-endpoint
  go.mod                       # replaces pkg/enclaveproto and pkg/api with local paths
```

Public surface:

| Area | Symbols |
| --- | --- |
| Session | `InitEnclaveSecureSession`, `SessionConfig`, `EnclaveSecureSession` |
| Session methods | `GenerateDEK`, `GetTableIEK`, `SessionUnwrap`, `UnsealDEK`, `GetAttestationInfo`, `Close` |
| Sessionless | `UnwrapSingleDEK` |
| Keys | `GenerateEphemeralKeypair` |
| Encrypt/decrypt | `Encrypt/DecryptObjectProbabilistic`, `Encrypt/DecryptObjectDeterministic`, `EncryptObjectDeterministicFixed`, `Encrypt/DecryptObjectOPE` |
| OPE normalizers | `Normalize/Denormalize` for `Int64`, `Int32`, `Uint32`, `Float32`, `Float64`, `Time`, `TimeExtended`, plus the reflective `NormalizeValueOPE` |

## Behavior

**Session establishment.** `InitEnclaveSecureSession` generates an X25519 keypair and a 32-byte random nonce, POSTs them to `<endpoint>/secure-session/init`, and receives the enclave's ephemeral public key, session id, expiry, and an attestation document. It derives the session key with `curve25519.X25519` → HKDF-SHA256, info `"gardbase-enclave-session-v1"` — byte-identical to the enclave's derivation, so both sides land on the same 32-byte XChaCha20-Poly1305 key. It then verifies the attestation *before* returning a usable session; on failure it zeroes the session key and returns an error alongside the (unusable) session object. Every subsequent method re-checks `AttestationVerified` and `ExpiresAt`.

**Attestation verification** (`verifyAttestation`, eight recorded steps):

1. CBOR-decode the COSE_Sign1 envelope.
2. CBOR-decode the attestation document payload.
3. Verify the certificate chain from the leaf through the CA bundle to the AWS Nitro Root CA — the root is PEM-embedded in the source, overridable via `SessionConfig.RootCA`.
4. Verify the ES384 signature: rebuild the canonical `Sig_structure` (`["Signature1", protected, b"", payload]`), SHA-384 it, and check the raw 96-byte R‖S against the leaf's ECDSA P-384 key.
5. Reject documents older than `MaxAttestationAge`.
6. Require the document's nonce to equal the client's — replay protection.
7. Require the document's `public_key` to equal the enclave ephemeral key just received — this is what binds the attestation to *this* handshake rather than some other genuine enclave.
8. If `VerifyPCRs` is set and `ExpectedPCRs` is non-empty, compare each hex-encoded expected PCR against the document's.

`GetAttestationInfo()` returns the verified steps, timestamp, and hex PCRs for logging or display.

**Key operations.** All go over HTTPS to the API, which relays to the enclave; the client sees only session-sealed blobs and opens them locally with `openDEK` (XChaCha20-Poly1305, 24-byte nonce).

- `GenerateDEK(ctx, tableHash, count)` → per DEK: the plaintext key (opened from the session seal), the KMS ciphertext blob, and the master-key-encrypted copy with its nonce — the latter two are what get persisted alongside the object. Also returns the table's index encryption key.
- `GetTableIEK(ctx, tableHash)` → the IEK alone, for read paths that only need to build index tokens.
- `SessionUnwrap(ctx, items)` → batch DEK recovery; each sealed DEK is bound to its object id as AEAD associated data, so `UnsealDEK(ctx, sealed, nonce, objectID)` must be called with the matching id.
- `UnwrapSingleDEK` is the sessionless path: it generates a NaCl box keypair, posts to `/decrypt`, verifies the returned attestation, and opens the box. It performs full verification per call rather than amortizing it over a session.

Authentication is transparent: `tenantRoundTripper` injects `X-Tenant-ID` and `X-API-Key` on every request from `SessionConfig`.

**Encryption modes.**

- *Probabilistic* — AES-256-GCM with a fresh random 12-byte nonce prepended. The default for object bodies; the same plaintext encrypts differently every time.
- *Deterministic* — AES-256-GCM where the nonce is `HMAC-SHA256(dek, contextType ‖ 0x00 ‖ context)[:12]` and the context is also the AAD. Same plaintext + same context ⇒ same ciphertext, which is what makes equality queries on encrypted index tokens possible. `EncryptObjectDeterministicFixed` is the one-way variant: a 32-byte HMAC over context and plaintext, with no decryption path.
- *OPE* — `goope` over a 32-bit input range mapped into a wide 62-bit output range to limit collisions, producing an 8-byte big-endian token. The normalizers exist because OPE only accepts a bounded integer domain: signed integers get their sign bit flipped so ordering survives, floats are converted to order-preserving bit patterns, `int64`/`float64` are truncated to their high 32 bits, and `time.Time` maps to Unix seconds (with an extended variant for pre-1970 / post-2106).

**Memory hygiene.** `Close()` zeros the session key and client private key; `deriveSessionKey` zeros the raw shared secret immediately after HKDF.

## Dependencies

- `github.com/fxamacker/cbor/v2` — COSE/CBOR decoding of attestation documents.
- `golang.org/x/crypto` — `curve25519`, `hkdf`, `chacha20poly1305`, `nacl/box`.
- `github.com/alessandrofoglia07/goope` — the order-preserving encryption scheme.
- Stdlib `crypto/*` for AES-GCM, HMAC, ECDSA verification, and X.509.
- **Workspace packages** via local `replace` directives: `pkg/api` (endpoint request/response types) and `pkg/enclaveproto` (shared unwrap item types).
- **Runtime peer:** a reachable Gardbase API endpoint (`SessionConfig.Endpoint` points at `.../api/encryption`); it never talks to AWS directly.

## Notes

- `VerifyPCRs` defaults to `false`. Sessions still verify chain, signature, freshness, nonce, and key binding — but without PCRs the client accepts *any* genuine Nitro enclave, not specifically Gardbase's. Production callers must set `VerifyPCRs: true` and populate `ExpectedPCRs` (obtainable via `nitro-cli describe-eif`, or from the SSM parameter the deployment publishes at `/<project>/<environment>/enclave/pcr-values`). This is the single most important configuration decision when using this SDK.
- OPE is documented in-file with a `CRITICAL SECURITY WARNING`: it leaks ordering, approximate values, distribution, and frequency. Use it only for low-sensitivity fields that genuinely need range queries.
- Deterministic encryption leaks equality — two records with the same value for an indexed field are visibly the same. That is the deliberate trade for searchability.
- Sealed DEKs from `SessionUnwrap` are AAD-bound to their object id; unsealing with the wrong id fails authentication rather than returning garbage.
- `SessionConfig.HTTPTimeout` is declared but unused — both `InitEnclaveSecureSession` and `UnwrapSingleDEK` hardcode a 15-second `http.Client` timeout.
- The HTTP error paths decode into `errBody` and then read `res.Body` again for the message; since the decoder has already consumed the stream, the reported body is typically empty. Status codes are still accurate.
- `NormalizeInt64OPE` and `NormalizeFloat64OPE` keep only the high 32 bits, so 64-bit values are lossy under OPE: ordering is preserved but nearby values collide. `NormalizeValueOPE` rejects `uint64` outright for the same reason.
- `kms_test.go` requires a live deployment (`go test -enclave-endpoint=…`); with no endpoint the tests fail rather than skip. Only session init and DEK generation are covered — unwrap/unseal are left to SDK-level integration tests.
- `AESKeySize` is 32 and enforced on every DEK-taking function, except `EncryptObjectDeterministicFixed`, which accepts any non-empty key since it is a plain HMAC.
