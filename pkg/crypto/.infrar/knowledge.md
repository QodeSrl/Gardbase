---
schema_version: 1
id: 980a19a0-3ad8-4f09-95a7-a4012ac0ea95
name: crypto-sdk
node: pkg/crypto
category: other
---
## Purpose

`crypto-sdk` is the client-side half of Gardbase's zero-trust model, published as the standalone Go module `github.com/qodesrl/gardbase/pkg/crypto`. It is a library, not a deployed service — applications import it to talk to a Gardbase deployment safely.

It carries the two responsibilities the server cannot be trusted with:

1. **Verifying the enclave.** Full AWS Nitro attestation verification — COSE/CBOR decoding, certificate chain validation to the AWS Nitro root CA, ECDSA P-384 signature check, document freshness, nonce match, enclave public-key binding, and PCR comparison against expected measurements. Everything else in the SDK refuses to run until this passes.
2. **Encrypting and decrypting data.** Probabilistic AES-GCM for payloads, deterministic AES-GCM (and an HMAC variant) for equality-searchable index tokens, and order-preserving encryption for range-queryable index tokens — plus the session handling that gets DEKs from the enclave in the first place.

The design contract: plaintext exists only in the calling application and inside the enclave. This module is what makes that true on the client side.

## Structure

Five source files and a test, all in package `crypto`:

- `kms.go` — the session API and HTTP client. `EnclaveSecureSession` (session id, X25519 keypair, enclave public key, derived session key, expiry, attestation state, endpoint, http client), `SessionConfig` (endpoint, tenant id, API key, `ExpectedPCRs`, optional `RootCA`, `MaxAttestationAge`, `VerifyPCRs`, `HTTPTimeout`), and `tenantRoundTripper` which injects `X-Tenant-ID`/`X-API-Key` on every request. Entry points: `InitEnclaveSecureSession`, and on the session `GenerateDEK`, `GetTableIEK`, `SessionUnwrap`, `UnsealDEK`, `GetAttestationInfo`, `Close`. Plus the standalone `UnwrapSingleDEK`.
- `attestationVerification.go` — the attestation machinery. `attestationDocument` and `coseSign1` CBOR structs, the embedded AWS Nitro root CA PEM, `verifyAttestation` (the eight-step pipeline), `verifyCertificateChain`, `verifyCOSESignature`, `verifyPCRs`, and `verificationResult` which records which steps passed.
- `probabilistic.go` — `EncryptObjectProbabilistic` / `DecryptObjectProbabilistic`. AES-256-GCM with a random 12-byte nonce; ciphertext format is `nonce ‖ gcmCiphertext`.
- `deterministic.go` — `EncryptObjectDeterministic` / `DecryptObjectDeterministic` (AES-GCM with an HMAC-derived nonce so the same plaintext+context always yields the same ciphertext) and `EncryptObjectDeterministicFixed` (a one-way 32-byte HMAC-SHA256 tag, no decryption).
- `ope.go` — order-preserving encryption. A large family of `Normalize*OPE`/`Denormalize*OPE` helpers mapping int8/16/32/64, uint variants, float32/64, `time.Time` and `time.Duration` into a monotonic 32-bit domain, the generic `NormalizeValueOPE(any)` dispatcher, and `EncryptObjectOPE`/`DecryptObjectOPE` producing 8-byte big-endian ciphertexts.
- `utils.go` — constants (`AESKeySize` 32, `GMCNonceSize` 12), `generateRandomBytes`, `deriveNonceHMAC`, `GenerateEphemeralKeypair`, `deriveSessionKey`, `zero`, `openDEK`.
- `kms_test.go` — integration-style tests gated on an `-enclave-endpoint` flag; they require a live deployment and do nothing useful without one.

## Behavior

**Session establishment.** `InitEnclaveSecureSession` generates an ephemeral X25519 keypair and a 32-byte random nonce, POSTs them to `<endpoint>/secure-session/init`, and receives the session id, the enclave's ephemeral public key, an expiry, and an attestation document. It derives the session key with X25519 + HKDF-SHA256 using info `"gardbase-enclave-session-v1"` — byte-for-byte the same derivation the enclave performs, which is what makes the channel end-to-end rather than terminating at the API. It then verifies the attestation; on failure it zeroes the session key and returns an error alongside the (unusable) session.

