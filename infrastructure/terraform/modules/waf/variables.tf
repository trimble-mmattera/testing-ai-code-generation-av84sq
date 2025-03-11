# WAF Module Variables
# This file defines the variables used to customize the WAF (Web Application Firewall) deployment
# for the Document Management Platform.

variable "project_name" {
  description = "Name of the project for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod"
  }
}

variable "alb_arn" {
  description = "ARN of the Application Load Balancer to associate with the WAF Web ACL"
  type        = string
}

# Rate limiting configuration to prevent DDoS and brute force attacks
variable "rate_limit_threshold" {
  description = "Maximum number of requests allowed from a single IP within 5 minutes"
  type        = number
  default     = 3000
}

# Whitelist configuration for trusted IP addresses
variable "ip_whitelist" {
  description = "List of IP addresses or CIDR blocks to whitelist (exempt from WAF rules)"
  type        = list(string)
  default     = []
}

# Security rule configurations for common attack vectors
variable "enable_sql_injection_protection" {
  description = "Enable AWS managed rule set for SQL injection protection"
  type        = bool
  default     = true
}

variable "enable_common_vulnerabilities_protection" {
  description = "Enable AWS managed rule set for common vulnerabilities protection"
  type        = bool
  default     = true
}

# Logging configuration for security monitoring and compliance
variable "enable_logging" {
  description = "Enable logging of WAF requests to S3"
  type        = bool
  default     = true
}

variable "waf_log_retention_days" {
  description = "Number of days to retain WAF logs in S3"
  type        = number
  default     = 90
}

# Resource tagging
variable "tags" {
  description = "Additional tags to apply to all resources"
  type        = map(string)
  default     = {}
}