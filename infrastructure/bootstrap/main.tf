terraform {
  required_version = ">= 1.0.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
  backend "s3" {
    bucket = "gardbase-terraform-state"
    key    = "bootstrap/terraform.tfstate"
    region = "eu-central-1"
    encrypt = true
  }
}

provider "aws" {
  region = var.region
}

resource "aws_s3_bucket" "lambdas_bucket" {
  bucket        = "${var.project_name}-lambdas-bucket-${var.environment}"
  force_destroy = var.environment == "dev" ? true : false
}

resource "aws_ecr_repository" "api" {
  name         = "${var.project_name}-api"
  force_delete = var.environment == "dev" ? true : false
}
