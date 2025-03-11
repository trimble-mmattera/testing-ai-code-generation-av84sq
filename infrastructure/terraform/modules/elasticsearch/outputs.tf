# Outputs for the Elasticsearch module

output "endpoint" {
  description = "Elasticsearch domain endpoint URL"
  value       = aws_elasticsearch_domain.main.endpoint
}

output "domain_name" {
  description = "Elasticsearch domain name"
  value       = aws_elasticsearch_domain.main.domain_name
}

output "arn" {
  description = "Elasticsearch domain ARN"
  value       = aws_elasticsearch_domain.main.arn
}

output "snapshot_bucket" {
  description = "S3 bucket name for Elasticsearch snapshots"
  value       = aws_s3_bucket.elasticsearch_snapshots.bucket
}

output "security_group_id" {
  description = "Security group ID for Elasticsearch domain"
  value       = aws_security_group.elasticsearch.id
}

output "kibana_endpoint" {
  description = "Kibana endpoint URL for the Elasticsearch domain"
  value       = aws_elasticsearch_domain.main.kibana_endpoint
}