# -----------------------------------------------------------
# General Project Variables
# -----------------------------------------------------------

variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment name"
  type        = string
  default     = "dev"
}

# -----------------------------------------------------------
# AWS Provider Variables
# -----------------------------------------------------------

variable "aws_region" {
  description = "AWS region where resources will be deployed"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS CLI profile to use for authentication"
  type        = string
  default     = "document-mgmt-dev"
}

# -----------------------------------------------------------
# Networking Variables
# -----------------------------------------------------------

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for the public subnets"
  type        = list(string)
  default     = ["10.0.0.0/24", "10.0.1.0/24"]
}

variable "private_app_subnet_cidrs" {
  description = "CIDR blocks for the private application subnets"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24"]
}

variable "private_data_subnet_cidrs" {
  description = "CIDR blocks for the private data subnets"
  type        = list(string)
  default     = ["10.0.20.0/24", "10.0.21.0/24"]
}

# -----------------------------------------------------------
# EKS (Kubernetes) Variables
# -----------------------------------------------------------

variable "eks_cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "document-mgmt-dev"
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
    general    = ["t3.medium"]
    processing = ["t3.large"]
    search     = ["t3.large"]
  }
}

variable "eks_node_group_desired_sizes" {
  description = "Desired number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 2
    processing = 1
    search     = 1
  }
}

variable "eks_node_group_min_sizes" {
  description = "Minimum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 1
    processing = 1
    search     = 1
  }
}

variable "eks_node_group_max_sizes" {
  description = "Maximum number of nodes in each EKS node group"
  type        = map(number)
  default     = {
    general    = 3
    processing = 3
    search     = 3
  }
}

# -----------------------------------------------------------
# Database Variables
# -----------------------------------------------------------

variable "db_instance_class" {
  description = "Instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.t3.small"
}

variable "db_allocated_storage" {
  description = "Allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 20
}

variable "db_max_allocated_storage" {
  description = "Maximum allocated storage for the RDS PostgreSQL database in GB"
  type        = number
  default     = 100
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
  default     = false
}

# -----------------------------------------------------------
# Elasticsearch Variables
# -----------------------------------------------------------

variable "elasticsearch_instance_type" {
  description = "Instance type for Elasticsearch nodes"
  type        = string
  default     = "t3.small.elasticsearch"
}

variable "elasticsearch_instance_count" {
  description = "Number of Elasticsearch nodes"
  type        = number
  default     = 1
}

variable "elasticsearch_volume_size" {
  description = "Size of EBS volumes attached to Elasticsearch nodes in GB"
  type        = number
  default     = 20
}

# -----------------------------------------------------------
# S3 Storage Variables
# -----------------------------------------------------------

variable "s3_document_bucket_name" {
  description = "Name of the S3 bucket for document storage"
  type        = string
  default     = "document-mgmt-dev-documents"
}

variable "s3_temporary_bucket_name" {
  description = "Name of the S3 bucket for temporary document storage"
  type        = string
  default     = "document-mgmt-dev-temp"
}

variable "s3_quarantine_bucket_name" {
  description = "Name of the S3 bucket for quarantined documents"
  type        = string
  default     = "document-mgmt-dev-quarantine"
}

variable "enable_s3_versioning" {
  description = "Whether to enable versioning on S3 buckets"
  type        = bool
  default     = false
}

variable "document_retention_days" {
  description = "Number of days to retain documents in the document bucket"
  type        = number
  default     = 30
}

variable "quarantine_retention_days" {
  description = "Number of days to retain documents in the quarantine bucket"
  type        = number
  default     = 30
}

variable "temporary_retention_days" {
  description = "Number of days to retain documents in the temporary bucket"
  type        = number
  default     = 1
}

# -----------------------------------------------------------
# SQS Queue Variables
# -----------------------------------------------------------

variable "sqs_document_processing_queue_name" {
  description = "Name of the SQS queue for document processing"
  type        = string
  default     = "document-mgmt-dev-processing"
}

variable "sqs_virus_scanning_queue_name" {
  description = "Name of the SQS queue for virus scanning"
  type        = string
  default     = "document-mgmt-dev-virus-scanning"
}

variable "sqs_indexing_queue_name" {
  description = "Name of the SQS queue for document indexing"
  type        = string
  default     = "document-mgmt-dev-indexing"
}

# -----------------------------------------------------------
# Security and Monitoring Variables
# -----------------------------------------------------------

variable "enable_cloudfront" {
  description = "Whether to enable CloudFront for content delivery"
  type        = bool
  default     = false
}

variable "enable_waf" {
  description = "Whether to enable WAF for API protection"
  type        = bool
  default     = false
}

variable "enable_guardduty" {
  description = "Whether to enable GuardDuty for threat detection"
  type        = bool
  default     = false
}

variable "enable_flow_logs" {
  description = "Enable VPC flow logs for network traffic monitoring and security"
  type        = bool
  default     = true
}

variable "flow_logs_retention_days" {
  description = "Number of days to retain VPC flow logs"
  type        = number
  default     = 30
}

# -----------------------------------------------------------
# Tagging Variables
# -----------------------------------------------------------

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {
    Environment = "Development"
    Project     = "Document Management Platform"
    ManagedBy   = "Terraform"
  }
}