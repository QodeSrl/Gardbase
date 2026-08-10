---
schema_version: 1
id: 20e9b506-b49c-4c40-8e85-068c406a1847
name: crypto-sdk
node: pkg/crypto
category: app
---
## Purpose

`crypto-sdk` is the client-side cryptographic library for Gardbase. It is the piece that runs on the *user's* machine, outside the trust boundary of both the API and AWS, and it is what makes the zero-trust claim verifiable rather than merely asserted.

It has two distinct jobs:

1. **Establish and verify a secure session with the Nitro Enclave.** It generates an ephemeral X25519 keypair, initiates a session through the API, and then independently verifies the enclave's attestation document — walking the certificate chain to AWS's Nitro Root CA, checking the COSE/ECDSA signature, comparing PCR measurements against expected values, validating freshness, and confirming the attested public key is the one it just did a key exchange with. Only after all of that does it unseal any key material. If verification fails, the session is unusable: every subsequent call checks `AttestationVerified`.

2. **Provide the encryption primitives for the data itself** — probabilistic AES-GCM for ordinary payloads, deterministic AES-GCM for equality-searchable index tokens, and order-preserving encryption for range-queryable fields.

The security argument the whole product rests on lives here. The API is assumed hostile; this package is what lets a client prove it is talking to a specific, measured piece of enclave code before handing over anything.

## Structure

A standalone Go module (`pkg/crypto/go.mod`) with a flat file layout — no `internal/`, no subpackages. Everything is `package crypto`.

- **`kms.go`** (~400 lines) — the session client. Defines `SessionConfig` (endpoint, tenant ID, API key, expected PCRs, optional root CA, max attestation age, `VerifyPCRs` toggle, HTTP timeout), `EnclaveSecureSession` (session ID, client keypair, enclave public key, derived session key, expiry, attestation document and result, verification flag), and `GeneratedDEK`. Holds `InitEnclaveSecureSession`, the session methods `SessionUnwrap` / `GenerateDEK` / `GetTableIEK` / `UnsealDEK` / `Close` / `GetAttestationInfo`, and the sessionless `UnwrapSingleDEK`. Also `tenantRoundTripper`, an `http.RoundTripper` that injects `X-Tenant-ID` and `X-API-Key` on every request.

- **`attestationVerification.go`** (~256 lines) — the verifier. CBOR structs for the COSE_Sign1 envelope and the Nitro attestation document, the AWS Nitro Root CA embedded as a PEM constant, the eight-step `verifyAttestation` function, and helpers `verifyCertificateChain`, `verifyCOSESignature`, `verifyPCRs`. `verificationResult` records which steps passed.

- **`probabilistic.go`** — `EncryptObjectProbabilistic` / `DecryptObjectProbabilistic`. AES-256-GCM with a random 12-byte nonce; ciphertext format is `nonce || gcmct`.

- **`deterministic.go`** — `EncryptObjectDeterministic` / `DecryptObjectDeterministic` (AES-GCM with an HMAC-derived nonce and the context as AAD, so the same plaintext under the same context always yields the same ciphertext) and `EncryptObjectDeterministicFixed` (a plain 32-byte HMAC-SHA256 tag, one-way).

- **`ope.go`** (~223 lines) — order-preserving encryption. `EncryptObjectOPE` / `DecryptObjectOPE` over an 8-byte big-endian encoding, plus a family of `Normalize*OPE` / `Denormalize*OPE` functions mapping `int64`, `int32`, `uint32`, `float32`, `float64`, `time.Time`, and `time.Duration` into the OPE input domain, and a reflective `NormalizeValueOPE(any)` dispatcher.

- **`utils.go`** — `AESKeySize` (32) and `GMCNonceSize` (12), `generateRandomBytes`, `deriveNonceHMAC`, `GenerateEphemeralKeypair`, `deriveSessionKey`, `zero`, `openDEK`.

- **`kms_test.go`** — integration-only tests driven by an `-enclave-endpoint` flag; they require a live enclave proxy and are skipped in effect without one.

## Behavior

### Session establishment

`InitEnclaveSecureSession` generates an X25519 keypair and a 32-byte random nonce, POSTs `{ClientEphemeralPublicKey, Nonce}` to `{Endpoint}/secure-session/init`, and receives back a session ID, the enclave's ephemeral public key, an expiry, and an attestation document. It derives the session key with X25519 + HKDF-SHA256 using info `"gardbase-enclave-session-v1"` — the same construction the enclave performs on its side — then immediately verifies the attestation. On verification failure it **zeroes the session key and returns both the session object and an error**, leaving `AttestationVerified` false.

