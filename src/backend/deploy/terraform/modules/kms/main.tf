# Required providers configuration with AWS provider version ~> 4.0
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Get the current AWS account ID for policy definition
data "aws_caller_identity" "current" {}

locals {
  # This is a basic key policy that grants full control to the AWS account root user.
  # In a production environment, this should be customized to restrict access based on 
  # principles of least privilege. For example, you might want to:
  # 1. Limit key administrators to specific IAM roles
  # 2. Restrict usage operations (encrypt/decrypt) to specific IAM roles or services
  # 3. Add conditions based on tags, IP address ranges, or other attributes
  key_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "EnableIAMUserPermissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      }
    ]
  })
}

# KMS key resource for data encryption
resource "aws_kms_key" "main" {
  description             = "KMS key for ${var.project_name} ${var.environment} environment"
  deletion_window_in_days = var.deletion_window_in_days
  enable_key_rotation     = var.enable_key_rotation
  policy                  = local.key_policy
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-kms-key"
    Environment = var.environment
    Project     = var.project_name
  }
}

# KMS alias resource for easier reference
resource "aws_kms_alias" "main" {
  name          = "alias/${var.project_name}-${var.environment}"
  target_key_id = aws_kms_key.main.key_id
}

# Input variables
variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "enable_key_rotation" {
  description = "Whether to enable automatic key rotation"
  type        = bool
  default     = true
}

variable "deletion_window_in_days" {
  description = "Duration in days before the key is deleted after being scheduled for deletion"
  type        = number
  default     = 30
}

# Output values
output "key_id" {
  description = "The globally unique identifier for the KMS key"
  value       = aws_kms_key.main.key_id
}

output "key_arn" {
  description = "The Amazon Resource Name (ARN) of the KMS key"
  value       = aws_kms_key.main.arn
}

output "alias_name" {
  description = "The display name of the KMS key alias"
  value       = aws_kms_alias.main.name
}

output "alias_arn" {
  description = "The Amazon Resource Name (ARN) of the KMS key alias"
  value       = aws_kms_alias.main.arn
}