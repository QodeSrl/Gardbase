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

output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.api.id
}

output "instance_public_dns" {
  description = "Public DNS name of the instance"
  value       = aws_instance.api.public_dns
}

output "ec2_instance_public_ip" {
  value = aws_instance.api.public_ip
}

output "ssh_private_key_ssm_parameter" {
  description = "SSM Parameter Store path for SSH private key"
  value       = aws_ssm_parameter.api_private_key.name
  sensitive   = true
}

output "ssh_command" {
  description = "SSH command to connect to the instance"
  value       = "aws ssm get-parameter --name ${aws_ssm_parameter.api_private_key.name} --with-decryption --query Parameter.Value --output text > /tmp/key.pem && chmod 400 /tmp/key.pem && ssh -i /tmp/key.pem ec2-user@${aws_eip.api.public_ip}"
  sensitive   = true
}

output "enclave_configuration" {
  description = "Enclave configuration"
  value = {
    cpu_count  = var.enclave_cpus
    memory_mib = var.enclave_memory_mib
    debug_mode = var.enable_debug_mode
  }
}

output "kms_key_id" {
  description = "KMS key ID for enclave"
  value       = aws_kms_key.enclave_key.id
}

output "kms_key_arn" {
  description = "KMS key ARN for enclave"
  value       = aws_kms_key.enclave_key.arn
}
