# Define required providers
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.16"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.8"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
  }
}

# Local variables for Prometheus configuration
locals {
  prometheus_config = {
    namespace             = var.monitoring_namespace
    service_account_name  = "prometheus"
    prometheus_storage_class = "gp2"
    prometheus_storage_size = "50Gi"
    retention_period      = "30d"
    version               = "2.45.0"
    replicas              = 1
  }
}

# Variables
variable "project_name" {
  type        = string
  description = "Name of the project"
  default     = "document-mgmt"
}

variable "environment" {
  type        = string
  description = "Deployment environment (dev, staging, prod)"
  default     = "dev"
}

variable "monitoring_namespace" {
  type        = string
  description = "Kubernetes namespace for monitoring resources"
  default     = "monitoring"
}

variable "prometheus_version" {
  type        = string
  description = "Version of the Prometheus Helm chart"
  default     = "15.10.1"
}

# S3 bucket for storing Prometheus data backups
resource "aws_s3_bucket" "prometheus_backups" {
  bucket = "${var.project_name}-${var.environment}-prometheus-backups"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-prometheus-backups"
    Environment = var.environment
    Project     = var.project_name
    Component   = "Monitoring"
  }
}

# Configure server-side encryption for the Prometheus backups bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "prometheus_backups_encryption" {
  bucket = aws_s3_bucket.prometheus_backups.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# IAM role for Prometheus to access AWS resources
resource "aws_iam_role" "prometheus_role" {
  name = "${var.project_name}-${var.environment}-prometheus-role"
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      },
      {
        Effect = "Allow"
        Principal = {
          Federated = "${aws_iam_openid_connect_provider.eks_oidc.arn}"
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "${aws_iam_openid_connect_provider.eks_oidc.url}:sub": "system:serviceaccount:${local.prometheus_config.namespace}:${local.prometheus_config.service_account_name}"
          }
        }
      }
    ]
  })
}

# IAM policy for Prometheus to access S3 and CloudWatch
resource "aws_iam_policy" "prometheus_policy" {
  name        = "${var.project_name}-${var.environment}-prometheus-policy"
  description = "Policy for Prometheus to access S3 and CloudWatch"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:DeleteObject"
        ]
        Resource = [
          "${aws_s3_bucket.prometheus_backups.arn}",
          "${aws_s3_bucket.prometheus_backups.arn}/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:GetMetricData",
          "cloudwatch:ListMetrics",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:DescribeAlarms"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach the Prometheus policy to the Prometheus role
resource "aws_iam_role_policy_attachment" "prometheus_policy_attachment" {
  role       = aws_iam_role.prometheus_role.name
  policy_arn = aws_iam_policy.prometheus_policy.arn
}

# Kubernetes service account for Prometheus
resource "kubernetes_service_account" "prometheus" {
  metadata {
    name      = local.prometheus_config.service_account_name
    namespace = local.prometheus_config.namespace
    
    annotations = {
      "eks.amazonaws.com/role-arn" = aws_iam_role.prometheus_role.arn
    }
  }
}

# ConfigMap containing Prometheus configuration
resource "kubernetes_config_map" "prometheus_config" {
  metadata {
    name      = "prometheus-config"
    namespace = local.prometheus_config.namespace
  }
  
  data = {
    "prometheus.yml"   = file("${path.module}/../prometheus/prometheus.yml")
    "alert-rules.yml" = file("${path.module}/../prometheus/alert-rules.yml")
  }
}

# PVC for Prometheus data storage
resource "kubernetes_persistent_volume_claim" "prometheus_storage" {
  metadata {
    name      = "prometheus-storage"
    namespace = local.prometheus_config.namespace
  }
  
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = local.prometheus_config.prometheus_storage_class
    
    resources {
      requests = {
        storage = local.prometheus_config.prometheus_storage_size
      }
    }
  }
}

