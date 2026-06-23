---
schema_version: 1
id: 4735bb63-8b50-4d02-a520-4eb5fa5cc0f3
name: infra-bootstrap
node: infrastructure/bootstrap
category: iac
---

## Purpose
Terraform root module that provisions the foundational ('bootstrap') infrastructure required before other IaC stacks can run — typically remote state backend resources (e.g. S3 bucket / DynamoDB lock table or GCS bucket), IAM roles/service accounts, and KMS keys. This is the chicken-and-egg layer applied first, usually with local state, to enable remote-state-backed stacks downstream.

## Structure
Standard Terraform module layout: main.tf (primary resource/provider declarations), variables.tf (input variable definitions), outputs.tf (exported values consumed by downstream stacks, e.g. bucket name, lock table name, role ARNs). .terraform.lock.hcl pins provider versions and checksums for reproducible installs. .infrar/knowledge.md holds node-local documentation/metadata. No backend.tf is listed, consistent with bootstrap modules using local state.

## Behavior
On `terraform init/plan/apply`, creates the bootstrap resources defined in main.tf and surfaces their identifiers via outputs. Provider versions are constrained by the lock file. Inputs in variables.tf parameterize environment/region/naming. Idempotent on repeated apply; destroying this stack would invalidate any downstream stacks that depend on its outputs (notably the remote state backend).

## Dependencies
Requires Terraform CLI and the providers pinned in .terraform.lock.hcl (cloud provider plus likely random/tls). Implicit dependency on cloud credentials/permissions sufficient to create state-backend and IAM resources. Downstream IaC stacks depend on this node's outputs; this node should have no IaC dependencies itself.

## Notes
Apply order matters: this must run before any stack that references the remote backend it creates. Verify whether state is local or migrated to the created backend after first apply. Treat destroy operations with caution due to backend/state coupling. Keep .terraform.lock.hcl committed for deterministic provider resolution.
