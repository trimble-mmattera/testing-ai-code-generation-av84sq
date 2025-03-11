variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"

  validation {
    condition     = length(var.project_name) > 0
    error_message = "The project_name value must not be empty."
  }
}

variable "environment" {
  description = "Environment name (dev, staging, prod) used for resource naming and tagging"
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "The environment value must be one of: dev, staging, prod."
  }
}

variable "repository_names" {
  description = "List of ECR repository names to create for microservices"
  type        = list(string)
  default     = ["api-gateway", "document-service", "storage-service", "search-service", "folder-service", "virus-scanning-service", "event-service"]

  validation {
    condition     = length(var.repository_names) > 0
    error_message = "At least one repository name must be provided."
  }
}

variable "image_retention_count" {
  description = "Number of images to retain per repository in lifecycle policy"
  type        = number
  default     = 30

  validation {
    condition     = var.image_retention_count > 0
    error_message = "Image retention count must be greater than 0."
  }
}

variable "kms_key_id" {
  description = "KMS key ID for ECR repository encryption. If not provided, AWS managed keys will be used"
  type        = string
  default     = null
}

variable "cicd_role_arn" {
  description = "ARN of the IAM role used by CI/CD pipeline for ECR access. If not provided, no repository policy will be created"
  type        = string
  default     = null
}

variable "scan_on_push" {
  description = "Enable vulnerability scanning on image push"
  type        = bool
  default     = true
}

variable "immutable_tags" {
  description = "Prevent image tags from being overwritten"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Additional tags to apply to ECR repositories"
  type        = map(string)
  default     = {}
}