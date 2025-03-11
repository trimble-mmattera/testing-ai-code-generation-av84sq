# AWS Config configuration for Document Management Platform
# This file implements continuous monitoring of AWS resource configurations
# for compliance with security best practices and standards like SOC2 and ISO27001

provider "aws" {
  # AWS provider version ~> 4.0
  version = "~> 4.0"
}

# Get current AWS account ID
data "aws_caller_identity" "current" {}

# Get current AWS region
data "aws_region" "current" {}

# IAM policy document for Config to assume role
data "aws_iam_policy_document" "config_assume_role" {
  count = var.enable_config ? 1 : 0

  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["config.amazonaws.com"]
    }
    actions = ["sts:AssumeRole"]
  }
}

# IAM policy document for Config bucket access
data "aws_iam_policy_document" "config_bucket_access" {
  count = var.enable_config ? 1 : 0

  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:GetBucketAcl"
    ]
    resources = [
      "${aws_s3_bucket.config_bucket[0].arn}",
      "${aws_s3_bucket.config_bucket[0].arn}/*"
    ]
  }
}

# IAM policy document for Config bucket policy
data "aws_iam_policy_document" "config_bucket_policy" {
  count = var.enable_config ? 1 : 0

  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["config.amazonaws.com"]
    }
    actions   = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.config_bucket[0].arn}/config/AWSLogs/${data.aws_caller_identity.current.account_id}/Config/*"]
    condition {
      test     = "StringEquals"
      variable = "s3:x-amz-acl"
      values   = ["bucket-owner-full-control"]
    }
  }

  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["config.amazonaws.com"]
    }
    actions   = ["s3:GetBucketAcl"]
    resources = ["${aws_s3_bucket.config_bucket[0].arn}"]
  }
}

# IAM policy document for the Config SNS topic
data "aws_iam_policy_document" "config_sns_topic_policy" {
  count = var.enable_config && !var.enable_guardduty ? 1 : 0

  statement {
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
    actions   = ["sns:Publish"]
    resources = ["${aws_sns_topic.config_alerts[0].arn}"]
  }
}

# S3 bucket for storing AWS Config configuration snapshots and history
resource "aws_s3_bucket" "config_bucket" {
  count = var.enable_config ? 1 : 0

  bucket        = "${var.project_name}-${var.environment}-config-${data.aws_caller_identity.current.account_id}-${data.aws_region.current.name}"
  force_destroy = var.environment != "prod"

  tags = {
    Name        = "${var.project_name}-${var.environment}-config-bucket"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Configure server-side encryption for the Config S3 bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "config_bucket_encryption" {
  count  = var.enable_config ? 1 : 0
  bucket = aws_s3_bucket.config_bucket[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Configure lifecycle rules for the Config S3 bucket
resource "aws_s3_bucket_lifecycle_configuration" "config_bucket_lifecycle" {
  count  = var.enable_config ? 1 : 0
  bucket = aws_s3_bucket.config_bucket[0].id

  rule {
    id     = "config-transition-to-standard-ia"
    status = "Enabled"

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
  }

  rule {
    id     = "config-transition-to-glacier"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }
  }

  rule {
    id     = "config-expiration"
    status = "Enabled"

    expiration {
      days = var.config_history_retention_days
    }
  }
}

# Configure bucket policy for the Config S3 bucket
resource "aws_s3_bucket_policy" "config_bucket_policy" {
  count  = var.enable_config ? 1 : 0
  bucket = aws_s3_bucket.config_bucket[0].id
  policy = data.aws_iam_policy_document.config_bucket_policy[0].json
}

# IAM role for AWS Config to assume
resource "aws_iam_role" "config_role" {
  count = var.enable_config ? 1 : 0

  name               = "${var.project_name}-${var.environment}-config-role"
  assume_role_policy = data.aws_iam_policy_document.config_assume_role[0].json

  tags = {
    Name        = "${var.project_name}-${var.environment}-config-role"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Attach the AWS managed policy for Config to the IAM role
resource "aws_iam_role_policy_attachment" "config_role_policy" {
  count      = var.enable_config ? 1 : 0
  role       = aws_iam_role.config_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWS_ConfigRole"
}

# Create an IAM policy for AWS Config to access the S3 bucket
resource "aws_iam_role_policy" "config_bucket_access" {
  count  = var.enable_config ? 1 : 0
  name   = "${var.project_name}-${var.environment}-config-bucket-access"
  role   = aws_iam_role.config_role[0].id
  policy = data.aws_iam_policy_document.config_bucket_access[0].json
}

# Create an AWS Config configuration recorder to track resource configurations
resource "aws_config_configuration_recorder" "main" {
  count = var.enable_config ? 1 : 0

  name     = "${var.project_name}-${var.environment}-config-recorder"
  role_arn = aws_iam_role.config_role[0].arn

  recording_group {
    all_supported                 = true
    include_global_resource_types = true
  }
}

# Create an AWS Config delivery channel to specify where configuration snapshots are delivered
resource "aws_config_delivery_channel" "main" {
  count = var.enable_config ? 1 : 0

  name           = "${var.project_name}-${var.environment}-config-delivery-channel"
  s3_bucket_name = aws_s3_bucket.config_bucket[0].bucket
  s3_key_prefix  = "config"

  snapshot_delivery_properties {
    delivery_frequency = "Six_Hours"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Manage the status of the AWS Config configuration recorder
resource "aws_config_configuration_recorder_status" "main" {
  count      = var.enable_config ? 1 : 0
  name       = aws_config_configuration_recorder.main[0].name
  is_enabled = true
  depends_on = [aws_config_delivery_channel.main]
}

# Create AWS Config rules for security best practices - S3 bucket encryption
resource "aws_config_config_rule" "s3_bucket_encryption" {
  count = var.enable_config ? 1 : 0

  name = "s3-bucket-server-side-encryption-enabled"
  source {
    owner             = "AWS"
    source_identifier = "S3_BUCKET_SERVER_SIDE_ENCRYPTION_ENABLED"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - RDS encryption
resource "aws_config_config_rule" "rds_encryption" {
  count = var.enable_config ? 1 : 0

  name = "rds-storage-encrypted"
  source {
    owner             = "AWS"
    source_identifier = "RDS_STORAGE_ENCRYPTED"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - EBS encryption
resource "aws_config_config_rule" "ebs_encryption" {
  count = var.enable_config ? 1 : 0

  name = "encrypted-volumes"
  source {
    owner             = "AWS"
    source_identifier = "ENCRYPTED_VOLUMES"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - Root account MFA
resource "aws_config_config_rule" "root_mfa" {
  count = var.enable_config ? 1 : 0

  name = "root-account-mfa-enabled"
  source {
    owner             = "AWS"
    source_identifier = "ROOT_ACCOUNT_MFA_ENABLED"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - IAM password policy
resource "aws_config_config_rule" "iam_password_policy" {
  count = var.enable_config ? 1 : 0

  name = "iam-password-policy"
  source {
    owner             = "AWS"
    source_identifier = "IAM_PASSWORD_POLICY"
  }
  
  input_parameters = <<EOF
{
  "RequireUppercaseCharacters":"true",
  "RequireLowercaseCharacters":"true",
  "RequireSymbols":"true",
  "RequireNumbers":"true",
  "MinimumPasswordLength":"14",
  "PasswordReusePrevention":"24",
  "MaxPasswordAge":"90"
}
EOF

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - VPC flow logs enabled
resource "aws_config_config_rule" "vpc_flow_logs" {
  count = var.enable_config ? 1 : 0

  name = "vpc-flow-logs-enabled"
  source {
    owner             = "AWS"
    source_identifier = "VPC_FLOW_LOGS_ENABLED"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create AWS Config rules for security best practices - CloudTrail enabled
resource "aws_config_config_rule" "cloudtrail_enabled" {
  count = var.enable_config ? 1 : 0

  name = "cloudtrail-enabled"
  source {
    owner             = "AWS"
    source_identifier = "CLOUD_TRAIL_ENABLED"
  }

  depends_on = [aws_config_configuration_recorder.main]
}

# Create a CloudWatch Event rule to capture AWS Config compliance changes
resource "aws_cloudwatch_event_rule" "config_compliance_changes" {
  count       = var.enable_config ? 1 : 0
  name        = "${var.project_name}-${var.environment}-config-compliance-changes"
  description = "Captures AWS Config compliance status changes"

  event_pattern = <<EOF
{
  "source": ["aws.config"],
  "detail-type": ["Config Rules Compliance Change"],
  "detail": {
    "messageType": ["ComplianceChangeNotification"],
    "newEvaluationResult": {
      "complianceType": ["NON_COMPLIANT"]
    }
  }
}
EOF
}

# Create an SNS topic for Config alerts if GuardDuty is not enabled
resource "aws_sns_topic" "config_alerts" {
  count = var.enable_config && !var.enable_guardduty ? 1 : 0

  name = "${var.project_name}-${var.environment}-config-alerts"
  tags = {
    Name        = "${var.project_name}-${var.environment}-config-alerts"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Configure the policy for the Config alerts SNS topic
resource "aws_sns_topic_policy" "config_alerts_policy" {
  count  = var.enable_config && !var.enable_guardduty ? 1 : 0
  arn    = aws_sns_topic.config_alerts[0].arn
  policy = data.aws_iam_policy_document.config_sns_topic_policy[0].json
}

# Create a CloudWatch Event target to send Config compliance changes to SNS
resource "aws_cloudwatch_event_target" "config_compliance_target" {
  count = var.enable_config ? 1 : 0
  rule  = aws_cloudwatch_event_rule.config_compliance_changes[0].name
  
  # If GuardDuty is enabled, use its security_alerts topic, otherwise use our config_alerts topic
  arn       = var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : aws_sns_topic.config_alerts[0].arn
  target_id = "send-to-sns"

  input_transformer {
    input_paths = {
      rule     = "$.detail.configRuleName"
      resource = "$.detail.resourceId"
      account  = "$.account"
      region   = "$.region"
      time     = "$.time"
    }
    input_template = "\"AWS Config compliance violation detected:\\nRule: <rule>\\nResource: <resource>\\nAccount: <account>\\nRegion: <region>\\nTime: <time>\""
  }
}

# Output: ID of the AWS Config configuration recorder
output "config_recorder_id" {
  description = "ID of the AWS Config configuration recorder"
  value       = var.enable_config ? aws_config_configuration_recorder.main[0].id : null
}

# Output: Name of the S3 bucket storing AWS Config data
output "config_bucket_name" {
  description = "Name of the S3 bucket storing AWS Config data"
  value       = var.enable_config ? aws_s3_bucket.config_bucket[0].bucket : null
}

# Output: ARN of the SNS topic for Config alerts
output "config_alerts_topic_arn" {
  description = "ARN of the SNS topic for Config alerts"
  value       = var.enable_config && !var.enable_guardduty ? aws_sns_topic.config_alerts[0].arn : var.enable_config && var.enable_guardduty ? aws_sns_topic.security_alerts[0].arn : null
}

# Output: List of AWS Config rules created
output "config_rules" {
  description = "List of AWS Config rules created"
  value       = var.enable_config ? ["s3-bucket-server-side-encryption-enabled", "rds-storage-encrypted", "encrypted-volumes", "root-account-mfa-enabled", "iam-password-policy", "vpc-flow-logs-enabled", "cloudtrail-enabled"] : []
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

variable "enable_config" {
  type        = bool
  description = "Whether to enable AWS Config"
  default     = true
}

variable "enable_guardduty" {
  type        = bool
  description = "Whether GuardDuty is enabled (for SNS topic sharing)"
  default     = true
}

variable "config_history_retention_days" {
  type        = number
  description = "Number of days to retain Config history"
  default     = 2555 # ~7 years
}