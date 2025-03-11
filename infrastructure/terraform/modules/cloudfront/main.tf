# CloudFront Terraform module for Document Management Platform
# Creates a CloudFront distribution with appropriate security settings for document delivery
# version: hashicorp/aws ~> 4.0

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Create an Origin Access Identity for CloudFront to access S3
resource "aws_cloudfront_origin_access_identity" "main" {
  comment = "Origin Access Identity for ${var.project_name}-${var.environment} S3 bucket"
}

# Create CloudFront Distribution
resource "aws_cloudfront_distribution" "main" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "${var.project_name}-${var.environment} distribution"
  default_root_object = "index.html"
  price_class         = var.price_class
  web_acl_id          = var.web_acl_id
  
  # Custom domain names if provided
  aliases = var.domain_names

  # Logging configuration
  dynamic "logging_config" {
    for_each = var.log_bucket != null ? [1] : []
    content {
      include_cookies = false
      bucket          = var.log_bucket
      prefix          = "cloudfront/"
    }
  }

  # S3 origin configuration
  origin {
    domain_name = var.s3_bucket_domain_name
    origin_id   = "S3-${var.s3_bucket_name}"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.main.cloudfront_access_identity_path
    }
  }

  # Default cache behavior
  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${var.s3_bucket_name}"

    forwarded_values {
      query_string = true
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy     = "redirect-to-https"
    min_ttl                    = var.min_ttl
    default_ttl                = var.default_ttl
    max_ttl                    = var.max_ttl
    compress                   = var.compress
    field_level_encryption_id  = var.field_level_encryption_id
  }

  # SSL/TLS certificate configuration
  viewer_certificate {
    cloudfront_default_certificate = length(var.domain_names) == 0
    acm_certificate_arn            = length(var.domain_names) > 0 ? var.acm_certificate_arn : null
    ssl_support_method             = length(var.domain_names) > 0 ? "sni-only" : null
    minimum_protocol_version       = length(var.domain_names) > 0 ? "TLSv1.2_2021" : null
  }

  # Geo restrictions - none by default
  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # Tags
  tags = merge(var.tags, {
    Name = "${var.project_name}-${var.environment}-cloudfront"
  })
}