**Attestation verification pipeline.** In order: decode COSE_Sign1 → decode the attestation document from its payload → verify the certificate chain from the leaf through the CA bundle to the root (the embedded AWS root CA unless `SessionConfig.RootCA` overrides it) → verify the COSE signature (SHA-384 over the canonical `Sig_structure`, with AWS's raw 96-byte R‖S signature split into two 48-byte P-384 halves rather than ASN.1) → check the document is no older than `MaxAttestationAge` → check the nonce equals the one this client generated → check the embedded public key equals the enclave ephemeral key received → if `VerifyPCRs`, compare each expected PCR against the document. Every step appends to `VerifiedSteps`, surfaced by `GetAttestationInfo` for logging or display. The nonce and public-key bindings are what stop a malicious API from replaying an old document or substituting its own key.

**Key retrieval.** `GenerateDEK(ctx, tableHash, count)` asks for a batch of DEKs plus the table's index encryption key; each returned DEK arrives sealed under the session key and is opened locally with `openDEK` (XChaCha20-Poly1305, 24-byte nonce). The caller gets, per DEK, the plaintext key plus the KMS-wrapped and master-key-wrapped forms to persist alongside the object. `GetTableIEK` fetches just the IEK. `SessionUnwrap` bulk-unwraps DEKs for reads; `UnsealDEK` opens one of those results using the **object ID as associated data**, so a sealed DEK cannot be moved to a different object. Every session method first checks expiry and `AttestationVerified` and refuses to proceed otherwise.

**The one-shot path.** `UnwrapSingleDEK` needs no prior session: it generates a NaCl box keypair, POSTs to `/decrypt`, verifies the returned attestation, and opens the box. Useful for a single decryption where establishing a session is overkill.

**Encryption modes and their trade-offs.** Payloads use probabilistic AES-GCM — semantically secure, not searchable. Index tokens for equality use deterministic encryption, where the nonce is HMAC-derived from the key and a caller-supplied context string (the context also serves as AEAD associated data), so identical values produce identical tokens; that is precisely what enables equality queries and precisely what leaks which records share a value. Range-queryable fields use OPE, which leaks far more — order, distribution, approximate magnitude, frequency — and the file opens with an explicit warning to use it only for low-sensitivity data where range queries are genuinely required. The normalization helpers exist because OPE operates on a 32-bit ordered integer domain: signed integers get their sign bit flipped, floats get the IEEE-754 sign/magnitude transform, and 64-bit values are truncated to their high 32 bits (so `int64` and extended timestamps lose precision by design). `Close()` zeroes the session key and client private key.

## Dependencies

- **Internal modules** (via local `replace` directives): `pkg/api` (request/response types for the HTTP calls) and `pkg/enclaveproto` (`SessionUnwrapItem`, `GeneratedDEK`, and friends). It does **not** depend on `pkg/models` or on either application.
- **Libraries**: `github.com/fxamacker/cbor/v2` (COSE/CBOR), `github.com/alessandrofoglia07/goope` (the OPE primitive), `golang.org/x/crypto` (curve25519, hkdf, chacha20poly1305, nacl/box). Everything else is the Go standard library — no AWS SDK, no HTTP framework.
- **Runtime peer**: a reachable Gardbase API endpoint, plus valid `TenantID`/`APIKey` with the `crypto` permission. The API relays to the enclave; this module never talks to the enclave or to AWS directly.
- **Operational input**: expected PCR values, which `infrastructure-main` publishes to SSM at `/${project}/${env}/enclave/pcr-values` after each EIF build (also obtainable via `nitro-cli describe-eif`).

## Notes

- This module is the mirror image of `enclave-service`. The HKDF info string, the XChaCha20-Poly1305 sealing format, and the associated-data conventions must match exactly on both sides — changing one without the other silently breaks every session.
- `VerifyPCRs` defaults to false in `SessionConfig`; with it off, the SDK confirms it is talking to *a* genuine Nitro enclave but not to *your* enclave image. Production clients must set it true and supply `ExpectedPCRs`, and must update those values whenever the enclave image is rebuilt.
- OPE ciphertexts are 8 bytes and OPE range values are assumed to be exactly 8 bytes by the server-side index layout (`models.OPERangeValueLength`, itself `TODO`-flagged); the two must stay in sync.
- `EncryptObjectDeterministicFixed` is one-way (HMAC), unlike `EncryptObjectDeterministic` (reversible AES-GCM). It also accepts any non-empty key length while the AES variants require exactly 32 bytes.
- `NormalizeTimeOPE` rejects timestamps before 1970 or after 2106 and points at `NormalizeTimeExtendedOPE`, whose precision is roughly 136 years per unit — usable for coarse ordering only. `uint64` is rejected outright as unrepresentable.
- Error handling in the HTTP helpers is awkward: the response body is decoded into `errBody` and then `io.ReadAll` is called on the already-consumed body, so non-200 error messages come back empty. Several errors also read "failed to start decrypt session" regardless of which call actually failed.
- `UnwrapSingleDEK` slices `resBody.Ciphertext[:24]` before validating its length, so a short or empty ciphertext from a misbehaving server panics rather than erroring.
- `zero()` is best-effort — Go's garbage collector may have already copied the buffer elsewhere.
- The tests require a live enclave endpoint passed via `-enclave-endpoint`; there is no unit-test coverage of the crypto primitives or the attestation pipeline in this module.
