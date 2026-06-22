---
schema_version: 1
id: dfe8a543-11f2-4d76-b699-a89daa5e2d61
name: infrastructure-main
node: infrastructure/main
category: iac
---

## Purpose
Root Terraform configuration defining the primary cloud infrastructure (AWS) for the project. Provisions compute, storage, data, encryption, and monitoring resources, parameterized per environment (dev/prod).

## Structure
Flat Terraform root module. main.tf holds provider/backend and core wiring; resource concerns split by file: ec2.tf (compute instances), s3.tf (object storage), dynamodb.tf (NoSQL tables), kms.tf (encryption keys), cloudwatch.tf (logging/alarms/metrics). variables.tf declares inputs; outputs.tf exposes computed values. environments/*.tfvars supply per-environment variable values. user_data.sh is the EC2 bootstrap/cloud-init script. .terraform.lock.hcl pins provider versions.

## Behavior
Applied via `terraform init/plan/apply` with a selected var-file (e.g. `-var-file=environments/dev.tfvars`). Terraform reconciles declared resources against actual cloud state. EC2 instances run user_data.sh on first boot for provisioning. KMS keys encrypt S3/DynamoDB/CloudWatch resources where referenced. Outputs surface IDs/endpoints for downstream consumption.

## Dependencies
Terraform CLI and the AWS provider (versions pinned in .terraform.lock.hcl). AWS credentials/region context required at apply time. Likely a remote backend (e.g. S3 + DynamoDB state lock) configured in main.tf. Inter-resource dependencies: KMS keys referenced by S3/DynamoDB/CloudWatch; EC2 may reference S3/DynamoDB outputs.

## Notes
Verify backend and state-locking configuration before applying to prod. Confirm tfvars do not contain secrets (use a secrets manager/SSM instead). The dynamodb.tf table may double as the state-lock table—check for circular bootstrap concerns. Review user_data.sh for idempotency and secret handling.
