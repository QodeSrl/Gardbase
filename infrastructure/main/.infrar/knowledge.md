---
schema_version: 1
id: 89d01231-f4b5-44e8-8446-34c69f8d391c
name: infrastructure-main
node: infrastructure/main
category: iac
---
## Purpose

`infrastructure-main` is the Terraform workspace that provisions the entire Gardbase runtime environment on AWS: the storage layer, the KMS key that anchors the encryption hierarchy, the enclave-enabled EC2 host that runs both applications, the IAM permissions binding them together, and the observability around it.

Its defining responsibility is the **KMS key policy**. That policy is where Gardbase's zero-trust claim is enforced in infrastructure rather than in code: KMS will only release key material to a caller presenting a Nitro attestation document whose PCR0 measurement matches an expected value. Neither the API process, nor an operator with SSH access, nor anyone holding the instance's IAM role can obtain plaintext keys without running the exact enclave image that was measured.

It is applied *after* `infrastructure/bootstrap` and after both container images have been pushed to ECR, because the EC2 instance pulls those images during boot.

## Structure

Seven `.tf` files, a user-data script, and per-environment variable files:

- **`main.tf`** — provider requirements (AWS `~> 6.0`, TLS `~> 4.0`), the S3 backend (`main/terraform.tfstate`), a `terraform_remote_state` data source reading the bootstrap workspace, the AWS provider with `default_tags` (Project / Environment / ManagedBy), and `data.aws_caller_identity.current`.
- **`variables.tf`** — 10 variables, all with defaults. Notable: `instance_type` (`m5a.xlarge`, with a comment noting not all instance types support Nitro Enclaves), `enclave_cpus` (2), `enclave_memory_mib` (2048), `enable_debug_mode` (false), `max_attestation_age_minutes` (5), `allowed_ssh_cidr_blocks`, `enclave_pcr0_sha384` (default `"PLACEHOLDER_PCR0"`), `kms_key_deletion_window_days` (30).
- **`kms.tf`** — `aws_kms_key.enclave_key` with rotation enabled and the attestation-gated policy, plus an alias.
- **`ec2.tf`** (the largest, ~250 lines) — default VPC/subnet lookups, the security group, the IAM role, inline policy and instance profile, the AL2023 AMI lookup, a generated RSA-4096 SSH keypair stored in SSM, and `aws_instance.api` with `enclave_options.enabled = true`.
- **`dynamodb.tf`** — five tables: `objects`, `indexes`, `table_configs`, `tenant_configs`, `api_keys`.
- **`s3.tf`** — the `uploads` bucket plus lifecycle, versioning, SSE, and public-access-block configurations.
- **`cloudwatch.tf`** — two log groups (api, enclave) at 7-day retention and two metric alarms (high CPU, instance health).
- **`user_data.sh`** — 444 lines of bash, templated by Terraform, that turns a bare AL2023 instance into a running Gardbase host.
- **`outputs.tf`** — bucket and table names, instance ID / public DNS / public IP, the SSM path for the SSH key (marked sensitive), enclave configuration, and the KMS key ID and ARN.
- **`environments/dev.tfvars`, `environments/prod.tfvars`** — three lines each: `environment`, `project_name`, `region`.

## Behavior

### The KMS key policy — the security anchor

`aws_kms_key.enclave_key` has rotation enabled and two policy statements. The second is the important one: it grants the EC2 instance role `GenerateDataKey`, `Decrypt`, `ReEncrypt`, `Encrypt`, and `DescribeKey`, but only under the condition

```hcl
"kms:RecipientAttestation:ImageSha384" = var.enable_debug_mode ? "*" : var.enclave_pcr0_sha384
```

When debug mode is off, KMS evaluates the attestation document supplied in the `Recipient` field of the API's `GenerateDataKey`/`Decrypt` calls and refuses the request unless PCR0 matches. Because the response is then returned as `CiphertextForRecipient` — encrypted to the enclave's NSM public key — even a successful call yields nothing usable to the caller. Setting `enable_debug_mode = true` replaces the condition with `*`, disabling this gate entirely.

The first statement grants the account root full `kms:*`, which is standard KMS practice to avoid locking the key out.

### Storage

**DynamoDB** — five provisioned-capacity tables, all named `${project_name}-<table>-${environment}`:

