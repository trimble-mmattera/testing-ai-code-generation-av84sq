# AWS Provider version ~> 4.0
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Local variables for common values
locals {
  cluster_name = "${var.project_name}-${var.environment}-cluster"
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
  }
}

#------------------------------------------------
# IAM Roles and Policies for EKS
#------------------------------------------------

# IAM role for the EKS cluster
resource "aws_iam_role" "eks_cluster_role" {
  name = "${var.project_name}-${var.environment}-eks-cluster-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "eks.amazonaws.com"
        }
      }
    ]
  })

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-eks-cluster-role"
    }
  )
}

# Attach the required EKS cluster policy
resource "aws_iam_role_policy_attachment" "eks_cluster_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.eks_cluster_role.name
}

# IAM role for EKS node groups
resource "aws_iam_role" "eks_node_role" {
  name = "${var.project_name}-${var.environment}-eks-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-eks-node-role"
    }
  )
}

# Attach required policies for EKS worker nodes
resource "aws_iam_role_policy_attachment" "eks_worker_node_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "eks_cni_policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.eks_node_role.name
}

resource "aws_iam_role_policy_attachment" "ecr_read_only" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.eks_node_role.name
}

# Custom policy for document management platform specific needs
resource "aws_iam_policy" "eks_node_policy" {
  name        = "${var.project_name}-${var.environment}-eks-node-policy"
  description = "Policy for EKS nodes to access required AWS services"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:DeleteObject",
          "s3:ListBucket",
        ]
        Resource = [
          "arn:aws:s3:::${var.project_name}-${var.environment}-*/*",
          "arn:aws:s3:::${var.project_name}-${var.environment}-*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
        ]
        Resource = "arn:aws:sqs:*:*:${var.project_name}-${var.environment}-*"
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey"
        ]
        Resource = var.kms_key_arn != null ? var.kms_key_arn : "*"
      }
    ]
  })
}

# Attach the custom policy to the node role
resource "aws_iam_role_policy_attachment" "eks_node_custom_policy" {
  policy_arn = aws_iam_policy.eks_node_policy.arn
  role       = aws_iam_role.eks_node_role.name
}

#------------------------------------------------
# Security Groups for EKS
#------------------------------------------------

# Security group for the EKS cluster
resource "aws_security_group" "eks_cluster_sg" {
  name        = "${var.project_name}-${var.environment}-eks-cluster-sg"
  description = "Security group for EKS cluster"
  vpc_id      = var.vpc_id

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-eks-cluster-sg"
    }
  )
}

# Allow outbound traffic from the cluster
resource "aws_security_group_rule" "cluster_egress" {
  security_group_id = aws_security_group.eks_cluster_sg.id
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  description       = "Allow all outbound traffic"
}

# Allow nodes to communicate with the cluster API server
resource "aws_security_group_rule" "cluster_nodes_ingress" {
  security_group_id        = aws_security_group.eks_cluster_sg.id
  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.eks_cluster_sg.id
  description              = "Allow nodes to communicate with the cluster API server"
}

#------------------------------------------------
# EKS Cluster
#------------------------------------------------

# EKS Cluster
resource "aws_eks_cluster" "main" {
  name     = local.cluster_name
  role_arn = aws_iam_role.eks_cluster_role.arn
  version  = var.cluster_version

  # Enable control plane logging
  enabled_cluster_log_types = ["api", "audit", "authenticator", "controllerManager", "scheduler"]

  vpc_config {
    subnet_ids              = var.subnet_ids
    security_group_ids      = [aws_security_group.eks_cluster_sg.id]
    endpoint_private_access = true
    endpoint_public_access  = true
  }

  # Enable secrets encryption if KMS key ARN is provided
  dynamic "encryption_config" {
    for_each = var.kms_key_arn != null ? [1] : []
    content {
      provider {
        key_arn = var.kms_key_arn
      }
      resources = ["secrets"]
    }
  }

  depends_on = [
    aws_iam_role_policy_attachment.eks_cluster_policy
  ]

  tags = merge(
    local.common_tags,
    {
      Name = local.cluster_name
    }
  )
}

#------------------------------------------------
# EKS Node Groups
#------------------------------------------------

