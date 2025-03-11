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

variable "s3_bucket_name" {
  description = "Name of the S3 bucket that serves as the origin for CloudFront"
  type        = string
}

variable "s3_bucket_domain_name" {
  description = "Domain name of the S3 bucket that serves as the origin for CloudFront"
  type        = string
}

variable "domain_names" {
  description = "List of domain names for the CloudFront distribution (CNAME aliases)"
  type        = list(string)
  default     = []
}

variable "acm_certificate_arn" {
  description = "ARN of the ACM certificate for HTTPS with custom domains"
  type        = string
  default     = null
}

variable "price_class" {
  description = "CloudFront price class (PriceClass_All, PriceClass_200, PriceClass_100)"
  type        = string
  default     = "PriceClass_100"
}

variable "min_ttl" {
  description = "Minimum TTL for cached objects in seconds"
  type        = number
  default     = 0
}

variable "default_ttl" {
  description = "Default TTL for cached objects in seconds"
  type        = number
  default     = 3600
}

variable "max_ttl" {
  description = "Maximum TTL for cached objects in seconds"
  type        = number
  default     = 86400
}

variable "compress" {
  description = "Whether CloudFront should automatically compress content"
  type        = bool
  default     = true
}

variable "web_acl_id" {
  description = "ID of the AWS WAF Web ACL to associate with the CloudFront distribution"
  type        = string
  default     = null
}

variable "log_bucket" {
  description = "Name of the S3 bucket for CloudFront access logs"
  type        = string
  default     = null
}

variable "field_level_encryption_id" {
  description = "ID of the CloudFront field-level encryption configuration"
  type        = string
  default     = null
}

variable "tags" {
  description = "Additional tags to apply to the CloudFront distribution"
  type        = map(string)
  default     = {}
}