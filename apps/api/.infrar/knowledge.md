---
schema_version: 1
id: 711bb2d1-9fb7-4dd3-8b8d-bb1d1b67434a
name: api
node: apps/api
category: app
---

## Purpose
HTTP API service for a multi-tenant object storage/encryption platform. Exposes endpoints for tenant management, object CRUD, encryption operations, and health checks, backed by AWS DynamoDB and S3, with KMS/enclave-based cryptography.

## Structure
Go application following a standard layered layout. cmd/server/main.go is the entrypoint that wires up the server. internal/handlers holds HTTP request handlers grouped by domain (encryption, healthCheck, objects, tenants). internal/middleware provides cross-cutting HTTP concerns: CORS, permission checks, rate limiting, tenant resolution, and Zap-based request logging. internal/services contains business/integration logic for KMS and an enclave client over VSock. internal/storage abstracts persistence via DynamoDB (dynamo.go) and S3 (s3.go) with shared error types (errors.go). go.mod/go.sum define module dependencies; project.json integrates the app into an Nx monorepo build/task pipeline.

## Behavior
main.go bootstraps the HTTP server, registers routes and middleware, then serves requests. Incoming requests pass through the middleware chain (CORS, logging, rate limiting, tenant extraction, permission enforcement) before reaching handlers. Handlers delegate to services (KMS, enclave VSock) for crypto operations and to storage adapters for reading/writing tenants and objects in DynamoDB and S3. Storage errors are normalized through the shared errors package for consistent HTTP responses.

## Dependencies
Go standard library plus AWS SDK for DynamoDB, S3, and KMS; Zap for structured logging; likely an HTTP router/framework and rate-limiting library (see go.mod for exact modules). Runtime relies on AWS services (DynamoDB, S3, KMS) and a Nitro Enclave reachable via VSock. Built and orchestrated within the Nx monorepo via project.json.

## Notes
The enclaveVSock service implies deployment inside/alongside an AWS Nitro Enclave for isolated cryptographic operations. Multi-tenancy is enforced at the middleware layer, so correct tenant and permission middleware ordering is security-critical. Confirm AWS credentials/IAM roles and enclave availability are provisioned in the target environment.
