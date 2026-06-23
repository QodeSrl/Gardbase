---
schema_version: 1
id: c7d907ef-0c6e-4a65-a6dc-effb15254246
name: enclave-service
node: apps/enclave-service
category: app
---

## Purpose
Go service intended to run inside a secure enclave (e.g. AWS Nitro / TEE). Provides cryptographic operations — remote attestation, KEK/DEK/IEK key preparation, session-scoped key derivation, data decryption, and table hashing — over an HTTP/RPC handler surface. Acts as the trusted boundary where plaintext keys and data are handled in isolation from the host.

## Structure
Standard Go layout. cmd/enclave/main.go is the entrypoint. internal/handlers/ holds one file per endpoint: health, get_attestation, prepare_kek, decrypt, and a session lifecycle group (session_init, session_prepare_dek, session_prepare_iek, session_generate_table_hash, session_unwrap). internal/session/ provides an in-memory session store (store.go) and a background cleanup/expiry routine (cleanup.go). internal/utils/ contains crypto and infra helpers: attestation document generation, connection setup, session key derivation (deriveSessionKey.go), hashing (hash.go), OpenSSL wrapper (openssl.go), and zeroing of sensitive memory (zero.go). go.mod/go.sum pin dependencies; project.json defines Nx build/serve targets; .infrar/knowledge.md holds node docs.

## Behavior
Boots in main.go, registers handlers, and serves requests. Typical flow: client requests an attestation document (get_attestation) to verify the enclave, then runs session_init to establish a session and derive a session key. Subsequent calls (prepare_kek, session_prepare_dek, session_prepare_iek, session_unwrap) wrap/unwrap and prepare key material within the session; decrypt performs data decryption; session_generate_table_hash computes deterministic hashes. Session store tracks sessions with TTL; cleanup.go periodically evicts expired entries. Sensitive buffers are explicitly zeroed (zero.go) after use to limit exposure.

## Dependencies
Go runtime and modules per go.mod. Relies on enclave attestation infrastructure (utils/attestation.go — likely Nitro vsock/NSM or analogous). Shells out to or links OpenSSL (utils/openssl.go) for crypto primitives. Nx tooling (project.json) for build/serve orchestration. Consumed by an external host/proxy that relays requests into the enclave and presumably by KMS-like callers needing attested key operations.

## Notes
Security-critical boundary: review attestation verification, key zeroing completeness, and session store concurrency/expiry for leaks. internal/ packages are not importable outside this module by design. OpenSSL dependency implies CGO or external binary — confirm enclave image includes it. Handler-per-file convention makes the API surface auditable; keep session lifecycle handlers consistent with store TTL semantics.
