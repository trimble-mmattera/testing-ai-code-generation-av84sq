# Terraform variables definition file for the staging environment of the Document Management Platform
# This file defines all input variables required for provisioning AWS resources in the staging environment,
# including networking, compute, storage, and database configurations.

# Project and AWS configuration variables
variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "aws_region" {
  description = "AWS region where resources will be deployed"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS CLI profile to use for authentication"
  type        = string
  default     = "document-mgmt-staging"
}

# Networking variables
variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.1.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for the public subnets"
  type        = list(string)
  default     = ["10.1.0.0/24", "10.1.1.0/24", "10.1.2.0/24"]
}

variable "private_app_subnet_cidrs" {
  description = "CIDR blocks for the private application subnets"
  type        = list(string)
  default     = ["10.1.10.0/24", "10.1.11.0/24", "10.1.12.0/24"]
}

variable "private_data_subnet_cidrs" {
  description = "CIDR blocks for the private data subnets"
  type        = list(string)
  default     = ["10.1.20.0/24", "10.1.21.0/24", "10.1.22.0/24"]
}

# EKS (Kubernetes) variables
variable "eks_cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "document-mgmt-staging"
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
    general    = ["m5.xlarge"]
    processing = ["c5.2xlarge"]
    search     = ["r5.2xlarge"]
  }
}

variable "eks_node_group_desired_sizes" {
  description = "Desired number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 3
    processing = 2
    search     = 2
  }
}

variable "eks_node_group_min_sizes" {
  description = "Minimum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 2
    processing = 1
    search     = 1
  }
}

variable "eks_node_group_max_sizes" {
  description = "Maximum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 6
    processing = 4
    search     = 4
  }
}

# Database variables
variable "db_instance_class" {
  description = "Instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.r5.large"
}

variable "db_allocated_storage" {
  description = "Allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 100
}

variable "db_max_allocated_storage" {
  description = "Maximum allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 500
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
  default     = null
}

variable "db_multi_az" {
  description = "Whether to enable Multi-AZ deployment for the RDS instance"
  type        = bool
  default     = true
}

variable "db_backup_retention_period" {
  description = "Number of days to retain database backups"
  type        = number
  default     = 14
}

# Elasticsearch variables
variable "elasticsearch_instance_type" {
  description = "Instance type for Elasticsearch nodes"
  type        = string
  default     = "r5.large.elasticsearch"
}

variable "elasticsearch_instance_count" {
  description = "Number of Elasticsearch nodes"
  type        = number
  default     = 2
}

variable "elasticsearch_volume_size" {
  description = "Size of EBS volumes attached to Elasticsearch nodes in GB"
  type        = number
  default     = 100
}

# S3 Bucket variables
variable "s3_document_bucket_name" {
  description = "Name of the S3 bucket for document storage"
  type        = string
  default     = "document-mgmt-staging-documents"
}

variable "s3_temporary_bucket_name" {
  description = "Name of the S3 bucket for temporary document storage"
  type        = string
  default     = "document-mgmt-staging-temp"
}

variable "s3_quarantine_bucket_name" {
  description = "Name of the S3 bucket for quarantined documents"
  type        = string
  default     = "document-mgmt-staging-quarantine"
}

variable "enable_s3_versioning" {
  description = "Whether to enable versioning on S3 buckets"
  type        = bool
  default     = true
}

variable "document_retention_days" {
  description = "Number of days to retain documents in the document bucket"
  type        = number
  default     = 90
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

# SQS Queue variables
variable "sqs_document_processing_queue_name" {
  description = "Name of the SQS queue for document processing"
  type        = string
  default     = "document-mgmt-staging-processing"
}

variable "sqs_virus_scanning_queue_name" {
  description = "Name of the SQS queue for virus scanning"
  type        = string
  default     = "document-mgmt-staging-virus-scanning"
}

variable "sqs_indexing_queue_name" {
  description = "Name of the SQS queue for document indexing"
  type        = string
  default     = "document-mgmt-staging-indexing"
}

# Security variables
variable "enable_waf" {
  description = "Whether to enable WAF for API protection"
  type        = bool
  default     = true
}

variable "admin_role_arn" {
  description = "ARN of the IAM role for administrators"
  type        = string
  default     = null
}

variable "certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS"
  type        = string
  default     = null
}

variable "enable_flow_logs" {
  description = "Enable VPC flow logs for network traffic monitoring and security"
  type        = bool
  default     = true
}

variable "flow_logs_retention_days" {
  description = "Number of days to retain VPC flow logs"
  type        = number
  default     = 90
}

# Tagging variables
variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {
    Environment = "Staging"
    Project     = "Document Management Platform"
    ManagedBy   = "Terraform"
  }
}