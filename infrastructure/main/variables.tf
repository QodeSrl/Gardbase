variable "environment" {
  description = "The environment for the deployment (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "The name of the project"
  type        = string
  default     = "gardbase"
}

variable "region" {
  description = "The cloud region to deploy resources in"
  type        = string
  default     = "eu-central-1"
}

variable "instance_type" {
  description = "The type of EC2 instance to use"
  type        = string
  // note: nitro enclaves doesn't support all instance types (see https://docs.aws.amazon.com/enclaves/latest/user/nitro-enclave.html#nitro-enclave-reqs) - c6g.large is a good choice for optimized compute, also consider m6g.large, r6g.large, m5.xlarge
  default = "c6g.large"
}

variable "enclave_cpus" {
  description = "The number of vCPUs to allocate to the Nitro Enclave"
  type        = number
  default     = 2
}

variable "enclave_memory_mib" {
  description = "The amount of memory (in MiB) to allocate to the Nitro Enclave"
  type        = number
  default     = 2048
}

variable "kms_key_deletion_window_days" {
  description = "The number of days before a KMS key is deleted after being scheduled for deletion"
  type        = number
  default     = 30
}

variable "enable_debug_mode" {
  description = "Enable debug mode for the application (disables PCR verification)"
  type        = bool
  default     = false
}

variable "max_attestation_age_minutes" {
  description = "Maximum age of attestation document in minutes"
  type        = number
  default     = 5
}

variable "allowed_ssh_cidr_blocks" {
  description = "List of CIDR blocks allowed to access the EC2 instance via SSH"
  type        = list(string)
  default     = [""]
}

variable "enclave_pcr0_sha384" {
  description = "The expected SHA384 hash of the enclave image for PCR0 verification"
  type        = string
  default     = "PLACEHOLDER_PCR0"
}
