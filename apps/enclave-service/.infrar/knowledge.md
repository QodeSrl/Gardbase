---
schema_version: 1
id: 7b748114-a59d-47ab-8b28-cb1e913dfad6
name: enclave-service
node: apps/enclave-service
category: app
---

## Purpose
A Go-based service intended to run inside a secure enclave (e.g. AWS Nitro Enclaves). It provides cryptographic operations bound to remote attestation: producing attestation documents, establishing secure sessions, preparing/unwrapping key material (KEK/DEK/IEK), and performing decryption. It acts as the trusted computing boundary where sensitive keys are handled in isolation.

## Structure
Standard Go application layout. Entry point at cmd/enclave/main.go wires up the service. Business logic lives under internal/ (unexported to external modules):
- internal/handlers/: request handlers, one file per operation — health, get_attestation (attestation document retrieval), prepare_kek (key-encryption-key setup), decrypt, and session-scoped handlers: session_init, session_prepare_dek, session_prepare_iek, session_generate_table_hash, session_unwrap.
- internal/session/: session lifecycle management — store.go (in-memory session state) and cleanup.go (expiry/garbage collection of sessions).
- internal/utils/: cryptographic and I/O helpers — attestation.go (attestation doc generation/verification), connection.go (transport, likely vsock for enclave comms), deriveSessionKey.go (session key derivation), hash.go, openssl.go (OpenSSL-backed crypto primitives), zero.go (secure memory zeroing).
go.mod/go.sum define module deps; project.json integrates the app into an Nx monorepo build/target graph.

## Behavior
The service exposes handlers implementing an attestation-gated key exchange and decryption protocol. Typical flow: client requests an attestation document (get_attestation) proving enclave identity; a session is initialized (session_init) and a shared key derived (deriveSessionKey); key material is prepared inside the enclave (prepare_kek, session_prepare_dek, session_prepare_iek); wrapped keys are unwrapped (session_unwrap) and used for decryption (decrypt). session_generate_table_hash computes a hash over tabular data. Sessions are held in an in-memory store with periodic cleanup of stale entries. Secrets are wiped from memory via zero.go after use. health provides a liveness/readiness endpoint. Communication likely occurs over a vsock connection rather than standard TCP given the enclave context.

## Dependencies
Go module with its own go.mod (independent of other monorepo Go workspaces unless replaced). Relies on OpenSSL (via openssl.go) for cryptographic primitives. Depends on an enclave attestation runtime (e.g. AWS Nitro Secure Module / NSM) and vsock transport for host-enclave communication (connection.go). Integrated into the Nx workspace via project.json for build/serve/test targets. Consumers are host-side services that relay requests into the enclave.

## Notes
internal/ packages are deliberately unexported, so this app is not a reusable library. Security-sensitive: correctness of attestation verification, session key derivation, and memory zeroing (zero.go) is critical — review these paths carefully. The use of an external openssl.go wrapper implies shelling out to or binding native OpenSSL; ensure it matches the enclave's available crypto stack. Confirm session store concurrency safety and that cleanup goroutine lifecycle is tied to service shutdown.
