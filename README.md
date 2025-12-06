<div align="center">

<br>

# Gardbase

</div>

Gardbase is a secure Database-as-a-Service (DBaaS) that provides client-side encrypted storage with searchable encryption. It stores encrypted data objects in S3 and encrypted, searchable indexes in DynamoDB, using AWS Nitro Enclaves to securely manage encryption keys in a zero-trust architecture: the system ensures that sensitive data is never exposed in plaintext outside of trusted environments, leveraging hardware isolation and attestation to maintain data confidentiality and integrity.

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

### Storage

S3 (Objects):

- Each object encrypted with unique DEK
- DEK wrapped with KMS key, stored as metadata
- Large binary data (documents, files, blobs)
- Envelope encryption provides key rotation flexibility

DynamoDB (Indexes):

- Searchable fields encrypted with deterministic encryption
- Enables exact-match queries on encrypted data
- Sortable fields use order-preserving encryption
- Metadata and pointers to S3 objects

### Apps

#### Parent Application (API Server)

Location `apps/api`
<br>
Runs on: EC2 host instance
<br>
Language: Go

Purpose: Main API server for Gardbase.

Responsibilities:

- Manages user authentication and authorization
- Handles file uploads from users
- Manages AWS resources (S3, DynamoDB) for storing encrypted data and metadata
- Acts as a proxy to communicate with the Enclave Service for sensitive data processing

Key Features:

- RESTful API endpoints for client interactions
- Uses AWS SDK to interact with S3 and DynamoDB
- vsock client to communicate securely with the Enclave Service
- Does NOT have direct access to unencrypted data or encryption keys (zero-trust design)

#### Enclave Service

Location `apps/enclave-service`
<br>
Runs on: AWS Nitro Enclave (isolated environment on EC2)
<br>
Language: Go

Purpose: Performs sensitive cryptographic operations in a hardware-isolated environment.

Responsibilities:

- Handles sessions with the client through ephemeral X25519 keypairs (ECDSA + HKDF key derivation)
- Provides attestation documents (signed by AWS) to prove the enclave's identity and integrity
- Decrypts Data Encryption Keys (DEKs) using AWS KMS within the enclave
- Seals DEKs with session keys before returning to the API server
- Zeros out sensitive data in memory after use to prevent leakage

Key Features:

- Uses AWS Nitro Enclaves SDK for secure operations
- vsock server to communicate with the API server
- NSM (Nitro Secure Module) for attestation, code measurements (PCRs) verifiable by clients
- KMS client
- No network access, no persistent storage, memory isolation

#### Lambdas

- Upload Processor: Processes file uploads, extracts metadata, and stores information in DynamoDB.

### Packages

#### Crypto

Location `pkg/crypto`
<br>
Language: Go

Purpose: Client-side cryptographic operations for secure data handling, enables applications to securely request DEK unwrapping with full attestation verification.

Responsibilities:

- Simplifies cryptographic operations for clients
- Generates ephemeral X25519 keypairs
- Initiates sessions with the enclave
- Verifies attestation documents (critical for security):
  - Validates certificate chain to AWS Root CA
  - Verifies COSE signatures
  - Checks PCR values match expected code measurements
  - Validates nonce freshness
  - Confirms public key binding
- Unseals DEKs received from enclave

#### Enclaveproto

Location `pkg/enclaveproto`
<br>
Language: Go

Purpose: Protocol definitions for parent-enclave communication.

Contains:

- Base Request and Response structures
- Specific message types (`SessionInitRequest`, `SessionInitResponse`, `SessionGenerateDEKRequest`, `SessionGenerateDEKResponse`, etc.)

#### Models

Location `pkg/models`
<br>
Language: Go

Purpose: Common data structures shared across components.

Contains:

- Database models (objects, indexes)

## Runtime flow

### Session Initialization

```
1. Client generates ephemeral keypair + nonce
2. Client → Parent: { clientPubKey, nonce }
3. Parent → Enclave: Forward request
4. Enclave:
   - Generates ephemeral keypair
   - Derives session key
   - Requests attestation from NSM (includes nonce, pubKey)
5. Enclave → Parent → Client: { enclavePubKey, attestation }
6. Client:
   - Verifies attestation (8 checks)
   - Derives same session key
   - Session established ✓
```

### DEK Generation

```
1. Client → Parent: { sessionID, kmsKeyID, count }
2. Parent → Enclave: Forward request
3. Enclave:
   - Validates session
   - For i in 1..count:
      - Calls KMS.GenerateDataKey with attestation
      - KMS returns { plaintextDEK, wrappedDEK }
      - Seals plaintextDEK with session key + objectID as associated data
      - Zeros plaintext DEK from memory
4. Enclave → Parent → Client: { { sealedDEK, wrappedDEK, nonce }[] }
5. Client:
    - Unseals with session key
    - Uses SealedDEK for local encryption
    - Stores wrappedDEK with object metadata for later unwrapping
```

### DEK Unwrapping

```
1. Client → Parent: { sessionID, { objectID, wrappedDEK }[] }
2. Parent → Enclave: Forward request
3. Enclave:
   - Validates session
   - For each { objectID, wrappedDEK }:
      - Calls KMS.Decrypt with attestation
      - KMS returns DEK encrypted for enclave's RSA key
      - Decrypts with enclave's RSA private key
      - Seals with session key + objectID as associated data
      - Zeros plaintext DEK from memory
4. Enclave → Parent → Client: { { objectID, sealedDEK }[], nonce }
5. Client:
   - Unseals with session key
   - Uses DEK to decrypt application data
```
