terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
  backend "s3" {
    bucket = "gardbase-terraform-state"
    key    = "main/terraform.tfstate"
    region = "eu-central-1"
    encrypt = true
  }
}

data "terraform_remote_state" "bootstrap" {
  backend = "s3"
  config = {
    bucket = "gardbase-terraform-state"
    key    = "bootstrap/terraform.tfstate"
    region = "eu-central-1"
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      Project = var.project_name
      Environment = var.environment
      ManagedBy = "Terraform"
    }
  }
}

data "aws_caller_identity" "current" {}