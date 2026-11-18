# Gardbase crypto package

Client-side Go primitives for object encryption, index tokens, and attested enclave sessions. Import path: `github.com/qodesrl/gardbase/pkg/crypto` (the Go package name is `crypto`).

[Security model](../../README.md#security-model) · [API routes](../../apps/api/README.md) · [Enclave protocol](../enclaveproto)

## Package map

| File                                                       | Responsibility                                                                |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------- |
| [`probabilistic.go`](probabilistic.go)                     | AES-256-GCM object encryption and decryption                                  |
| [`deterministic.go`](deterministic.go)                     | Deterministic encryption and HMAC equality tokens                             |
| [`ope.go`](ope.go)                                         | Order-preserving encryption and type normalization                            |
| [`kms.go`](kms.go)                                         | Session configuration, session initialization, key generation, and unwrapping |
| [`attestationVerification.go`](attestationVerification.go) | Certificate chain, signature, nonce, public key, age, and optional PCR checks |

These are lower-level building blocks. The package does not provide an ORM, collection builder, or automatic object lifecycle client.

## Encrypt an object

This example uses existing package functions. The caller supplies a 32-byte DEK obtained through its configured key workflow:

```go
package example

import gardcrypto "github.com/qodesrl/gardbase/pkg/crypto"

func roundTrip(plaintext, dek []byte) ([]byte, error) {
    ciphertext, err := gardcrypto.EncryptObjectProbabilistic(plaintext, dek)
    if err != nil {
        return nil, err
    }
    return gardcrypto.DecryptObjectProbabilistic(ciphertext, dek)
}
```

`EncryptObjectProbabilistic` creates a fresh 12-byte nonce and returns `nonce || ciphertext || authentication tag`. The decrypt function reads the nonce from that blob. Keep object DEKs out of storage payloads in plaintext; the object API uses wrapped-key fields alongside the encrypted content.

## Configure an enclave session

`InitEnclaveSecureSession(ctx, config)` accepts a `SessionConfig` containing:

- `Endpoint`: the deployment URL **including `/api/encryption`**, without a trailing slash.
- `TenantID` and `APIKey`: credentials for a key with `crypto` permission.
- `VerifyPCRs: true` and a nonempty `ExpectedPCRs` map: the expected enclave measurements as hex strings.
- A nonzero `MaxAttestationAge`: the accepted attestation age.
- `RootCA`: optional custom trust root; otherwise the embedded AWS Nitro root is used.

Check the initialization error before using a returned session. On verification failure the function may return a session together with an error; that session must not be treated as verified. Call `Close()` when finished. It clears the local session key; it is not a server-side session-revocation request.

Session methods include `GenerateDEK`, `GetTableIEK`, `SessionUnwrap`, and `UnsealDEK`. Read their request/response handling in [`kms.go`](kms.go) when assembling the full object workflow. `HTTPTimeout` is currently a configuration field, but session initialization constructs an HTTP client with a fixed 15-second timeout.

## Index tokens and leakage

`EncryptObjectDeterministicFixed` produces a 32-byte HMAC-SHA256 token using the key, context, and plaintext value. Matching tokens reveal equal values under the same key and context. Keep the context and token encoding consistent between stored indexes and queries.

The separate `EncryptObjectDeterministic` / `DecryptObjectDeterministic` functions use AES-GCM with a context-derived nonce. They are distinct from the fixed HMAC token function; review their nonce/context requirements before selecting a primitive.

Order-preserving encryption exposes ordering and can expose distributions, frequency, and approximate values. Use it only for low-sensitivity fields where that tradeoff is acceptable. Several normalizers reduce precision: `int64` and `float64` discard lower bits, and extended timestamps outside the normal unsigned-32-bit Unix-seconds range are very coarse. Do not assume a lossless round trip for those types.

## Validation

Use the toolchain declared in the repository's `go.work` and `go.mod` files. Existing tests in [`kms_test.go`](kms_test.go) are integration tests requiring a running proxy. Their current setup uses development verification settings and does not populate tenant credentials; review and adapt that setup before running against your own deployment. It is not a production client-configuration example.
