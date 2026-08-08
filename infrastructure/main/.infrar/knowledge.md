---
schema_version: 1
id: c157939d-0326-4749-91ef-0c3f60c8789e
name: infrastructure-main
node: infrastructure/main
category: iac
---
## Purpose

`infrastructure-main` provisions the entire running Gardbase deployment: the Nitro-Enclave-capable EC2 host that runs both the API container and the enclave, the KMS key whose policy is cryptographically pinned to the enclave's code measurement, the DynamoDB tables and S3 bucket that hold ciphertext, the IAM role tying them together, and the CloudWatch logging and alarms around it.

The security-critical piece lives here, not in application code: the KMS key policy condition on `kms:RecipientAttestation:ImageSha384`. That single condition is what makes "the server cannot decrypt your data" true — KMS will only return recipient-encrypted key material to an enclave whose measured image matches the expected PCR0.

## Structure

```
infrastructure/main/
  main.tf                # terraform + provider config, S3 backend, bootstrap remote state, caller identity
  variables.tf           # environment, sizing, enclave config, debug mode, PCR0, SSH CIDRs
  outputs.tf             # bucket, table names, instance id/DNS/IP, SSH key SSM path, KMS key id/ARN
  kms.tf                 # the attestation-gated CMK + alias
  dynamodb.tf            # five tables
  s3.tf                  # uploads bucket + versioning, SSE, public-access block, lifecycle
  ec2.tf                 # VPC/subnet lookup, SG, IAM role & policy, AMI, keypair, instance
  cloudwatch.tf          # two log groups + CPU and status-check alarms
  user_data.sh           # host provisioning: Docker, nitro-cli, EIF build, systemd units
  environments/dev.tfvars, environments/prod.tfvars
  .terraform.lock.hcl
```

Resources by concern:

| Concern | Resources |
| --- | --- |
| Compute | `aws_instance.api` (default `m5a.xlarge`, `enclave_options.enabled = true`, 30 GB encrypted gp3 root, default VPC, first default subnet, public IP) |
| Access | `aws_security_group.api_sg` (80/443 open; 22 restricted to `allowed_ssh_cidr_blocks` in prod, open in dev), `tls_private_key` + `aws_key_pair`, private key stored in SSM as `SecureString` |
| Keys | `aws_kms_key.enclave_key` with rotation enabled, `aws_kms_alias` |
| Data | `aws_dynamodb_table.{objects,indexes,table_configs,tenant_configs,api_keys}`, `aws_s3_bucket.uploads` |
| Identity | `aws_iam_role.api_role`, inline `aws_iam_role_policy.api_policy`, `aws_iam_instance_profile` |
| Observability | two `aws_cloudwatch_log_group`s (7-day retention), high-CPU and StatusCheckFailed alarms |

## Behavior

**State and inputs.** Backend is `gardbase-terraform-state` at key `main/terraform.tfstate`. A `terraform_remote_state` data source reads the bootstrap workspace's state for `ecr_repository_url`. The provider applies `default_tags` (`Project`, `Environment`, `ManagedBy = Terraform`) to everything. Networking is *discovered*, not created: `data.aws_vpc.default` plus `data.aws_subnets.default`, taking the first subnet.

**KMS policy.** Two statements: account root gets `kms:*`, and the EC2 instance role gets `Encrypt`, `Decrypt`, `GenerateDataKey`, `ReEncrypt`, `DescribeKey` — but only under the condition

```
"kms:RecipientAttestation:ImageSha384" = var.enable_debug_mode ? "*" : var.enclave_pcr0_sha384
```

With debug mode off, this is the zero-trust anchor. With debug mode on it collapses to a wildcard, and any code on the host can use the key.

**Data model in Terraform.** `objects` and `api_keys` are `pk`/`sk` string tables; `objects` also has TTL and point-in-time recovery enabled. `indexes` uses a **binary** sort key (the encrypted index token) plus a `gsi1` GSI keyed `gsi1pk`/`gsi1sk` with `ALL` projection, used to find and clean up an object's index entries. `table_configs` and `tenant_configs` are `pk`-only. All five are `PROVISIONED` with 5 RCU/WCU when `environment == "production"` and 1 otherwise.

The S3 uploads bucket blocks all public access, enables versioning and AES256 SSE, and carries a lifecycle rule that expires objects tagged `status=deleted` after 30 days — the durable half of the API's soft-delete flow (`TagForDeletion` sets that tag; `/recover` removes it).

**Host provisioning (`user_data.sh`).** Templated with region, project, environment, ECR URL, all five table names, bucket, KMS key id, enclave CPU/memory, debug mode, and max attestation age; `user_data_replace_on_change = true` means editing it replaces the instance. On boot it:

