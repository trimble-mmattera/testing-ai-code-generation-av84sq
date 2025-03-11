# Outputs for the ECR module
# These outputs provide necessary information for CI/CD pipelines
# and other Terraform modules to interact with the created ECR repositories

output "repository_urls" {
  description = "Map of repository names to their complete URLs"
  value       = {for k, v in aws_ecr_repository.this : k => v.repository_url}
}

output "repository_names" {
  description = "Map of repository keys to their full names"
  value       = {for k, v in aws_ecr_repository.this : k => v.name}
}

output "registry_id" {
  description = "The registry ID where the repositories are created"
  value       = length(aws_ecr_repository.this) > 0 ? values(aws_ecr_repository.this)[0].registry_id : null
}

output "registry_url" {
  description = "The base URL of the ECR registry without repository names"
  value       = length(aws_ecr_repository.this) > 0 ? replace(values(aws_ecr_repository.this)[0].repository_url, "/${values(aws_ecr_repository.this)[0].name}", "") : null
}