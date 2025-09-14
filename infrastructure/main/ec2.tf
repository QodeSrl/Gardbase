// get the default VPC
data "aws_vpc" "default" {
  default = true
}

// get all subnets in the default VPC
data "aws_subnet_ids" "default" {
  vpc_id = data.aws_vpc.default.id
}

// security group for the API server
resource "aws_security_group" "api_sg" {
  name        = "api-sg-${var.environment}"
  description = "Allow HTTP and SSH"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "Allow HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] // Allow from anywhere
  }

  ingress {
    description = "Allow SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] // Allow from anywhere - restrict to specific IPs for better security
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1" // all protocols
    cidr_blocks = ["0.0.0.0/0"]
  }
}

// IAM role for the EC2 instance
resource "aws_iam_role" "api_role" {
  name = "api-role-${var.environment}"
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
  name = "api-policy-${var.environment}"
  role = aws_iam_role.api_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.uploads.arn}/*"
      },
      {
        Effect = "Allow"
        Action = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:UpdateItem", "dynamodb:Query", "dynamodb:Scan"]
        Resource = [
          aws_dynamodb_table.objects.arn,
          aws_dynamodb_table.indexes.arn
        ]
      }
    ]
  })
}

// IAM instance profile for the EC2 instance
resource "aws_iam_instance_profile" "api_instance_profile" {
  name = "api-instance-profile-${var.environment}"
  role = aws_iam_role.api_role.name
}

// AMI for ARM-based instances (Graviton)
data "aws_ami" "ubuntu_arm64" {
  most_recent = true
  owners      = ["099720109477"] // Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-focal-20.04-arm64-server-*"]
  }
}

resource "aws_instance" "api" {
  ami           = data.aws_ami.ubuntu_arm64.id
  instance_type = var.instance_type

  subnet_id              = element(data.aws_subnets_ids.default.ids, 0)
  vpc_security_group_ids = [aws_security_group.api_sg.id]

  associate_public_ip_address = true // ensure the instance gets a public IP
  iam_instance_profile        = aws_iam_instance_profile.api_instance_profile.name

  user_data = <<-EOF
                #!/bin/bash
                set -e
                apt update -y
                apt install -y docker.io
                systemctl start docker
                systemctl enable docker
                # login to ECR
                aws ecr get-login --no-include-email --region ${var.region}
                # pull and run the container
                docker run -d -p 80:80 \
                -e S3_BUCKET="${aws_s3_bucket.uploads.bucket}" \
                -e DYNAMO_OBJECTS_TABLE="${aws_dynamodb_table.objects.name}" \
                -e DYNAMO_INDEXES_TABLE="${aws_dynamodb_table.indexes.name}" \
                -e AWS_REGION="${var.region}" \
                -e ENVIRONMENT="${var.environment}" \
                -e PORT="80" \
                ${data.terraform_remote_state.bootstrap.outputs.ecr_repository_url}:latest
                EOF

  tags = {
    Name        = "api-${var.environment}"
    Environment = var.environment
    Project     = var.project_name
  }
}