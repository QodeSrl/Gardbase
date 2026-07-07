---
schema_version: 1
id: 4f3f58ba-53aa-4c3a-bb4f-ce0e15499d40
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---

## Purpose
Terraform root module responsible for provisioning the foundational infrastructure required before other IaC stacks can run. Typically bootstraps remote state backend resources (e.g., state bucket, lock table) and any baseline account-level primitives that subsequent stacks depend on.

## Structure
Standard flat Terraform module layout: main.tf defines the provider configuration and resources; variables.tf declares input variables; outputs.tf exposes computed values (such as backend resource identifiers) for consumption by other stacks or documentation. The .terraform.lock.hcl pins provider dependency versions and their verified hashes for reproducible initialization.

## Behavior
Executed via the standard Terraform workflow (terraform init, plan, apply). On apply it creates the declared bootstrap resources and emits outputs. Because it commonly manages the remote state backend itself, it is often applied with local state first, then optionally migrated to the created backend. It is intended to be run once per environment/account before dependent stacks.

## Dependencies
Requires Terraform CLI and the providers pinned in .terraform.lock.hcl (typically a cloud provider such as AWS/GCP/Azure). Depends on valid provider credentials and the input values supplied through variables.tf. Downstream infrastructure stacks depend on the outputs produced here (e.g., state backend configuration).

## Notes
Review main.tf to confirm the exact resources and provider. Treat this module as a prerequisite for other IaC nodes. Keep .terraform.lock.hcl committed to ensure consistent provider versions across runs and CI.
