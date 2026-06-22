---
schema_version: 1
id: 990ff5f1-5622-40b7-9c36-475d1e7cb949
name: enclave-service
node: apps/enclave-service
category: app
---

## Purpose
A Go service intended to run inside a secure enclave (e.g. AWS Nitro Enclaves) that handles attestation, session-based key management, and cryptographic operations (DEK/IEK/KEK preparation, unwrapping, decryption). It exposes handlers for establishing trusted sessions with a host/client, deriving session keys, and performing encryption/decryption while keeping key material isolated within the enclave.

## Structure
Standard Go application layout. Entry point at cmd/enclave/main.go. Business logic under internal/ (not importable outside the module):
- internal/handlers/: request handlers for each operation — health, get_attestation, prepare_kek, decrypt, and session lifecycle (session_init, session_prepare_dek, session_prepare_iek, session_unwrap, session_generate_table_hash).
- internal/session/: in-memory session state (store.go) and expiry/cleanup logic (cleanup.go).
- internal/utils/: crypto and infra helpers — attestation document handling (attestation.go), enclave/host connection (connection.go, likely vsock), session key derivation (deriveSessionKey.go), hashing (hash.go), OpenSSL wrappers (openssl.go), and secure memory zeroing (zero.go).
Module defined by go.mod/go.sum; project.json integrates the app into an Nx monorepo build/task graph.

## Behavior
main.go starts a listener (likely vsock for enclave isolation) and routes incoming requests to handlers. Typical flow: client requests an attestation document (get_attestation) to verify enclave identity; a session is initialized (session_init) and a session key derived (deriveSessionKey); the host then prepares key material (prepare_kek, session_prepare_iek, session_prepare_dek) which is unwrapped inside the enclave (session_unwrap); decryption (decrypt) and table-hash generation (session_generate_table_hash) run against session-scoped keys. Sessions are tracked in the store and periodically evicted by cleanup. Sensitive buffers are zeroed after use (zero.go). health.go provides a liveness/readiness probe.

## Dependencies
Go toolchain and the modules pinned in go.mod/go.sum (attestation/crypto libraries, likely AWS Nitro Enclaves SDK or vsock transport). Shells out to or links against OpenSSL (openssl.go). Nx workspace tooling consumes project.json for build/serve/test targets. Runtime expects an enclave host providing the vsock connection and key-wrapping inputs.

## Notes
internal/ enforces package privacy — handlers and crypto utils are not consumable by other apps. Security-critical: correctness of attestation verification, session key derivation, secure zeroing, and session expiry directly affect the trust model. The OpenSSL utility implies subprocess invocation or cgo — review for command-injection and proper memory handling. No tests are visible in the file list.
