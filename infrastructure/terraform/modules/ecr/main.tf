# Local variables for naming consistency and common tags
locals {
  repository_prefix = "${var.project_name}-${var.environment}"
  common_tags = {
    Project     = "${var.project_name}"
    Environment = "${var.environment}"
    ManagedBy   = "terraform"
  }
}

# Create ECR repositories for each microservice
resource "aws_ecr_repository" "this" {
  for_each = toset(var.repository_names)

  name                 = "${local.repository_prefix}-${each.value}"
  image_tag_mutability = "IMMUTABLE"
  
  image_scanning_configuration {
    scan_on_push = true
  }
  
  encryption_configuration {
    encryption_type = var.kms_key_id != null ? "KMS" : "AES256"
    kms_key         = var.kms_key_id != null ? var.kms_key_id : null
  }
  
  tags = local.common_tags
}

# Configure lifecycle policies for ECR repositories to manage image retention
resource "aws_ecr_lifecycle_policy" "this" {
  for_each = toset(var.repository_names)

  repository = aws_ecr_repository.this[each.key].name
  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep only ${var.image_retention_count} images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = var.image_retention_count
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}

# Sets access policies for ECR repositories to allow CI/CD pipeline access
resource "aws_ecr_repository_policy" "this" {
  for_each = var.cicd_role_arn != null ? toset(var.repository_names) : []

  repository = aws_ecr_repository.this[each.key].name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowPushPull"
        Effect    = "Allow"
        Principal = {
          AWS = var.cicd_role_arn
        }
        Action = [
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload"
        ]
      }
    ]
  })
}

# Output for use in CI/CD pipelines and Kubernetes deployments
output "repository_urls" {
  description = "Map of repository names to their URLs for use in CI/CD pipelines and Kubernetes deployments"
  value = {
    for k, v in aws_ecr_repository.this : k => v.repository_url
  }
}

output "repository_names" {
  description = "Map of repository keys to their full names including environment and project prefixes"
  value = {
    for k, v in aws_ecr_repository.this : k => v.name
  }
}

output "registry_id" {
  description = "The registry ID where the repositories are created"
  value = length(aws_ecr_repository.this) > 0 ? values(aws_ecr_repository.this)[0].registry_id : null
}

output "registry_url" {
  description = "The base URL of the ECR registry without repository names"
  value = length(aws_ecr_repository.this) > 0 ? replace(values(aws_ecr_repository.this)[0].repository_url, "/${values(aws_ecr_repository.this)[0].name}", "") : null
}