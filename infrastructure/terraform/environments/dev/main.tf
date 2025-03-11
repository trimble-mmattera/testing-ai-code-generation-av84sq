# ------------------------------------------------------------------------------
# Terraform configuration for Document Management Platform - Development Environment
# 
# This configuration orchestrates the provisioning of AWS resources required for
# the development environment, including networking, compute, storage, and database.
# ------------------------------------------------------------------------------

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

  # Note: Region cannot use variables in backend configuration
  # Use partial configuration with CLI: terraform init -backend-config="region=us-east-1"
  backend "s3" {
    bucket         = "document-mgmt-terraform-state"
    key            = "dev/terraform.tfstate"
    dynamodb_table = "document-mgmt-terraform-locks"
    encrypt        = true
  }
}

# ------------------------------------------------------------------------------
# Providers
# ------------------------------------------------------------------------------

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile

  default_tags {
    tags = {
      Environment = "dev"
      Project     = "document-management-platform"
      ManagedBy   = "terraform"
    }
  }
}

provider "random" {}

# ------------------------------------------------------------------------------
# Local variables for resource naming and tagging consistency
# ------------------------------------------------------------------------------

locals {
  # Common tags to be applied to all resources
  tags = merge(var.tags, {
    Environment = var.environment
    Project     = var.project_name
  })
}

# ------------------------------------------------------------------------------
# Random password for database (if not explicitly provided)
# ------------------------------------------------------------------------------

resource "random_password" "db_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# ------------------------------------------------------------------------------
# Secrets Manager for database credentials
# ------------------------------------------------------------------------------

resource "aws_secretsmanager_secret" "db_credentials" {
  name        = "${var.project_name}-${var.environment}-db-credentials"
  description = "Database credentials for ${var.project_name} ${var.environment} environment"
  kms_key_id  = module.kms.key_id
  tags        = var.tags
}

resource "aws_secretsmanager_secret_version" "db_credentials" {
  secret_id = aws_secretsmanager_secret.db_credentials.id
  secret_string = jsonencode({
    username = var.db_username
    password = var.db_password != null ? var.db_password : random_password.db_password.result
  })
}

# ------------------------------------------------------------------------------
# VPC Module - Creates the VPC and networking infrastructure
# ------------------------------------------------------------------------------

module "vpc" {
  source = "../../modules/vpc"

  project_name          = var.project_name
  environment           = var.environment
  region                = var.aws_region
  vpc_cidr              = var.vpc_cidr
  availability_zones    = ["${var.aws_region}a", "${var.aws_region}b"]
  public_subnet_cidrs   = var.public_subnet_cidrs
  private_subnet_cidrs  = var.private_app_subnet_cidrs
  private_data_subnet_cidrs = var.private_data_subnet_cidrs
  
  single_nat_gateway = true  # Use single NAT gateway for dev environment to reduce costs
  enable_flow_logs   = true
  enable_s3_endpoint = true
  
  tags = var.tags
}

# ------------------------------------------------------------------------------
# KMS Module - Creates KMS keys for encryption of data at rest
# ------------------------------------------------------------------------------

module "kms" {
  source = "../../modules/kms"

  project_name = var.project_name
  environment  = var.environment
  tags         = var.tags
}

# ------------------------------------------------------------------------------
# S3 Module - Creates S3 buckets for document storage
# ------------------------------------------------------------------------------

module "s3" {
  source = "../../modules/s3"

  project_name          = var.project_name
  environment           = var.environment
  document_bucket_name  = var.s3_document_bucket_name
  temporary_bucket_name = var.s3_temporary_bucket_name
  quarantine_bucket_name = var.s3_quarantine_bucket_name
  
  kms_key_id        = module.kms.key_id
  enable_versioning = var.enable_s3_versioning
  
  document_retention_days   = 30
  temporary_retention_days  = 1
  quarantine_retention_days = 30
  
  tags = var.tags
}

# ------------------------------------------------------------------------------
# RDS Module - Creates RDS PostgreSQL database for document metadata
# ------------------------------------------------------------------------------

module "rds" {
  source = "../../modules/rds"

  project_name = var.project_name
  environment  = var.environment
  
  vpc_id      = module.vpc.vpc_id
  subnet_ids  = module.vpc.private_data_subnet_ids
  
  instance_class        = var.db_instance_class
  allocated_storage     = var.db_allocated_storage
  max_allocated_storage = var.db_max_allocated_storage
  
  db_name   = var.db_name
  username  = var.db_username
  password  = var.db_password != null ? var.db_password : random_password.db_password.result
  
  multi_az                = var.db_multi_az
  kms_key_id              = module.kms.key_id
  backup_retention_period = 7
  deletion_protection     = false  # Set to false for dev environment for easy cleanup
  
  tags = var.tags
}

# ------------------------------------------------------------------------------
# EKS Module - Creates EKS cluster for container orchestration
# ------------------------------------------------------------------------------

module "eks" {
  source = "../../modules/eks"

  project_name    = var.project_name
  environment     = var.environment
  cluster_name    = var.eks_cluster_name
  cluster_version = var.eks_cluster_version
  
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_app_subnet_ids
  
  node_groups = {
    general = {
      instance_types = var.eks_node_group_instance_types.general
      desired_size   = var.eks_node_group_desired_sizes.general
      min_size       = var.eks_node_group_min_sizes.general
      max_size       = var.eks_node_group_max_sizes.general
    }
    processing = {
      instance_types = var.eks_node_group_instance_types.processing
      desired_size   = var.eks_node_group_desired_sizes.processing
      min_size       = var.eks_node_group_min_sizes.processing
      max_size       = var.eks_node_group_max_sizes.processing
    }
    search = {
      instance_types = var.eks_node_group_instance_types.search
      desired_size   = var.eks_node_group_desired_sizes.search
      min_size       = var.eks_node_group_min_sizes.search
      max_size       = var.eks_node_group_max_sizes.search
    }
  }
  
  tags = var.tags
}

# ------------------------------------------------------------------------------
# Elasticsearch Module - Creates Elasticsearch domain for document content search
# ------------------------------------------------------------------------------

module "elasticsearch" {
  source = "../../modules/elasticsearch"

  project_name = var.project_name
  environment  = var.environment
  
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_data_subnet_ids
  
  instance_type   = var.elasticsearch_instance_type
  instance_count  = var.elasticsearch_instance_count
  ebs_volume_size = var.elasticsearch_volume_size
  
  kms_key_id = module.kms.key_id
  tags       = var.tags
}

# ------------------------------------------------------------------------------
# SQS Module - Creates SQS queues for document processing and virus scanning
# ------------------------------------------------------------------------------

module "sqs" {
  source = "../../modules/sqs"

  project_name = var.project_name
  environment  = var.environment
  
  document_processing_queue_name = var.sqs_document_processing_queue_name
  virus_scanning_queue_name      = var.sqs_virus_scanning_queue_name
  indexing_queue_name            = var.sqs_indexing_queue_name
  
  kms_key_id = module.kms.key_id
  tags       = var.tags
}

# ------------------------------------------------------------------------------
# SNS Module - Creates SNS topics for event notifications
# ------------------------------------------------------------------------------

module "sns" {
  source = "../../modules/sns"

  project_name = var.project_name
  environment  = var.environment
  
  kms_key_id = module.kms.key_id
  tags       = var.tags
}