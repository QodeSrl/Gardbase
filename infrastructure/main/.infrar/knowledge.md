---
schema_version: 1
id: 8664482d-aaa5-4778-b79b-06d97547213d
name: infrastructure-main
node: infrastructure/main
category: iac
---

## Purpose
Terraform root module that provisions the primary AWS infrastructure stack for the project, including compute (EC2), storage (S3, DynamoDB), encryption (KMS), and observability (CloudWatch). Supports multiple deployment environments via tfvars files.

## Structure
Flat root module organized by AWS service into separate .tf files: main.tf (provider/backend and core configuration), variables.tf (input variable declarations), outputs.tf (exported resource attributes), and per-service resource files — ec2.tf, s3.tf, dynamodb.tf, kms.tf, cloudwatch.tf. user_data.sh is the EC2 instance bootstrap script. environments/ holds environment-specific variable values (dev.tfvars, prod.tfvars). .terraform.lock.hcl pins provider versions.

## Behavior
Applied with `terraform apply -var-file=environments/<env>.tfvars`. Terraform reconciles declared resources against remote state and the live AWS account. EC2 instances execute user_data.sh at launch for instance initialization. KMS keys provide encryption for S3 buckets and DynamoDB tables; CloudWatch defines metrics/alarms/log groups. Outputs expose identifiers (e.g., instance IDs, bucket names, table names) for consumption by other tooling or modules.

## Dependencies
Requires Terraform CLI and the AWS provider (version-pinned in .terraform.lock.hcl). Depends on valid AWS credentials and permissions for EC2, S3, DynamoDB, KMS, and CloudWatch. Assumes a configured Terraform backend for state (defined in main.tf). user_data.sh may depend on external package repositories or bootstrap artifacts.

## Notes
Environment separation is achieved via tfvars rather than distinct workspaces/directories — verify state isolation between dev and prod to avoid cross-environment drift. Keep the lock file committed for reproducible provider versions. Sensitive values should not be hardcoded in tfvars; prefer secrets management or KMS-backed parameters.
