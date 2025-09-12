resource "aws_iam_role" "lambda_exec" {
    name = "lambda-exec-${var.environment}"
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

// todo: add policies for logging, s3 access, and dynamodb access for upload-processor

resource "aws_lambda_function" "upload-processor" {
    function_name = "upload-processor-${var.environment}"
    // todo: setup nx build+package+deploy to create this zip file and upload to s3
    s3_bucket = aws_s3_bucket.uploads.bucket
    s3_key = "lambda-builds/upload-processor.zip"
    handler = "main"
    runtime = "go1.x"
    role = aws_iam_role.lambda_exec.arn
}

resource "aws_s3_bucket_notification" "uploads-notification" {
    bucket = aws_s3_bucket.uploads.id

    lambda_function {
      lambda_function_arn = aws_lambda_function.upload-processor.arn
      events = ["s3:ObjectCreated:*"]
    }
}