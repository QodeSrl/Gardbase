resource "aws_dynamodb_table" "objects" {
    name = "objects"
    billing_mode = "PROVISIONED"
    read_capacity = var.environment == "production" ? 5 : 1
    write_capacity = var.environment == "production" ? 5 : 1
    hash_key = "pk"
    range_key = "sk"

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
        enabled = true
    }

    tags = {
        Environment = var.environment
        Project     = var.project_name
    }
}

resource "aws_dynamodb_table" "indexes" {
    name = "indexes"
    billing_mode = "PROVISIONED"
    read_capacity = var.environment == "production" ? 5 : 1
    write_capacity = var.environment == "production" ? 5 : 1
    hash_key = "pk"
    range_key = "sk"
    
    attribute {
        name = "pk"
        type = "S"
    }
    attribute {
        name = "sk"
        type = "S"
    }
    
    tags = {
        Environment = var.environment
        Project     = var.project_name
    }
}