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
  default     = "eu-north-1"
}

variable "instance_type" {
  description = "The type of EC2 instance to use"
  type        = string
  // note: nitro enclaves doesn't support all instance types (see https://docs.aws.amazon.com/enclaves/latest/user/nitro-enclave.html#nitro-enclave-reqs) - c6g.large is a good choice for optimized compute, also consider m6g.large, r6g.large, m5.xlarge
  default     = "c6g.large"
}