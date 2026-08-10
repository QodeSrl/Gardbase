---
schema_version: 1
id: 1da63b23-3341-4cec-83b4-ab2e2a128a8d
name: infrastructure-main
node: infrastructure/main
category: iac
---
## Purpose

`infrastructure-main` is the Terraform workspace that provisions the entire Gardbase runtime on AWS: the Nitro-Enclave-capable EC2 host that runs both application containers, the five DynamoDB tables and the S3 bucket they persist to, the KMS key that anchors the whole key hierarchy, and the CloudWatch logging and alarms around it.

Its defining responsibility is the trust boundary. The KMS key policy is what makes the zero-trust claim enforceable: `kms:Decrypt` and `kms:GenerateDataKey` are gated on `kms:RecipientAttestation:ImageSha384` matching the enclave's PCR0 measurement, so even the EC2 instance role cannot unwrap keys outside an attested enclave running the expected image. Everything else here — the instance, the storage, the IAM role — exists to serve that arrangement.

## Structure

All files sit flat in `infrastructure/main`:

- `main.tf` — `terraform` block (required version `>= 1.0.0`, providers `hashicorp/aws ~> 6.0` and `hashicorp/tls ~> 4.0`, S3 backend at key `main/terraform.tfstate`), the `terraform_remote_state.bootstrap` data source, the AWS provider with `default_tags` (`Project`, `Environment`, `ManagedBy`), and `data.aws_caller_identity.current`.
- `variables.tf` — `environment`, `project_name`, `region`, `instance_type` (default `m5a.xlarge`), `enclave_cpus` (2), `enclave_memory_mib` (2048), `kms_key_deletion_window_days` (30), `enable_debug_mode` (false), `max_attestation_age_minutes` (5), `allowed_ssh_cidr_blocks`, `enclave_pcr0_sha384` (default `PLACEHOLDER_PCR0`).
- `dynamodb.tf` — five tables: `objects`, `indexes`, `table_configs`, `tenant_configs`, `api_keys`.
- `s3.tf` — the `uploads` bucket plus lifecycle, versioning, server-side encryption, and public-access-block configurations.
- `kms.tf` — the enclave KMS key and its alias.
- `ec2.tf` — default-VPC/subnet lookups, the security group, the instance IAM role/policy/profile, the AL2023 AMI lookup, a generated SSH keypair stored in SSM, and the `aws_instance.api` resource.
- `cloudwatch.tf` — two log groups and two metric alarms.
- `outputs.tf` — bucket name, table names, instance id/DNS/IP, the SSM parameter holding the SSH key, the enclave configuration, and the KMS key id/ARN.
- `user_data.sh` — the ~440-line instance bootstrap script, rendered through `templatefile`.
- `environments/dev.tfvars`, `environments/prod.tfvars` — three values each (`environment`, `project_name`, `region`).
- `.terraform.lock.hcl` — provider pins.

## Behavior

**Storage schema.** All tables use `PROVISIONED` billing at 5 RCU/WCU when `environment == "production"` and 1 otherwise.

- `objects` — `pk` (S) / `sk` (S), TTL on `ttl` (used for the 30-day soft-delete window), point-in-time recovery, and server-side encryption enabled.
- `indexes` — `pk` (S) / `sk` (**B**, the binary index token), plus a `gsi1` GSI on `gsi1pk`/`gsi1sk` with `ALL` projection that lets the API find every index row belonging to one object.
- `table_configs` and `tenant_configs` — `pk`-only tables holding the KMS-wrapped table index key and the wrapped tenant master key / table salt respectively.
- `api_keys` — `pk` / `sk`, one row per key.

**Object bucket.** `${project_name}-uploads-${environment}`, versioned, SSE-S3 (`AES256`), fully public-access-blocked, `force_destroy` only in dev. A lifecycle rule expires objects tagged `status=deleted` after 30 days — this is the back half of the API's soft-delete flow, which tags rather than deletes.

**KMS.** One symmetric key with automatic rotation and the configured deletion window. The policy has two statements: full `kms:*` for the account root, and a scoped grant to the EC2 instance role for `GenerateDataKey`, `Decrypt`, `ReEncrypt`, `Encrypt`, and `DescribeKey` — conditioned on `kms:RecipientAttestation:ImageSha384` equalling `var.enclave_pcr0_sha384`, or `*` when `enable_debug_mode` is true. An alias `alias/${project_name}-enclave-${environment}` is created alongside.

**Compute.** The instance is placed in the account's **default VPC**, first available subnet, with a public IP. The security group allows 80 and 443 from anywhere and SSH from `var.allowed_ssh_cidr_blocks` when `environment == "prod"`, otherwise from `0.0.0.0/0`; egress is unrestricted. `enclave_options.enabled = true`, detailed monitoring on, and a 30 GB encrypted gp3 root volume. `lifecycle` sets `create_before_destroy` and `ignore_changes = [ami]`, and `user_data_replace_on_change = true` means editing the bootstrap script replaces the instance.

**Instance IAM.** Scoped to exactly what the API needs: S3 object CRUD plus `ListBucket`/`HeadBucket` on the uploads bucket; DynamoDB item and query/scan operations on the five tables and their indexes; ECR pull permissions; `kms:Decrypt`/`GenerateDataKey`/`DescribeKey` on the one key; CloudWatch Logs writes scoped to `/aws/ec2/${project_name}*`; and `ssm:PutParameter` under `/${project_name}/${environment}/*` so the boot script can publish PCR values.

