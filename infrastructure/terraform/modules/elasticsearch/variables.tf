variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod) used for resource naming and tagging"
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region where Elasticsearch domain will be deployed"
  type        = string
  default     = "us-east-1"
}

variable "domain_name" {
  description = "Custom name for the Elasticsearch domain (optional, defaults to {project_name}-{environment}-es)"
  type        = string
  default     = ""
}

variable "vpc_id" {
  description = "ID of the VPC where Elasticsearch domain will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs where Elasticsearch domain will be deployed (must be in different AZs for zone awareness)"
  type        = list(string)
  default     = []
}

variable "instance_type" {
  description = "Instance type for Elasticsearch nodes"
  type        = string
  default     = "r5.large.elasticsearch"
}

variable "instance_count" {
  description = "Number of instances in the Elasticsearch cluster"
  type        = number
  default     = 3
}

variable "ebs_volume_size" {
  description = "Size of EBS volumes attached to Elasticsearch instances in GB"
  type        = number
  default     = 100
}

variable "ebs_volume_type" {
  description = "Type of EBS volumes attached to Elasticsearch instances"
  type        = string
  default     = "gp3"
}

variable "ebs_iops" {
  description = "IOPS for EBS volumes when using io1 or gp3 volume types"
  type        = number
  default     = 3000
}

variable "elasticsearch_version" {
  description = "Version of Elasticsearch to deploy"
  type        = string
  default     = "7.10"
}

variable "kms_key_id" {
  description = "KMS key ID for encrypting data at rest"
  type        = string
  default     = null
}

variable "snapshot_bucket_name" {
  description = "Custom name for the S3 bucket used for Elasticsearch snapshots (optional, defaults to {project_name}-{environment}-es-snapshots)"
  type        = string
  default     = ""
}

variable "snapshot_retention_days" {
  description = "Number of days to retain Elasticsearch snapshots"
  type        = number
  default     = 30
}

variable "allowed_security_groups" {
  description = "List of security group IDs allowed to access the Elasticsearch domain"
  type        = list(string)
  default     = []
}

variable "allowed_cidr_blocks" {
  description = "List of CIDR blocks allowed to access the Elasticsearch domain"
  type        = list(string)
  default     = []
}

variable "create_log_group" {
  description = "Whether to create a CloudWatch log group for Elasticsearch logs"
  type        = bool
  default     = true
}

variable "log_group_arn" {
  description = "ARN of an existing CloudWatch log group for Elasticsearch logs (required if create_log_group is false)"
  type        = string
  default     = null
}

variable "zone_awareness_enabled" {
  description = "Whether to enable zone awareness for the Elasticsearch domain"
  type        = bool
  default     = true
}

variable "availability_zone_count" {
  description = "Number of availability zones to use when zone awareness is enabled"
  type        = number
  default     = 3
}

variable "advanced_options" {
  description = "Advanced options for Elasticsearch configuration"
  type        = map(string)
  default = {
    "rest.action.multi.allow_explicit_index" = "true"
    "indices.fielddata.cache.size"           = "20"
    "indices.query.bool.max_clause_count"    = "1024"
  }
}

variable "tags" {
  description = "Additional tags to apply to Elasticsearch resources"
  type        = map(string)
  default     = {}
}