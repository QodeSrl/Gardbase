---
schema_version: 1
id: 8fabb931-798c-4315-9564-4c3f9c8ddd1f
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---
## Purpose

`infrastructure-bootstrap` is the first-run Terraform workspace. It creates the small set of AWS resources that must exist *before* application images can be built and before the main stack can be deployed — specifically, somewhere to push container images and somewhere to store Lambda artifacts.

It exists to break a chicken-and-egg dependency: `infrastructure-main` launches an EC2 instance whose user-data pulls `:latest-parent` and `:latest-enclave` from ECR, so the ECR repository has to be created and populated first. This workspace is that step.

It is intentionally minimal and long-lived — you apply it once per environment and then rarely touch it again.

## Structure

A flat Terraform workspace at `infrastructure/bootstrap`, four files:

- `main.tf` — the `terraform` block (required version `>= 1.0.0`, AWS provider `~> 6.0`, S3 backend), the `aws` provider configuration, and both resources:
  - `aws_s3_bucket.lambdas_bucket` — named `${project_name}-lambdas-bucket-${environment}`, for Lambda deployment packages.
  - `aws_ecr_repository.api` — named `${project_name}-api`, the single repository holding both application images (differentiated by tag, not by repository).
- `variables.tf` — `environment` (default `dev`), `project_name` (default `gardbase`), `region` (default `eu-central-1`).
- `outputs.tf` — `lambdas_bucket_name`, `lambda_bucket_arn`, `ecr_repository_url`, `ecr_repository_name`.
- `.terraform.lock.hcl` — provider version lock.

There is no `environments/` directory here; variables are passed on the command line (`terraform apply -var="environment=dev"`).

## Behavior

**State.** The S3 backend is hard-coded: bucket `gardbase-terraform-state`, key `bootstrap/terraform.tfstate`, region `eu-central-1`, encrypted. That state bucket is itself a prerequisite this workspace does not create — it must exist before `terraform init`. There is no DynamoDB lock table configured, so concurrent applies are not protected against.

**Consumption by the main stack.** `infrastructure-main` reads this workspace's outputs through a `terraform_remote_state` data source pointing at the same bucket and the `bootstrap/terraform.tfstate` key. Only `ecr_repository_url` is actually consumed today — it is templated into the EC2 user-data so the instance can `docker login`, pull the parent image, and pull the enclave image that `nitro-cli build-enclave` converts into an EIF. This remote-state read is what makes bootstrap a hard ordering dependency rather than just a convention.

**Environment-conditional destruction.** Both resources are protective by environment: `force_destroy` on the bucket and `force_delete` on the ECR repository are `true` only when `environment == "dev"`. In any other environment a `terraform destroy` fails while objects or images remain, which is deliberate — it prevents accidentally deleting published images that running instances depend on.

**Typical flow.**
1. `terraform init && terraform apply -var="environment=dev"` here.
2. Build and push both images via the Nx targets (`nx run api:build-and-push`, `nx run enclave-service:build-and-push`), passing the AWS account ID and region.
3. Apply `infrastructure/main`, whose instances pull those images on boot.

Re-applying is effectively a no-op once the resources exist. Changing `environment` produces a *new* bucket (name is environment-suffixed) but reuses the same ECR repository, since its name is not environment-suffixed.

## Dependencies

- **Terraform** `>= 1.0.0`; **AWS provider** `~> 6.0`.
- **Pre-existing AWS resources**: the `gardbase-terraform-state` S3 bucket in `eu-central-1`.
- **AWS permissions**: create/manage S3 buckets and ECR repositories, plus read/write on the state bucket.
- **Downstream**: `infrastructure-main` (via remote state), and the `api` / `enclave-service` Docker push targets which target the ECR repository this creates.
- **Upstream**: nothing.

## Notes

- The state backend configuration is hard-coded rather than partial, so the same bucket and region are used for every environment — environments are separated by resource naming, not by state isolation. Two environments share one state file per workspace.
- The ECR repository name has no `${environment}` suffix while the S3 bucket does. Deploying multiple environments therefore shares a single image repository across them; the images are distinguished only by the `latest-parent` / `latest-enclave` tags, which are also mutable. There is no image immutability setting, no lifecycle policy, and no scan-on-push configured.
- Both images live in the *same* ECR repository despite very different trust levels — the enclave image is the one whose PCR0 measurement gates KMS access.
- The Lambda bucket is provisioned but nothing in the repository currently uploads to it; the README's "Lambdas" section under Components is an empty placeholder.
- The bucket has no explicit versioning, encryption, or public-access-block configuration (unlike the uploads bucket in `infrastructure-main`, which sets all three).
- No DynamoDB state lock table is configured for either workspace.
