# Main Terraform configuration file for the security infrastructure of the Document Management Platform
# This file defines AWS provider configuration, common variables, and imports various security modules
# including WAF, GuardDuty, SecurityHub, Config, and Inspector to implement a comprehensive security
# posture that meets SOC2 and ISO27001 compliance requirements.

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }

  # NOTE: The values below should be replaced with actual values or provided 
  # via backend-config flags during terraform init
  # Example: terraform init -backend-config="bucket=${terraform_state_bucket}" ...
  backend "s3" {
    bucket         = "document-mgmt-terraform-state"
    key            = "security/terraform.tfstate"
    region         = "us-west-2"
    encrypt        = true
    dynamodb_table = "document-mgmt-terraform-locks"
  }
}

# Configure the AWS provider with region, profile, and default tags
provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile

  default_tags {
    tags = {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# Configure the random provider for generating random values when needed
provider "random" {}

#################################################
# Variables
#################################################

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

variable "aws_region" {
  type        = string
  description = "AWS region to deploy resources"
  default     = "us-west-2"
}

variable "aws_profile" {
  type        = string
  description = "AWS profile to use for deployment"
  default     = "default"
}

variable "terraform_state_bucket" {
  type        = string
  description = "S3 bucket for storing Terraform state"
  default     = "document-mgmt-terraform-state"
}

variable "terraform_lock_table" {
  type        = string
  description = "DynamoDB table for Terraform state locking"
  default     = "document-mgmt-terraform-locks"
}

variable "alb_arn" {
  type        = string
  description = "ARN of the Application Load Balancer to protect with WAF"
}

variable "enable_waf" {
  type        = bool
  description = "Whether to enable AWS WAF"
  default     = true
}

variable "enable_guardduty" {
  type        = bool
  description = "Whether to enable AWS GuardDuty"
  default     = true
}

variable "enable_securityhub" {
  type        = bool
  description = "Whether to enable AWS SecurityHub"
  default     = true
}

variable "enable_config" {
  type        = bool
  description = "Whether to enable AWS Config"
  default     = true
}

variable "enable_inspector" {
  type        = bool
  description = "Whether to enable AWS Inspector"
  default     = true
}

#################################################
# Local values
#################################################

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
  }
}

#################################################
# WAF Module - Web Application Firewall configuration
#################################################

module "waf" {
  source       = "./modules/waf"
  project_name = var.project_name
  environment  = var.environment
  alb_arn      = var.alb_arn
  enable_waf   = var.enable_waf
}

#################################################
# GuardDuty - Threat detection service
#################################################

resource "aws_guardduty_detector" "main" {
  count = var.enable_guardduty ? 1 : 0

  enable                       = true
  finding_publishing_frequency = "FIFTEEN_MINUTES"

  tags = local.common_tags
}

# SNS Topic for Security Alerts
resource "aws_sns_topic" "security_alerts" {
  count = var.enable_guardduty ? 1 : 0

  name = "${var.project_name}-${var.environment}-security-alerts"
  tags = local.common_tags
}

# GuardDuty-to-SNS Event Rule - Routes GuardDuty findings to SNS
resource "aws_cloudwatch_event_rule" "guardduty_findings" {
  count = var.enable_guardduty ? 1 : 0

  name        = "${var.project_name}-${var.environment}-guardduty-findings"
  description = "Capture GuardDuty findings and send to SNS"

  event_pattern = jsonencode({
    source      = ["aws.guardduty"]
    detail-type = ["GuardDuty Finding"]
  })

  tags = local.common_tags
}

resource "aws_cloudwatch_event_target" "guardduty_findings_to_sns" {
  count = var.enable_guardduty ? 1 : 0

  rule      = aws_cloudwatch_event_rule.guardduty_findings[0].name
  target_id = "SendToSNS"
  arn       = aws_sns_topic.security_alerts[0].arn
}

#################################################
# SecurityHub - Security posture management
#################################################

resource "aws_securityhub_account" "main" {
  count = var.enable_securityhub ? 1 : 0
}

# Enable standard subscriptions for SecurityHub
resource "aws_securityhub_standards_subscription" "cis_aws_foundations" {
  count = var.enable_securityhub ? 1 : 0

  depends_on    = [aws_securityhub_account.main]
  standards_arn = "arn:aws:securityhub:${var.aws_region}::standards/cis-aws-foundations-benchmark/v/1.2.0"
}

