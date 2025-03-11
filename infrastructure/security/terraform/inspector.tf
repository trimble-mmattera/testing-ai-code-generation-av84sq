# AWS Inspector configuration for the Document Management Platform
# This file defines AWS Inspector resources for automated vulnerability assessment and
# security compliance scanning to meet SOC2 and ISO27001 requirements.
# AWS provider version: ~> 4.0

# Get current AWS account ID
data "aws_caller_identity" "current" {}

# Enable AWS Inspector for the account
resource "aws_inspector2_enabler" "aws_inspector2_enabler" {
  count          = var.enable_inspector ? 1 : 0
  account_ids    = [data.aws_caller_identity.current.account_id]
  resource_types = ["EC2", "ECR"]
}

# Configure AWS Inspector for organization if applicable
resource "aws_inspector2_organization_configuration" "aws_inspector2_organization_configuration" {
  count = var.enable_inspector && var.is_organization_member ? 1 : 0
  auto_enable = {
    ec2 = true
    ecr = true
  }
}

# CloudWatch Event rule to capture Inspector findings
resource "aws_cloudwatch_event_rule" "inspector_findings" {
  count       = var.enable_inspector ? 1 : 0
  name        = "${var.project_name}-${var.environment}-inspector-findings"
  description = "Captures Inspector findings"
  event_pattern = jsonencode({
    source      = ["aws.inspector2"]
    detail-type = ["Inspector2 Finding"]
  })
}

# SNS topic for Inspector alerts (if GuardDuty is not enabled)
resource "aws_sns_topic" "inspector_alerts" {
  count = var.enable_inspector && !var.enable_guardduty ? 1 : 0
  name  = "${var.project_name}-${var.environment}-inspector-alerts"
  tags = {
    Name        = "${var.project_name}-${var.environment}-inspector-alerts"
    Environment = var.environment
    Project     = var.project_name
  }
}

# IAM policy document for the Inspector SNS topic
data "aws_iam_policy_document" "inspector_sns_topic_policy" {
  count = var.enable_inspector && !var.enable_guardduty ? 1 : 0
  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.inspector_alerts[0].arn]
  }
}

# Attach policy to the SNS topic
resource "aws_sns_topic_policy" "inspector_sns_policy" {
  count  = var.enable_inspector && !var.enable_guardduty ? 1 : 0
  arn    = aws_sns_topic.inspector_alerts[0].arn
  policy = data.aws_iam_policy_document.inspector_sns_topic_policy[0].json
}

# CloudWatch Event target to send Inspector findings to SNS
resource "aws_cloudwatch_event_target" "inspector_findings_to_sns" {
  count     = var.enable_inspector ? 1 : 0
  rule      = aws_cloudwatch_event_rule.inspector_findings[0].name
  target_id = "send-to-sns"
  arn       = var.enable_guardduty ? var.security_alerts_topic_arn : aws_sns_topic.inspector_alerts[0].arn
  
  input_transformer {
    input_paths = {
      finding     = "$.detail.finding"
      account     = "$.account"
      region      = "$.region"
      severity    = "$.detail.finding.severity"
      title       = "$.detail.finding.title"
      description = "$.detail.finding.description"
    }
    input_template = "\"Inspector finding in account <account> region <region>:\\nSeverity: <severity>\\nTitle: <title>\\nDescription: <description>\\n\""
  }
}

# Filter for critical severity Inspector findings
resource "aws_inspector2_filter" "critical_findings" {
  count = var.enable_inspector ? 1 : 0
  name  = "${var.project_name}-${var.environment}-critical-findings"
  
  filter_criteria = {
    aws_account_id = [{
      comparison = "EQUALS"
      value      = data.aws_caller_identity.current.account_id
    }]
    
    severity = [{
      comparison = "EQUALS"
      value      = "CRITICAL"
    }]
  }
}

# Filter for high severity Inspector findings
resource "aws_inspector2_filter" "high_findings" {
  count = var.enable_inspector ? 1 : 0
  name  = "${var.project_name}-${var.environment}-high-findings"
  
  filter_criteria = {
    aws_account_id = [{
      comparison = "EQUALS"
      value      = data.aws_caller_identity.current.account_id
    }]
    
    severity = [{
      comparison = "EQUALS"
      value      = "HIGH"
    }]
  }
}

# Variables
variable "project_name" {
  type        = string
  description = "Name of the project"
  default     = "document-mgmt"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
}

variable "enable_inspector" {
  type        = bool
  description = "Whether to enable AWS Inspector"
  default     = true
}

variable "enable_guardduty" {
  type        = bool
  description = "Whether GuardDuty is enabled (for SNS topic sharing)"
  default     = true
}

variable "is_organization_member" {
  type        = bool
  description = "Whether the account is part of an AWS Organization"
  default     = false
}

variable "security_alerts_topic_arn" {
  type        = string
  description = "ARN of the security alerts SNS topic (from GuardDuty)"
  default     = ""
}

# Outputs
output "inspector_enabled" {
  description = "Whether Inspector is enabled"
  value       = var.enable_inspector
}

output "inspector_alerts_topic_arn" {
  description = "ARN of the SNS topic for Inspector alerts"
  value       = var.enable_inspector && !var.enable_guardduty ? aws_sns_topic.inspector_alerts[0].arn : var.enable_inspector && var.enable_guardduty ? var.security_alerts_topic_arn : null
}

output "inspector_resource_types" {
  description = "Resource types scanned by Inspector"
  value       = var.enable_inspector ? ["EC2", "ECR"] : []
}