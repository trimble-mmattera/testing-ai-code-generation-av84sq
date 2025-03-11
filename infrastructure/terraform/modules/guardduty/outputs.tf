# Outputs from the GuardDuty module for integration with security monitoring
# and incident response automation components

output "detector_id" {
  description = "The ID of the GuardDuty detector"
  value       = aws_guardduty_detector.detector.id
}

output "detector_arn" {
  description = "The ARN of the GuardDuty detector"
  value       = aws_guardduty_detector.detector.arn
}

output "security_alerts_topic_arn" {
  description = "The ARN of the SNS topic for security alerts"
  value       = aws_sns_topic.security_alerts.arn
}

output "security_alerts_topic_name" {
  description = "The name of the SNS topic for security alerts"
  value       = aws_sns_topic.security_alerts.name
}

output "guardduty_findings_event_rule_arn" {
  description = "The ARN of the CloudWatch Event rule for GuardDuty findings"
  value       = aws_cloudwatch_event_rule.guardduty_findings.arn
}

output "guardduty_findings_event_rule_name" {
  description = "The name of the CloudWatch Event rule for GuardDuty findings"
  value       = aws_cloudwatch_event_rule.guardduty_findings.name
}