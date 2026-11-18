# Deploying Gardbase on AWS

The Terraform configuration deploys the parent API and an AWS Nitro Enclave on EC2, with KMS, DynamoDB, S3, IAM, and CloudWatch resources. This is a self-hosting workflow: the deployment owner supplies the AWS account, configuration, cloud costs, and ongoing operations.

[Project overview](../README.md) · [API configuration](../apps/api/README.md) · [Enclave service](../apps/enclave-service/README.md)

## Prerequisites

- An AWS account with permission to provision the resources in this directory.
- AWS CLI credentials for the Terraform provider's named profile, `terraform`.
- Terraform compatible with the checked-in requirements (`>= 1.0.0`; AWS provider `~> 6.0`).
- Docker and a Linux build/deployment environment compatible with the target instance architecture.
- Nitro Enclaves tooling and an enclave-capable EC2 instance for EIF creation and runtime verification.
- Node.js/pnpm for the repository's Nx Docker targets, and a Go toolchain matching `go.work` for local Go builds.

```bash
aws sts get-caller-identity --profile terraform
```

The configurations are starting points to review and configure. They are not a zero-configuration production deployment.

## 1. Configure state and environment

There are two Terraform roots:

| Root                     | State key                     | Resources                                                      |
| ------------------------ | ----------------------------- | -------------------------------------------------------------- |
| [`bootstrap`](bootstrap) | `bootstrap/terraform.tfstate` | ECR repository and an S3 bucket reserved for Lambda code       |
| [`main`](main)           | `main/terraform.tfstate`      | EC2/enclave host, encrypted storage, keys, IAM, and monitoring |

Both backends currently reference an **existing** S3 bucket named `gardbase-terraform-state` in `eu-central-1`. The bootstrap root does not create that state bucket. Create/configure your own backend first, or point the backend configuration at an existing bucket you control.

If you change the bucket, keys, or region, update both backend blocks and the `terraform_remote_state.bootstrap` configuration in [`main/main.tf`](main/main.tf). Each environment needs its own state selection; changing only the `environment` variable does not change the hardcoded backend key.

The provider explicitly uses the `terraform` AWS profile. Review the region and project naming variables. Environment examples are under [`main/environments`](main/environments).

## 2. Provision bootstrap resources

From the repository root, after configuring state:

```bash
terraform -chdir=infrastructure/bootstrap init
terraform -chdir=infrastructure/bootstrap plan -var='environment=dev' -out=bootstrap.tfplan
terraform -chdir=infrastructure/bootstrap apply bootstrap.tfplan
terraform -chdir=infrastructure/bootstrap output
```

The ECR repository defaults to `gardbase-api`. The bootstrap also creates a Lambda-code bucket, but no Lambda function is deployed by the current roots.

## 3. Build and publish the container images

The API and enclave images use separate tags in the same ECR repository. Set your account/region values, authenticate the AWS CLI for that account, then run:

```bash
export AWS_PROFILE=terraform
export GARDBASE_AWS_ACCOUNT_ID="<your-account-id>"
export GARDBASE_AWS_REGION="eu-central-1"

pnpm exec nx run @gardbase/api:build-and-push \
  --aws_account_id="$GARDBASE_AWS_ACCOUNT_ID" \
  --aws_region="$GARDBASE_AWS_REGION"

pnpm exec nx run @gardbase/enclave-service:build-and-push \
  --aws_account_id="$GARDBASE_AWS_ACCOUNT_ID" \
  --aws_region="$GARDBASE_AWS_REGION"
```

The current targets publish `gardbase-api:latest-parent` and `gardbase-api:latest-enclave`. If you changed `project_name` or the ECR repository name, adjust the Docker targets too: their repository name is currently hardcoded. Match the container architecture to the target EC2 host and enclave. These commands build and push images; they do not apply Terraform.

## 4. Measure and configure the enclave

Build an EIF from the intended enclave image on a host with Nitro Enclaves tooling, then inspect its measurements:

```bash
nitro-cli describe-eif --eif-path /path/to/enclave.eif
```

Set `enclave_pcr0_sha384` to the measured PCR0 value. Its checked-in default is `PLACEHOLDER_PCR0` and must be replaced. Keep `enable_debug_mode = false` for a deployment relying on code-identity verification. Configure the client `ExpectedPCRs` with the corresponding measurements and enable client PCR verification.

Also review `allowed_ssh_cidr_blocks`, instance type, enclave CPU/memory allocation, KMS policy, public ingress, transport security, and tenant-provisioning access. The default SSH CIDR list contains an empty string and needs an explicit valid value. Pass environment-specific settings in an untracked `.tfvars` file or explicit variables; do not commit credentials or state.

## 5. Plan and apply the main stack

After supplying the environment and enclave settings above:

```bash
terraform -chdir=infrastructure/main init
terraform -chdir=infrastructure/main plan \
  -var-file=environments/dev.tfvars \
  -var-file=terraform.tfvars \
  -out=main.tfplan
terraform -chdir=infrastructure/main apply main.tfplan
terraform -chdir=infrastructure/main output
```

`terraform.tfvars` here is your untracked, reviewed configuration for PCR0, SSH access, and any other overrides. Keep `environment` consistent with bootstrap. Review the plan before applying it. Development resources include force-delete settings, so plan resource removal separately and retain required data before destroying an environment.

The EC2 startup script pulls the Docker images, builds the EIF, and configures the API/enclave services. It explicitly sets the API to enclave CID `16`, port `5000`. When running manually, note that the API's code default is `8080` while the enclave defaults to `5000`.

## 6. Verify the deployment

Use the returned instance outputs and your configured public API origin to check:

- `/api/health/` for API liveness.
- `/api/health/enclave` for vsock/enclave connectivity.
- `/api/health/storage` and `/api/health/kms` for dependency checks. Inspect the response body as well as the HTTP status; storage health can return HTTP 200 with an unhealthy status.
- A client session with expected PCRs, certificate/signature verification, and a freshness limit.
- An object round trip: client encryption, write, read, and client decryption.

A health response does not establish the complete security boundary. Review the [security model](../README.md#security-model), client integration, IAM/KMS policy, recovery strategy, and operational controls together.
