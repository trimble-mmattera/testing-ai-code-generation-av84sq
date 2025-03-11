# variables.tf - Production Environment
# 
# This file defines input variables for the Document Management Platform's production environment.
# It configures AWS resources with settings appropriate for a high-availability, fault-tolerant
# production environment that meets SOC2 and ISO27001 compliance requirements.

# Project and Environment Settings
variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment name"
  type        = string
  default     = "prod"
}

# AWS Configuration
variable "aws_region" {
  description = "AWS region where resources will be deployed"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS CLI profile to use for authentication"
  type        = string
  default     = "document-mgmt-prod"
}

# Networking Configuration
variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.2.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for the public subnets"
  type        = list(string)
  default     = ["10.2.0.0/24", "10.2.1.0/24", "10.2.2.0/24"]
}

variable "private_app_subnet_cidrs" {
  description = "CIDR blocks for the private application subnets"
  type        = list(string)
  default     = ["10.2.10.0/24", "10.2.11.0/24", "10.2.12.0/24"]
}

variable "private_data_subnet_cidrs" {
  description = "CIDR blocks for the private data subnets"
  type        = list(string)
  default     = ["10.2.20.0/24", "10.2.21.0/24", "10.2.22.0/24"]
}

# EKS Configuration
variable "eks_cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "document-mgmt-prod"
}

variable "eks_cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.25"
}

variable "eks_node_group_instance_types" {
  description = "Instance types for EKS node groups by purpose"
  type        = map(list(string))
  default     = {
    general    = ["m5.2xlarge"]
    processing = ["c5.4xlarge"]
    search     = ["r5.4xlarge"]
  }
}

variable "eks_node_group_desired_sizes" {
  description = "Desired number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 6
    processing = 4
    search     = 4
  }
}

variable "eks_node_group_min_sizes" {
  description = "Minimum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 4
    processing = 2
    search     = 2
  }
}

variable "eks_node_group_max_sizes" {
  description = "Maximum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 12
    processing = 8
    search     = 8
  }
}

# Database Configuration
variable "db_instance_class" {
  description = "Instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.r5.2xlarge"
}

variable "db_allocated_storage" {
  description = "Allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 500
}

variable "db_max_allocated_storage" {
  description = "Maximum allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 2000
}

variable "db_name" {
  description = "Name of the PostgreSQL database"
  type        = string
  default     = "documentmgmt"
}

variable "db_username" {
  description = "Username for the PostgreSQL database"
  type        = string
  default     = "dbadmin"
}

variable "db_password" {
  description = "Password for the PostgreSQL database (should be provided via environment variable)"
  type        = string
  sensitive   = true
}

variable "db_multi_az" {
  description = "Whether to enable Multi-AZ deployment for the RDS instance"
  type        = bool
  default     = true
}

variable "db_backup_retention_period" {
  description = "Number of days to retain database backups"
  type        = number
  default     = 30
}

variable "db_deletion_protection" {
  description = "Whether to enable deletion protection for the RDS instance"
  type        = bool
  default     = true
}

variable "db_performance_insights_enabled" {
  description = "Whether to enable Performance Insights for the RDS instance"
  type        = bool
  default     = true
}

variable "db_performance_insights_retention_period" {
  description = "Retention period for Performance Insights data in days"
  type        = number
  default     = 7
}

# Elasticsearch Configuration
variable "elasticsearch_instance_type" {
  description = "Instance type for Elasticsearch nodes"
  type        = string
  default     = "r5.2xlarge.elasticsearch"
}

variable "elasticsearch_instance_count" {
  description = "Number of Elasticsearch nodes"
  type        = number
  default     = 3
}

variable "elasticsearch_volume_size" {
  description = "Size of EBS volumes attached to Elasticsearch nodes in GB"
  type        = number
  default     = 500
}

