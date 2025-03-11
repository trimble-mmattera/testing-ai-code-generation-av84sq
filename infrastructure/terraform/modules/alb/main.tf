# AWS provider version ~> 4.0
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Local variables for naming consistency and tags
locals {
  name_prefix = "${var.project_name}-${var.environment}"
  common_tags = {
    Name        = "${var.project_name}-${var.environment}-alb"
    Environment = var.environment
    Project     = var.project_name
    ManagedBy   = "terraform"
  }
}

# Application Load Balancer
resource "aws_lb" "this" {
  name               = "${local.name_prefix}-alb"
  load_balancer_type = "application"
  internal           = false
  subnets            = var.public_subnet_ids
  security_groups    = [aws_security_group.alb.id]
  idle_timeout       = var.idle_timeout
  
  enable_deletion_protection = var.enable_deletion_protection
  
  access_logs {
    bucket  = var.access_logs_bucket
    prefix  = "alb-logs"
    enabled = true
  }
  
  tags = local.common_tags
}

# Target group for API services
resource "aws_lb_target_group" "api" {
  name     = "${local.name_prefix}-api-tg"
  port     = 80
  protocol = "HTTP"
  vpc_id   = var.vpc_id
  target_type = "ip"
  deregistration_delay = 30
  
  health_check {
    path                = var.health_check_path
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = var.health_check_timeout
    interval            = var.health_check_interval
    healthy_threshold   = var.health_check_healthy_threshold
    unhealthy_threshold = var.health_check_unhealthy_threshold
    matcher             = "200-299"
  }
  
  tags = local.common_tags
}

# HTTPS listener with TLS 1.3 support
resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.this.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.certificate_arn
  
  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }
  
  tags = local.common_tags
}

# HTTP listener with redirect to HTTPS
resource "aws_lb_listener" "http_redirect" {
  load_balancer_arn = aws_lb.this.arn
  port              = 80
  protocol          = "HTTP"
  
  default_action {
    type = "redirect"
    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
  
  tags = local.common_tags
}

# Security group for the ALB
resource "aws_security_group" "alb" {
  name        = "${local.name_prefix}-alb-sg"
  description = "Security group for ${local.name_prefix} ALB"
  vpc_id      = var.vpc_id
  
  tags = local.common_tags
}

# HTTP ingress rule
resource "aws_security_group_rule" "alb_ingress_http" {
  security_group_id = aws_security_group.alb.id
  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  description       = "Allow HTTP traffic from internet"
}

# HTTPS ingress rule
resource "aws_security_group_rule" "alb_ingress_https" {
  security_group_id = aws_security_group.alb.id
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  description       = "Allow HTTPS traffic from internet"
}

# Egress rule to EKS cluster
resource "aws_security_group_rule" "alb_egress_eks" {
  security_group_id        = aws_security_group.alb.id
  type                     = "egress"
  from_port                = 0
  to_port                  = 65535
  protocol                 = "tcp"
  source_security_group_id = var.eks_security_group_id
  description              = "Allow all outbound traffic to EKS cluster"
}

# Ingress rule for EKS from ALB
resource "aws_security_group_rule" "eks_ingress_alb" {
  security_group_id        = var.eks_security_group_id
  type                     = "ingress"
  from_port                = 0
  to_port                  = 65535
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.alb.id
  description              = "Allow all inbound traffic from ALB"
}

# Variables
variable "project_name" {
  type        = string
  description = "Name of the project used for resource naming and tagging"
  default     = "document-mgmt"
  
  validation {
    condition     = length(var.project_name) > 0
    error_message = "Project name cannot be empty"
  }
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
  
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be one of: dev, staging, prod"
  }
}

variable "vpc_id" {
  type        = string
  description = "ID of the VPC where the ALB will be deployed"
  
  validation {
    condition     = length(var.vpc_id) > 0
    error_message = "VPC ID cannot be empty"
  }
}

variable "public_subnet_ids" {
  type        = list(string)
  description = "List of public subnet IDs where the ALB will be deployed"
  
  validation {
    condition     = length(var.public_subnet_ids) >= 2
    error_message = "At least two public subnets are required for high availability"
  }
}

variable "certificate_arn" {
  type        = string
  description = "ARN of the ACM certificate for HTTPS"
  
  validation {
    condition     = length(var.certificate_arn) > 0
    error_message = "Certificate ARN cannot be empty"
  }
}

variable "access_logs_bucket" {
  type        = string
  description = "Name of the S3 bucket for ALB access logs"
  
  validation {
    condition     = length(var.access_logs_bucket) > 0
    error_message = "Access logs bucket name cannot be empty"
  }
}

variable "eks_security_group_id" {
  type        = string
  description = "ID of the EKS cluster security group"
  
  validation {
    condition     = length(var.eks_security_group_id) > 0
    error_message = "EKS security group ID cannot be empty"
  }
}

variable "tags" {
  type        = map(string)
  description = "Additional tags to apply to all resources"
  default     = {}
}

variable "idle_timeout" {
  type        = number
  description = "The time in seconds that the connection is allowed to be idle"
  default     = 60
}

variable "enable_deletion_protection" {
  type        = bool
  description = "If true, deletion of the load balancer will be disabled via the AWS API"
  default     = var.environment == "prod" ? true : false
}

variable "health_check_path" {
  type        = string
  description = "Path for ALB health checks"
  default     = "/health"
}

variable "health_check_timeout" {
  type        = number
  description = "Timeout for health checks in seconds"
  default     = 5
}

variable "health_check_interval" {
  type        = number
  description = "Interval between health checks in seconds"
  default     = 15
}

variable "health_check_healthy_threshold" {
  type        = number
  description = "Number of consecutive successful health checks required to consider target healthy"
  default     = 2
}

variable "health_check_unhealthy_threshold" {
  type        = number
  description = "Number of consecutive failed health checks required to consider target unhealthy"
  default     = 2
}

# Outputs
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