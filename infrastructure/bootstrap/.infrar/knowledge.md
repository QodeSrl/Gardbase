---
schema_version: 1
id: d95a39f8-086c-48a2-aba2-0dad3965a295
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---

## Purpose
Terraform root module that provisions the foundational infrastructure required before other IaC stacks can run — typically the remote state backend (e.g. S3 bucket + DynamoDB lock table or equivalent), and possibly baseline IAM/project resources. This is the 'chicken-and-egg' bootstrap that establishes the state store other modules consume.

## Structure
Standard flat Terraform root module layout: main.tf declares providers and resources; variables.tf defines input variables; outputs.tf exposes computed values (e.g. backend bucket name, lock table name) for consumption by downstream stacks; .terraform.lock.hcl pins provider dependency versions for reproducible runs. No submodules present.

## Behavior
Run via the standard Terraform lifecycle: `terraform init`, `plan`, `apply`. Because it bootstraps the remote backend itself, this module is usually applied with local state first, after which the backend it creates is used by other stacks. Outputs are referenced by other modules (via data sources or wired variables) to configure their backend block.

## Dependencies
Requires Terraform CLI and the provider(s) pinned in .terraform.lock.hcl (commonly a cloud provider such as AWS/GCP/Azure). Depends on valid cloud credentials/permissions to create state-backing and baseline resources. Downstream IaC stacks depend on the outputs/resources this module creates.

## Notes
Bootstrap modules are sensitive: destroying them can orphan the remote state of all dependent stacks. Inspect main.tf to confirm exact provider and resources before applying. Confirm whether its own state is local or migrated to the backend it provisions. The lock file should be committed to ensure consistent provider versions across operators.