1. Installs Docker, `aws-nitro-enclaves-cli`, jq; raises `user.max_user_namespaces` for vsock.
2. Writes `/etc/nitro_enclaves/allocator.yaml` from the CPU/memory variables and starts the allocator.
3. Logs into ECR and pulls `:latest-parent` and `:latest-enclave`.
4. Creates `gardbase-enclave.service` (oneshot, ordered `Before` the parent) which pulls the enclave image, runs `nitro-cli build-enclave` to produce `/opt/gardbase/enclave.eif`, terminates any running enclave, and launches the new one at CID 16 with the configured resources (adding `--debug-mode` when enabled).
5. Creates `gardbase-parent.service`, which runs the API container with `--device=/dev/vsock`, ports 80/443, and the full env var set (`ENCLAVE_CID=16`, `ENCLAVE_PORT=5000`, `BASE_URL` from instance metadata's public hostname).
6. Runs `extract-pcrs.sh`, which parses PCR0/1/2 out of the build log, writes `/opt/gardbase/pcr-values.json`, and publishes them to SSM at `/<project>/<environment>/enclave/pcr-values` for clients to fetch.
7. Installs the CloudWatch agent, runs `health-check.sh` (curls `/api/health` and checks `nitro-cli describe-enclaves`), and writes an operator MOTD.

**IAM.** The instance role gets scoped S3 access to the uploads bucket, DynamoDB item/query/scan on the five tables and their indexes, ECR pull, the three KMS actions on this key only, CloudWatch Logs under `/aws/ec2/<project>*`, and `ssm:PutParameter` scoped to `/<project>/<environment>/*`.

**Apply.** `cd infrastructure/main && terraform init && terraform apply -var="environment=dev"` (or `-var-file=environments/dev.tfvars`), after bootstrap and after both images are pushed.

## Dependencies

- **Providers:** `hashicorp/aws ~> 6.0`, `hashicorp/tls ~> 4.0`; Terraform >= 1.0.0.
- **Upstream:** `infrastructure-bootstrap` via remote state (ECR URL); the unmanaged `gardbase-terraform-state` bucket; both Docker images already pushed to ECR.
- **Downstream:** every runtime env var the `api` node reads is set by `user_data.sh`; the `enclave-service` EIF is built and launched by the systemd unit defined here; `pkg/crypto` clients verify against the PCR values this stack publishes to SSM.
- **AMI:** most-recent Amazon Linux 2023 x86_64 HVM (comment says "ARM-based / Graviton" but the filters select x86_64, matching the `m5a.xlarge` default).

## Notes

- **PCR0 is a manual loop.** `enclave_pcr0_sha384` defaults to the literal `"PLACEHOLDER_PCR0"`. Any change to the enclave image changes its measurement, so a real deployment is: push image → apply → read the PCR0 the host wrote to SSM → set `enclave_pcr0_sha384` → apply again. Until that second apply, KMS calls from the enclave fail the attestation condition. There is no automation closing this loop.
- `enable_debug_mode = true` disables both PCR verification in the KMS policy and the enclave's isolation guarantees (`--debug-mode` makes enclave memory inspectable and yields zeroed PCRs). It is a development-only switch.
- Single instance, no ASG, no load balancer, no Elastic IP — the public DNS changes on replacement, and `BASE_URL` is baked from metadata at boot. `create_before_destroy` is set but there is nothing in front to shift traffic.
- The security group opens 80 and 443 to `0.0.0.0/0`, but nothing terminates TLS: the API container listens on plain HTTP port 80. Port 443 is exposed without a certificate story.
- In dev, SSH is open to `0.0.0.0/0`. `allowed_ssh_cidr_blocks` defaults to `[""]`, which is not a valid CIDR — prod applies must supply a real value.
- The instance's SSH private key is generated by Terraform and therefore stored in plaintext in the state file, in addition to the encrypted SSM parameter.
- `lifecycle.ignore_changes = [ami]` prevents replacement on AMI updates, so hosts do not pick up new base images without manual intervention.
- The CloudWatch agent config collects `/var/log/user-data.log`, but the script's own `exec > >(tee /var/log/user_data.log)` writes to an underscore filename — the paths don't match, so user-data output isn't shipped. Likewise it collects `/opt/gardbase/logs/enclave-console.log` while `capture-console.sh` writes to `/opt/gardbase/enclave-console.log`.
- The `indexes` and `table_configs` tables lack the point-in-time recovery and TTL settings applied to `objects`.
- `outputs.tf` references `aws_eip.api` in a commented-out `ssh_command` output; no EIP resource exists.
