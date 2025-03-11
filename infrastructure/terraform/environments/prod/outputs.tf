# outputs.tf
# Production environment infrastructure outputs for Document Management Platform

# VPC and Networking Outputs
output "vpc_id" {
  description = "ID of the VPC created for the production environment"
  value       = module.vpc.vpc_id
}

output "vpc_cidr" {
  description = "CIDR block of the VPC"
  value       = module.vpc.vpc_cidr
}

output "public_subnet_ids" {
  description = "List of public subnet IDs"
  value       = module.vpc.public_subnet_ids
}

output "private_app_subnet_ids" {
  description = "List of private application subnet IDs"
  value       = module.vpc.private_app_subnet_ids
}

output "private_data_subnet_ids" {
  description = "List of private data subnet IDs"
  value       = module.vpc.private_data_subnet_ids
}

# EKS Cluster Outputs
output "eks_cluster_name" {
  description = "Name of the EKS cluster"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint for the EKS cluster API server"
  value       = module.eks.cluster_endpoint
}

output "eks_cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.eks.cluster_security_group_id
}

output "eks_node_security_group_id" {
  description = "Security group ID attached to the EKS worker nodes"
  value       = module.eks.node_security_group_id
}

# S3 Bucket Outputs
output "document_bucket_name" {
  description = "Name of the S3 bucket for document storage"
  value       = module.s3.document_bucket_name
}

output "document_bucket_arn" {
  description = "ARN of the S3 bucket for document storage"
  value       = module.s3.document_bucket_arn
}

output "temporary_bucket_name" {
  description = "Name of the S3 bucket for temporary document storage"
  value       = module.s3.temporary_bucket_name
}

output "quarantine_bucket_name" {
  description = "Name of the S3 bucket for quarantined documents"
  value       = module.s3.quarantine_bucket_name
}

output "logs_bucket_name" {
  description = "Name of the S3 bucket for logs storage"
  value       = module.s3.logs_bucket_name
}

# RDS Outputs
output "rds_endpoint" {
  description = "Connection endpoint for the RDS PostgreSQL instance"
  value       = module.rds.db_endpoint
}

output "rds_instance_name" {
  description = "Name of the RDS PostgreSQL instance"
  value       = module.rds.db_instance_name
}

output "rds_read_replica_endpoint" {
  description = "Connection endpoint for the RDS PostgreSQL read replica"
  value       = module.rds.read_replica_endpoint
}

# Elasticsearch Outputs
output "elasticsearch_endpoint" {
  description = "Endpoint for the Elasticsearch domain"
  value       = module.elasticsearch.endpoint
}

output "elasticsearch_kibana_endpoint" {
  description = "Kibana endpoint for the Elasticsearch domain"
  value       = module.elasticsearch.kibana_endpoint
}

# SQS Queue Outputs
output "document_processing_queue_url" {
  description = "URL of the SQS queue for document processing"
  value       = module.sqs.document_processing_queue_url
}

output "virus_scanning_queue_url" {
  description = "URL of the SQS queue for virus scanning"
  value       = module.sqs.virus_scanning_queue_url
}

output "indexing_queue_url" {
  description = "URL of the SQS queue for document indexing"
  value       = module.sqs.indexing_queue_url
}

# KMS Key Outputs
output "document_kms_key_id" {
  description = "ID of the KMS key used for document encryption"
  value       = module.kms.document_kms_key_id
}

output "document_kms_key_arn" {
  description = "ARN of the KMS key used for document encryption"
  value       = module.kms.document_kms_key_arn
}

output "database_kms_key_id" {
  description = "ID of the KMS key used for database encryption"
  value       = module.kms.database_kms_key_id
}

output "elasticsearch_kms_key_id" {
  description = "ID of the KMS key used for Elasticsearch encryption"
  value       = module.kms.elasticsearch_kms_key_id
}

# ALB Outputs
output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = module.alb.dns_name
}

output "alb_zone_id" {
  description = "Route 53 zone ID of the Application Load Balancer"
  value       = module.alb.zone_id
}

output "alb_security_group_id" {
  description = "Security group ID of the Application Load Balancer"
  value       = module.alb.security_group_id
}

output "alb_arn" {
  description = "ARN of the Application Load Balancer"
  value       = module.alb.alb_arn
}

# WAF Outputs
output "waf_web_acl_id" {
  description = "ID of the WAF Web ACL protecting the API"
  value       = module.waf.web_acl_id
}

output "waf_web_acl_arn" {
  description = "ARN of the WAF Web ACL protecting the API"
  value       = module.waf.web_acl_arn
}

# CloudFront Outputs
output "cloudfront_distribution_id" {
  description = "ID of the CloudFront distribution for content delivery"
  value       = module.cloudfront.distribution_id
}

output "cloudfront_domain_name" {
  description = "Domain name of the CloudFront distribution"
  value       = module.cloudfront.domain_name
}

# SNS Topic Outputs
output "sns_document_topic_arn" {
  description = "ARN of the SNS topic for document events"
  value       = module.sns.document_topic_arn
}

# Secrets Manager Outputs
output "db_credentials_secret_arn" {
  description = "ARN of the Secrets Manager secret for database credentials"
  value       = aws_secretsmanager_secret.db_credentials.arn
}

# Security and Monitoring Outputs
output "guardduty_detector_id" {
  description = "ID of the GuardDuty detector for threat detection"
  value       = module.guardduty.detector_id
}

output "flow_logs_group_name" {
  description = "Name of the CloudWatch Logs group for VPC flow logs"
  value       = module.vpc.flow_logs_group_name
}

# Disaster Recovery Outputs
output "dr_replication_role_arn" {
  description = "ARN of the IAM role used for cross-region replication to disaster recovery region"
  value       = module.s3.replication_role_arn
}