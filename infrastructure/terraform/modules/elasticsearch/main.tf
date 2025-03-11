# Required providers
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

# Local variables for resource naming and configuration
locals {
  domain_name = var.domain_name != "" ? var.domain_name : "${var.project_name}-${var.environment}-es"
  snapshot_bucket_name = var.snapshot_bucket_name != "" ? var.snapshot_bucket_name : "${var.project_name}-${var.environment}-es-snapshots"
  log_group_arn = var.create_log_group ? aws_cloudwatch_log_group.elasticsearch_logs[0].arn : var.log_group_arn
}

# AWS Elasticsearch domain for document content and metadata search
resource "aws_elasticsearch_domain" "main" {
  domain_name           = local.domain_name
  elasticsearch_version = var.elasticsearch_version

  cluster_config {
    instance_type            = var.instance_type
    instance_count           = var.instance_count
    zone_awareness_enabled   = var.zone_awareness_enabled
    
    dynamic "zone_awareness_config" {
      for_each = var.zone_awareness_enabled ? [1] : []
      content {
        availability_zone_count = var.availability_zone_count
      }
    }
  }

  vpc_options {
    subnet_ids         = var.subnet_ids
    security_group_ids = [aws_security_group.elasticsearch.id]
  }

  ebs_options {
    ebs_enabled = true
    volume_type = var.ebs_volume_type
    volume_size = var.ebs_volume_size
    iops        = var.ebs_volume_type == "gp3" || var.ebs_volume_type == "io1" ? var.ebs_iops : null
  }

  encrypt_at_rest {
    enabled    = true
    kms_key_id = var.kms_key_id
  }

  node_to_node_encryption {
    enabled = true
  }

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }

  advanced_options = var.advanced_options

  snapshot_options {
    automated_snapshot_start_hour = 23
  }

  log_publishing_options {
    enabled                  = true
    log_type                 = "INDEX_SLOW_LOGS"
    cloudwatch_log_group_arn = local.log_group_arn
  }

  log_publishing_options {
    enabled                  = true
    log_type                 = "SEARCH_SLOW_LOGS"
    cloudwatch_log_group_arn = local.log_group_arn
  }

  log_publishing_options {
    enabled                  = true
    log_type                 = "ES_APPLICATION_LOGS"
    cloudwatch_log_group_arn = local.log_group_arn
  }

  tags = merge(
    {
      Name        = local.domain_name
      Environment = var.environment
      Project     = var.project_name
      ManagedBy   = "terraform"
    },
    var.tags
  )
}

# Security group for Elasticsearch domain
resource "aws_security_group" "elasticsearch" {
  name        = "${local.domain_name}-sg"
  description = "Security group for ${local.domain_name} Elasticsearch domain"
  vpc_id      = var.vpc_id

  ingress {
    description     = "HTTPS from allowed security groups"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = var.allowed_security_groups
  }

  ingress {
    description = "HTTPS from allowed CIDR blocks"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${local.domain_name}-sg"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# CloudWatch log group for Elasticsearch logs
resource "aws_cloudwatch_log_group" "elasticsearch_logs" {
  count             = var.create_log_group ? 1 : 0
  name              = "/aws/elasticsearch/${local.domain_name}"
  retention_in_days = 30

  tags = {
    Name        = "${local.domain_name}-logs"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# S3 bucket for Elasticsearch snapshots
resource "aws_s3_bucket" "elasticsearch_snapshots" {
  bucket        = local.snapshot_bucket_name
  force_destroy = false

  tags = {
    Name        = local.snapshot_bucket_name
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# Server-side encryption configuration for snapshot bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "snapshot_encryption" {
  bucket = aws_s3_bucket.elasticsearch_snapshots.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Lifecycle configuration for snapshot bucket
resource "aws_s3_bucket_lifecycle_configuration" "snapshot_lifecycle" {
  bucket = aws_s3_bucket.elasticsearch_snapshots.id

  rule {
    id     = "snapshot-retention"
    status = "Enabled"

    expiration {
      days = var.snapshot_retention_days
    }
  }
}

# IAM role for Elasticsearch snapshot functionality
resource "aws_iam_role" "elasticsearch_snapshot_role" {
  name               = "${local.domain_name}-snapshot-role"
  assume_role_policy = data.aws_iam_policy_document.elasticsearch_assume_role.json

  tags = {
    Name        = "${local.domain_name}-snapshot-role"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# IAM policy for Elasticsearch snapshot functionality
resource "aws_iam_policy" "elasticsearch_snapshot_policy" {
  name   = "${local.domain_name}-snapshot-policy"
  policy = data.aws_iam_policy_document.elasticsearch_snapshot_policy.json

  tags = {
    Name        = "${local.domain_name}-snapshot-policy"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# Attaches the snapshot policy to the snapshot role
resource "aws_iam_role_policy_attachment" "elasticsearch_snapshot_attachment" {
  role       = aws_iam_role.elasticsearch_snapshot_role.name
  policy_arn = aws_iam_policy.elasticsearch_snapshot_policy.arn
}

# IAM policy document for Elasticsearch service to assume role
data "aws_iam_policy_document" "elasticsearch_assume_role" {
  statement {
    effect = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["es.amazonaws.com"]
    }
  }
}

# IAM policy document for Elasticsearch snapshot permissions
data "aws_iam_policy_document" "elasticsearch_snapshot_policy" {
  statement {
    effect = "Allow"
    actions = [
      "s3:ListBucket",
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:GetBucketLocation"
    ]
    resources = [
      aws_s3_bucket.elasticsearch_snapshots.arn,
      "${aws_s3_bucket.elasticsearch_snapshots.arn}/*"
    ]
  }
}