**SSH.** A 4096-bit RSA key is generated by the `tls` provider, registered as an `aws_key_pair`, and the private half is written to SSM Parameter Store as a `SecureString` at `/${project_name}/${environment}/api/ssh-private-key`.

**Bootstrap (`user_data.sh`).** Rendered with the region, resource names, ECR URL, and enclave sizing. It installs Docker and the Nitro Enclaves CLI, raises `user.max_user_namespaces` / `user.max_mnt_namespaces` for vsock, writes `/etc/nitro_enclaves/allocator.yaml` from `enclave_cpus`/`enclave_memory_mib`, starts the allocator, logs into ECR, and pulls both image tags. It then writes two systemd units:

- `gardbase-enclave.service` (oneshot, ordered `Before=gardbase-parent.service`) — pulls `:latest-enclave`, runs `nitro-cli build-enclave` to produce `/opt/gardbase/enclave.eif`, terminates any running enclave, and starts the new one at CID 16 with the configured CPU/memory, adding `--debug-mode` when `enable_debug_mode` is true.
- `gardbase-parent.service` — `docker run` of `:latest-parent` with `--device=/dev/vsock`, ports 80/443 published, and every environment variable the API requires (bucket, five table names, region, KMS key id, `BASE_URL` from the instance's public hostname, `ENCLAVE_PORT=5000`, `ENCLAVE_CID=16`, `ENVIRONMENT`, `PORT=80`).

Helper scripts are installed alongside: `run-enclave.sh`, `capture-console.sh` (debug-mode console capture), `extract-pcrs.sh` (parses PCR0/1/2 out of the build log and publishes them to SSM at `/${project}/${env}/enclave/pcr-values`), and `health-check.sh` (curls `/api/health` and asserts the enclave is `RUNNING`). Finally it installs the CloudWatch agent, starts both services in order, runs the health check, and writes an operator MOTD.

**Observability.** Two log groups (`/aws/ec2/${project}-api-${env}` and `.../enclave-${env}`) with 7-day retention, and two alarms on the instance: average CPU > 80% over two 5-minute periods, and `StatusCheckFailed` > 0 over two 1-minute periods.

## Dependencies

**Upstream:**
- `infrastructure/bootstrap` — consumed via `terraform_remote_state` for `ecr_repository_url`. The bootstrap workspace must be applied first.
- Container images already pushed to that ECR repository as `latest-parent` and `latest-enclave`; the instance fails to start its services otherwise.
- The `gardbase-terraform-state` S3 bucket (unmanaged) and a pre-existing default VPC in the target region.
- The enclave's PCR0 value, which only exists after the enclave image has been built at least once.

**Providers:** `hashicorp/aws ~> 6.0`, `hashicorp/tls ~> 4.0`.

**Downstream consumers:** the `api` node reads every table name, the bucket, the KMS key id, and the enclave CID/port from the environment variables this workspace injects; the `enclave-service` node depends on the enclave options, allocator sizing, and the KMS attestation condition; `pkg/crypto` clients verify against the PCR values published to SSM by `extract-pcrs.sh`.

## Notes

- **Capacity and SSH conditionals disagree with the tfvars.** `dynamodb.tf` compares `var.environment == "production"`, but `environments/prod.tfvars` sets `environment = "prod"`. As written, a prod apply gets 1 RCU/WCU tables. `ec2.tf` and `s3.tf` compare against `"prod"` and `"dev"` respectively, so the three files use three different environment string conventions.
- **`enclave_pcr0_sha384` defaults to `PLACEHOLDER_PCR0`.** Until it is set to the real measurement, the KMS condition can never match and every enclave key operation fails — unless `enable_debug_mode` is true, which replaces the condition with `*` and removes the attestation guarantee entirely. This is a chicken-and-egg step: build the enclave, read PCR0 from `/opt/gardbase/pcr-values.json` or SSM, then re-apply with the real value. Every new enclave image push changes PCR0 and requires repeating it.
- **Single instance, no redundancy.** There is no Auto Scaling group, no load balancer, no TLS termination (port 443 is open in the security group but nothing terminates it), and `BASE_URL` is set to `http://<public-dns>`. Replacing the instance changes the public IP; there is no Elastic IP (the `aws_eip.api` reference survives only in a commented-out output) and no DNS record.
- **Default VPC.** No VPC, subnets, NAT, or private networking are managed here; the host sits on a public subnet with a public IP.
- `allowed_ssh_cidr_blocks` defaults to `[""]`, which is not a valid CIDR — a prod apply must override it. Outside prod, SSH is open to `0.0.0.0/0`.
- The CloudWatch agent config collects `/var/log/user-data.log` and `/opt/gardbase/logs/enclave-console.log`, but the script actually writes `/var/log/user_data.log` (underscore) and `/opt/gardbase/enclave-console.log`, so neither stream is picked up as configured.
- Both alarms are declared without `alarm_actions`, so they change state but notify nobody.
- The enclave service unit removes and rebuilds the EIF on every start, and the parent unit re-pulls `:latest-parent` on every restart — so a `systemctl restart` is effectively a redeploy of whatever is currently tagged `latest`.
- `user_data_replace_on_change = true` combined with `create_before_destroy` means any edit to `user_data.sh` or to a templated value replaces the instance in place of an in-place update.
- A stray `terraform.tfstate` exists at the repository root; it is not this workspace's state (which lives in S3) and should not be treated as authoritative.
