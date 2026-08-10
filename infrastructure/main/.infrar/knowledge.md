---
schema_version: 1
id: bdf97b20-e8f0-47e9-a456-2115bce4cb44
name: infrastructure-main
node: infrastructure/main
category: iac
---
## Purpose

`infrastructure-main` provisions the entire running Gardbase deployment: the Nitro-Enclave-capable EC2 host that runs both the API container and the enclave, the KMS key whose policy is cryptographically pinned to the enclave's code measurement, the DynamoDB tables and S3 bucket that hold ciphertext, the IAM role tying them together, and the CloudWatch logging and alarms around it.

The security-critical piece lives here rather than in application code: the KMS key policy condition on `kms:RecipientAttestation:ImageSha384`. That single condition is what makes "the server cannot decrypt your data" true — KMS only returns recipient-encrypted key material to an enclave whose measured image matches the expected PCR0.

## Structure

```
infrastructure/main/
  main.tf                # terraform + provider config, S3 backend, bootstrap remote state, caller identity
  variables.tf           # environment, sizing, enclave config, debug mode, PCR0, SSH CIDRs
  outputs.tf             # bucket, table names, instance id/DNS/IP, SSH key SSM path, KMS key id/ARN
  kms.tf                 # the attestation-gated CMK + alias
  dynamodb.tf            # five tables
  s3.tf                  # uploads bucket + versioning, SSE, public-access block, lifecycle
  ec2.tf                 # VPC/subnet lookup, security group, IAM role & policy, AMI, keypair, instance
  cloudwatch.tf          # two log groups + CPU and status-check alarms
  user_data.sh           # host provisioning: Docker, nitro-cli, EIF build, systemd units
  environments/dev.tfvars, environments/prod.tfvars
  .terraform.lock.hcl
```

Resources by concern:

| Concern | Resources |
| --- | --- |
| Compute | `aws_instance.api` — default `m5a.xlarge`, `enclave_options.enabled = true`, 30 GB encrypted gp3 root, default VPC, first default subnet, public IP, detailed monitoring |
| Access | `aws_security_group.api_sg` (80/443 open, 22 restricted to `allowed_ssh_cidr_blocks` in prod and open in dev), `tls_private_key` + `aws_key_pair`, private key stored in SSM as a `SecureString` |
| Keys | `aws_kms_key.enclave_key` with rotation enabled, plus `aws_kms_alias` |
| Data | `aws_dynamodb_table.{objects,indexes,table_configs,tenant_configs,api_keys}`, `aws_s3_bucket.uploads` |
| Identity | `aws_iam_role.api_role`, inline `aws_iam_role_policy.api_policy`, `aws_iam_instance_profile` |
| Observability | two `aws_cloudwatch_log_group`s (7-day retention), high-CPU and `StatusCheckFailed` alarms |

## Behavior

**State and inputs.** The backend is `gardbase-terraform-state` at key `main/terraform.tfstate`. A `terraform_remote_state` data source reads the bootstrap workspace's state for `ecr_repository_url`. The provider applies `default_tags` (`Project`, `Environment`, `ManagedBy = Terraform`) to everything. Networking is *discovered*, not created: `data.aws_vpc.default` plus `data.aws_subnets.default`, taking the first subnet.

**KMS policy.** Two statements. The account root gets `kms:*`. The EC2 instance role gets `Encrypt`, `Decrypt`, `GenerateDataKey`, `ReEncrypt`, and `DescribeKey`, but only under the condition

```
"kms:RecipientAttestation:ImageSha384" = var.enable_debug_mode ? "*" : var.enclave_pcr0_sha384
```

With debug mode off, this is the zero-trust anchor. With debug mode on it collapses to a wildcard and any code on the host can use the key.

**Data model.** `objects` and `api_keys` are `pk`/`sk` string-keyed tables; `objects` additionally enables TTL (used by the 30-day soft-delete window) and point-in-time recovery, and has server-side encryption on. `indexes` uses a **binary** sort key — the encrypted index token — plus a `gsi1` GSI keyed `gsi1pk`/`gsi1sk` with `ALL` projection, used to find and clean up an object's index entries. `table_configs` and `tenant_configs` are `pk`-only. All five use `PROVISIONED` billing with 5 RCU/WCU when `environment == "production"` and 1 otherwise.

The S3 uploads bucket blocks all public access, enables versioning and AES256 server-side encryption, and carries a lifecycle rule expiring objects tagged `status=deleted` after 30 days — the durable half of the API's soft-delete flow, where `TagForDeletion` sets the tag and `/recover` removes it.

**Host provisioning (`user_data.sh`).** Templated with region, project, environment, ECR URL, all five table names, the bucket, the KMS key id, enclave CPU/memory, debug mode, and max attestation age. `user_data_replace_on_change = true` means editing the script replaces the instance. On boot it:

