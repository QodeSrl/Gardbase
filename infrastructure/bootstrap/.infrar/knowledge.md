---
schema_version: 1
id: cf2a5c36-586a-4a1a-94d3-36eb284262dc
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---
## Purpose

`infrastructure-bootstrap` is the first Terraform workspace applied to a fresh AWS account. It creates the small set of resources that must exist *before* application images can be built and pushed, and therefore before the main infrastructure can be deployed at all.

It solves an ordering problem. `infrastructure/main` provisions an EC2 host whose user-data pulls container images from ECR at boot — but the ECR repository has to exist, and images have to be in it, before that instance can start successfully. Bootstrap breaks the cycle by creating the registry (and a Lambda artifact bucket) as a separate, independently-applied step.

The deployment sequence is: **bootstrap → build and push images → main**.

## Structure

Four files, ~65 lines of HCL total:

- **`main.tf`** — the `terraform` block (required version `>= 1.0.0`, AWS provider `~> 6.0`, S3 backend), the `aws` provider, and both resources:
  - `aws_s3_bucket.lambdas_bucket` — `${project_name}-lambdas-bucket-${environment}`, for Lambda deployment packages.
  - `aws_ecr_repository.api` — `${project_name}-api`, the single registry that holds *both* container images, distinguished by tag rather than by repository.
- **`variables.tf`** — three variables, all with defaults: `environment` (`dev`), `project_name` (`gardbase`), `region` (`eu-central-1`).
- **`outputs.tf`** — four outputs: `lambdas_bucket_name`, `lambda_bucket_arn`, `ecr_repository_url`, `ecr_repository_name`.
- **`.terraform.lock.hcl`** — provider version lock.

There is no `environments/` directory here (unlike `infrastructure/main`); the environment is passed with `-var="environment=dev"` on the command line.

## Behavior

**State.** An S3 backend, hardcoded in `main.tf`:

```
bucket  = "gardbase-terraform-state"
key     = "bootstrap/terraform.tfstate"
region  = "eu-central-1"
encrypt = true
```

That bucket is *not* created by this workspace — it must pre-exist. There is no DynamoDB lock table configured, so concurrent applies are not protected against.

**Outputs are consumed cross-workspace.** `infrastructure/main` reads this state directly:

```hcl
data "terraform_remote_state" "bootstrap" {
  backend = "s3"
  config  = { bucket = "gardbase-terraform-state", key = "bootstrap/terraform.tfstate", region = "eu-central-1" }
}
```

and uses `outputs.ecr_repository_url` to template the EC2 user-data script, which pulls `$ECR_REPOSITORY_URL:latest-parent` and `$ECR_REPOSITORY_URL:latest-enclave`. This remote-state read is the hard coupling between the two workspaces.

**Environment-conditional destruction guards.** Both resources use the same pattern:

```hcl
force_destroy = var.environment == "dev" ? true : false   # S3
force_delete  = var.environment == "dev" ? true : false   # ECR
```

In `dev`, `terraform destroy` will delete a non-empty bucket and a repository containing images. In any other environment, destruction fails unless the contents are removed first — a deliberate safety rail against accidentally deleting production artifacts.

**Naming.** Every name is `${project_name}-...-${environment}` except the ECR repository, which is `${project_name}-api` with no environment suffix. Multiple environments therefore *share one registry*, separated only by image tag.

**Typical usage:**

```bash
cd infrastructure/bootstrap
terraform init
terraform plan  -var="environment=dev"
terraform apply -var="environment=dev"
```

Then push images (`nx run @gardbase/api:build-and-push`, `nx run @gardbase/enclave-service:build-and-push`), then apply `infrastructure/main`.

## Dependencies

**Upstream (must exist first):**

- An S3 bucket named `gardbase-terraform-state` in `eu-central-1` for Terraform state.
- AWS credentials with permission to create S3 buckets and ECR repositories.
- Terraform `>= 1.0.0`.

**Providers:**

- `hashicorp/aws` `~> 6.0` — the only provider.

**Downstream (depends on this):**

- `infrastructure/main`, via `terraform_remote_state`, for `ecr_repository_url`.
- The Nx Docker targets in `apps/api/project.json` and `apps/enclave-service/project.json`. These do *not* read Terraform outputs — they reconstruct the registry URL from `--aws_account_id` and `--aws_region` arguments and hardcode the repository name as `gardbase-api`. The two must be kept consistent by hand.

**Not created here:** the state bucket, the KMS key, DynamoDB tables, the uploads bucket, EC2, IAM, CloudWatch — all of that belongs to `infrastructure/main`.

## Notes

- **The state bucket is a chicken-and-egg gap.** `gardbase-terraform-state` is required by the backend but created by no workspace in this repository. First-time setup requires creating it manually (or running bootstrap once with a local backend and migrating).
- **The Lambda bucket is currently unused.** No Lambda functions exist in this repository — `infrastructure/main` provisions none, and the root README's "Lambdas" section under Apps is an empty heading. The bucket and its two outputs are provisioning for planned work.
- **No state locking.** The S3 backend has no `dynamodb_table` (or `use_lockfile`) configured, so simultaneous applies can corrupt state.
- **The S3 bucket has no hardening.** Unlike `aws_s3_bucket.uploads` in `infrastructure/main` — which gets versioning, server-side encryption, and a public access block — `lambdas_bucket` is a bare bucket resource with none of those.
- **The ECR repository has no lifecycle policy, no image scanning, and no tag immutability.** Both images are pushed to fixed `latest-parent` / `latest-enclave` tags, so every deploy overwrites the previous image and no history is retained. For the enclave image in particular this matters: the PCR0 measurement that gates KMS access is derived from the image, and a mutable tag means the measurement can change under a running deployment.
- **The environment suffix is inconsistent.** The S3 bucket is per-environment; the ECR repository is not. Running bootstrap for `dev` and `prod` yields two buckets but one shared repository — and since both environments would push to the same `latest-parent` / `latest-enclave` tags, they would overwrite each other.
- **Region is doubly specified.** The backend region is hardcoded to `eu-central-1` while the provider region comes from `var.region` (also defaulting to `eu-central-1`). Overriding `region` moves the resources but not the state.
- S3 bucket names are globally unique across all of AWS, so `gardbase-lambdas-bucket-dev` will collide for anyone forking this project without changing `project_name`.
