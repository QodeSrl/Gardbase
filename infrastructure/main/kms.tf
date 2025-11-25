resource "aws_kms_key" "enclave_key" {
  description             = "KMS key for ${var.project_name} Nitro Enclave in ${var.environment} environment"
  deletion_window_in_days = var.kms_key_deletion_window_days
  enable_key_rotation     = true

  // IAM policy allowing the enclave to decrypt data
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Allow Enclave to Generate and Decrypt Data Keys"
        Effect = "Allow"
        Principal = {
          AWS = aws_iam_role.api_role.arn
        }
        Action = [
          "kms:GenerateDataKey",
          "kms:Decrypt",
          "kms:DescribeKey"
        ],
        Resource = "*"
        Condition = {
          StringEqualsIgnoreCase = {
            "kms:RecipientAttestation:ImageSha384" = var.enable_debug_mode ? "*" : var.enclave_pcr0_sha384
          }
        }
      }
    ]
  })

  tags = {
    Name = "${var.project_name}-enclave-key-${var.environment}"
  }
}

resource "aws_kms_alias" "enclave_key_alias" {
  name          = "alias/${var.project_name}-enclave-${var.environment}"
  target_key_id = aws_kms_key.enclave_key.key_id
}
