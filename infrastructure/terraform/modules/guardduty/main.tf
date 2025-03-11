# AWS GuardDuty module for threat detection and security monitoring
# Version: 1.0.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

# GuardDuty detector with configurable protections
resource "aws_guardduty_detector" "main" {
  name                        = "${var.project_name}-${var.environment}-detector"
  enable                      = true
  finding_publishing_frequency = var.finding_publishing_frequency

  datasources {
    s3_logs {
      enable = var.enable_s3_protection
    }
    kubernetes {
      audit_logs {
        enable = var.enable_kubernetes_protection
      }
    }
    malware_protection {
      scan_ec2_instance_with_findings {
        ebs_volumes {
          enable = var.enable_malware_protection
        }
      }
    }
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-guardduty-detector"
    }
  )
}

# SNS topic for security alerts
resource "aws_sns_topic" "security_alerts" {
  name = "${var.project_name}-${var.environment}-security-alerts"
  
  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-security-alerts"
    }
  )
}

# IAM policy document for the SNS topic
data "aws_iam_policy_document" "sns_topic_policy" {
  statement {
    effect = "Allow"
    
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.security_alerts.arn]
  }
}

# SNS topic policy to allow CloudWatch Events to publish to it
resource "aws_sns_topic_policy" "security_alerts" {
  arn    = aws_sns_topic.security_alerts.arn
  policy = data.aws_iam_policy_document.sns_topic_policy.json
}

# Email subscriptions for security alerts
resource "aws_sns_topic_subscription" "security_alerts_email" {
  count     = length(var.sns_topic_subscription_emails)
  topic_arn = aws_sns_topic.security_alerts.arn
  protocol  = "email"
  endpoint  = element(var.sns_topic_subscription_emails, count.index)
}

# CloudWatch Event rule to capture GuardDuty findings
resource "aws_cloudwatch_event_rule" "guardduty_findings" {
  name        = "${var.project_name}-${var.environment}-guardduty-findings"
  description = "Captures GuardDuty findings with severity at or above the threshold"
  
  # This pattern captures GuardDuty findings with severity at or above the configured threshold
  # GuardDuty severity levels range from 1 (lowest) to 8 (highest)
  event_pattern = jsonencode({
    source      = ["aws.guardduty"]
    detail-type = ["GuardDuty Finding"]
    detail      = {
      severity = range(var.alert_severity_threshold, 9)
    }
  })
}

# CloudWatch Event target to send GuardDuty findings to SNS
resource "aws_cloudwatch_event_target" "guardduty_to_sns" {
  rule      = aws_cloudwatch_event_rule.guardduty_findings.name
  target_id = "send-to-sns"
  arn       = aws_sns_topic.security_alerts.arn
}