| Table | Keys | Notes |
| --- | --- | --- |
| `objects` | `pk` (S) / `sk` (S) | TTL on `ttl`, point-in-time recovery, SSE |
| `indexes` | `pk` (S) / `sk` (**B**) | GSI `gsi1` on `gsi1pk`/`gsi1sk`, `projection_type = ALL` |
| `table_configs` | `pk` (S) | |
| `tenant_configs` | `pk` (S) | holds wrapped master key and wrapped table salt |
| `api_keys` | `pk` (S) / `sk` (S) | |

The `indexes` table's binary sort key is what makes encrypted range queries work: order-preserving ciphertexts sort correctly as raw bytes.

**S3** — `aws_s3_bucket.uploads` for large encrypted blobs, with versioning enabled, AES256 server-side encryption, all four public-access blocks on, `force_destroy` only in dev, and a lifecycle rule expiring objects tagged `status=deleted` after 30 days. That tag is what the API's `TagForDeletion` / `UntagForDeletion` methods write, implementing soft delete with a recovery window.

### Compute

The instance is created in the **default VPC**, on the first default subnet, with a public IP. `enclave_options { enabled = true }` is what makes Nitro Enclaves available. Root volume is 30 GB gp3, encrypted. Detailed monitoring is on. Lifecycle rules set `create_before_destroy = true` and `ignore_changes = [ami]` so routine AMI updates do not force replacement — but `user_data_replace_on_change = true` means any edit to the user-data template *does* replace the instance.

The security group allows 80 and 443 from anywhere, egress to anywhere, and SSH from `var.allowed_ssh_cidr_blocks` in `prod` or `0.0.0.0/0` in every other environment.

An RSA-4096 keypair is generated by the `tls` provider; the public half becomes an `aws_key_pair` and the private half is stored as an SSM `SecureString` at `/${project_name}/${environment}/api/ssh-private-key`.

The IAM inline policy grants: scoped S3 object operations on the uploads bucket; DynamoDB read/write on all five tables and their indexes; ECR pull (`Resource = "*"`, as ECR auth requires); the five KMS actions on the enclave key ARN; CloudWatch Logs writes scoped to `/aws/ec2/${project_name}*`; and `ssm:PutParameter` scoped to `/${project_name}/${environment}/*` (used by the PCR extraction script).

### Boot sequence (`user_data.sh`)

Templated with the region, project name, environment, ECR URL, all five table names, the bucket, the KMS key ID, and the enclave sizing/debug settings. It then:

1. Fetches the public hostname from IMDSv2 (token-based) — used to set `BASE_URL`.
2. `dnf install` Docker, `aws-nitro-enclaves-cli`, `aws-nitro-enclaves-cli-devel`, `jq`, `git`; raises `user.max_user_namespaces` / `user.max_mnt_namespaces` for vsock.
3. Writes `/etc/nitro_enclaves/allocator.yaml` with the configured CPU count and memory, then starts the allocator service.
4. Logs into ECR and pulls `:latest-parent` and `:latest-enclave`.
5. Writes two systemd units:
   - **`gardbase-enclave.service`** — `Type=oneshot`, `RemainAfterExit=yes`, ordered `Before=gardbase-parent.service`. Its `ExecStartPre` chain re-pulls the enclave image, deletes any stale EIF, runs `nitro-cli build-enclave` (logging the PCR measurements), and terminates existing enclaves. `ExecStart` runs `/opt/gardbase/run-enclave.sh`, which invokes `nitro-cli run-enclave --enclave-cid 16`, appending `--debug-mode` when enabled.
   - **`gardbase-parent.service`** — `Restart=always`, runs the parent container with `--device=/dev/vsock`, `--security-opt seccomp=unconfined`, ports 80 and 443 published, and the full environment (`ENCLAVE_CID=16`, `ENCLAVE_PORT=5000`, table names, `KMS_KEY_ID`, `BASE_URL=http://$PUBLIC_DNS`, …).
6. Writes helper scripts: `run-enclave.sh`, `capture-console.sh` (streams enclave console output, debug mode only), `extract-pcrs.sh`, `health-check.sh`.
7. Installs the CloudWatch agent and configures it to ship user-data and enclave console logs to the two log groups.
8. Starts the enclave service, waits, runs `extract-pcrs.sh` — which parses PCR0/PCR1/PCR2 out of the build log, writes `/opt/gardbase/pcr-values.json`, and **publishes them to SSM at `/${project_name}/${environment}/enclave/pcr-values`** so clients can fetch the values they need for attestation verification.
9. Starts the parent service, runs a health check, and writes an operator MOTD.

The ordering matters: the enclave must be built and running before the parent starts, because the parent's first act on any crypto request is a vsock round-trip.

### Usage