1. Installs Docker, `aws-nitro-enclaves-cli`, and jq, and raises `user.max_user_namespaces` / `user.max_mnt_namespaces` for vsock.
2. Writes `/etc/nitro_enclaves/allocator.yaml` from the CPU/memory variables and starts the allocator service.
3. Logs into ECR and pulls `:latest-parent` and `:latest-enclave`.
4. Creates `gardbase-enclave.service` — a oneshot unit ordered `Before` the parent — which re-pulls the enclave image, runs `nitro-cli build-enclave` to produce `/opt/gardbase/enclave.eif`, terminates any running enclave, and launches the new one at CID 16 with the configured resources, adding `--debug-mode` when enabled.
5. Creates `gardbase-parent.service`, which runs the API container with `--device=/dev/vsock`, ports 80 and 443, and the full environment set (`ENCLAVE_CID=16`, `ENCLAVE_PORT=5000`, `BASE_URL` derived from the instance metadata public hostname).
6. Runs `extract-pcrs.sh`, which parses PCR0/1/2 out of the build log, writes `/opt/gardbase/pcr-values.json`, and publishes them to SSM at `/<project>/<environment>/enclave/pcr-values` for clients to fetch.
7. Installs the CloudWatch agent, runs `health-check.sh` (curls `/api/health` and checks `nitro-cli describe-enclaves`), and writes an operator MOTD documenting the log locations and restart commands.

**IAM.** The instance role gets scoped S3 access to the uploads bucket, DynamoDB item/query/scan operations on the five tables and their indexes, ECR pull permissions, the KMS actions on this key only, CloudWatch Logs under `/aws/ec2/<project>*`, and `ssm:PutParameter` scoped to `/<project>/<environment>/*`.

**Apply.** `cd infrastructure/main && terraform init && terraform apply -var="environment=dev"` (or `-var-file=environments/dev.tfvars`), after bootstrap and after both images have been pushed.

## Dependencies

- **Providers:** `hashicorp/aws ~> 6.0` and `hashicorp/tls ~> 4.0`; Terraform `>= 1.0.0`.
- **Upstream:** `infrastructure-bootstrap` via remote state for the ECR URL; the unmanaged `gardbase-terraform-state` bucket; both Docker images already pushed to ECR.
- **Downstream:** every runtime environment variable the `api` node reads is set by `user_data.sh`; the `enclave-service` EIF is built and launched by the systemd unit defined here; `crypto-sdk` clients verify against the PCR values this stack publishes to SSM.
- **AMI:** the most recent Amazon Linux 2023 x86_64 HVM image.

## Notes

- **PCR0 is a manual loop.** `enclave_pcr0_sha384` defaults to the literal `"PLACEHOLDER_PCR0"`. Any change to the enclave image changes its measurement, so a real deployment is: push image → apply → read the PCR0 the host wrote to SSM → set `enclave_pcr0_sha384` → apply again. Until that second apply, KMS calls from the enclave fail the attestation condition. Nothing automates this loop.
- `enable_debug_mode = true` disables both the PCR condition in the KMS policy and the enclave's isolation guarantees — `--debug-mode` makes enclave memory inspectable and yields zeroed PCRs. It is a development-only switch.
- Single instance, no autoscaling group, no load balancer, no Elastic IP. The public DNS changes on replacement and `BASE_URL` is baked from instance metadata at boot. `create_before_destroy` is set, but there is nothing in front to shift traffic.
- The security group opens 80 and 443 to `0.0.0.0/0`, but nothing terminates TLS: the API container listens on plain HTTP on port 80, and port 443 is exposed with no certificate story.
- In dev, SSH is open to `0.0.0.0/0`. `allowed_ssh_cidr_blocks` defaults to `[""]`, which is not a valid CIDR, so prod applies must supply a real value.
- The instance SSH private key is generated by Terraform and therefore stored in plaintext in the state file, in addition to the encrypted SSM parameter.
- `lifecycle.ignore_changes = [ami]` prevents replacement on AMI updates, so hosts do not pick up new base images without manual intervention.
- The CloudWatch agent config collects `/var/log/user-data.log`, but the script writes to `/var/log/user_data.log` (underscore), so user-data output is not shipped. Similarly it collects `/opt/gardbase/logs/enclave-console.log` while `capture-console.sh` writes to `/opt/gardbase/enclave-console.log`.
- The `indexes`, `table_configs`, `tenant_configs`, and `api_keys` tables lack the point-in-time recovery, TTL, and server-side encryption settings applied to `objects`.
- The AMI data source is commented "ARM-based instances (Graviton)" but its filters select x86_64, consistent with the `m5a.xlarge` default instance type.
- `outputs.tf` contains a commented-out `ssh_command` output referencing `aws_eip.api`; no EIP resource exists in this workspace.
