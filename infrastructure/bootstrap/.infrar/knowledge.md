---
schema_version: 1
id: 54285f00-f0a2-453a-a05c-ee18f01ab03b
name: infrastructure-bootstrap
node: infrastructure/bootstrap
category: iac
---
## Purpose

`infrastructure-bootstrap` is the chicken-and-egg workspace: the minimal set of AWS resources that must exist *before* anything else can be built or deployed. It creates the ECR repository that container images are pushed to and an S3 bucket for Lambda artifacts.

It is separate from `infrastructure-main` because the main stack consumes ECR image URIs at plan time — the repository has to already exist, and the Docker images have to already be pushed, before the EC2 instance that pulls them can be provisioned. Bootstrap is applied once per environment and then left largely alone.

## Structure

```
infrastructure/bootstrap/
  main.tf                # terraform block, S3 backend, aws provider, the two resources
  variables.tf           # environment, project_name, region
  outputs.tf             # bucket name/ARN, ECR repository URL/name
  .terraform.lock.hcl    # pins hashicorp/aws 6.13.0
```

Resources created:

| Resource | Name pattern |
| --- | --- |
| `aws_s3_bucket.lambdas_bucket` | `<project_name>-lambdas-bucket-<environment>` |
| `aws_ecr_repository.api` | `<project_name>-api` |

Variables (all with defaults): `environment` = `dev`, `project_name` = `gardbase`, `region` = `eu-central-1`.

## Behavior

State lives in the S3 backend `gardbase-terraform-state` under key `bootstrap/terraform.tfstate` in `eu-central-1`, encrypted. That bucket is not managed by this workspace — it must be created out-of-band before the first `terraform init`.

Both resources are environment-aware in exactly one way: destructive protection is relaxed in dev. `force_destroy` on the bucket and `force_delete` on the ECR repository are `true` when `environment == "dev"` and `false` otherwise, so a production repository holding images cannot be torn down by accident.

The ECR repository is deliberately *singular* and shared: both applications publish into `<project>-api`, distinguished only by tag — `latest-parent` for the API and `latest-enclave` for the enclave service. The Nx `docker-*` targets in `apps/api/project.json` and `apps/enclave-service/project.json` build, tag, log into, and push against this repository.

Standard usage:

```bash
cd infrastructure/bootstrap
terraform init
terraform plan  -var="environment=dev"
terraform apply -var="environment=dev"
```

Then push images, then apply `infrastructure/main`.

`infrastructure-main` reads this workspace's state through a `terraform_remote_state` data source and uses `ecr_repository_url` to template the EC2 user-data script. That output is the entire contract between the two stacks.

## Dependencies

- **Provider:** `hashicorp/aws ~> 6.0` (lockfile pins 6.13.0); Terraform >= 1.0.0.
- **Pre-existing, unmanaged:** the `gardbase-terraform-state` S3 bucket.
- **Credentials:** default AWS provider chain; the principal needs S3 and ECR create permissions.
- **Consumers:** `infrastructure-main` (remote state); the `api` and `enclave-service` Nx docker targets (push destination).

## Notes

- Apply order matters and is not enforced by Terraform: bootstrap → build & push both images → main. The main stack's EC2 user-data pulls `:latest-parent` and `:latest-enclave` on boot, and its enclave systemd unit builds the EIF from the pulled image — missing tags mean a host that comes up without a working enclave.
- The `lambdas_bucket` is provisioned ahead of need; the repo has no Lambda functions yet (the README lists "Lambdas" as an empty section).
- The state bucket name and region are hardcoded in the `backend` block, so `var.region` only affects where *resources* land, not where state lives. Deploying to a second region means editing the backend or using `-backend-config`.
- Neither resource sets tags, unlike `infrastructure-main`, which applies provider-level `default_tags`.
- ECR image scanning, lifecycle policies, and immutable tags are not configured; `latest-*` tags are overwritten in place on every push, which means a redeploy of the enclave image silently changes its PCR0 measurement.