### Attestation verification — the eight steps

`verifyAttestation` records each step it completes in `VerifiedSteps`:

1. **Decode the COSE_Sign1 envelope** from CBOR (`[protected, unprotected, payload, signature]`).
2. **Decode the attestation document** from the payload — module ID, timestamp, PCR map, leaf certificate, CA bundle, public key, user data, nonce.
3. **Verify the certificate chain.** Parse the leaf, load the CA bundle as intermediates, and verify against a root pool containing either `config.RootCA` or the embedded AWS Nitro Root CA constant (`aws.nitro-enclaves`, valid 2019–2049).
4. **Verify the COSE signature.** Reconstruct the `Sig_structure` (`["Signature1", protected, "", payload]`), CBOR-encode it, hash with SHA-384, split the 96-byte raw signature into 48-byte R and S, and `ecdsa.Verify` against the leaf's P-384 public key. AWS uses raw R||S, not ASN.1 — the code checks the length explicitly.
5. **Verify freshness** — reject if `time.Since(docTime) > config.MaxAttestationAge` (the document timestamp is in milliseconds). Skipped when `MaxAttestationAge` is 0.
6. **Verify the nonce** matches the one the client generated — this is the replay defense.
7. **Verify the public key binding** — the `public_key` field in the signed document must equal the enclave ephemeral public key the client just used for its key exchange. Without this step a valid attestation could be paired with an attacker's key.
8. **Verify PCRs** against `config.ExpectedPCRs` (a `map[uint]string` of hex digests), but **only when `config.VerifyPCRs` is true and the map is non-empty**.

Steps 3, 4, 6, and 7 together are what bind "a genuine AWS Nitro enclave" to "this specific key exchange, right now". Step 8 is what binds it to "running exactly the code I expect".

### Session operations

All three session methods check expiry and `AttestationVerified` before doing anything:

- **`SessionUnwrap(items)`** — POSTs a batch of `{ObjectId, Ciphertext}` and returns per-item sealed DEKs. The caller then opens each with `UnsealDEK`, which uses the object ID as AEAD associated data.
- **`GenerateDEK(tableHash, count)`** — requests `count` fresh DEKs plus the table's index encryption key. Each response DEK is unsealed with `openDEK` and returned alongside its KMS-wrapped and master-key-wrapped forms, so the caller can persist the wrapped copies and use the plaintext locally. The IEK is unsealed too.
- **`GetTableIEK(tableHash)`** — fetches and unseals just the index key.
- **`Close()`** zeroes the session key and the client private key.
- **`GetAttestationInfo()`** returns a map of verification status, timestamp, hex-encoded PCRs, and the list of verified steps — for logging or display.

Unsealing everywhere is XChaCha20-Poly1305 (`chacha20poly1305.NewX`) with a 24-byte nonce, matching the enclave's sealing.

### Sessionless path

`UnwrapSingleDEK` is a one-shot alternative: generate a NaCl box keypair, POST the wrapped DEK to `/decrypt`, verify the returned attestation, and `box.Open` the response. Useful when a single decryption doesn't justify session setup.

### Data encryption

- **Probabilistic** — the default for payloads. Random nonce, so identical plaintexts produce different ciphertexts. No searchability.
- **Deterministic** — for equality-searchable index tokens. The nonce is `HMAC-SHA256(dek, contextType || 0x00 || context)` truncated to 12 bytes, and `context` is also passed as AAD. Determinism is scoped to the context string: the same value under a different context encrypts differently. The trade-off, stated in the module docs and the root README, is that equal plaintexts are visibly equal within a context.
- **OPE** — for range-queryable fields, and carrying an explicit `CRITICAL SECURITY WARNING` header: it leaks order, approximate values, distribution, and frequency. The normalization layer flattens signed integers, floats, and timestamps into a monotone `[0, 2^32-1]` domain (flipping sign bits for integers, the IEEE-754 total-order trick for floats), encrypts through `goope` into a `[0, 2^61]` output range chosen to reduce collisions, and encodes big-endian so raw byte comparison in DynamoDB preserves order.

## Dependencies

**Go module** — `github.com/qodesrl/gardbase/pkg/crypto`, Go 1.24.4, in the root `go.work` workspace.

