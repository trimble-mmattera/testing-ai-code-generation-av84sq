# Grafana configuration for the Document Management Platform monitoring infrastructure
# This file defines AWS and Kubernetes resources needed for Grafana deployment

# Required providers
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws" # v4.0
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes" # v2.16
      version = "~> 2.16"
    }
    helm = {
      source  = "hashicorp/helm" # v2.8
      version = "~> 2.8"
    }
    random = {
      source  = "hashicorp/random" # v3.4
      version = "~> 3.4"
    }
  }
}

# Local variables for Grafana configuration
locals {
  grafana_config = {
    namespace            = var.monitoring_namespace
    service_account_name = "grafana"
    grafana_storage_class = "gp2"
    grafana_storage_size  = "10Gi"
    version              = var.grafana_version
    replicas             = 1
    admin_user           = "admin"
  }
  
  monitoring_domain = "${var.environment}.${var.project_name}.com"
}

# Generate a random password for Grafana admin user
resource "random_password" "grafana_admin_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# S3 bucket for storing Grafana dashboard backups
resource "aws_s3_bucket" "grafana_backups" {
  bucket = "${var.project_name}-${var.environment}-grafana-backups"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-grafana-backups"
    Environment = var.environment
    Project     = var.project_name
    Component   = "Monitoring"
  }
}

# Configures server-side encryption for the Grafana backups bucket
resource "aws_s3_bucket_server_side_encryption_configuration" "grafana_backups" {
  bucket = aws_s3_bucket.grafana_backups.id
  
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# IAM role for Grafana to access AWS resources
resource "aws_iam_role" "grafana_role" {
  name = "${var.project_name}-${var.environment}-grafana-role"
  
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
            "${aws_iam_openid_connect_provider.eks_oidc.url}:sub": "system:serviceaccount:${local.grafana_config.namespace}:${local.grafana_config.service_account_name}"
          }
        }
      }
    ]
  })
}

# IAM policy for Grafana to access S3 and CloudWatch
resource "aws_iam_policy" "grafana_policy" {
  name        = "${var.project_name}-${var.environment}-grafana-policy"
  description = "Policy for Grafana to access S3 and CloudWatch"
  
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
          "${aws_s3_bucket.grafana_backups.arn}",
          "${aws_s3_bucket.grafana_backups.arn}/*"
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

# Attaches the Grafana policy to the Grafana role
resource "aws_iam_role_policy_attachment" "grafana_policy_attachment" {
  role       = aws_iam_role.grafana_role.name
  policy_arn = aws_iam_policy.grafana_policy.arn
}

# Kubernetes service account for Grafana
resource "kubernetes_service_account" "grafana" {
  metadata {
    name      = local.grafana_config.service_account_name
    namespace = local.grafana_config.namespace
    annotations = {
      "eks.amazonaws.com/role-arn" = aws_iam_role.grafana_role.arn
    }
  }
}

# Secret containing Grafana admin credentials
resource "kubernetes_secret" "grafana_admin_credentials" {
  metadata {
    name      = "grafana-admin-credentials"
    namespace = local.grafana_config.namespace
  }

  data = {
    "admin-user"     = base64encode(local.grafana_config.admin_user)
    "admin-password" = base64encode(random_password.grafana_admin_password.result)
  }
}

# ConfigMap containing Grafana datasources configuration
resource "kubernetes_config_map" "grafana_datasources" {
  metadata {
    name      = "grafana-datasources"
    namespace = local.grafana_config.namespace
  }

  data = {
    "datasources.yaml" = file("${path.module}/../grafana/datasources/datasources.yml")
  }
}

# ConfigMap containing Grafana dashboards configuration
resource "kubernetes_config_map" "grafana_dashboards_config" {
  metadata {
    name      = "grafana-dashboards-config"
    namespace = local.grafana_config.namespace
  }

  data = {
    "dashboard-provider.yaml" = <<EOF
apiVersion: 1
providers:
- name: 'default'
  orgId: 1
  folder: ''
  type: file
  disableDeletion: false
  updateIntervalSeconds: 30
  options:
    path: /var/lib/grafana/dashboards
EOF
  }
}

# ConfigMap containing Grafana API dashboard
resource "kubernetes_config_map" "grafana_dashboard_api" {
  metadata {
    name      = "grafana-dashboard-api"
    namespace = local.grafana_config.namespace
  }

  data = {
    "api-dashboard.json" = file("${path.module}/../grafana/dashboards/api-dashboard.json")
  }
}

# ConfigMap containing Grafana Document Service dashboard
resource "kubernetes_config_map" "grafana_dashboard_document_service" {
  metadata {
    name      = "grafana-dashboard-document-service"
    namespace = local.grafana_config.namespace
  }

  data = {
    "document-service-dashboard.json" = file("${path.module}/../grafana/dashboards/document-service-dashboard.json")
  }
}

# ConfigMap containing Grafana Search Service dashboard
resource "kubernetes_config_map" "grafana_dashboard_search_service" {
  metadata {
    name      = "grafana-dashboard-search-service"
    namespace = local.grafana_config.namespace
  }

  data = {
    "search-service-dashboard.json" = file("${path.module}/../grafana/dashboards/search-service-dashboard.json")
  }
}

# ConfigMap containing Grafana Storage Service dashboard
resource "kubernetes_config_map" "grafana_dashboard_storage_service" {
  metadata {
    name      = "grafana-dashboard-storage-service"
    namespace = local.grafana_config.namespace
  }

  data = {
    "storage-service-dashboard.json" = file("${path.module}/../grafana/dashboards/storage-service-dashboard.json")
  }
}

# ConfigMap containing Grafana Business Metrics dashboard
resource "kubernetes_config_map" "grafana_dashboard_business_metrics" {
  metadata {
    name      = "grafana-dashboard-business-metrics"
    namespace = local.grafana_config.namespace
  }

  data = {
    "business-metrics-dashboard.json" = file("${path.module}/../grafana/dashboards/business-metrics-dashboard.json")
  }
}

# PVC for Grafana data storage
resource "kubernetes_persistent_volume_claim" "grafana_storage" {
  metadata {
    name      = "grafana-storage"
    namespace = local.grafana_config.namespace
  }
  
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = local.grafana_config.grafana_storage_class
    
    resources {
      requests = {
        storage = local.grafana_config.grafana_storage_size
      }
    }
  }
}

