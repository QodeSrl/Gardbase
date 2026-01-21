resource "aws_dynamodb_table" "objects" {
  name           = "${var.project_name}-objects-${var.environment}"
  billing_mode   = "PROVISIONED"
  read_capacity  = var.environment == "production" ? 5 : 1
  write_capacity = var.environment == "production" ? 5 : 1
  hash_key       = "pk"
  range_key      = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  point_in_time_recovery {
    enabled = true
  }

  server_side_encryption {
    enabled = true
  }

  tags = {
    Name        = "${var.project_name}-objects-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_dynamodb_table" "indexes" {
  name           = "${var.project_name}-indexes-${var.environment}"
  billing_mode   = "PROVISIONED"
  read_capacity  = var.environment == "production" ? 5 : 1
  write_capacity = var.environment == "production" ? 5 : 1
  hash_key       = "pk"
  range_key      = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  tags = {
    Name        = "${var.project_name}-indexes-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_dynamodb_table" "tenant_configs" {
  name           = "${var.project_name}-tenant-configs-${var.environment}"
  billing_mode   = "PROVISIONED"
  read_capacity  = var.environment == "production" ? 5 : 1
  write_capacity = var.environment == "production" ? 5 : 1
  hash_key       = "pk"

  attribute {
    name = "pk"
    type = "S"
  }

  tags = {
    Name        = "${var.project_name}-tenant-configs-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_dynamodb_table" "api_keys" {
  name           = "${var.project_name}-api-keys-${var.environment}"
  billing_mode   = "PROVISIONED"
  read_capacity  = var.environment == "production" ? 5 : 1
  write_capacity = var.environment == "production" ? 5 : 1
  hash_key       = "pk"
  range_key      = "sk"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  tags = {
    Name        = "${var.project_name}-api-keys-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}
