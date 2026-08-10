---
schema_version: 1
id: bf5f9da9-97e4-44d5-a019-ddc1e69960bc
name: crypto-sdk
node: pkg/crypto
category: app
---
## Purpose

`crypto-sdk` is the client-side half of Gardbase's zero-trust design: a standalone Go module (`github.com/qodesrl/gardbase/pkg/crypto`) that applications embed to encrypt their own data before it ever reaches the API.

It carries the two responsibilities that cannot be delegated to the server:

1. **Verification.** Before trusting the enclave with anything, it independently validates the Nitro attestation document — certificate chain to the AWS Nitro root CA, COSE/ECDSA signature, freshness, nonce echo, public-key binding, and optionally PCR code measurements. Only after that does it accept the session key it just negotiated.
2. **Encryption.** It implements the three field encryption modes the storage layer expects — probabilistic AES-256-GCM for opaque payloads, deterministic AES-GCM for equality-searchable index tokens, and order-preserving encryption for range-queryable fields — along with the session handshake and DEK unsealing that feed them keys.

It is a library, not a service. Nothing in this repo deploys it; the `api` and `enclave-service` modules do not import it. It is the reference implementation that higher-level SDKs are built on.

## Structure

The node root is `pkg/crypto`, a self-contained Go module with `replace` directives to the sibling `pkg/enclaveproto` and `pkg/api` modules.

- `kms.go` — the session client. Defines `EnclaveSecureSession` and `SessionConfig`, the `tenantRoundTripper` that injects auth headers, and the five operations: `InitEnclaveSecureSession`, `SessionUnwrap`, `GenerateDEK`, `GetTableIEK`, `UnsealDEK`, plus `Close`, `GetAttestationInfo`, and the standalone `UnwrapSingleDEK`.
- `attestationVerification.go` — the attestation verifier. Holds the CBOR structs (`attestationDocument`, `coseSign1`), the embedded AWS Nitro root CA PEM, the eight-step `verifyAttestation`, and the helpers `verifyCertificateChain`, `verifyCOSESignature`, `verifyPCRs`.
- `deterministic.go` — `EncryptObjectDeterministic` / `DecryptObjectDeterministic` (AES-GCM with an HMAC-derived nonce) and `EncryptObjectDeterministicFixed` (a raw HMAC-SHA256 token).
- `probabilistic.go` — `EncryptObjectProbabilistic` / `DecryptObjectProbabilistic` (AES-GCM, `nonce || ciphertext`).
- `ope.go` — order-preserving encryption over `goope`, plus the full set of `Normalize*OPE` / `Denormalize*OPE` converters and the reflective `NormalizeValueOPE`.
- `utils.go` — shared primitives: `AESKeySize`/`GMCNonceSize` constants, `generateRandomBytes`, `deriveNonceHMAC`, `GenerateEphemeralKeypair`, `deriveSessionKey`, `openDEK`, `zero`.
- `kms_test.go` — integration-style tests for session init and DEK generation.

## Behavior

**Session establishment.** `InitEnclaveSecureSession` generates an ephemeral X25519 keypair and a 32-byte nonce, POSTs them to `<Endpoint>/secure-session/init`, and receives the enclave's ephemeral public key, a session ID, an expiry, and an attestation document. It derives the shared session key with `X25519 → HKDF-SHA256(info: "gardbase-enclave-session-v1")` — the same derivation the enclave performs — then verifies the attestation. If verification fails it zeroes the session key and returns an error (alongside a non-nil session value, so callers must check `err`, not just the pointer). Only on success is `AttestationVerified` set, and every subsequent operation refuses to run without it and without a non-expired session.

**Attestation verification.** `verifyAttestation` runs in a fixed order, appending to `VerifiedSteps` as it goes: decode the COSE_Sign1 envelope → decode the CBOR attestation document → verify the leaf certificate against the CA bundle up to the root (the embedded AWS Nitro root CA unless `config.RootCA` overrides it) → verify the ES384 signature by reconstructing the `Signature1` structure, hashing with SHA-384, and checking the raw 96-byte `R || S` against the leaf's ECDSA key → check the document age against `MaxAttestationAge` → compare the echoed nonce → compare the embedded public key against the enclave key just used for the handshake → and, when `VerifyPCRs` is set and `ExpectedPCRs` is non-empty, compare each expected PCR. The nonce and public-key checks are what make the document specific to *this* handshake rather than replayable.

**Key operations.** All of them go through the API server, which relays to the enclave:

- `GenerateDEK(tableHash, count)` — returns `count` DEKs, each with its plaintext (unsealed locally from the session-sealed form), its KMS-wrapped form, and its master-key-wrapped form plus nonce for storage. Also returns the table's index encryption key.
- `GetTableIEK(tableHash)` — fetches just the IEK, unsealed with the session key.
- `SessionUnwrap(items)` — batch-unwraps stored DEKs; results are session-sealed per object.
- `UnsealDEK(sealed, nonce, objectID)` — opens a session-sealed DEK with XChaCha20-Poly1305 using the object ID as associated data, matching how the enclave sealed it.
- `UnwrapSingleDEK` — a sessionless path: generates a NaCl box keypair, POSTs to `/decrypt`, verifies the returned attestation, and opens the box.

