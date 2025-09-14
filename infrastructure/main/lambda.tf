resource "aws_iam_role" "lambda_role" {
  count = 1
  name  = "lambda-role-${var.environment}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

// attach the AWSLambdaBasicExecutionRole policy to the Lambda role
resource "aws_iam_role_policy_attachment" "lambda_basic_execution" {
  count      = 1
  role       = aws_iam_role.lambda_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

// policy for the upload processor Lambda function to allow updating DynamoDB items
resource "aws_iam_policy" "upload-processor-policy" {
  name = "upload-processor-policy-${var.environment}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "dynamodb:UpdateItem",
        ]
        Effect   = "Allow"
        Resource = aws_dynamodb_table.objects.arn
      }
    ]
  })
}

// attach the custom policy to the Lambda role
resource "aws_iam_role_policy_attachment" "lambda_upload_processor_policy" {
  role       = aws_iam_role.lambda_role[0].name
  policy_arn = aws_iam_policy.upload-processor-policy.arn
}

resource "aws_lambda_function" "upload-processor" {
  function_name = "upload-processor-${var.environment}"
  s3_bucket     = data.terraform_remote_state.bootstrap.outputs.lambdas_bucket_name
  s3_key        = "upload-processor.zip"
  handler       = "main"
  runtime       = "go1.x"
  role          = aws_iam_role.lambda_role[0].arn
  environment {
    variables = {
      "DYNAMO_OBJECTS_TABLE" = aws_dynamodb_table.objects.name
    }
  }
}

// give S3 permission to invoke the Lambda function on object creation
resource "aws_s3_bucket_notification" "uploads-notification" {
  bucket = aws_s3_bucket.uploads.id

  lambda_function {
    lambda_function_arn = aws_lambda_function.upload-processor.arn
    events              = ["s3:ObjectCreated:*"]
  }
}