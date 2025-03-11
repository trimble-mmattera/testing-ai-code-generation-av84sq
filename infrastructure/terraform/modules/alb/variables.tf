variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
  
  validation {
    condition     = length(var.project_name) > 0
    error_message = "Project name cannot be empty"
  }
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod"
  }
}

variable "vpc_id" {
  description = "ID of the VPC where the ALB will be deployed"
  type        = string
  
  validation {
    condition     = length(var.vpc_id) > 0
    error_message = "VPC ID cannot be empty"
  }
}

variable "public_subnet_ids" {
  description = "List of public subnet IDs where the ALB will be deployed"
  type        = list(string)
  
  validation {
    condition     = length(var.public_subnet_ids) >= 2
    error_message = "At least two public subnets are required for high availability"
  }
}

variable "certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS"
  type        = string
  
  validation {
    condition     = length(var.certificate_arn) > 0
    error_message = "Certificate ARN cannot be empty"
  }
}

variable "access_logs_bucket" {
  description = "Name of the S3 bucket for ALB access logs"
  type        = string
  
  validation {
    condition     = length(var.access_logs_bucket) > 0
    error_message = "Access logs bucket name cannot be empty"
  }
}

variable "eks_security_group_id" {
  description = "ID of the EKS cluster security group"
  type        = string
  
  validation {
    condition     = length(var.eks_security_group_id) > 0
    error_message = "EKS security group ID cannot be empty"
  }
}

variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "idle_timeout" {
  description = "The time in seconds that the connection is allowed to be idle"
  type        = number
  default     = 60
}

variable "enable_deletion_protection" {
  description = "If true, deletion of the load balancer will be disabled via the AWS API"
  type        = bool
  default     = null # This will be determined based on environment
}

# Default value for enable_deletion_protection based on environment
locals {
  enable_deletion_protection = var.enable_deletion_protection == null ? (var.environment == "prod" ? true : false) : var.enable_deletion_protection
}

variable "health_check_path" {
  description = "Path for ALB health checks"
  type        = string
  default     = "/health"
}

variable "health_check_timeout" {
  description = "Timeout for health checks in seconds"
  type        = number
  default     = 5
}

variable "health_check_interval" {
  description = "Interval between health checks in seconds"
  type        = number
  default     = 15
}

variable "health_check_healthy_threshold" {
  description = "Number of consecutive successful health checks required to consider target healthy"
  type        = number
  default     = 2
}

variable "health_check_unhealthy_threshold" {
  description = "Number of consecutive failed health checks required to consider target unhealthy"
  type        = number
  default     = 2
}