# Kubernetes deployment for Prometheus
resource "kubernetes_deployment" "prometheus" {
  metadata {
    name      = "prometheus"
    namespace = local.prometheus_config.namespace
    
    labels = {
      app       = "prometheus"
      component = "monitoring"
    }
  }
  
  spec {
    replicas = local.prometheus_config.replicas
    
    selector {
      match_labels = {
        app = "prometheus"
      }
    }
    
    strategy {
      type = "Recreate"
    }
    
    template {
      metadata {
        labels = {
          app = "prometheus"
        }
        
        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/port"   = "9090"
        }
      }
      
      spec {
        service_account_name = local.prometheus_config.service_account_name
        
        security_context {
          fs_group        = 65534
          run_as_non_root = true
          run_as_user     = 65534
        }
        
        container {
          name  = "prometheus"
          image = "prom/prometheus:v${local.prometheus_config.version}"
          
          args = [
            "--config.file=/etc/prometheus/prometheus.yml",
            "--storage.tsdb.path=/prometheus",
            "--storage.tsdb.retention.time=${local.prometheus_config.retention_period}",
            "--web.console.libraries=/etc/prometheus/console_libraries",
            "--web.console.templates=/etc/prometheus/consoles",
            "--web.enable-lifecycle",
            "--web.external-url=http://prometheus.${local.prometheus_config.namespace}.svc:9090"
          ]
          
          port {
            container_port = 9090
            name           = "http"
            protocol       = "TCP"
          }
          
          resources {
            limits = {
              cpu    = "1000m"
              memory = "4Gi"
            }
            requests = {
              cpu    = "500m"
              memory = "2Gi"
            }
          }
          
          volume_mount {
            name       = "prometheus-config"
            mount_path = "/etc/prometheus"
          }
          
          volume_mount {
            name       = "prometheus-storage"
            mount_path = "/prometheus"
          }
          
          liveness_probe {
            http_get {
              path = "/-/healthy"
              port = "http"
            }
            
            initial_delay_seconds = 30
            timeout_seconds       = 5
            period_seconds        = 15
          }
          
          readiness_probe {
            http_get {
              path = "/-/ready"
              port = "http"
            }
            
            initial_delay_seconds = 30
            timeout_seconds       = 5
            period_seconds        = 15
          }
        }
        
        volume {
          name = "prometheus-config"
          
          config_map {
            name = "prometheus-config"
          }
        }
        
        volume {
          name = "prometheus-storage"
          
          persistent_volume_claim {
            claim_name = "prometheus-storage"
          }
        }
        
        node_selector = {
          "kubernetes.io/os" = "linux"
        }
        
        affinity {
          node_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              
              preference {
                match_expressions {
                  key      = "node-role.kubernetes.io/monitoring"
                  operator = "Exists"
                }
              }
            }
          }
        }
        
        toleration {
          key      = "monitoring"
          operator = "Equal"
          value    = "true"
          effect   = "NoSchedule"
        }
      }
    }
  }
}

# Kubernetes service for Prometheus
resource "kubernetes_service" "prometheus" {
  metadata {
    name      = "prometheus"
    namespace = local.prometheus_config.namespace
    
    labels = {
      app       = "prometheus"
      component = "monitoring"
    }
    
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "9090"
    }
  }
  
  spec {
    selector = {
      app = "prometheus"
    }
    
    port {
      port        = 9090
      target_port = 9090
      protocol    = "TCP"
      name        = "http"
    }
    
    type = "ClusterIP"
  }
}

# Alternative Helm deployment of Prometheus (commented out but available as an option)
/*
resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  version    = var.prometheus_version
  namespace  = local.prometheus_config.namespace
  create_namespace = false
  
  values = [
    file("${path.module}/../prometheus/values.yaml")
  ]
  
  set {
    name  = "server.persistentVolume.enabled"
    value = "true"
  }
  
  set {
    name  = "server.persistentVolume.size"
    value = local.prometheus_config.prometheus_storage_size
  }
  
  set {
    name  = "server.persistentVolume.storageClass"
    value = local.prometheus_config.prometheus_storage_class
  }
  
  set {
    name  = "server.retention"
    value = local.prometheus_config.retention_period
  }
  
  set {
    name  = "serviceAccounts.server.name"
    value = local.prometheus_config.service_account_name
  }
  
  set {
    name  = "serviceAccounts.server.annotations.eks\\.amazonaws\\.com/role-arn"
    value = aws_iam_role.prometheus_role.arn
  }
  
  set {
    name  = "server.resources.limits.cpu"
    value = "1000m"
  }
  
  set {
    name  = "server.resources.limits.memory"
    value = "4Gi"
  }
  
  set {
    name  = "server.resources.requests.cpu"
    value = "500m"
  }
  
  set {
    name  = "server.resources.requests.memory"
    value = "2Gi"
  }
}
*/

# Outputs
output "prometheus_endpoint" {
  description = "Endpoint URL for Prometheus server"
  value       = "${kubernetes_service.prometheus.metadata[0].name}.${kubernetes_service.prometheus.metadata[0].namespace}.svc.cluster.local:9090"
}

output "prometheus_service_account" {
  description = "Service account used by Prometheus"
  value       = kubernetes_service_account.prometheus.metadata[0].name
}

output "prometheus_backups_bucket" {
  description = "S3 bucket name for Prometheus backups"
  value       = aws_s3_bucket.prometheus_backups.bucket
}

output "prometheus_role_arn" {
  description = "ARN of the IAM role for Prometheus"
  value       = aws_iam_role.prometheus_role.arn
}