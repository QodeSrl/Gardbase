<br>

# Gardbase

Gardbase is a fully encrypted NoSQL DBaaS (Database-as-a-Service) built on AWS infrastructure that provides true zero-trust security. All data is encrypted client-side before leaving your application, while searchable encryption enables secure server-side indexing and queries. AWS Nitro Enclaves manage encryption keys in hardware-isolated environments, ensuring the backend never sees plaintext data. Think MongoDB Atlas meets end-to-end encryption! Ideal for healthcare, finance, and any application requiring verifiable data confidentiality.

## Features

- 🔒 Zero-Trust Encryption - All data encrypted client-side before transmission
- 🔐 End-to-End Encryption - Server never sees plaintext data
- 🛡️ AWS Nitro Enclaves - Cryptographic operations in isolated, attested environment
- 📊 DynamoDB + S3 Storage - Scalable hybrid storage (inline for small objects, S3 for large)
- 🔍 Encrypted Indexes - Search encrypted data using deterministic encryption
- 🔄 Optimistic Locking - Version-based concurrency control prevents conflicts
- 🚀 Type-Safe Generics - Fully typed SDK with Go 1.18+ generics
- 📖 ORM-like API - Intuitive API inspired by Mongoose and GORM

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

DynamoDB (Objects & Indexes):

- Lightweight encrypted binary data
- Searchable fields encrypted with deterministic encryption
- Enables exact-match queries on encrypted data
- Sortable fields use order-preserving encryption

S3 (Objects):

- Each object encrypted with unique DEK
- DEK wrapped with KMS key, stored as metadata
- Large binary data (documents, files, blobs)
- Envelope encryption provides key rotation flexibility

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
- No network access, no persistent storage, memory isolation

#### Lambdas

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

#### API

Location `pkg/api`
<br>
Language: Go

Purpose: API request and response structures for the Parent Application.

Contains:

- Request and response types for API endpoints (e.g., `GenerateDEKRequest`, `GenerateDEKResponse`, `UnwrapDEKRequest`, `UnwrapDEKResponse`)

#### Models

Location `pkg/models`
<br>
Language: Go

Purpose: Common data structures shared across components.

Contains:

- Database models (objects, indexes)

## Security Model

### Encryption Hierarchy

Gardbase uses a 4-level key hierarchy:

```
Level 1: AWS KMS Key (per environment)
           ↓ wraps
Level 2: Tenant Master Key (per tenant, random 32 bytes)
           ↓ encrypts
Level 3: Data Encryption Keys (DEKs, per object, random 32 bytes)
           ↓ encrypts
Level 4: Your Data (encrypted with DEK using AES-256-GCM)
```

### Properties

- Master Key stored encrypted (KMS-wrapped) on server
- Master Key only decrypted inside AWS Nitro Enclave
- DEKs generated fresh for each object
- All encryption happens client-side or in enclave
- Server never sees plaintext data or unencrypted keys

### Enclave Attestation

Every cryptographic operation goes through a verified AWS Nitro Enclave:

- Client requests enclave session
- Enclave provides cryptographic attestation document
- Client verifies attestation (proves code running in genuine enclave)
- Encrypted channel established
- Enclave performs key operations
- Keys never leave enclave in plaintext

What this means:

- Even Gardbase operators cannot access your keys
- Compromised backend cannot decrypt data
- Hardware-level isolation for sensitive operations

### Index Token Security

Searchable indexes use deterministic encryption:

- Same value always produces same encrypted token
- Enables equality queries on encrypted data
- Index tokens generated in enclave, never exposed
- Cannot reverse token back to original value

**Trade-off**: Deterministic encryption reveals if two records have the same value for an indexed field. Don't index highly sensitive fields if this is a concern.

## Contributing

We welcome contributions! Please fork the repository and submit a pull request with your changes. For major changes, please open an issue first to discuss what you would like to change.

## License

The project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
