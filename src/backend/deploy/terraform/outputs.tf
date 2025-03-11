# outputs.tf
# 
# This file defines all exported values from the infrastructure deployment for the 
# Document Management Platform. These outputs provide essential information about 
# created resources that can be used by other systems or for reference.

# VPC Outputs
output "vpc_id" {
  description = "The ID of the VPC where resources are deployed"
  value       = module.vpc.vpc_id
}

output "public_subnet_ids" {
  description = "List of public subnet IDs"
  value       = module.vpc.public_subnet_ids
}

output "private_app_subnet_ids" {
  description = "List of private subnet IDs for application components"
  value       = module.vpc.private_app_subnet_ids
}

output "private_data_subnet_ids" {
  description = "List of private subnet IDs for data components"
  value       = module.vpc.private_data_subnet_ids
}

# KMS Outputs
output "kms_key_id" {
  description = "The ID of the KMS key used for encryption"
  value       = module.kms.key_id
}

output "kms_key_arn" {
  description = "The ARN of the KMS key used for encryption"
  value       = module.kms.key_arn
}

# S3 Bucket Outputs
output "document_bucket_name" {
  description = "The name of the S3 bucket for permanent document storage"
  value       = module.s3.document_bucket_name
}

output "temp_bucket_name" {
  description = "The name of the S3 bucket for temporary document storage"
  value       = module.s3.temp_bucket_name
}

output "quarantine_bucket_name" {
  description = "The name of the S3 bucket for quarantined documents"
  value       = module.s3.quarantine_bucket_name
}

output "document_bucket_arn" {
  description = "The ARN of the S3 bucket for permanent document storage"
  value       = module.s3.document_bucket_arn
}

output "temp_bucket_arn" {
  description = "The ARN of the S3 bucket for temporary document storage"
  value       = module.s3.temp_bucket_arn
}

output "quarantine_bucket_arn" {
  description = "The ARN of the S3 bucket for quarantined documents"
  value       = module.s3.quarantine_bucket_arn
}

# RDS Database Outputs
output "db_endpoint" {
  description = "The connection endpoint for the RDS PostgreSQL database"
  value       = module.rds.db_endpoint
}

output "db_port" {
  description = "The port for the RDS PostgreSQL database"
  value       = module.rds.db_port
}

output "db_name" {
  description = "The name of the PostgreSQL database"
  value       = module.rds.db_name
}

output "db_instance_id" {
  description = "The ID of the RDS instance"
  value       = module.rds.db_instance_id
}

# Security Group Outputs
output "db_security_group_id" {
  description = "The ID of the security group for the RDS instance"
  value       = module.rds.db_security_group_id
}

output "app_security_group_id" {
  description = "The ID of the security group for application components"
  value       = module.rds.app_security_group_id
}

# SQS Queue Outputs
output "document_processing_queue_url" {
  description = "The URL of the SQS queue for document processing"
  value       = module.sqs.document_processing_queue_url
}

output "document_processing_queue_arn" {
  description = "The ARN of the SQS queue for document processing"
  value       = module.sqs.document_processing_queue_arn
}

output "virus_scanning_queue_url" {
  description = "The URL of the SQS queue for virus scanning"
  value       = module.sqs.virus_scanning_queue_url
}

output "virus_scanning_queue_arn" {
  description = "The ARN of the SQS queue for virus scanning"
  value       = module.sqs.virus_scanning_queue_arn
}

output "indexing_queue_url" {
  description = "The URL of the SQS queue for document indexing"
  value       = module.sqs.indexing_queue_url
}

output "indexing_queue_arn" {
  description = "The ARN of the SQS queue for document indexing"
  value       = module.sqs.indexing_queue_arn
}

output "quarantine_queue_url" {
  description = "The URL of the SQS queue for quarantined documents"
  value       = module.sqs.quarantine_queue_url
}

output "quarantine_queue_arn" {
  description = "The ARN of the SQS queue for quarantined documents"
  value       = module.sqs.quarantine_queue_arn
}

# EKS Cluster Outputs
output "eks_cluster_name" {
  description = "The name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "The endpoint for the EKS cluster API server"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_certificate_authority_data" {
  description = "The certificate authority data for the EKS cluster"
  value       = module.eks.cluster_certificate_authority_data
}

output "eks_cluster_security_group_id" {
  description = "The ID of the security group for the EKS cluster"
  value       = module.eks.cluster_security_group_id
}

output "eks_node_group_role_arn" {
  description = "The ARN of the IAM role for the EKS node group"
  value       = module.eks.node_group_role_arn
}

# Secrets Manager Outputs
output "secrets_manager_secret_id" {
  description = "The ID of the Secrets Manager secret containing configuration"
  value       = aws_secretsmanager_secret.config.id
}

output "secrets_manager_secret_arn" {
  description = "The ARN of the Secrets Manager secret containing configuration"
  value       = aws_secretsmanager_secret.config.arn
}