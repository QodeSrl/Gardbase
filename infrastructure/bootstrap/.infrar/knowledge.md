---
schema_version: 1
id: 4a5a52d3-51e4-4cbd-82b6-19c90257dc3b
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---
## Purpose

`infrastructure-bootstrap` is the small Terraform workspace that has to run **before** anything else can be deployed. It creates the two long-lived artifact stores that the rest of the pipeline assumes already exist:

- an S3 bucket for Lambda deployment packages;
- an ECR repository that holds both container images built from this repo (the API server and the enclave service).

It is deliberately separated from `infrastructure/main` because of ordering: images must be pushed to ECR before the EC2 host boots and tries to pull them, and the main stack reads this workspace's outputs through a remote-state data source rather than re-declaring the resources. Splitting also means the bucket and repository survive a `destroy` of the application stack.

## Structure

Four files, all at `infrastructure/bootstrap`:

- `main.tf` — the `terraform` block (required version `>= 1.0.0`, `hashicorp/aws ~> 6.0`, S3 backend), the AWS provider, and both resources.
- `variables.tf` — `environment` (default `dev`), `project_name` (default `gardbase`), `region` (default `eu-central-1`).
- `outputs.tf` — `lambdas_bucket_name`, `lambda_bucket_arn`, `ecr_repository_url`, `ecr_repository_name`.
- `.terraform.lock.hcl` — pins the AWS provider to `6.13.0`.

There is no `environments/` directory here; the environment is passed on the command line (`terraform apply -var="environment=dev"`).

## Behavior

**State.** Remote S3 backend: bucket `gardbase-terraform-state`, key `bootstrap/terraform.tfstate`, region `eu-central-1`, `encrypt = true`. There is no DynamoDB lock table configured, so concurrent applies are not serialized.

**Resources.**

- `aws_s3_bucket.lambdas_bucket` — named `${project_name}-lambdas-bucket-${environment}`. `force_destroy` is `true` only when `environment == "dev"`, so a dev bucket can be torn down with objects still in it while other environments cannot.
- `aws_ecr_repository.api` — named `${project_name}-api`, with the same dev-only `force_delete` guard. Both application images live in this one repository, distinguished only by tag: `latest-parent` for the API server and `latest-enclave` for the enclave service.

**Consumption.** `infrastructure/main/main.tf` declares a `terraform_remote_state` data source against `bootstrap/terraform.tfstate` and passes `outputs.ecr_repository_url` into the EC2 user data, where it is used for `docker login`, `docker pull ...:latest-parent`, and `nitro-cli build-enclave --docker-uri ...:latest-enclave`. The bucket outputs are exported but nothing in the repo consumes them yet.

**Operational order.** `terraform apply` here → `nx run @gardbase/api:build-and-push` and `nx run @gardbase/enclave-service:build-and-push` (which tag and push into this ECR repository) → `terraform apply` in `infrastructure/main`. The Nx docker targets take `--aws_account_id` and `--aws_region` arguments and reconstruct the registry URL themselves rather than reading it from these outputs.

## Dependencies

**Upstream (must exist before first run, not managed here):** the `gardbase-terraform-state` S3 bucket that holds this workspace's own state, and AWS credentials able to create S3 buckets and ECR repositories.

**Providers:** `hashicorp/aws ~> 6.0` (locked at `6.13.0`).

**Downstream consumers:**
- `infrastructure/main` — reads this state for `ecr_repository_url`; the EC2 IAM policy grants the instance `ecr:GetAuthorizationToken`, `ecr:BatchCheckLayerAvailability`, `ecr:GetDownloadUrlForLayer`, and `ecr:BatchGetImage` so it can pull from here.
- The `api` and `enclave-service` nodes — their Nx `docker-push` targets are the only writers to this repository.

## Notes

- The ECR repository name is **not** environment-suffixed (`gardbase-api`, not `gardbase-api-dev`), while the Lambda bucket is. Applying this workspace against a second environment in the same account reuses the same repository, so `latest-parent` / `latest-enclave` tags are shared across environments and a push for dev also changes what prod would pull.
- Because tags are mutable and always `latest-*`, there is no image immutability or digest pinning; the enclave's PCR0 changes whenever a new enclave image is pushed and rebuilt, which invalidates the `enclave_pcr0_sha384` value baked into the KMS key policy in `infrastructure/main`.
- The Lambda bucket is provisioned but unused — there are no Lambda functions anywhere in the repo yet. The README lists a "Lambdas" component with no content.
- No lifecycle policy, versioning, encryption configuration, or public-access block is declared on the Lambda bucket, and no image scanning or lifecycle policy on the ECR repository.
- The backend block hardcodes the state bucket, key, and region, so the same three values are duplicated verbatim in `infrastructure/main/main.tf` (both in its own backend and in the remote-state data source).
- `terraform.tfvars` files are gitignored (`infrastructure/*/terraform.tfvars`), so per-environment values are expected to be passed with `-var` or an untracked tfvars file.
