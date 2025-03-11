# AWS SQS Module for Document Management Platform
# This module creates the necessary SQS queues for document processing,
# virus scanning, indexing, and quarantine notifications

# AWS Provider version: ~> 4.0
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Input Variables
variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

variable "kms_key_id" {
  description = "KMS key ID for encrypting queue messages"
  type        = string
}

variable "message_retention_seconds" {
  description = "Number of seconds to retain messages in the queues"
  type        = number
  default     = 345600 # 4 days
}

variable "visibility_timeout_seconds" {
  description = "Visibility timeout for messages in the queues"
  type        = number
  default     = 300 # 5 minutes
}

variable "max_receive_count" {
  description = "Maximum number of times a message can be received before being sent to the DLQ"
  type        = number
  default     = 5
}

variable "delay_seconds" {
  description = "Number of seconds to delay delivery of messages"
  type        = number
  default     = 0
}

#------------------------------------------------------
# Document Processing Queue and DLQ
#------------------------------------------------------
resource "aws_sqs_queue" "document_processing_dlq" {
  name                      = "${var.project_name}-${var.environment}-document-processing-dlq"
  message_retention_seconds = 1209600 # 14 days for DLQ
  kms_master_key_id         = var.kms_key_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-document-processing-dlq"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_sqs_queue" "document_processing" {
  name                      = "${var.project_name}-${var.environment}-document-processing"
  delay_seconds             = var.delay_seconds
  message_retention_seconds = var.message_retention_seconds
  visibility_timeout_seconds = var.visibility_timeout_seconds
  kms_master_key_id         = var.kms_key_id
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.document_processing_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-document-processing"
    Environment = var.environment
    Project     = var.project_name
  }
}

#------------------------------------------------------
# Virus Scanning Queue and DLQ
#------------------------------------------------------
resource "aws_sqs_queue" "virus_scanning_dlq" {
  name                      = "${var.project_name}-${var.environment}-virus-scanning-dlq"
  message_retention_seconds = 1209600 # 14 days for DLQ
  kms_master_key_id         = var.kms_key_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-virus-scanning-dlq"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_sqs_queue" "virus_scanning" {
  name                      = "${var.project_name}-${var.environment}-virus-scanning"
  delay_seconds             = var.delay_seconds
  message_retention_seconds = var.message_retention_seconds
  visibility_timeout_seconds = var.visibility_timeout_seconds
  kms_master_key_id         = var.kms_key_id
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.virus_scanning_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-virus-scanning"
    Environment = var.environment
    Project     = var.project_name
  }
}

#------------------------------------------------------
# Document Indexing Queue and DLQ
#------------------------------------------------------
resource "aws_sqs_queue" "indexing_dlq" {
  name                      = "${var.project_name}-${var.environment}-indexing-dlq"
  message_retention_seconds = 1209600 # 14 days for DLQ
  kms_master_key_id         = var.kms_key_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-indexing-dlq"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_sqs_queue" "indexing" {
  name                      = "${var.project_name}-${var.environment}-indexing"
  delay_seconds             = var.delay_seconds
  message_retention_seconds = var.message_retention_seconds
  visibility_timeout_seconds = var.visibility_timeout_seconds
  kms_master_key_id         = var.kms_key_id
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.indexing_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-indexing"
    Environment = var.environment
    Project     = var.project_name
  }
}

#------------------------------------------------------
# Quarantine Notification Queue and DLQ
#------------------------------------------------------
resource "aws_sqs_queue" "quarantine_dlq" {
  name                      = "${var.project_name}-${var.environment}-quarantine-dlq"
  message_retention_seconds = 1209600 # 14 days for DLQ
  kms_master_key_id         = var.kms_key_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-quarantine-dlq"
    Environment = var.environment
    Project     = var.project_name
  }
}

resource "aws_sqs_queue" "quarantine" {
  name                      = "${var.project_name}-${var.environment}-quarantine"
  delay_seconds             = var.delay_seconds
  message_retention_seconds = var.message_retention_seconds
  visibility_timeout_seconds = var.visibility_timeout_seconds
  kms_master_key_id         = var.kms_key_id
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.quarantine_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Name        = "${var.project_name}-${var.environment}-quarantine"
    Environment = var.environment
    Project     = var.project_name
  }
}

#------------------------------------------------------
# Module Outputs
#------------------------------------------------------
output "document_processing_queue_url" {
  description = "URL of the document processing queue"
  value       = aws_sqs_queue.document_processing.url
}

output "virus_scanning_queue_url" {
  description = "URL of the virus scanning queue"
  value       = aws_sqs_queue.virus_scanning.url
}

output "indexing_queue_url" {
  description = "URL of the document indexing queue"
  value       = aws_sqs_queue.indexing.url
}

output "quarantine_queue_url" {
  description = "URL of the quarantine notification queue"
  value       = aws_sqs_queue.quarantine.url
}