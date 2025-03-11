terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Variables
variable "project_name" {
  type        = string
  description = "Name of the project used for resource naming and tagging"
  default     = "document-mgmt"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
}

variable "kms_key_id" {
  type        = string
  description = "KMS key ID for encrypting documents at rest"
}

variable "document_retention_days" {
  type        = number
  description = "Number of days to retain documents before deletion (0 for indefinite)"
  default     = 0
}

variable "quarantine_retention_days" {
  type        = number
  description = "Number of days to retain quarantined documents before deletion"
  default     = 90
}

# Local variables
locals {
  document_bucket_name   = "${var.project_name}-${var.environment}-documents"
  temp_bucket_name       = "${var.project_name}-${var.environment}-temp"
  quarantine_bucket_name = "${var.project_name}-${var.environment}-quarantine"
}

# Main document bucket
resource "aws_s3_bucket" "document_bucket" {
  bucket        = local.document_bucket_name
  force_destroy = var.environment != "prod"

  tags = {
    Name        = local.document_bucket_name
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_s3_bucket_versioning" "document_bucket" {
  bucket = aws_s3_bucket.document_bucket.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "document_bucket" {
  bucket = aws_s3_bucket.document_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = var.kms_key_id
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "document_bucket" {
  bucket = aws_s3_bucket.document_bucket.id

  rule {
    id     = "document-retention"
    status = "Enabled"

    filter {
      prefix = ""
    }

    expiration {
      days = var.document_retention_days
    }
  }
}

resource "aws_s3_bucket_public_access_block" "document_bucket" {
  bucket = aws_s3_bucket.document_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Temporary document bucket
resource "aws_s3_bucket" "temp_bucket" {
  bucket        = local.temp_bucket_name
  force_destroy = true

  tags = {
    Name        = local.temp_bucket_name
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "temp_bucket" {
  bucket = aws_s3_bucket.temp_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = var.kms_key_id
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "temp_bucket" {
  bucket = aws_s3_bucket.temp_bucket.id

  rule {
    id     = "temp-cleanup"
    status = "Enabled"

    filter {
      prefix = ""
    }

    expiration {
      days = 1
    }
  }
}

resource "aws_s3_bucket_public_access_block" "temp_bucket" {
  bucket = aws_s3_bucket.temp_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Quarantine bucket for infected documents
resource "aws_s3_bucket" "quarantine_bucket" {
  bucket        = local.quarantine_bucket_name
  force_destroy = true

  tags = {
    Name        = local.quarantine_bucket_name
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "quarantine_bucket" {
  bucket = aws_s3_bucket.quarantine_bucket.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = var.kms_key_id
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "quarantine_bucket" {
  bucket = aws_s3_bucket.quarantine_bucket.id

  rule {
    id     = "quarantine-retention"
    status = "Enabled"

    filter {
      prefix = ""
    }

    expiration {
      days = var.quarantine_retention_days
    }
  }
}

resource "aws_s3_bucket_public_access_block" "quarantine_bucket" {
  bucket = aws_s3_bucket.quarantine_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Outputs
output "document_bucket_name" {
  description = "Name of the main document storage bucket"
  value       = aws_s3_bucket.document_bucket.bucket
}

output "temp_bucket_name" {
  description = "Name of the temporary document storage bucket"
  value       = aws_s3_bucket.temp_bucket.bucket
}

output "quarantine_bucket_name" {
  description = "Name of the quarantine storage bucket"
  value       = aws_s3_bucket.quarantine_bucket.bucket
}

output "document_bucket_arn" {
  description = "ARN of the main document storage bucket"
  value       = aws_s3_bucket.document_bucket.arn
}

output "temp_bucket_arn" {
  description = "ARN of the temporary document storage bucket"
  value       = aws_s3_bucket.temp_bucket.arn
}

output "quarantine_bucket_arn" {
  description = "ARN of the quarantine storage bucket"
  value       = aws_s3_bucket.quarantine_bucket.arn
}