# Main Terraform configuration for Document Management Platform's monitoring infrastructure

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.16"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.8"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
  }

  # Backend configuration stub - actual values provided during terraform init
  backend "s3" {}
}

# Configure AWS provider
provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
  
  default_tags {
    tags = {
      Environment = var.environment
      Project     = var.project_name
      Component   = "Monitoring"
      ManagedBy   = "terraform"
    }
  }
}

# Configure Kubernetes provider
provider "kubernetes" {
  host                   = var.kubernetes_host
  token                  = var.kubernetes_token
  cluster_ca_certificate = base64decode(var.kubernetes_cluster_ca_certificate)
  
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args = [
      "eks",
      "get-token",
      "--cluster-name",
      var.eks_cluster_name,
      "--region",
      var.aws_region,
      "--profile",
      var.aws_profile
    ]
  }
}

# Configure Helm provider
provider "helm" {
  kubernetes {
    host                   = var.kubernetes_host
    token                  = var.kubernetes_token
    cluster_ca_certificate = base64decode(var.kubernetes_cluster_ca_certificate)
    
    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args = [
        "eks",
        "get-token",
        "--cluster-name",
        var.eks_cluster_name,
        "--region",
        var.aws_region,
        "--profile",
        var.aws_profile
      ]
    }
  }
}

# Configure Random provider
provider "random" {}

# Local variables for common configuration values
locals {
  common_tags = {
    Environment = var.environment
    Project     = var.project_name
    Component   = "Monitoring"
    ManagedBy   = "terraform"
  }
  
  monitoring_domain = var.environment == "prod" ? "monitoring.${var.domain_name}" : "monitoring.${var.environment}.${var.domain_name}"
}

# OIDC provider for EKS to allow pod-based IAM roles
resource "aws_iam_openid_connect_provider" "eks_oidc" {
  url             = var.eks_oidc_issuer_url
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = [var.eks_oidc_thumbprint]
}

# Kubernetes namespace for all monitoring resources
resource "kubernetes_namespace" "monitoring" {
  metadata {
    name = var.monitoring_namespace
    labels = {
      name    = var.monitoring_namespace
      purpose = "monitoring"
    }
  }
}

# S3 bucket for storing monitoring configuration backups
resource "aws_s3_bucket" "monitoring_backups" {
  bucket = "${var.project_name}-${var.environment}-monitoring-backups"
  tags   = local.common_tags
}

# Configure server-side encryption for the monitoring backups bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "backups_encryption" {
  bucket = aws_s3_bucket.monitoring_backups.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# Configure lifecycle rules for monitoring backups retention
resource "aws_s3_bucket_lifecycle_configuration" "backups_lifecycle" {
  bucket = aws_s3_bucket.monitoring_backups.id

  rule {
    id     = "backups-retention"
    status = "Enabled"

    expiration {
      days = var.backups_retention_days
    }
  }
}

# IAM role for monitoring components to access AWS resources
resource "aws_iam_role" "monitoring_role" {
  name = "${var.project_name}-${var.environment}-monitoring-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      },
      {
        Effect = "Allow"
        Principal = {
          Federated = aws_iam_openid_connect_provider.eks_oidc.arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${replace(aws_iam_openid_connect_provider.eks_oidc.url, "https://", "")}:sub" = "system:serviceaccount:${var.monitoring_namespace}:*"
          }
        }
      }
    ]
  })
}

# IAM policy for monitoring components to access AWS resources
resource "aws_iam_policy" "monitoring_policy" {
  name        = "${var.project_name}-${var.environment}-monitoring-policy"
  description = "Policy for monitoring components to access AWS resources"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:DeleteObject"
        ]
        Resource = [
          aws_s3_bucket.monitoring_backups.arn,
          "${aws_s3_bucket.monitoring_backups.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:GetMetricData",
          "cloudwatch:ListMetrics",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:DescribeAlarms"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams",
          "logs:GetLogEvents",
          "logs:FilterLogEvents"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach the monitoring policy to the monitoring role
resource "aws_iam_role_policy_attachment" "monitoring_policy_attachment" {
  role       = aws_iam_role.monitoring_role.name
  policy_arn = aws_iam_policy.monitoring_policy.arn
}

# ConfigMap containing common monitoring configuration
resource "kubernetes_config_map" "monitoring_config" {
  metadata {
    name      = "monitoring-config"
    namespace = var.monitoring_namespace
  }

  data = {
    "environment.json" = jsonencode({
      project     = var.project_name
      environment = var.environment
      region      = var.aws_region
      domain      = local.monitoring_domain
    })
  }
}

# Variables
variable "project_name" {
  type        = string
  description = "Name of the project"
  default     = "document-mgmt"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
}

variable "aws_region" {
  type        = string
  description = "AWS region for resources"
  default     = "us-west-2"
}

variable "aws_profile" {
  type        = string
  description = "AWS CLI profile to use for authentication"
  default     = "document-mgmt"
}

variable "domain_name" {
  type        = string
  description = "Base domain name for the application"
  default     = "document-mgmt.com"
}

variable "monitoring_namespace" {
  type        = string
  description = "Kubernetes namespace for monitoring resources"
  default     = "monitoring"
}

variable "eks_cluster_name" {
  type        = string
  description = "Name of the EKS cluster"
  default     = "document-mgmt-cluster"
}

variable "eks_oidc_issuer_url" {
  type        = string
  description = "OIDC issuer URL for the EKS cluster"
}

variable "eks_oidc_thumbprint" {
  type        = string
  description = "OIDC thumbprint for the EKS cluster"
}

variable "kubernetes_host" {
  type        = string
  description = "Kubernetes API server endpoint"
}

variable "kubernetes_token" {
  type        = string
  description = "Kubernetes API server token"
  sensitive   = true
}

variable "kubernetes_cluster_ca_certificate" {
  type        = string
  description = "Kubernetes cluster CA certificate"
  sensitive   = true
}

variable "terraform_state_bucket" {
  type        = string
  description = "S3 bucket for storing Terraform state"
  default     = "document-mgmt-terraform-state"
}

variable "terraform_lock_table" {
  type        = string
  description = "DynamoDB table for Terraform state locking"
  default     = "document-mgmt-terraform-locks"
}

variable "backups_retention_days" {
  type        = number
  description = "Number of days to retain monitoring backups"
  default     = 30
}

# Outputs
output "monitoring_namespace" {
  description = "Kubernetes namespace where monitoring resources are deployed"
  value       = kubernetes_namespace.monitoring.metadata[0].name
}

output "monitoring_domain" {
  description = "Domain name for monitoring components"
  value       = local.monitoring_domain
}

output "monitoring_role_arn" {
  description = "ARN of the IAM role for monitoring components"
  value       = aws_iam_role.monitoring_role.arn
}

output "monitoring_backups_bucket" {
  description = "S3 bucket name for monitoring backups"
  value       = aws_s3_bucket.monitoring_backups.bucket
}