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
