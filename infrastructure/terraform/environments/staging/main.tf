# Terraform configuration for Document Management Platform - Staging Environment
# This file defines all the infrastructure resources required for the staging environment

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

  backend "s3" {
    bucket         = "document-mgmt-terraform-state"
    key            = "staging/terraform.tfstate"
    region         = "${var.aws_region}"
    dynamodb_table = "document-mgmt-terraform-locks"
    encrypt        = true
  }
}

provider "aws" {
  region  = var.aws_region
  profile = var.aws_profile
  
  default_tags {
    tags = {
      Environment = "staging"
      Project     = "document-management-platform"
      ManagedBy   = "terraform"
    }
  }
}

provider "random" {}

# Local variables for resource naming and tagging consistency
locals {
  environment = "staging"
  name_prefix = "${var.project_name}-${local.environment}"
}

# Generate random password for RDS if not provided
resource "random_password" "db_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# Store database credentials in Secrets Manager
resource "aws_secretsmanager_secret" "db_credentials" {
  name        = "${var.project_name}-staging-db-credentials"
  description = "Database credentials for ${var.project_name} staging environment"
  kms_key_id  = module.kms.database_kms_key_id
  tags        = var.tags
}

resource "aws_secretsmanager_secret_version" "db_credentials" {
  secret_id     = aws_secretsmanager_secret.db_credentials.id
  secret_string = "{\"username\":\"${var.db_username}\",\"password\":\"${var.db_password != null ? var.db_password : random_password.db_password.result}\"}"
}

# VPC Module - Creates the VPC and networking infrastructure
module "vpc" {
  source = "../../modules/vpc"
  
  project_name          = var.project_name
  environment           = "staging"
  region                = var.aws_region
  vpc_cidr              = var.vpc_cidr
  availability_zones    = ["${var.aws_region}a", "${var.aws_region}b", "${var.aws_region}c"]
  public_subnet_cidrs   = var.public_subnet_cidrs
  private_subnet_cidrs  = var.private_app_subnet_cidrs
  private_data_subnet_cidrs = var.private_data_subnet_cidrs
  single_nat_gateway    = false
  enable_flow_logs      = var.enable_flow_logs
  flow_logs_retention_days = var.flow_logs_retention_days
  enable_s3_endpoint    = true
  tags                  = var.tags
}

# KMS Module - Creates KMS keys for encryption of data at rest
module "kms" {
  source = "../../modules/kms"
  
  project_name    = var.project_name
  environment     = "staging"
  admin_role_arn  = var.admin_role_arn
  tags            = var.tags
}

# S3 Module - Creates S3 buckets for document storage
module "s3" {
  source = "../../modules/s3"
  
  project_name           = var.project_name
  environment            = "staging"
  document_bucket_name   = var.s3_document_bucket_name
  temporary_bucket_name  = var.s3_temporary_bucket_name
  quarantine_bucket_name = var.s3_quarantine_bucket_name
  kms_key_id             = module.kms.document_kms_key_id
  enable_versioning      = var.enable_s3_versioning
  document_retention_days    = var.document_retention_days
  temporary_retention_days   = var.temporary_retention_days
  quarantine_retention_days  = var.quarantine_retention_days
  enable_replication     = false
  tags                   = var.tags
}

# RDS Module - Creates PostgreSQL database for document metadata
module "rds" {
  source = "../../modules/rds"
  
  project_name           = var.project_name
  environment            = "staging"
  vpc_id                 = module.vpc.vpc_id
  subnet_ids             = module.vpc.private_data_subnet_ids
  instance_class         = var.db_instance_class
  allocated_storage      = var.db_allocated_storage
  max_allocated_storage  = var.db_max_allocated_storage
  db_name                = var.db_name
  username               = var.db_username
  password               = var.db_password != null ? var.db_password : random_password.db_password.result
  multi_az               = var.db_multi_az
  kms_key_id             = module.kms.database_kms_key_id
  backup_retention_period = var.db_backup_retention_period
  deletion_protection    = true
  performance_insights_enabled = true
  monitoring_interval    = 60
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]
  tags                   = var.tags
}

# EKS Module - Creates Kubernetes cluster for container orchestration
module "eks" {
  source = "../../modules/eks"
  
  project_name        = var.project_name
  environment         = "staging"
  cluster_name        = var.eks_cluster_name
  cluster_version     = var.eks_cluster_version
  vpc_id              = module.vpc.vpc_id
  subnet_ids          = module.vpc.private_app_subnet_ids
  
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
  
  cluster_log_types = ["api", "audit", "authenticator"]
  tags              = var.tags
}

# Elasticsearch Module - Creates Elasticsearch domain for document content search
module "elasticsearch" {
  source = "../../modules/elasticsearch"
  
  project_name          = var.project_name
  environment           = "staging"
  vpc_id                = module.vpc.vpc_id
  subnet_ids            = module.vpc.private_data_subnet_ids
  instance_type         = var.elasticsearch_instance_type
  instance_count        = var.elasticsearch_instance_count
  ebs_volume_size       = var.elasticsearch_volume_size
  kms_key_id            = module.kms.elasticsearch_kms_key_id
  snapshot_retention_days = 15
  create_log_group      = true
  allowed_security_groups = [module.eks.node_security_group_id]
  tags                  = var.tags
}

# SQS Module - Creates SQS queues for document processing and virus scanning
module "sqs" {
  source = "../../modules/sqs"
  
  project_name                  = var.project_name
  environment                   = "staging"
  document_processing_queue_name = var.sqs_document_processing_queue_name
  virus_scanning_queue_name     = var.sqs_virus_scanning_queue_name
  indexing_queue_name           = var.sqs_indexing_queue_name
  kms_key_id                    = module.kms.document_kms_key_id
  visibility_timeout            = 300
  message_retention_seconds     = 86400
  max_receive_count             = 5
  tags                          = var.tags
}

# ALB Module - Creates Application Load Balancer for API traffic
module "alb" {
  source = "../../modules/alb"
  
  project_name        = var.project_name
  environment         = "staging"
  vpc_id              = module.vpc.vpc_id
  public_subnet_ids   = module.vpc.public_subnet_ids
  certificate_arn     = var.certificate_arn
  access_logs_bucket  = module.s3.logs_bucket_name
  eks_security_group_id = module.eks.node_security_group_id
  tags                = var.tags
}

# WAF Module - Creates WAF for API protection
module "waf" {
  source = "../../modules/waf"
  
  project_name  = var.project_name
  environment   = "staging"
  alb_arn       = module.alb.alb_arn
  logs_bucket   = module.s3.logs_bucket_name
  enable_waf    = var.enable_waf
  tags          = var.tags
}

# SNS Module - Creates SNS topics for event notifications
module "sns" {
  source = "../../modules/sns"
  
  project_name  = var.project_name
  environment   = "staging"
  kms_key_id    = module.kms.document_kms_key_id
  tags          = var.tags
}