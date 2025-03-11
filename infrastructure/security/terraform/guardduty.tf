terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Variables for configuring GuardDuty resources
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "enable_guardduty" {
  description = "Whether to enable AWS GuardDuty"
  type        = bool
  default     = true
}

variable "trusted_ip_list_enabled" {
  description = "Whether to enable trusted IP list for GuardDuty"
  type        = bool
  default     = false
}

variable "trusted_ip_list_location" {
  description = "S3 location for trusted IP list"
  type        = string
  default     = ""
}

variable "threat_intel_set_enabled" {
  description = "Whether to enable threat intelligence set for GuardDuty"
  type        = bool
  default     = false
}

variable "threat_intel_set_location" {
  description = "S3 location for threat intelligence set"
  type        = string
  default     = ""
}

# IAM policy document for SNS topic
data "aws_iam_policy_document" "sns_topic_policy" {
  count = var.enable_guardduty ? 1 : 0

  statement {
    effect = "Allow"
    
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.security_alerts[0].arn]
  }
}

# GuardDuty Detector - Main threat detection resource
resource "aws_guardduty_detector" "main" {
  count = var.enable_guardduty ? 1 : 0

  name                     = "${var.project_name}-${var.environment}-detector"
  enable                   = true
  finding_publishing_frequency = "SIX_HOURS"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-guardduty-detector"
    Environment = var.environment
    Project     = var.project_name
  }
}

# SNS Topic for security alerts including GuardDuty findings
resource "aws_sns_topic" "security_alerts" {
  count = var.enable_guardduty ? 1 : 0
  
  name = "${var.project_name}-${var.environment}-security-alerts"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-security-alerts"
    Environment = var.environment
    Project     = var.project_name
  }
}

# SNS Topic Policy - Controls who can publish to the security alerts topic
resource "aws_sns_topic_policy" "security_alerts_policy" {
  count = var.enable_guardduty ? 1 : 0
  
  arn    = aws_sns_topic.security_alerts[0].arn
  policy = data.aws_iam_policy_document.sns_topic_policy[0].json
}

# CloudWatch Event Rule to capture GuardDuty findings
resource "aws_cloudwatch_event_rule" "guardduty_findings" {
  count = var.enable_guardduty ? 1 : 0
  
  name        = "${var.project_name}-${var.environment}-guardduty-findings"
  description = "Captures GuardDuty findings"
  
  event_pattern = <<PATTERN
{
  "source": ["aws.guardduty"],
  "detail-type": ["GuardDuty Finding"]
}
PATTERN
}

# CloudWatch Event Target to send GuardDuty findings to SNS
resource "aws_cloudwatch_event_target" "guardduty_to_sns" {
  count = var.enable_guardduty ? 1 : 0
  
  rule      = aws_cloudwatch_event_rule.guardduty_findings[0].name
  target_id = "send-to-sns"
  arn       = aws_sns_topic.security_alerts[0].arn
}

# Optional: GuardDuty IP set for trusted IP list
resource "aws_guardduty_ipset" "trusted_ips" {
  count = var.enable_guardduty && var.trusted_ip_list_enabled ? 1 : 0
  
  name        = "${var.project_name}-${var.environment}-trusted-ips"
  detector_id = aws_guardduty_detector.main[0].id
  format      = "TXT"
  location    = var.trusted_ip_list_location
  activate    = true
}

# Optional: GuardDuty threat intelligence set
resource "aws_guardduty_threatintelset" "threat_intel" {
  count = var.enable_guardduty && var.threat_intel_set_enabled ? 1 : 0
  
  name        = "${var.project_name}-${var.environment}-threat-intel"
  detector_id = aws_guardduty_detector.main[0].id
  format      = "TXT"
  location    = var.threat_intel_set_location
  activate    = true
}

# Outputs for use by other Terraform configurations
output "guardduty_detector_id" {
  description = "ID of the GuardDuty detector"
  value       = var.enable_guardduty ? aws_guardduty_detector.main[0].id : null
}

output "security_alerts_topic_arn" {
  description = "ARN of the SNS topic for security alerts"
  value       = var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : null
}