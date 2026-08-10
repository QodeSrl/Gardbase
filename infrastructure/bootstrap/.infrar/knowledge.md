---
schema_version: 1
id: 2d31af7f-282e-4dfd-af8e-440ced643563
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---
## Purpose

`infrastructure-bootstrap` is the chicken-and-egg workspace: the minimal set of AWS resources that must exist *before* anything else can be built or deployed. It creates the ECR repository that container images are pushed to, and an S3 bucket reserved for Lambda artifacts.

It is separate from `infrastructure-main` because the main stack consumes an ECR image URI when it templates EC2 user data — the repository has to exist and the images have to be pushed before the instance that pulls them can be provisioned. Bootstrap is applied once per environment and then left largely alone.

## Structure

```
infrastructure/bootstrap/
  main.tf                # terraform block, S3 backend, aws provider, the two resources
  variables.tf           # environment, project_name, region
  outputs.tf             # bucket name/ARN, ECR repository URL/name
  .terraform.lock.hcl    # provider lockfile
```

Resources created:

| Resource | Name pattern |
| --- | --- |
| `aws_s3_bucket.lambdas_bucket` | `<project_name>-lambdas-bucket-<environment>` |
| `aws_ecr_repository.api` | `<project_name>-api` |

Variables, all with defaults: `environment` = `dev`, `project_name` = `gardbase`, `region` = `eu-central-1`.

Outputs: `lambdas_bucket_name`, `lambda_bucket_arn`, `ecr_repository_url`, `ecr_repository_name`.

## Behavior

State lives in the S3 backend bucket `gardbase-terraform-state` under key `bootstrap/terraform.tfstate` in `eu-central-1`, encrypted. That bucket is not managed by this workspace — it must exist before the first `terraform init`.

Both resources are environment-aware in exactly one way: destructive protection relaxes in dev. `force_destroy` on the bucket and `force_delete` on the ECR repository are `true` when `environment == "dev"` and `false` otherwise, so a production repository holding images cannot be torn down by accident.

The ECR repository is deliberately singular and shared. Both applications publish into `<project>-api`, distinguished only by tag — `latest-parent` for the API and `latest-enclave` for the enclave service. The Nx `docker-build` / `docker-tag` / `docker-login` / `docker-push` targets in `apps/api/project.json` and `apps/enclave-service/project.json` (and the `build-and-push` composite) target this repository, taking the account id and region as arguments.

Standard usage:

```bash
cd infrastructure/bootstrap
terraform init
terraform plan  -var="environment=dev"
terraform apply -var="environment=dev"
```

Then push both images, then apply `infrastructure/main`.

`infrastructure-main` reads this workspace's state through a `terraform_remote_state` data source and uses `ecr_repository_url` to template the EC2 user-data script. That single output is the entire contract between the two stacks.

## Dependencies

- **Providers:** `hashicorp/aws ~> 6.0`; Terraform `>= 1.0.0`.
- **Pre-existing and unmanaged:** the `gardbase-terraform-state` S3 bucket used as the backend.
- **Credentials:** the default AWS provider chain; the principal needs permission to create S3 buckets and ECR repositories.
- **Consumers:** `infrastructure-main` via remote state; the `api` and `enclave-service` Nx docker targets, which push here.

## Notes

- Apply order matters and Terraform does not enforce it: bootstrap → build and push both images → main. The main stack's user data pulls `:latest-parent` and `:latest-enclave` on boot, and its enclave systemd unit builds the EIF from the pulled image, so missing tags produce a host that comes up without a working enclave.
- The `lambdas_bucket` is provisioned ahead of need — the repository contains no Lambda functions yet.
- The state bucket name, key, and region are hardcoded in the `backend` block, so `var.region` only affects where *resources* land, not where state lives. Deploying to a second region means editing the backend or passing `-backend-config`.
- Neither resource sets tags, unlike `infrastructure-main`, which applies provider-level `default_tags`.
- ECR image scanning, lifecycle policies, and tag immutability are not configured. The `latest-*` tags are overwritten in place on every push, which means a redeploy of the enclave image silently changes its PCR0 measurement and invalidates the value pinned in the KMS key policy.
- Because one repository serves both images, an ECR lifecycle policy added later would have to be tag-aware to avoid expiring one application's image while pruning the other's.
