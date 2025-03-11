# Terraform module for RDS PostgreSQL database
# This module creates and configures a PostgreSQL database for the Document Management Platform

terraform {
  required_providers {
    aws = { # hashicorp/aws ~> 4.0
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Variables
variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "vpc_id" {
  description = "ID of the VPC where the RDS instance will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs where the RDS instance will be deployed"
  type        = list(string)
}

variable "db_name" {
  description = "Name of the PostgreSQL database"
  type        = string
  default     = "documentdb"
}

variable "db_username" {
  description = "Username for the PostgreSQL database"
  type        = string
  default     = "dbadmin"
}

variable "db_password" {
  description = "Password for the PostgreSQL database"
  type        = string
  sensitive   = true
}

variable "db_instance_class" {
  description = "Instance class for the RDS PostgreSQL database"
  type        = string
  default     = "db.t3.medium"
}

variable "multi_az" {
  description = "Whether to enable Multi-AZ deployment for RDS"
  type        = bool
  default     = true
}

variable "kms_key_id" {
  description = "ID of the KMS key used for encryption"
  type        = string
}

variable "allocated_storage" {
  description = "Allocated storage size in GB"
  type        = number
  default     = 20
}

variable "max_allocated_storage" {
  description = "Maximum storage size in GB for autoscaling"
  type        = number
  default     = 100
}

# Local variables
locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

# Security group for the RDS instance
resource "aws_security_group" "db" {
  name        = "${var.project_name}-${var.environment}-db-sg"
  description = "Security group for ${var.project_name} ${var.environment} RDS instance"
  vpc_id      = var.vpc_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-db-sg"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Security group for application components that need to access the database
resource "aws_security_group" "app" {
  name        = "${var.project_name}-${var.environment}-app-sg"
  description = "Security group for ${var.project_name} ${var.environment} application components"
  vpc_id      = var.vpc_id

  tags = {
    Name        = "${var.project_name}-${var.environment}-app-sg"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Allows PostgreSQL traffic from application security group to database
resource "aws_security_group_rule" "db_ingress" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  security_group_id        = aws_security_group.db.id
  source_security_group_id = aws_security_group.app.id
  description              = "Allow PostgreSQL traffic from application"
}

# Allows outbound traffic from database to internet
resource "aws_security_group_rule" "db_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  security_group_id = aws_security_group.db.id
  cidr_blocks       = ["0.0.0.0/0"]
  description       = "Allow all outbound traffic"
}

# Allows outbound traffic from application to internet
resource "aws_security_group_rule" "app_egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  security_group_id = aws_security_group.app.id
  cidr_blocks       = ["0.0.0.0/0"]
  description       = "Allow all outbound traffic"
}

# Subnet group for the RDS instance spanning multiple availability zones
resource "aws_db_subnet_group" "main" {
  name        = "${var.project_name}-${var.environment}-db-subnet-group"
  subnet_ids  = var.subnet_ids
  description = "Subnet group for ${var.project_name} ${var.environment} RDS instance"

  tags = {
    Name        = "${var.project_name}-${var.environment}-db-subnet-group"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Parameter group for the PostgreSQL database with optimized settings
resource "aws_db_parameter_group" "main" {
  name        = "${var.project_name}-${var.environment}-pg-param-group"
  family      = "postgres14"
  description = "Parameter group for ${var.project_name} ${var.environment} PostgreSQL database"

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_statement"
    value = "ddl"
  }

  parameter {
    name  = "shared_buffers"
    value = "{DBInstanceClassMemory/32768}MB"
  }

  tags = {
    Name        = "${var.project_name}-${var.environment}-pg-param-group"
    Environment = var.environment
    Project     = var.project_name
  }
}

# RDS PostgreSQL database instance for document metadata storage
resource "aws_db_instance" "main" {
  identifier                          = "${var.project_name}-${var.environment}-db"
  engine                              = "postgres"
  engine_version                      = "14.6"
  instance_class                      = var.db_instance_class
  allocated_storage                   = var.allocated_storage
  max_allocated_storage               = var.max_allocated_storage
  storage_type                        = "gp3"
  storage_encrypted                   = true
  kms_key_id                          = var.kms_key_id
  db_name                             = var.db_name
  username                            = var.db_username
  password                            = var.db_password
  port                                = 5432
  vpc_security_group_ids              = [aws_security_group.db.id]
  db_subnet_group_name                = aws_db_subnet_group.main.name
  parameter_group_name                = aws_db_parameter_group.main.name
  backup_retention_period             = 7
  backup_window                       = "03:00-04:00"
  maintenance_window                  = "sun:04:30-sun:05:30"
  multi_az                            = var.multi_az
  publicly_accessible                 = false
  skip_final_snapshot                 = false
  final_snapshot_identifier           = "${var.project_name}-${var.environment}-db-final-snapshot"
  deletion_protection                 = true
  performance_insights_enabled        = true
  performance_insights_retention_period = 7
  enabled_cloudwatch_logs_exports     = ["postgresql", "upgrade"]
  auto_minor_version_upgrade          = true
  copy_tags_to_snapshot               = true

  tags = {
    Name        = "${var.project_name}-${var.environment}-db"
    Environment = var.environment
    Project     = var.project_name
  }
}

# Output values
output "db_endpoint" {
  description = "The connection endpoint for the RDS PostgreSQL database"
  value       = aws_db_instance.main.endpoint
}

output "db_port" {
  description = "The port for the RDS PostgreSQL database"
  value       = aws_db_instance.main.port
}

output "db_name" {
  description = "The name of the PostgreSQL database"
  value       = aws_db_instance.main.db_name
}

output "db_instance_id" {
  description = "The ID of the RDS instance"
  value       = aws_db_instance.main.id
}

output "db_security_group_id" {
  description = "The ID of the security group for the RDS instance"
  value       = aws_security_group.db.id
}

output "app_security_group_id" {
  description = "The ID of the security group for application components"
  value       = aws_security_group.app.id
}