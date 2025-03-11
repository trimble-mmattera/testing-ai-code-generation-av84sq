# AWS WAF (Web Application Firewall) Configuration for Document Management Platform
#
# This file configures AWS WAF to protect the platform's API endpoints from common web attacks
# including SQL injection, cross-site scripting (XSS), and implements rate limiting to prevent
# abuse. The WAF is attached to the Application Load Balancer that serves as the entry point
# for all API requests.
#
# The configuration leverages AWS managed rule sets for common vulnerability protection and
# implements custom rate-based rules to limit the number of requests from a single IP.
#
# Compliance:
# - Helps meet SOC2 requirements for system protection and access control
# - Supports ISO27001 controls A.13.1 (Network security management) and A.14.2 (Security in development)

terraform {
  required_version = ">= 0.14.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Variables
variable "enable_waf" {
  description = "Whether to enable AWS WAF"
  type        = bool
  default     = true
}

variable "alb_arn" {
  description = "ARN of the Application Load Balancer to protect with WAF"
  type        = string
}

variable "waf_log_retention_days" {
  description = "Number of days to retain WAF logs"
  type        = number
  default     = 90
}

variable "rate_limit_threshold" {
  description = "Maximum number of requests allowed from a single IP within 5 minutes"
  type        = number
  default     = 3000
}

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

variable "enable_logging" {
  description = "Enable logging of WAF requests to S3"
  type        = bool
  default     = true
}

variable "ip_whitelist" {
  description = "List of IP addresses or CIDR blocks to whitelist (exempt from WAF rules)"
  type        = list(string)
  default     = []
}

# Local variables
locals {
  waf_enabled = var.enable_waf == true
}

# WAF Module
module "waf" {
  source = "../../terraform/modules/waf"

  project_name                             = var.project_name
  environment                              = var.environment
  alb_arn                                  = var.alb_arn
  waf_log_retention_days                   = var.waf_log_retention_days
  rate_limit_threshold                     = var.rate_limit_threshold
  enable_sql_injection_protection          = var.enable_sql_injection_protection
  enable_common_vulnerabilities_protection = var.enable_common_vulnerabilities_protection
  enable_logging                           = var.enable_logging
  ip_whitelist                             = var.ip_whitelist
  tags                                     = local.common_tags
}

# Outputs
output "waf_web_acl_id" {
  description = "ID of the WAF Web ACL"
  value       = local.waf_enabled ? module.waf.web_acl_id : null
}

output "waf_web_acl_arn" {
  description = "ARN of the WAF Web ACL"
  value       = local.waf_enabled ? module.waf.web_acl_arn : null
}

output "waf_logs_bucket" {
  description = "Name of the S3 bucket storing WAF logs"
  value       = local.waf_enabled ? module.waf.waf_logs_bucket : null
}