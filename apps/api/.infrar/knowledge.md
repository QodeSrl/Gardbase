---
schema_version: 1
id: 51466eb8-e246-4157-9b04-30bbdcd1147e
name: api
node: apps/api
category: app
---

## Purpose
Go-based HTTP API service providing multi-tenant object storage with encryption. Exposes endpoints for tenant management, object CRUD, encryption operations, and health checks, backed by DynamoDB and S3, with KMS/enclave-based key management.

## Structure
Standard Go project layout. Entry point at cmd/server/main.go. internal/ holds non-exported packages: handlers/ (HTTP request handlers for encryption, health, objects, tenants), middleware/ (CORS, permission checks, rate limiting, tenant resolution, zap-based logging), services/ (enclaveVSock for secure enclave communication over VSock, kms for key management), storage/ (dynamo and s3 backends plus shared error types). go.mod/go.sum define module dependencies; project.json integrates with Nx monorepo tooling.

## Behavior
main.go bootstraps the server, wires middleware chain and routes handlers. Incoming requests pass through CORS, rate limiting, tenant identification, and permission middleware before reaching handlers. Handlers delegate to services (KMS/enclave for crypto) and storage (DynamoDB for metadata, S3 for object data). Encryption likely performed via AWS Nitro Enclave accessed over VSock. Storage layer normalizes backend errors via errors.go.

## Dependencies
Go modules (see go.sum). AWS SDK for DynamoDB, S3, and KMS. zap for structured logging. Nitro Enclave communication via VSock. Nx monorepo build orchestration via project.json. Consumed by other apps in the monorepo as the backend API.

## Notes
VSock usage and enclave service suggest deployment within AWS Nitro Enclaves for confidential compute. Multi-tenancy is enforced at the middleware layer—review tenant.go and permission.go for isolation correctness. Verify rate limiting is distributed-safe if running multiple instances.