```bash
cd infrastructure/main
terraform init
terraform apply -var="environment=dev"
# or: terraform apply -var-file=environments/prod.tfvars
```

## Dependencies

**Upstream:**

- `infrastructure/bootstrap` — must be applied first; its state is read for `ecr_repository_url`.
- Both container images pushed to ECR under `:latest-parent` and `:latest-enclave` before apply, or the instance boots into a broken state.
- The pre-existing `gardbase-terraform-state` S3 bucket.
- A default VPC in the target region with at least one subnet.
- The correct `enclave_pcr0_sha384` value — obtained from `nitro-cli describe-eif --eif-path enclave.eif`, or from the PCR values the previous deploy published to SSM.

**Providers:** `hashicorp/aws ~> 6.0`, `hashicorp/tls ~> 4.0`.

**Downstream consumers:**

- `apps/api` — every environment variable it reads is set from these outputs via user-data.
- `apps/enclave-service` — runs as the EIF built here; its `ENCLAVE_PORT` is set to 5000 implicitly by the parent's configuration and the enclave's own default.
- `pkg/crypto` — SDK clients need the PCR values published to SSM and the API endpoint (`instance_public_dns` / `ec2_instance_public_ip`).

**AWS services in play:** EC2 (with Nitro Enclaves), ECR, IAM, KMS, DynamoDB, S3, SSM Parameter Store, CloudWatch Logs and Alarms.

## Notes

- **`enclave_pcr0_sha384` defaults to the literal string `"PLACEHOLDER_PCR0"`.** Applying without overriding it produces a KMS policy that no real attestation can satisfy — every crypto operation fails. There is a bootstrapping order here: the PCR value is only known after the EIF is built on the instance, so a first deploy typically runs with `enable_debug_mode = true`, reads the PCRs out of SSM, and re-applies with the real value and debug off. Nothing in the workspace automates or documents that loop.
- **`enable_debug_mode = true` disables the entire zero-trust guarantee**, in two compounding ways: the KMS condition becomes `*`, accepting any attestation, and `nitro-cli run-enclave --debug-mode` produces an enclave whose PCRs are all zeros and whose console is readable from the host. It is a development-only setting.
- **`allowed_ssh_cidr_blocks` defaults to `[""]`** — a list containing one empty string, not an empty list. That is not a valid CIDR and will fail at apply time in `prod`, which is arguably the safe failure, but it is not an intentional-looking default.
- **SSH is open to `0.0.0.0/0` in every non-`prod` environment**, and the private key is retrievable by anyone with `ssm:GetParameter` on that path.
- **The environment string is compared inconsistently.** `dynamodb.tf` scales read/write capacity on `var.environment == "production"`, but `environments/prod.tfvars` sets `environment = "prod"`. The higher-capacity branch never fires as configured, so production would run on 1 RCU/1 WCU per table. `ec2.tf` and the destruction guards elsewhere correctly use `"prod"` / `"dev"`.
- **Single instance, no autoscaling, no load balancer.** `aws_instance.api` is one EC2 instance with a public IP; there is no ASG, no ALB, and no TLS termination despite port 443 being open in the security group and exposed by the container. `BASE_URL` is set to `http://$PUBLIC_DNS`, so presigned-URL callbacks are plain HTTP.
- **The public IP is not stable.** An `aws_eip.api` is referenced in a commented-out `ssh_command` output but no EIP resource exists, so instance replacement changes the address.
- **Default VPC usage** means no network segmentation — the instance sits on a public subnet in the account's default VPC.
- **CloudWatch log paths don't line up.** The agent is configured to collect `/var/log/user-data.log`, but `user_data.sh` writes to `/var/log/user_data.log` (underscore). It also collects `/opt/gardbase/logs/enclave-console.log`, while `capture-console.sh` writes to `/opt/gardbase/enclave-console.log`. Neither log stream will populate as written.
- **Alarms have no actions.** Both `aws_cloudwatch_metric_alarm` resources omit `alarm_actions`, so they change state but notify nobody. Log retention is 7 days.
- **No state locking** on the S3 backend, same as bootstrap.
- **`user_data_replace_on_change = true` combined with a single instance means any user-data edit is a full outage** — the instance is destroyed and rebuilt, and the enclave rebuilds its EIF from scratch on boot.
- The parent container runs with `--security-opt seccomp=unconfined`, which is broader than the vsock device access actually requires.
- The AMI filter comments reference ARM/Graviton, but the filters select `al2023-ami-*-x86_64` with `architecture = x86_64`, matching the `m5a.xlarge` default. The comment is stale relative to the filter.