Authentication is transparent: `tenantRoundTripper` sets `X-Tenant-ID` and `X-API-Key` on every request from `SessionConfig`. `Close()` zeroes the session key and the client private key.

**Encryption modes.**

- *Probabilistic* — random 12-byte nonce, AES-256-GCM, output `nonce || ciphertext`. Two encryptions of the same value differ; nothing is queryable.
- *Deterministic* — nonce derived as `HMAC-SHA256(dek, "gardbase-data-deterministic-encryption-v1" || 0x00 || context)` truncated to 12 bytes, with `context` also used as GCM associated data. Same plaintext plus same context plus same key always yields the same ciphertext, which is what makes equality lookups on encrypted index tokens possible. A non-empty context is mandatory.
- *Deterministic fixed* — `HMAC-SHA256(dek, context || 0x00 || plaintext)`, a one-way 32-byte token with no decryption path; this is the 32-byte hash segment the index sort keys are built from.
- *OPE* — values are first normalized to the `[0, 2^32-1]` input range by the `Normalize*` helpers (sign-bit flipping for signed integers, IEEE-754 bit reordering for floats, high-32-bit truncation for `int64` and out-of-range timestamps), encrypted with `goope` into a wide `[0, 2^61]` output range to limit collisions, and emitted as 8 big-endian bytes — the range segment of a hash+range index token.

**Testing.** `kms_test.go` takes an `-enclave-endpoint` flag and exercises `InitEnclaveSecureSession` and `GenerateDEK` against a live deployment. There are no unit tests for the encryption modes or the attestation verifier.

## Dependencies

**Internal (via `replace`):**
- `pkg/api` — the `encryption` package supplies every request/response type the session client marshals.
- `pkg/enclaveproto` — the underlying enclave message types those aliases resolve to (`SessionInitRequest/Response`, `SessionUnwrapItem`, `GeneratedDEK`, `PrepareIEKResponse`, `DecryptRequest/Response`).

**Third party:**
- `github.com/fxamacker/cbor/v2` — decodes the COSE envelope and the attestation document.
- `github.com/alessandrofoglia07/goope` — the order-preserving encryption primitive.
- `golang.org/x/crypto` — `curve25519`, `hkdf`, `chacha20poly1305` (XChaCha20-Poly1305 for unsealing), `nacl/box`.
- Standard library for AES-GCM, HMAC, ECDSA, and X.509.

**Runtime peers:** the `api` node (as the HTTP endpoint) and the `enclave-service` node (as the counterparty whose derivation, sealing, and attestation this module mirrors). Any change to the HKDF info string, the AEAD choice, the associated-data convention, or the index-token layout must be made on both sides simultaneously.

## Notes

- **`VerifyPCRs` defaults to false.** With it off, the verifier confirms the document came from a genuine Nitro enclave signed by AWS, but not *which code* is running in it. Production clients must set `VerifyPCRs: true` and populate `ExpectedPCRs` (obtainable from `nitro-cli describe-eif`, or from the SSM parameter `extract-pcrs.sh` publishes) for the zero-trust property to hold end to end.
- **OPE leaks.** `ope.go` opens with an explicit warning: order-preserving encryption reveals ordering, approximate values, distribution, and frequency. It is intended only for low-sensitivity fields where range queries are genuinely required. Deterministic encryption similarly reveals which records share a value for an indexed field.
- **OPE is lossy for wide types.** `NormalizeInt64OPE`, `NormalizeFloat64OPE`, and `NormalizeTimeExtendedOPE` keep only the high 32 bits, so round-tripping an `int64` or `float64` does not return the original value — they preserve order for range queries, not equality. `NormalizeTimeOPE` is exact but rejects timestamps outside 1970–2106, and `uint64` is rejected outright.
- **`InitEnclaveSecureSession` returns a non-nil session together with an error** when attestation fails. Callers that ignore the error will hold a session whose key has been zeroed.
- **This module is not in any Dockerfile and has no Nx project.** It participates in `go.work` and is versioned as its own module for consumption as a dependency; `go test ./...` here requires a reachable enclave endpoint and fails without one.
- The HTTP error paths in `kms.go` decode the response body into `errBody` and then call `io.ReadAll` on the already-drained body, so the reported error message is usually empty and the status code is the only signal.
- `UnwrapSingleDEK` copies the NaCl box nonce from the first 24 bytes of the ciphertext before verifying the attestation; verification does gate the final `box.Open`, but the ordering makes the flow harder to follow than the session path.
- The AWS Nitro root CA is embedded as a PEM constant with a 2049 expiry; `SessionConfig.RootCA` exists to override it if AWS ever rotates.
