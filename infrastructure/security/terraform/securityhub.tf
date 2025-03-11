# ---------------------------------------------------------------------------------------------------------------------
# AWS SecurityHub Configuration
# This file provisions AWS SecurityHub resources for the Document Management Platform
# It supports security monitoring, compliance auditing, and threat mitigation required by the technical specifications
# ---------------------------------------------------------------------------------------------------------------------

# ---------------------------------------------------------------------------------------------------------------------
# Enable SecurityHub Account
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_securityhub_account" "main" {
  count = var.enable_securityhub ? 1 : 0
}

# ---------------------------------------------------------------------------------------------------------------------
# Security Standards Subscriptions
# ---------------------------------------------------------------------------------------------------------------------
# AWS Foundational Security Best Practices
data "aws_securityhub_standards" "aws_foundational" {
  count = var.enable_securityhub ? 1 : 0
  name  = "AWS Foundational Security Best Practices v1.0.0"
}

resource "aws_securityhub_standards_subscription" "aws_foundational" {
  count         = var.enable_securityhub ? 1 : 0
  standards_arn = data.aws_securityhub_standards.aws_foundational[0].standards_arn
  depends_on    = [aws_securityhub_account.main]
}

# CIS AWS Foundations Benchmark
data "aws_securityhub_standards" "cis" {
  count = var.enable_securityhub ? 1 : 0
  name  = "CIS AWS Foundations Benchmark v1.2.0"
}

resource "aws_securityhub_standards_subscription" "cis" {
  count         = var.enable_securityhub ? 1 : 0
  standards_arn = data.aws_securityhub_standards.cis[0].standards_arn
  depends_on    = [aws_securityhub_account.main]
}

# PCI DSS (Optional)
data "aws_securityhub_standards" "pci_dss" {
  count = var.enable_securityhub && var.enable_pci_standard ? 1 : 0
  name  = "PCI DSS v3.2.1"
}

resource "aws_securityhub_standards_subscription" "pci_dss" {
  count         = var.enable_securityhub && var.enable_pci_standard ? 1 : 0
  standards_arn = data.aws_securityhub_standards.pci_dss[0].standards_arn
  depends_on    = [aws_securityhub_account.main]
}

# ---------------------------------------------------------------------------------------------------------------------
# Product Integrations
# ---------------------------------------------------------------------------------------------------------------------
# GuardDuty Integration
data "aws_securityhub_product" "guardduty" {
  count        = var.enable_securityhub && var.enable_guardduty ? 1 : 0
  product_name = "GuardDuty"
}

resource "aws_securityhub_product_subscription" "guardduty" {
  count       = var.enable_securityhub && var.enable_guardduty ? 1 : 0
  product_arn = data.aws_securityhub_product.guardduty[0].product_arn
  depends_on  = [aws_securityhub_account.main]
}

# Inspector Integration
data "aws_securityhub_product" "inspector" {
  count        = var.enable_securityhub && var.enable_inspector ? 1 : 0
  product_name = "Inspector"
}

resource "aws_securityhub_product_subscription" "inspector" {
  count       = var.enable_securityhub && var.enable_inspector ? 1 : 0
  product_arn = data.aws_securityhub_product.inspector[0].product_arn
  depends_on  = [aws_securityhub_account.main]
}

# ---------------------------------------------------------------------------------------------------------------------
# CloudWatch Event Rules and Targets for Security Findings
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_cloudwatch_event_rule" "securityhub_findings" {
  count       = var.enable_securityhub ? 1 : 0
  name        = "${var.project_name}-${var.environment}-securityhub-findings"
  description = "Captures SecurityHub findings"
  
  event_pattern = jsonencode({
    source      = ["aws.securityhub"]
    detail-type = ["Security Hub Findings - Imported"]
  })
}

resource "aws_cloudwatch_event_target" "securityhub_findings" {
  count     = var.enable_securityhub ? 1 : 0
  rule      = aws_cloudwatch_event_rule.securityhub_findings[0].name
  target_id = "send-to-sns"
  arn       = var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : aws_sns_topic.securityhub_alerts[0].arn
  
  input_transformer {
    input_paths = {
      finding     = "$.detail.findings[0]"
      account     = "$.detail.findings[0].AwsAccountId"
      region      = "$.region"
      severity    = "$.detail.findings[0].Severity.Label"
      title       = "$.detail.findings[0].Title"
      description = "$.detail.findings[0].Description"
    }
    
    input_template = "\"SecurityHub finding in account <account> region <region>:\\nSeverity: <severity>\\nTitle: <title>\\nDescription: <description>\\n\""
  }
}

# ---------------------------------------------------------------------------------------------------------------------
# SNS Topic for SecurityHub Alerts (if GuardDuty is not enabled)
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_sns_topic" "securityhub_alerts" {
  count = var.enable_securityhub && !var.enable_guardduty ? 1 : 0
  name  = "${var.project_name}-${var.environment}-securityhub-alerts"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-securityhub-alerts"
    Environment = var.environment
    Project     = var.project_name
  }
}

data "aws_iam_policy_document" "securityhub_sns_topic_policy" {
  count = var.enable_securityhub && !var.enable_guardduty ? 1 : 0
  
  statement {
    effect = "Allow"
    
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.securityhub_alerts[0].arn]
  }
}

resource "aws_sns_topic_policy" "securityhub_alerts" {
  count  = var.enable_securityhub && !var.enable_guardduty ? 1 : 0
  arn    = aws_sns_topic.securityhub_alerts[0].arn
  policy = data.aws_iam_policy_document.securityhub_sns_topic_policy[0].json
}

# ---------------------------------------------------------------------------------------------------------------------
# Custom Action Target for Remediation
# ---------------------------------------------------------------------------------------------------------------------
resource "aws_securityhub_action_target" "main" {
  count       = var.enable_securityhub ? 1 : 0
  name        = "Send to remediation"
  identifier  = "${var.project_name}-${var.environment}-remediation"
  description = "Sends finding to remediation workflow"
}

# ---------------------------------------------------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------------------------------------------------
output "securityhub_enabled" {
  description = "Whether SecurityHub is enabled"
  value       = var.enable_securityhub
}

output "securityhub_standards" {
  description = "List of security standards enabled in SecurityHub"
  value       = var.enable_securityhub ? ["AWS Foundational Security Best Practices", "CIS AWS Foundations Benchmark", var.enable_pci_standard ? "PCI DSS" : null] : []
}

output "securityhub_alerts_topic_arn" {
  description = "ARN of the SNS topic for SecurityHub alerts"
  value       = var.enable_securityhub && !var.enable_guardduty ? aws_sns_topic.securityhub_alerts[0].arn : var.enable_securityhub && var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : null
}

# ---------------------------------------------------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------------------------------------------------
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

variable "enable_securityhub" {
  type        = bool
  description = "Whether to enable AWS SecurityHub"
  default     = true
}

variable "enable_guardduty" {
  type        = bool
  description = "Whether to enable GuardDuty integration with SecurityHub"
  default     = true
}

variable "enable_inspector" {
  type        = bool
  description = "Whether to enable Inspector integration with SecurityHub"
  default     = true
}

variable "enable_pci_standard" {
  type        = bool
  description = "Whether to enable PCI DSS standard in SecurityHub"
  default     = false
}