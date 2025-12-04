// get the default VPC
data "aws_vpc" "default" {
  default = true
}

// get all subnets in the default VPC
data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

// security group for the API server
resource "aws_security_group" "api_sg" {
  name        = "${var.project_name}-api-sg-${var.environment}"
  description = "Allow HTTP and SSH"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "Allow HTTP from anywhere"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Allow HTTPS from anywhere"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Allow SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.environment == "prod" ? var.allowed_ssh_cidr_blocks : ["0.0.0.0/0"] // in prod, restrict SSH access
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1" // all protocols
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-api-sg-${var.environment}"
  }
}

// IAM role for the EC2 instance
resource "aws_iam_role" "api_role" {
  name = "${var.project_name}-api-role-${var.environment}"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })
}

// attach policies to the role
resource "aws_iam_role_policy" "api_policy" {
  name = "${var.project_name}-api-policy-${var.environment}"
  role = aws_iam_role.api_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject", "s3:DeleteObject", "s3:ListBucket", "s3:HeadBucket"]
        Resource = ["${aws_s3_bucket.uploads.arn}", "${aws_s3_bucket.uploads.arn}/*"]
      },
      {
        Effect = "Allow"
        Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:Scan", "dynamodb:DescribeTable"]
        Resource = [
          aws_dynamodb_table.objects.arn,
          "${aws_dynamodb_table.objects.arn}/index/*",
          aws_dynamodb_table.indexes.arn,
          "${aws_dynamodb_table.indexes.arn}/index/*"
        ]
      },
      {
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken", "ecr:BatchCheckLayerAvailability", "ecr:GetDownloadUrlForLayer", "ecr:BatchGetImage"]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:DescribeKey"
        ]
        Resource = aws_kms_key.enclave_key.arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ]
        Resource = "arn:aws:logs:${var.region}:${data.aws_caller_identity.current.account_id}:log-group:/aws/ec2/${var.project_name}*"
      },
      {
        Effect   = "Allow"
        Action   = ["ssm:PutParameter"]
        Resource = "arn:aws:ssm:${var.region}:${data.aws_caller_identity.current.account_id}:parameter/${var.project_name}/${var.environment}/*"
      }
    ]
  })
}

// IAM instance profile for the EC2 instance
resource "aws_iam_instance_profile" "api_instance_profile" {
  name = "${var.project_name}-api-instance-profile-${var.environment}"
  role = aws_iam_role.api_role.name

  tags = {
    Name = "${var.project_name}-api-instance-profile-${var.environment}"
  }
}

// AMI for ARM-based instances (Graviton)
data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

resource "tls_private_key" "api_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "api_key" {
  key_name   = "${var.project_name}-api-${var.environment}-key"
  public_key = tls_private_key.api_key.public_key_openssh

  tags = {
    Name = "${var.project_name}-api-${var.environment}-key"
  }
}

# Store private key in SSM Parameter Store (encrypted)
resource "aws_ssm_parameter" "api_private_key" {
  name        = "/${var.project_name}/${var.environment}/api/ssh-private-key"
  description = "SSH private key for API server"
  type        = "SecureString"
  value       = tls_private_key.api_key.private_key_pem

  tags = {
    Name = "${var.project_name}-api-private-key-${var.environment}"
  }
}

resource "aws_instance" "api" {

  ami           = data.aws_ami.amazon_linux_2023.id
  instance_type = var.instance_type

  subnet_id                   = element(data.aws_subnets.default.ids, 0) // use the first subnet
  vpc_security_group_ids      = [aws_security_group.api_sg.id]
  associate_public_ip_address = true // ensure the instance gets a public IP
  iam_instance_profile        = aws_iam_instance_profile.api_instance_profile.name
  key_name                    = aws_key_pair.api_key.key_name

  // Enable Nitro Enclaves
  enclave_options {
    enabled = true
  }

  monitoring = true

  # Increase root volume size
  root_block_device {
    volume_size           = 30
    volume_type           = "gp3"
    encrypted             = true
    delete_on_termination = true
  }

  user_data_base64 = base64encode(templatefile("${path.module}/user_data.sh", {
    region                      = var.region
    project_name                = var.project_name
    environment                 = var.environment
    ecr_repository_url          = data.terraform_remote_state.bootstrap.outputs.ecr_repository_url
    s3_bucket                   = aws_s3_bucket.uploads.bucket
    dynamo_objects_table        = aws_dynamodb_table.objects.name
    dynamo_indexes_table        = aws_dynamodb_table.indexes.name
    kms_key_id                  = aws_kms_key.enclave_key.id
    enclave_cpus                = var.enclave_cpus
    enclave_memory_mib          = var.enclave_memory_mib
    enable_debug_mode           = var.enable_debug_mode
    max_attestation_age_minutes = var.max_attestation_age_minutes
  }))

  user_data_replace_on_change = true

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [ami] // prevent replacement on AMI updates
  }

  tags = {
    Name        = "${var.project_name}-api-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}
