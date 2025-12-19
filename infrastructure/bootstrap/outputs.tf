output "lambdas_bucket_name" {
  value       = aws_s3_bucket.lambdas_bucket.bucket
  description = "Name of the S3 bucket for Lambda code"
}

output "lambda_bucket_arn" {
  value       = aws_s3_bucket.lambdas_bucket.arn
  description = "ARN of the S3 bucket for Lambda code"
}

output "ecr_repository_url" {
  value       = aws_ecr_repository.api.repository_url
  description = "URL of the ECR repository for the API service"
}

output "ecr_repository_name" {
  value       = aws_ecr_repository.api.name
  description = "Name of the ECR repository for the API service"
}
