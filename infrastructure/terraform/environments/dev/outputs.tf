# VPC Outputs
output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "vpc_cidr" {
  description = "CIDR block of the VPC"
  value       = module.vpc.vpc_cidr
}

# Subnet Outputs
output "public_subnet_ids" {
  description = "List of IDs of public subnets"
  value       = module.vpc.public_subnet_ids
}

output "private_app_subnet_ids" {
  description = "List of IDs of private application subnets"
  value       = module.vpc.private_app_subnet_ids
}

output "private_data_subnet_ids" {
  description = "List of IDs of private data subnets"
  value       = module.vpc.private_data_subnet_ids
}

# EKS Cluster Outputs
output "eks_cluster_name" {
  description = "Name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint URL of the EKS cluster"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_certificate_authority_data" {
  description = "Certificate authority data for the EKS cluster"
  value       = module.eks.cluster_certificate_authority_data
}

output "eks_node_security_group_id" {
  description = "Security group ID for EKS nodes"
  value       = module.eks.node_security_group_id
}

# RDS Outputs
output "rds_endpoint" {
  description = "Endpoint of the RDS PostgreSQL database"
  value       = module.rds.endpoint
}

output "rds_database_name" {
  description = "Name of the PostgreSQL database"
  value       = module.rds.database_name
}

output "rds_security_group_id" {
  description = "Security group ID for the RDS instance"
  value       = module.rds.security_group_id
}

# Elasticsearch Outputs
output "elasticsearch_endpoint" {
  description = "Endpoint URL of the Elasticsearch domain"
  value       = module.elasticsearch.endpoint
}

output "elasticsearch_kibana_endpoint" {
  description = "Kibana endpoint URL for the Elasticsearch domain"
  value       = module.elasticsearch.kibana_endpoint
}

output "elasticsearch_security_group_id" {
  description = "Security group ID for the Elasticsearch domain"
  value       = module.elasticsearch.security_group_id
}

# S3 Bucket Outputs
output "s3_document_bucket_name" {
  description = "Name of the S3 bucket for document storage"
  value       = module.s3.document_bucket_name
}

output "s3_temporary_bucket_name" {
  description = "Name of the S3 bucket for temporary document storage"
  value       = module.s3.temporary_bucket_name
}

output "s3_quarantine_bucket_name" {
  description = "Name of the S3 bucket for quarantined documents"
  value       = module.s3.quarantine_bucket_name
}

# KMS Outputs
output "kms_key_id" {
  description = "ID of the KMS key for encryption"
  value       = module.kms.key_id
}

output "kms_key_arn" {
  description = "ARN of the KMS key for encryption"
  value       = module.kms.key_arn
}

# SQS Outputs
output "sqs_document_processing_queue_url" {
  description = "URL of the SQS queue for document processing"
  value       = module.sqs.document_processing_queue_url
}

output "sqs_virus_scanning_queue_url" {
  description = "URL of the SQS queue for virus scanning"
  value       = module.sqs.virus_scanning_queue_url
}

output "sqs_indexing_queue_url" {
  description = "URL of the SQS queue for document indexing"
  value       = module.sqs.indexing_queue_url
}

# SNS Outputs
output "sns_document_topic_arn" {
  description = "ARN of the SNS topic for document events"
  value       = module.sns.document_topic_arn
}

# Secrets Manager Outputs
output "db_credentials_secret_arn" {
  description = "ARN of the Secrets Manager secret for database credentials"
  value       = aws_secretsmanager_secret.db_credentials.arn
}