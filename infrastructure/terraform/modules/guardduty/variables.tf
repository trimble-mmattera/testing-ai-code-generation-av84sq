variable "project_name" {
  type        = string
  description = "Name of the project used for resource naming and tagging"
  default     = "document-mgmt"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
}

variable "finding_publishing_frequency" {
  type        = string
  description = "Frequency of publishing findings from GuardDuty to CloudWatch Events"
  default     = "SIX_HOURS"
  
  validation {
    condition     = contains(["FIFTEEN_MINUTES", "ONE_HOUR", "SIX_HOURS"], self.value)
    error_message = "Valid values for finding_publishing_frequency are: FIFTEEN_MINUTES, ONE_HOUR, SIX_HOURS."
  }
}

variable "enable_s3_protection" {
  type        = bool
  description = "Whether to enable S3 protection in GuardDuty"
  default     = true
}

variable "enable_kubernetes_protection" {
  type        = bool
  description = "Whether to enable Kubernetes audit log monitoring in GuardDuty"
  default     = true
}

variable "enable_malware_protection" {
  type        = bool
  description = "Whether to enable malware protection for EC2 instances with findings"
  default     = true
}

variable "alert_severity_threshold" {
  type        = number
  description = "Minimum severity level (1-8) of GuardDuty findings to trigger alerts"
  default     = 4
  
  validation {
    condition     = self.value >= 1 && self.value <= 8
    error_message = "The alert_severity_threshold must be between 1 and 8, inclusive."
  }
}

variable "sns_topic_subscription_emails" {
  type        = list(string)
  description = "List of email addresses to subscribe to the security alerts SNS topic"
  default     = []
}

variable "trusted_ip_list_enabled" {
  type        = bool
  description = "Whether to enable trusted IP list for GuardDuty"
  default     = false
}

variable "trusted_ip_list_location" {
  type        = string
  description = "S3 location for trusted IP list (s3://bucket-name/prefix/object.txt)"
  default     = ""
}

variable "threat_intel_set_enabled" {
  type        = bool
  description = "Whether to enable threat intelligence set for GuardDuty"
  default     = false
}

variable "threat_intel_set_location" {
  type        = string
  description = "S3 location for threat intelligence set (s3://bucket-name/prefix/object.txt)"
  default     = ""
}