# General purpose node group for most workloads
resource "aws_eks_node_group" "general" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "${var.project_name}-${var.environment}-general-ng"
  node_role_arn   = aws_iam_role.eks_node_role.arn
  subnet_ids      = var.subnet_ids
  instance_types  = var.node_group_instance_types

  scaling_config {
    desired_size = var.node_group_desired_size
    min_size     = var.node_group_min_size
    max_size     = var.node_group_max_size
  }

  update_config {
    max_unavailable = 1
  }

  labels = {
    role = "general"
  }

  # Ensure IAM Role policies are attached before creating nodes
  depends_on = [
    aws_iam_role_policy_attachment.eks_worker_node_policy,
    aws_iam_role_policy_attachment.eks_cni_policy,
    aws_iam_role_policy_attachment.ecr_read_only,
    aws_iam_role_policy_attachment.eks_node_custom_policy
  ]

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-general-ng"
    }
  )
}

# Processing node group for CPU-intensive tasks (virus scanning, document processing)
resource "aws_eks_node_group" "processing" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "${var.project_name}-${var.environment}-processing-ng"
  node_role_arn   = aws_iam_role.eks_node_role.arn
  subnet_ids      = var.subnet_ids
  instance_types  = ["c5.2xlarge", "c5a.2xlarge"]

  scaling_config {
    desired_size = 2
    min_size     = 2
    max_size     = 6
  }

  update_config {
    max_unavailable = 1
  }

  labels = {
    role = "processing"
  }

  taint {
    key    = "workload"
    value  = "processing"
    effect = "NO_SCHEDULE"
  }

  # Ensure IAM Role policies are attached before creating nodes
  depends_on = [
    aws_iam_role_policy_attachment.eks_worker_node_policy,
    aws_iam_role_policy_attachment.eks_cni_policy,
    aws_iam_role_policy_attachment.ecr_read_only,
    aws_iam_role_policy_attachment.eks_node_custom_policy
  ]

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-processing-ng"
    }
  )
}

# Search node group for memory-intensive tasks (Elasticsearch, search operations)
resource "aws_eks_node_group" "search" {
  cluster_name    = aws_eks_cluster.main.name
  node_group_name = "${var.project_name}-${var.environment}-search-ng"
  node_role_arn   = aws_iam_role.eks_node_role.arn
  subnet_ids      = var.subnet_ids
  instance_types  = ["r5.2xlarge", "r5a.2xlarge"]

  scaling_config {
    desired_size = 2
    min_size     = 2
    max_size     = 6
  }

  update_config {
    max_unavailable = 1
  }

  labels = {
    role = "search"
  }

  taint {
    key    = "workload"
    value  = "search"
    effect = "NO_SCHEDULE"
  }

  # Ensure IAM Role policies are attached before creating nodes
  depends_on = [
    aws_iam_role_policy_attachment.eks_worker_node_policy,
    aws_iam_role_policy_attachment.eks_cni_policy,
    aws_iam_role_policy_attachment.ecr_read_only,
    aws_iam_role_policy_attachment.eks_node_custom_policy
  ]

  tags = merge(
    local.common_tags,
    {
      Name = "${var.project_name}-${var.environment}-search-ng"
    }
  )
}

#------------------------------------------------
# Outputs
#------------------------------------------------

output "cluster_name" {
  description = "The name of the EKS cluster"
  value       = aws_eks_cluster.main.name
}

output "cluster_endpoint" {
  description = "The endpoint for the EKS cluster API server"
  value       = aws_eks_cluster.main.endpoint
}

output "cluster_certificate_authority_data" {
  description = "The certificate authority data for the EKS cluster"
  value       = aws_eks_cluster.main.certificate_authority[0].data
}

output "cluster_security_group_id" {
  description = "The ID of the security group for the EKS cluster"
  value       = aws_security_group.eks_cluster_sg.id
}

output "node_group_role_arn" {
  description = "The ARN of the IAM role for the EKS node group"
  value       = aws_iam_role.eks_node_role.arn
}

#------------------------------------------------
# Variables
#------------------------------------------------

variable "project_name" {
  description = "Name of the project used for resource naming and tagging"
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

variable "vpc_id" {
  description = "ID of the VPC where the EKS cluster will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs where the EKS cluster and nodes will be deployed"
  type        = list(string)
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.25"
}

variable "node_group_instance_types" {
  description = "List of EC2 instance types for EKS node groups"
  type        = list(string)
  default     = ["m5.large", "m5a.large", "m5n.large"]
}

variable "node_group_desired_size" {
  description = "Desired number of nodes in the EKS node group"
  type        = number
  default     = 3
}

variable "node_group_min_size" {
  description = "Minimum number of nodes in the EKS node group"
  type        = number
  default     = 3
}

variable "node_group_max_size" {
  description = "Maximum number of nodes in the EKS node group"
  type        = number
  default     = 10
}

variable "kms_key_arn" {
  description = "ARN of the KMS key used for encryption"
  type        = string
  default     = null
}