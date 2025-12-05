<div align="center">

<br>

# Gardbase

</div>

## Getting Started

### Prerequisites

- Go 1.20+
- Docker
- Terraform 1.5+
- Node.js 22.14.0+
- pnpm 10.13.1+
- AWS CLI configured with an account that can create: VPC, Lambda, ECR, EC2, IAM, S3, DynamoDB tables, KMS keys, CloudWatch resources

```bash
aws configure
aws sts get-caller-identity
```

### Bootstrap Lambda S3 Bucket & ECR Repo

A minimal Terraform workspace sets up:

- S3 bucket for Lambda code
- ECR repository for API Docker images

```bash
cd infrastructure/bootstrap
terraform init
terraform plan -var="environment=dev"
terraform apply -var="environment=dev"
```

### Build & Deploy Lambda

```bash
nx run @gardbase/lambdas/upload-processor:build
nx run @gardbase/lambdas/upload-processor:package
nx run @gardbase/lambdas/upload-processor:push --bucket=<s3-bucket-name>
# Or
nx run @gardbase/lambdas/upload-processor:build-and-push --bucket=<s3-bucket-name>
```

### Build & Push API Docker Image

```bash
nx run api:docker-build
nx run api:docker-tag --aws_account_id=<aws-account-id> --aws_region=<aws-region>
nx run api:docker-login --aws_account_id=<aws-account-id> --aws_region=<aws-region>
nx run api:docker-push --aws_account_id=<aws-account-id> --aws_region=<aws-region>
# Or
nx run api:build-and-push --aws_account_id=<aws-account-id> --aws_region=<aws-region>
```

### Build & Push Enclave Service Docker Image

```bash
nx run enclave-service:docker-build
nx run enclave-service:docker-tag --aws_account_id=<aws-account-id> --aws_region=<aws-region>
nx run enclave-service:docker-login --aws_account_id=<aws-account-id> --aws_region=<aws-region>
nx run enclave-service:docker-push --aws_account_id=<aws-account-id> --aws_region=<aws-region>
# Or
nx run enclave-service:build-and-push --aws_account_id=<aws-account-id> --aws_region=<aws-region>
```

### Deploy Full Infrastructure

```bash
cd infrastructure/main
terraform init
terraform apply -var="environment=dev"
```

## Components

### Apps

#### API

#### Enclave Service

#### Lambdas

- Upload Processor

### Packages

#### Crypto

#### Enclaveproto

#### Models
