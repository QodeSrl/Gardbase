---
schema_version: 1
id: 52983633-ab3c-4cbd-83af-1255e9c08187
name: api
node: apps/api
category: app
---

## Purpose
Go HTTP API service providing tenant-scoped object storage and encryption operations. Exposes endpoints for health checks, tenant management, object CRUD, encryption/decryption, and dependency resolution, backed by DynamoDB/S3 and an enclave-based KMS for cryptographic operations.

## Structure
Standard Go layout. Entry point at cmd/server/main.go wires the server. internal/handlers holds HTTP request handlers (dependencies, encryption, healthCheck, objects, tenants). internal/middleware provides cross-cutting concerns: cors, permission, ratelimit, tenant resolution, and zapLogger structured logging. internal/services contains business/integration logic: dependencies, enclaveVSock (secure enclave communication over vsock), kms (key management). internal/storage abstracts persistence: dynamo.go (DynamoDB), s3.go (object storage), errors.go (storage error types). go.mod/go.sum declare modules; project.json integrates with Nx tooling.

## Behavior
main.go bootstraps the HTTP server and registers middleware and routes. Requests pass through middleware chain (CORS, rate limiting, tenant extraction, permission checks, logging) before reaching handlers. Handlers delegate to services and storage layers. Encryption/KMS flows communicate with a trusted enclave via VSock for sealing/unsealing or signing operations. Tenant middleware enforces multi-tenant isolation; storage layer maps operations to DynamoDB and S3.

## Dependencies
Go modules (see go.mod/go.sum). AWS SDK for DynamoDB and S3, zap for logging, vsock for enclave IPC, and an HTTP router/framework. Runtime depends on AWS DynamoDB and S3 services plus an available secure enclave reachable over VSock. Nx (project.json) for build/task orchestration in the monorepo.

## Notes
Multi-tenant security is central — tenant and permission middleware gate access, and KMS/enclave integration suggests handling of sensitive cryptographic material. Enclave VSock communication implies deployment in an AWS Nitro Enclaves context. Review .infrar/knowledge.md and commit.md for repo-specific context.
