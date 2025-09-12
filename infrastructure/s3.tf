resource "aws_s3_bucket" "uploads" {
    bucket = "${var.project_name}-uploads-${var.environment}"
    force_destroy = true
    
    tags = {
        Name       = "uploads-${var.environment}"
    }
}