# ------------------------------------------------------------------------------
# main.tf
# 
# Main Terraform configuration file that orchestrates the provisioning of AWS 
# infrastructure for the Document Management Platform. This file defines the AWS
# provider configuration and coordinates the creation of all required resources
# by calling specialized modules for VPC, EKS, RDS, S3, KMS, and SQS components.
# ------------------------------------------------------------------------------

# Define required providers and versions
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws" # v4.0
      version = "~> 4.0"
    }
    random = {
      source  = "hashicorp/random" # v3.0
      version = "~> 3.0"
    }
  }
  required_version = ">= 1.0.0"
  
  # Backend configuration uses partial configuration
  # Initialize with:
  # terraform init \
  #   -backend-config="bucket=<project_name>-<environment>-terraform-state" \
  #   -backend-config="region=<aws_region>" \
  #   -backend-config="dynamodb_table=<project_name>-<environment>-terraform-locks"
  backend "s3" {
    key     = "terraform.tfstate"
    encrypt = true
  }
}

# Configure the AWS Provider with region and default tags
provider "aws" {
  region = var.aws_region
  default_tags {
    tags = local.common_tags
  }
}

# Define local variables for use throughout the configuration
locals {
  # CIDR blocks for private data subnets where databases will be deployed
  private_data_subnet_cidrs = ["10.0.20.0/24", "10.0.21.0/24", "10.0.22.0/24"]
  
  # Common tags to apply to all resources
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

# Generate a secure random password for the RDS database
resource "random_password" "db_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
  min_special      = 2
  min_upper        = 2
  min_lower        = 2
  min_numeric      = 2
}

# Create a Secrets Manager secret to store sensitive configuration values
resource "aws_secretsmanager_secret" "config" {
  name                    = "${var.project_name}-${var.environment}-config"
  description             = "Configuration for ${var.project_name} ${var.environment} environment"
  recovery_window_in_days = 7
  tags = {
    Name        = "${var.project_name}-${var.environment}-config"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Create a version of the Secrets Manager secret with the actual configuration values
resource "aws_secretsmanager_secret_version" "config" {
  secret_id = aws_secretsmanager_secret.config.id
  secret_string = jsonencode({
    db_username = var.db_username
    db_password = random_password.db_password.result
    db_name     = var.db_name
    db_host     = module.rds.db_endpoint
    db_port     = module.rds.db_port
    document_bucket = module.s3.document_bucket_name
    temp_bucket     = module.s3.temp_bucket_name
    quarantine_bucket = module.s3.quarantine_bucket_name
    document_processing_queue_url = module.sqs.document_processing_queue_url
    virus_scanning_queue_url      = module.sqs.virus_scanning_queue_url
    indexing_queue_url            = module.sqs.indexing_queue_url
    quarantine_queue_url          = module.sqs.quarantine_queue_url
  })
}

# Create VPC with public and private subnets across multiple availability zones
module "vpc" {
  source = "../../infrastructure/terraform/modules/vpc"

  project_name   = var.project_name
  environment    = var.environment
  region         = var.aws_region
  vpc_cidr       = var.vpc_cidr
  availability_zones = var.availability_zones
  public_subnet_cidrs = var.public_subnet_cidrs
  private_subnet_cidrs = var.private_subnet_cidrs
  private_data_subnet_cidrs = local.private_data_subnet_cidrs
}

# Create KMS keys for encrypting sensitive data at rest
module "kms" {
  source = "./modules/kms"

  project_name = var.project_name
  environment  = var.environment
  enable_key_rotation = true
  deletion_window_in_days = 30
}

# Create S3 buckets for document storage with appropriate encryption and lifecycle policies
module "s3" {
  source = "./modules/s3"

  project_name = var.project_name
  environment  = var.environment
  kms_key_id   = module.kms.key_id
  document_retention_days = var.document_retention_days
  quarantine_retention_days = var.quarantine_retention_days
}

# Create RDS PostgreSQL database for document metadata storage
module "rds" {
  source = "./modules/rds"

  project_name = var.project_name
  environment  = var.environment
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_data_subnet_ids
  db_name      = var.db_name
  db_username  = var.db_username
  db_password  = random_password.db_password.result
  db_instance_class = var.db_instance_class
  multi_az    = var.multi_az
  kms_key_id  = module.kms.key_id
  allocated_storage = 20
  max_allocated_storage = 100
}

# Create SQS queues for asynchronous document processing and virus scanning
module "sqs" {
  source = "./modules/sqs"

  project_name = var.project_name
  environment  = var.environment
  kms_key_id   = module.kms.key_id
}

# Create EKS cluster for running containerized microservices
module "eks" {
  source = "./modules/eks"

  project_name = var.project_name
  environment  = var.environment
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_app_subnet_ids
  cluster_version = var.eks_cluster_version
  node_group_instance_types = var.node_group_instance_types
  node_group_desired_size = var.node_group_desired_size
  node_group_min_size = var.node_group_min_size
  node_group_max_size = var.node_group_max_size
  kms_key_arn = module.kms.key_arn
}