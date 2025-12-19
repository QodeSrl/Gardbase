resource "aws_s3_bucket" "uploads" {
  bucket        = "${var.project_name}-uploads-${var.environment}"
  force_destroy = var.environment == "dev" ? true : false

  tags = {
    Name = "uploads-${var.environment}"
  }
}

// Enable versioning for the S3 bucket
resource "aws_s3_bucket_versioning" "uploads" {
  bucket = aws_s3_bucket.uploads.id
  versioning_configuration {
    status = "Enabled"
  }
}

// Enable server-side encryption for the S3 bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "uploads" {
  bucket = aws_s3_bucket.uploads.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

// Block public access to the S3 bucket
resource "aws_s3_bucket_public_access_block" "uploads" {
  bucket = aws_s3_bucket.uploads.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