- `github.com/fxamacker/cbor/v2` — COSE and attestation document decoding.
- `golang.org/x/crypto` — `chacha20poly1305`, `curve25519`, `hkdf`, `nacl/box`.
- `github.com/alessandrofoglia07/goope` — the order-preserving encryption implementation.
- Standard library for everything else: `crypto/aes`, `crypto/cipher`, `crypto/ecdsa`, `crypto/hmac`, `crypto/sha256`, `crypto/sha512`, `crypto/x509`, `encoding/pem`, `math/big`, `net/http`.
- Internal, via `replace` to sibling directories: `github.com/qodesrl/gardbase/pkg/api` (the `encryption` request/response DTOs) and `github.com/qodesrl/gardbase/pkg/enclaveproto` (`SessionUnwrapItem`, `SessionUnwrapResponse`, and friends).

Notably **no AWS SDK dependency** — despite the filename `kms.go`, this package never calls KMS. All AWS interaction is proxied through the API.

**What it talks to:** the API's `/api/encryption` endpoints — `/secure-session/init`, `/unwrap`, `/generate-deks`, `/get-table-iek`, and `/decrypt` — all requiring the `crypto` permission on the API key.

**Configuration the caller must supply:** `Endpoint`, `TenantID`, `APIKey`, and — for a meaningful security guarantee — `VerifyPCRs: true` with `ExpectedPCRs` populated from `nitro-cli describe-eif` or from the SSM parameter that `infrastructure/main` publishes at `/${project}/${env}/enclave/pcr-values`.

**No Nx project file.** This module has no `project.json`, so unlike the other Go modules it has no Nx build/test/lint targets; it is built and tested with `go` directly.

## Notes

- **`VerifyPCRs` defaults to `false`, and PCR checking is the only step that ties the attestation to *this specific code*.** With it off, verification still proves "some genuine AWS Nitro enclave, freshly, with this key" — but not *which* enclave image. The struct comment says "Set to false during development, true in production"; nothing enforces that, and the shipped test config sets it false.
- **`verifyAttestation` sets `AttestationVerified = true` in two places.** The method `(*EnclaveSecureSession).verifyAttestation` sets it on success, and `InitEnclaveSecureSession` sets it again afterward. Harmless as written, but the duplication makes the invariant easy to break.
- **`InitEnclaveSecureSession` returns a non-nil session alongside its error** when attestation fails. The key is zeroed and `AttestationVerified` stays false, so the session methods will refuse to operate — but a caller that ignores the error and holds the returned pointer gets an object that looks usable.
- **`EncryptObjectDeterministicFixed` is one-way.** It returns a raw HMAC tag with no corresponding decrypt function, unlike every other primitive in the package. The file header comment describes AES-GCM with HMAC-derived nonces, which describes `EncryptObjectDeterministic` rather than the `Fixed` variant.
- **OPE precision is lossy by design.** 64-bit inputs are truncated to their high 32 bits, so `NormalizeInt64OPE`/`NormalizeFloat64OPE` round-trips are approximate, and `NormalizeTimeExtendedOPE` has roughly 136-year granularity. `NormalizeTimeOPE` errors outside 1970–2106. `uint64` is rejected outright. Callers using OPE for equality rather than ordering will get false matches.
- **OPE's security warning is not enforced anywhere.** The `CRITICAL SECURITY WARNING` is a comment; the API and enclave will happily store OPE ciphertexts for any field.
- **Error handling in the HTTP paths is confused.** The non-200 branches decode the body into `errBody` and then, inside the `err == nil` case, call `io.ReadAll` on an already-consumed body — so the reported error message is always empty. The same block is copy-pasted across five methods, and three of them report "failed to start decrypt session" regardless of which endpoint actually failed.
- **`SessionConfig.HTTPTimeout` is never read.** Both `InitEnclaveSecureSession` and `UnwrapSingleDEK` hardcode a 15-second `http.Client` timeout.
- **`UnwrapSingleDEK` slices `resBody.Ciphertext[:24]` before validating its length**, which panics on a short response — a malicious or malformed API response crashes the client. It also extracts the nonce from the ciphertext prefix while the response carries a separate `Nonce` field.
- **`Close()` is not called automatically and there is no finalizer.** Callers must invoke it to wipe the session key; `zero()` is also a plain loop with no `runtime.KeepAlive`, so the compiler is not prevented from optimizing it away, and Go's GC may have already copied the key elsewhere in memory.
- **Tests require live infrastructure.** `kms_test.go` needs a running enclave proxy at `-enclave-endpoint`; there are no unit tests for the encryption primitives, the normalization functions, or — most significantly — for `verifyAttestation`, despite it being the security-critical code in the package. There are no negative tests confirming that a tampered document, a wrong PCR, a stale timestamp, or a mismatched nonce is actually rejected.
