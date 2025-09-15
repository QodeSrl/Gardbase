output "uploads_bucket" {
  value = aws_s3_bucket.uploads.bucket
}

output "upload_processor_lambda_name" {
  value = aws_lambda_function.upload-processor.function_name
}

output "dynamodb_tables" {
  value = {
    objects = aws_dynamodb_table.objects.name
    index   = aws_dynamodb_table.indexes.name
  }
}

output "ec2_instance_public_ip" {
  value = aws_instance.api.public_ip
}

output "private_key_pem" {
  value     = tls_private_key.api_key.private_key_pem
  sensitive = true
}