variable "elasticsearch_dedicated_master_enabled" {
  description = "Whether to use dedicated master nodes for the Elasticsearch cluster"
  type        = bool
  default     = true
}

variable "elasticsearch_dedicated_master_type" {
  description = "Instance type for Elasticsearch dedicated master nodes"
  type        = string
  default     = "c5.large.elasticsearch"
}

variable "elasticsearch_dedicated_master_count" {
  description = "Number of dedicated master nodes in the Elasticsearch cluster"
  type        = number
  default     = 3
}

# S3 Bucket Configuration
variable "s3_document_bucket_name" {
  description = "Name of the S3 bucket for document storage"
  type        = string
  default     = "document-mgmt-prod-documents"
}

variable "s3_temporary_bucket_name" {
  description = "Name of the S3 bucket for temporary document storage"
  type        = string
  default     = "document-mgmt-prod-temp"
}

variable "s3_quarantine_bucket_name" {
  description = "Name of the S3 bucket for quarantined documents"
  type        = string
  default     = "document-mgmt-prod-quarantine"
}

variable "enable_s3_versioning" {
  description = "Whether to enable versioning on S3 buckets"
  type        = bool
  default     = true
}

variable "enable_s3_replication" {
  description = "Whether to enable cross-region replication for S3 buckets"
  type        = bool
  default     = true
}

variable "s3_replication_region" {
  description = "AWS region for S3 cross-region replication"
  type        = string
  default     = "us-west-2"
}

variable "document_retention_days" {
  description = "Number of days to retain documents in the document bucket"
  type        = number
  default     = 365
}

variable "quarantine_retention_days" {
  description = "Number of days to retain documents in the quarantine bucket"
  type        = number
  default     = 90
}

variable "temporary_retention_days" {
  description = "Number of days to retain documents in the temporary bucket"
  type        = number
  default     = 1
}

# SQS Configuration
variable "sqs_document_processing_queue_name" {
  description = "Name of the SQS queue for document processing"
  type        = string
  default     = "document-mgmt-prod-processing"
}

variable "sqs_virus_scanning_queue_name" {
  description = "Name of the SQS queue for virus scanning"
  type        = string
  default     = "document-mgmt-prod-virus-scanning"
}

variable "sqs_indexing_queue_name" {
  description = "Name of the SQS queue for document indexing"
  type        = string
  default     = "document-mgmt-prod-indexing"
}

variable "enable_sqs_dlq" {
  description = "Whether to enable dead-letter queues for SQS queues"
  type        = bool
  default     = true
}

variable "sqs_dlq_max_receive_count" {
  description = "Maximum number of times a message can be received before being sent to the DLQ"
  type        = number
  default     = 5
}

# Security and Compliance Configuration
variable "enable_cloudfront" {
  description = "Whether to enable CloudFront for content delivery"
  type        = bool
  default     = true
}

variable "enable_waf" {
  description = "Whether to enable WAF for API protection"
  type        = bool
  default     = true
}

variable "enable_guardduty" {
  description = "Whether to enable GuardDuty for threat detection"
  type        = bool
  default     = true
}

variable "enable_config" {
  description = "Whether to enable AWS Config for compliance monitoring"
  type        = bool
  default     = true
}

variable "enable_cloudtrail" {
  description = "Whether to enable CloudTrail for API activity logging"
  type        = bool
  default     = true
}

variable "admin_role_arn" {
  description = "ARN of the IAM role for administrators"
  type        = string
}

variable "certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS"
  type        = string
}

variable "enable_flow_logs" {
  description = "Enable VPC flow logs for network traffic monitoring and security"
  type        = bool
  default     = true
}

variable "flow_logs_retention_days" {
  description = "Number of days to retain VPC flow logs"
  type        = number
  default     = 365
}

# Resource Tagging
variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {
    Environment = "Production"
    Project     = "Document Management Platform"
    ManagedBy   = "Terraform"
    Compliance  = "SOC2,ISO27001"
  }
}