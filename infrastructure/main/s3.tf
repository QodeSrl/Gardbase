resource "aws_s3_bucket" "uploads" {
  bucket        = "${var.project_name}-uploads-${var.environment}"
  force_destroy = var.environment == "dev" ? true : false

  tags = {
    Name = "uploads-${var.environment}"
  }
}