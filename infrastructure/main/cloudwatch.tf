resource "aws_cloudwatch_log_group" "api" {
  name              = "/aws/ec2/${var.project_name}-api-${var.environment}"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-api-logs-${var.environment}"
  }
}

resource "aws_cloudwatch_log_group" "enclave" {
  name              = "/aws/ec2/${var.project_name}-enclave-${var.environment}"
  retention_in_days = 7

  tags = {
    Name = "${var.project_name}-enclave-logs-${var.environment}"
  }
}

# CloudWatch alarms
resource "aws_cloudwatch_metric_alarm" "high_cpu" {
  alarm_name          = "${var.project_name}-api-high-cpu-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors ec2 cpu utilization"

  dimensions = {
    InstanceId = aws_instance.api.id
  }

  tags = {
    Name = "${var.project_name}-high-cpu-alarm-${var.environment}"
  }
}

resource "aws_cloudwatch_metric_alarm" "instance_health" {
  alarm_name          = "${var.project_name}-api-health-${var.environment}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "StatusCheckFailed"
  namespace           = "AWS/EC2"
  period              = "60"
  statistic           = "Average"
  threshold           = "0"
  alarm_description   = "This metric monitors instance health"

  dimensions = {
    InstanceId = aws_instance.api.id
  }

  tags = {
    Name = "${var.project_name}-health-alarm-${var.environment}"
  }
}
