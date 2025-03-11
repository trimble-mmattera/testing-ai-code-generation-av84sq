# General project settings
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

# AWS region configuration
variable "aws_region" {
  description = "AWS region where resources will be deployed"
  type        = string
  default     = "us-west-2"
}

# Networking configuration
variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "List of availability zones for multi-AZ deployment"
  type        = list(string)
  default     = ["us-west-2a", "us-west-2b", "us-west-2c"]
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets"
  type        = list(string)
  default     = ["10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/24"]
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private application subnets"
  type        = list(string)
  default     = ["10.0.10.0/24", "10.0.11.0/24", "10.0.12.0/24"]
}

# Database configuration
variable "db_name" {
  description = "Name of the PostgreSQL database"
  type        = string
  default     = "documentdb"
}

variable "db_username" {
  description = "Username for the PostgreSQL database"
  type        = string
  default     = "dbadmin"
}

variable "db_instance_class" {
  description = "Instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.t3.medium"
}

variable "multi_az" {
  description = "Whether to enable Multi-AZ deployment for RDS"
  type        = bool
  default     = true
}

# Storage configuration
variable "document_retention_days" {
  description = "Number of days to retain documents in the permanent bucket"
  type        = number
  default     = 365
}

variable "quarantine_retention_days" {
  description = "Number of days to retain documents in the quarantine bucket"
  type        = number
  default     = 90
}

# EKS configuration
variable "eks_cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.25"
}

variable "node_group_instance_types" {
  description = "List of EC2 instance types for EKS node groups"
  type        = list(string)
  default     = ["m5.large", "m5a.large", "m5n.large"]
}

variable "node_group_desired_size" {
  description = "Desired number of nodes in the EKS node group"
  type        = number
  default     = 3
}

variable "node_group_min_size" {
  description = "Minimum number of nodes in the EKS node group"
  type        = number
  default     = 3
}

variable "node_group_max_size" {
  description = "Maximum number of nodes in the EKS node group"
  type        = number
  default     = 10
}