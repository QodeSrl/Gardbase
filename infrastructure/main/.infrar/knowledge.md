---
schema_version: 1
id: 3b184470-7eb0-4a73-b3a4-961e183776a9
name: infra-main
node: infrastructure/main
category: iac
---

## Purpose
Root Terraform module that provisions the primary AWS infrastructure stack, parameterized per environment (dev/prod) via tfvars. Defines compute, storage, encryption, and monitoring resources for the application.

## Structure
Flat root module organized by resource domain: main.tf (provider/backend/root config), ec2.tf (instances), s3.tf (buckets), dynamodb.tf (tables), kms.tf (encryption keys), cloudwatch.tf (logs/alarms/metrics), variables.tf (input declarations), outputs.tf (exported values). environments/ holds dev.tfvars and prod.tfvars for per-env values. user_data.sh is the EC2 bootstrap script. .terraform.lock.hcl pins provider versions. .infrar/knowledge.md is tooling-generated metadata.

## Behavior
Applied via `terraform plan/apply -var-file=environments/<env>.tfvars`. Provisions a complete stack: EC2 instances (bootstrapped with user_data.sh), S3 buckets, DynamoDB tables, KMS keys for at-rest encryption, and CloudWatch monitoring. Outputs in outputs.tf expose resource identifiers/endpoints for downstream consumers. State backend and provider configured in main.tf.

## Dependencies
Terraform CLI and AWS provider (versions locked in .terraform.lock.hcl). Requires AWS credentials and permissions for EC2, S3, DynamoDB, KMS, CloudWatch, IAM. KMS keys referenced cross-resource for encryption. Likely a remote state backend (e.g., S3/DynamoDB) declared in main.tf.

## Notes
No separate module abstraction — single root module per environment via tfvars rather than workspaces or distinct state dirs; confirm state isolation between dev and prod. Verify user_data.sh changes trigger intended instance replacement. Keep .terraform.lock.hcl committed for reproducible provider versions.
