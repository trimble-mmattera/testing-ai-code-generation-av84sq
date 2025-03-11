# AWS WAF Module for Document Management Platform
# AWS Provider version: ~> 4.0

locals {
  name_prefix = "${var.project_name}-${var.environment}"
  common_tags = {
    Name        = "${var.project_name}-${var.environment}-waf"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# AWS WAF IP Whitelist for exempting specific IP addresses from WAF rules
resource "aws_wafv2_ip_set" "whitelist" {
  name               = "${local.name_prefix}-ip-whitelist"
  description        = "Whitelisted IP addresses for ${var.project_name} ${var.environment} environment"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = var.ip_whitelist
  tags               = local.common_tags
}

# AWS WAF Web ACL with multiple protection rule sets
resource "aws_wafv2_web_acl" "main" {
  name        = "${local.name_prefix}-web-acl"
  description = "WAF Web ACL for ${var.project_name} ${var.environment} environment"
  scope       = "REGIONAL"

  default_action {
    allow {}
  }

  # Rate Limiting Rule - Protects against DDoS attacks by limiting request rate
  rule {
    name     = "rate-limit-rule"
    priority = 1

    action {
      block {}
    }

    statement {
      rate_based_statement {
        limit              = var.rate_limit_threshold
        aggregate_key_type = "IP"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${local.name_prefix}-rate-limit-rule-metric"
      sampled_requests_enabled   = true
    }
  }

  # IP Whitelist Rule - Allows trusted IP addresses to bypass WAF rules
  rule {
    name     = "ip-whitelist-rule"
    priority = 2

    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.whitelist.arn
      }
    }

    action {
      allow {}
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${local.name_prefix}-ip-whitelist-rule-metric"
      sampled_requests_enabled   = true
    }
  }

  # AWS Managed Rules - Common Rule Set for general protection
  rule {
    name     = "aws-managed-rules-common-rule"
    priority = 3

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        name         = "AWSManagedRulesCommonRuleSet"
        vendor_name  = "AWS"
        excluded_rule = []
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${local.name_prefix}-aws-managed-rules-common-rule-metric"
      sampled_requests_enabled   = true
    }
  }

  # AWS Managed Rules - SQL Injection Rule Set for protection against SQL injection attacks
  rule {
    name     = "aws-managed-rules-sql-rule"
    priority = 4

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        name         = "AWSManagedRulesSQLiRuleSet"
        vendor_name  = "AWS"
        excluded_rule = []
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "${local.name_prefix}-aws-managed-rules-sql-rule-metric"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${local.name_prefix}-web-acl-metric"
    sampled_requests_enabled   = true
  }

  tags = local.common_tags
}

# Associate WAF Web ACL with Application Load Balancer
resource "aws_wafv2_web_acl_association" "alb_association" {
  resource_arn = var.alb_arn
  web_acl_arn  = aws_wafv2_web_acl.main.arn
}

# S3 bucket for WAF logs
resource "aws_s3_bucket" "waf_logs" {
  bucket        = "${local.name_prefix}-waf-logs"
  force_destroy = var.environment != "prod"
  tags          = local.common_tags
}

# S3 bucket encryption for WAF logs
resource "aws_s3_bucket_server_side_encryption_configuration" "waf_logs_encryption" {
  bucket = aws_s3_bucket.waf_logs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 bucket lifecycle configuration for WAF logs to manage retention
resource "aws_s3_bucket_lifecycle_configuration" "waf_logs_lifecycle" {
  bucket = aws_s3_bucket.waf_logs.id

  rule {
    id     = "log-expiration"
    status = "Enabled"

    expiration {
      days = var.waf_log_retention_days
    }
  }
}

# Block public access to S3 bucket
resource "aws_s3_bucket_public_access_block" "waf_logs_public_access_block" {
  bucket                  = aws_s3_bucket.waf_logs.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# WAF logging configuration - Conditionally enabled
resource "aws_wafv2_web_acl_logging_configuration" "main" {
  count = var.enable_logging ? 1 : 0

  log_destination_configs = [aws_s3_bucket.waf_logs.arn]
  resource_arn            = aws_wafv2_web_acl.main.arn

  # Redact sensitive information in logs
  redacted_fields {
    single_header {
      name = "authorization"
    }
  }
}

# Output values for use in other Terraform resources
output "web_acl_id" {
  description = "ID of the WAF Web ACL"
  value       = aws_wafv2_web_acl.main.id
}

output "web_acl_arn" {
  description = "ARN of the WAF Web ACL"
  value       = aws_wafv2_web_acl.main.arn
}

output "waf_logs_bucket" {
  description = "Name of the S3 bucket storing WAF logs"
  value       = aws_s3_bucket.waf_logs.bucket
}