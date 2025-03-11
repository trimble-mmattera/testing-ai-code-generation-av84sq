output "alb_id" {
  description = "ID of the created Application Load Balancer"
  value       = aws_lb.this.id
}

output "alb_arn" {
  description = "ARN of the Application Load Balancer for WAF association and other integrations"
  value       = aws_lb.this.arn
}

output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer for creating DNS records"
  value       = aws_lb.this.dns_name
}

output "target_group_arns" {
  description = "List of ARNs of the target groups for service registration"
  value       = [aws_lb_target_group.api.arn]
}

output "security_group_id" {
  description = "ID of the ALB security group for reference by other resources"
  value       = aws_security_group.alb.id
}