# Kubernetes deployment for Grafana
resource "kubernetes_deployment" "grafana" {
  metadata {
    name      = "grafana"
    namespace = local.grafana_config.namespace
    labels = {
      app       = "grafana"
      component = "monitoring"
    }
  }

  spec {
    replicas = local.grafana_config.replicas
    
    selector {
      match_labels = {
        app = "grafana"
      }
    }
    
    strategy {
      type = "RollingUpdate"
    }
    
    template {
      metadata {
        labels = {
          app = "grafana"
        }
      }
      
      spec {
        service_account_name = local.grafana_config.service_account_name
        
        security_context {
          fs_group    = 472
          run_as_user = 472
        }
        
        container {
          name  = "grafana"
          image = "grafana/grafana:${local.grafana_config.version}"
          
          port {
            container_port = 3000
            name           = "http"
            protocol       = "TCP"
          }
          
          resources {
            limits = {
              cpu    = "500m"
              memory = "1Gi"
            }
            requests = {
              cpu    = "250m"
              memory = "512Mi"
            }
          }
          
          env {
            name = "GF_SECURITY_ADMIN_USER"
            value_from {
              secret_key_ref {
                name = "grafana-admin-credentials"
                key  = "admin-user"
              }
            }
          }
          
          env {
            name = "GF_SECURITY_ADMIN_PASSWORD"
            value_from {
              secret_key_ref {
                name = "grafana-admin-credentials"
                key  = "admin-password"
              }
            }
          }
          
          env {
            name  = "GF_INSTALL_PLUGINS"
            value = "grafana-piechart-panel,grafana-worldmap-panel,grafana-clock-panel"
          }
          
          env {
            name  = "GF_PATHS_PROVISIONING"
            value = "/etc/grafana/provisioning"
          }
          
          env {
            name  = "GF_PATHS_DASHBOARDS"
            value = "/var/lib/grafana/dashboards"
          }
          
          volume_mount {
            name       = "grafana-storage"
            mount_path = "/var/lib/grafana"
          }
          
          volume_mount {
            name       = "grafana-datasources"
            mount_path = "/etc/grafana/provisioning/datasources"
          }
          
          volume_mount {
            name       = "grafana-dashboards-config"
            mount_path = "/etc/grafana/provisioning/dashboards"
          }
          
          volume_mount {
            name       = "grafana-dashboard-api"
            mount_path = "/var/lib/grafana/dashboards/api"
          }
          
          volume_mount {
            name       = "grafana-dashboard-document-service"
            mount_path = "/var/lib/grafana/dashboards/document-service"
          }
          
          volume_mount {
            name       = "grafana-dashboard-search-service"
            mount_path = "/var/lib/grafana/dashboards/search-service"
          }
          
          volume_mount {
            name       = "grafana-dashboard-storage-service"
            mount_path = "/var/lib/grafana/dashboards/storage-service"
          }
          
          volume_mount {
            name       = "grafana-dashboard-business-metrics"
            mount_path = "/var/lib/grafana/dashboards/business-metrics"
          }
          
          liveness_probe {
            http_get {
              path = "/api/health"
              port = "http"
            }
            initial_delay_seconds = 60
            timeout_seconds       = 30
            period_seconds        = 10
            failure_threshold     = 5
          }
          
          readiness_probe {
            http_get {
              path = "/api/health"
              port = "http"
            }
            initial_delay_seconds = 30
            timeout_seconds       = 30
            period_seconds        = 10
            failure_threshold     = 5
          }
        }
        
        volume {
          name = "grafana-storage"
          persistent_volume_claim {
            claim_name = "grafana-storage"
          }
        }
        
        volume {
          name = "grafana-datasources"
          config_map {
            name = "grafana-datasources"
          }
        }
        
        volume {
          name = "grafana-dashboards-config"
          config_map {
            name = "grafana-dashboards-config"
          }
        }
        
        volume {
          name = "grafana-dashboard-api"
          config_map {
            name = "grafana-dashboard-api"
          }
        }
        
        volume {
          name = "grafana-dashboard-document-service"
          config_map {
            name = "grafana-dashboard-document-service"
          }
        }
        
        volume {
          name = "grafana-dashboard-search-service"
          config_map {
            name = "grafana-dashboard-search-service"
          }
        }
        
        volume {
          name = "grafana-dashboard-storage-service"
          config_map {
            name = "grafana-dashboard-storage-service"
          }
        }
        
        volume {
          name = "grafana-dashboard-business-metrics"
          config_map {
            name = "grafana-dashboard-business-metrics"
          }
        }
      }
    }
  }
  
  depends_on = [
    kubernetes_persistent_volume_claim.grafana_storage,
    kubernetes_config_map.grafana_datasources,
    kubernetes_config_map.grafana_dashboards_config,
    kubernetes_service_account.grafana
  ]
}

