resource "aws_iam_role" "lambda_role" {
  count = 1
  name  = "${var.project_name}-lambda-role-${var.environment}"

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
  name = "${var.project_name}-upload-processor-policy-${var.environment}"
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

resource "aws_lambda_permission" "allow_bucket" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.upload-processor.arn
  principal     = "s3.amazonaws.com"
  source_arn    = aws_s3_bucket.uploads.arn
}

resource "aws_lambda_function" "upload-processor" {
  function_name = "${var.project_name}-upload-processor-${var.environment}"
  s3_bucket     = data.terraform_remote_state.bootstrap.outputs.lambdas_bucket_name
  s3_key        = "upload-processor.zip"
  handler       = "main"
  runtime       = "provided.al2023"
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

  depends_on = [aws_lambda_permission.allow_bucket, aws_lambda_function.upload-processor, aws_s3_bucket.uploads]
}