resource "aws_securityhub_standards_subscription" "pci_dss" {
  count = var.enable_securityhub ? 1 : 0

  depends_on    = [aws_securityhub_account.main]
  standards_arn = "arn:aws:securityhub:${var.aws_region}::standards/pci-dss/v/3.2.1"
}

#################################################
# AWS Config - Configuration and compliance monitoring
#################################################

resource "aws_config_configuration_recorder" "main" {
  count = var.enable_config ? 1 : 0

  name     = "${var.project_name}-${var.environment}-config-recorder"
  role_arn = aws_iam_role.config_role[0].arn

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}

resource "aws_config_delivery_channel" "main" {
  count = var.enable_config ? 1 : 0

  name           = "${var.project_name}-${var.environment}-config-delivery-channel"
  s3_bucket_name = aws_s3_bucket.config_bucket[0].bucket
  depends_on     = [aws_config_configuration_recorder.main]
}

resource "aws_s3_bucket" "config_bucket" {
  count = var.enable_config ? 1 : 0

  bucket = "${var.project_name}-${var.environment}-config-bucket"
  tags   = local.common_tags
}

resource "aws_s3_bucket_acl" "config_bucket_acl" {
  count = var.enable_config ? 1 : 0

  bucket = aws_s3_bucket.config_bucket[0].id
  acl    = "private"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "config_bucket_encryption" {
  count = var.enable_config ? 1 : 0

  bucket = aws_s3_bucket.config_bucket[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_iam_role" "config_role" {
  count = var.enable_config ? 1 : 0

  name = "${var.project_name}-${var.environment}-config-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "config.amazonaws.com"
        }
      },
    ]
  })

  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWS_ConfigRole"
  ]

  tags = local.common_tags
}

resource "aws_config_configuration_recorder_status" "main" {
  count = var.enable_config ? 1 : 0

  name       = aws_config_configuration_recorder.main[0].name
  is_enabled = true
  depends_on = [aws_config_delivery_channel.main]
}

#################################################
# AWS Inspector - Automated security assessment service
#################################################

resource "aws_inspector_resource_group" "main" {
  count = var.enable_inspector ? 1 : 0

  tags = {
    Inspect = "true"
  }
}

resource "aws_inspector_assessment_target" "main" {
  count = var.enable_inspector ? 1 : 0

  name               = "${var.project_name}-${var.environment}-inspector-target"
  resource_group_arn = aws_inspector_resource_group.main[0].arn
}

resource "aws_inspector_assessment_template" "main" {
  count = var.enable_inspector ? 1 : 0

  name               = "${var.project_name}-${var.environment}-inspector-template"
  target_arn         = aws_inspector_assessment_target.main[0].arn
  duration           = 3600
  rules_package_arns = [
    "arn:aws:inspector:${var.aws_region}:${data.aws_caller_identity.current.account_id}:rulespackage/0-PmNV0Tcd", # Common Vulnerabilities and Exposures
    "arn:aws:inspector:${var.aws_region}:${data.aws_caller_identity.current.account_id}:rulespackage/0-gBONHN9h", # Network Reachability
    "arn:aws:inspector:${var.aws_region}:${data.aws_caller_identity.current.account_id}:rulespackage/0-JnA8Zp85", # Security Best Practices
    "arn:aws:inspector:${var.aws_region}:${data.aws_caller_identity.current.account_id}:rulespackage/0-IxQYbA1E", # Runtime Behavior Analysis
  ]
}

# Get account ID for ARN construction
data "aws_caller_identity" "current" {}

#################################################
# Outputs
#################################################

output "waf_web_acl_id" {
  description = "ID of the WAF Web ACL"
  value       = var.enable_waf ? module.waf.web_acl_id : null
}

output "guardduty_detector_id" {
  description = "ID of the GuardDuty detector"
  value       = var.enable_guardduty ? aws_guardduty_detector.main[0].id : null
}

output "securityhub_enabled" {
  description = "Whether SecurityHub is enabled"
  value       = var.enable_securityhub
}

output "config_recorder_id" {
  description = "ID of the AWS Config configuration recorder"
  value       = var.enable_config ? aws_config_configuration_recorder.main[0].id : null
}

output "inspector_enabled" {
  description = "Whether Inspector is enabled"
  value       = var.enable_inspector
}

output "security_alerts_topic_arn" {
  description = "ARN of the SNS topic for security alerts"
  value       = var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : null
}