# Kubernetes service for Grafana
resource "kubernetes_service" "grafana" {
  metadata {
    name      = "grafana"
    namespace = local.grafana_config.namespace
    labels = {
      app       = "grafana"
      component = "monitoring"
    }
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "3000"
    }
  }

  spec {
    selector = {
      app = "grafana"
    }
    
    port {
      port        = 80
      target_port = 3000
      protocol    = "TCP"
      name        = "http"
    }
    
    type = "ClusterIP"
  }
}

# Ingress for external access to Grafana
resource "kubernetes_ingress_v1" "grafana" {
  metadata {
    name      = "grafana"
    namespace = local.grafana_config.namespace
    annotations = {
      "kubernetes.io/ingress.class"             = "nginx"
      "cert-manager.io/cluster-issuer"          = "letsencrypt-prod"
      "nginx.ingress.kubernetes.io/ssl-redirect" = "true"
    }
  }

  spec {
    rule {
      host = "grafana.${local.monitoring_domain}"
      http {
        path {
          path      = "/"
          path_type = "Prefix"
          backend {
            service {
              name = "grafana"
              port {
                name = "http"
              }
            }
          }
        }
      }
    }
    
    tls {
      hosts       = ["grafana.${local.monitoring_domain}"]
      secret_name = "grafana-tls"
    }
  }
}

# Alternative Helm deployment of Grafana (commented out but available as an option)
/*
resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = "6.50.0"
  namespace  = local.grafana_config.namespace
  create_namespace = false

  values = [
    file("${path.module}/../grafana/values.yaml")
  ]

  set {
    name  = "persistence.enabled"
    value = "true"
  }

  set {
    name  = "persistence.size"
    value = local.grafana_config.grafana_storage_size
  }

  set {
    name  = "persistence.storageClassName"
    value = local.grafana_config.grafana_storage_class
  }

  set {
    name  = "adminUser"
    value = local.grafana_config.admin_user
  }

  set {
    name  = "adminPassword"
    value = random_password.grafana_admin_password.result
  }

  set {
    name  = "serviceAccount.name"
    value = local.grafana_config.service_account_name
  }

  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = aws_iam_role.grafana_role.arn
  }

  set {
    name  = "datasources.datasources\\.yaml.apiVersion"
    value = "1"
  }

  set {
    name  = "datasources.datasources\\.yaml.datasources[0].name"
    value = "Prometheus"
  }

  set {
    name  = "datasources.datasources\\.yaml.datasources[0].type"
    value = "prometheus"
  }

  set {
    name  = "datasources.datasources\\.yaml.datasources[0].url"
    value = "http://prometheus-server:9090"
  }

  set {
    name  = "datasources.datasources\\.yaml.datasources[0].isDefault"
    value = "true"
  }
}
*/

# Input variables
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

variable "grafana_version" {
  type        = string
  description = "Version of Grafana to deploy"
  default     = "9.3.6"
}

# Outputs
output "grafana_endpoint" {
  description = "Endpoint URL for Grafana service"
  value       = "grafana.${local.monitoring_domain}"
}

output "grafana_admin_password" {
  description = "Admin password for Grafana"
  value       = random_password.grafana_admin_password.result
  sensitive   = true
}

output "grafana_backups_bucket" {
  description = "S3 bucket name for Grafana backups"
  value       = aws_s3_bucket.grafana_backups.bucket
}