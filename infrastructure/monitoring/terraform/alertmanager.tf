# AlertManager Terraform Configuration
# Provisions and configures AlertManager resources for the Document Management Platform

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
    random = {
      source  = "hashicorp/random"
      version = "~> 3.4"
    }
  }
}

# Local variables for AlertManager configuration
locals {
  alertmanager_config = {
    namespace           = var.monitoring_namespace
    service_account_name = "alertmanager"
    storage_class       = "gp2"
    storage_size        = "10Gi"
    version             = "0.25.0"
    replicas            = 1
  }
}

# AWS Secrets Manager secret for AlertManager credentials
resource "aws_secretsmanager_secret" "alertmanager_credentials" {
  name        = "${var.project_name}-${var.environment}-alertmanager-credentials"
  description = "Credentials for AlertManager notification integrations"
  
  tags = {
    Name        = "${var.project_name}-${var.environment}-alertmanager-credentials"
    Environment = var.environment
    Project     = var.project_name
    Component   = "Monitoring"
  }
}

# Generate a random password for AlertManager SMTP authentication
resource "random_password" "alertmanager_smtp_password" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
  min_special      = 2
  min_upper        = 2
  min_lower        = 2
  min_numeric      = 2
}

# Store credentials in AWS Secrets Manager
resource "aws_secretsmanager_secret_version" "alertmanager_credentials" {
  secret_id = aws_secretsmanager_secret.alertmanager_credentials.id
  secret_string = jsonencode({
    SLACK_API_URL = var.slack_webhook_url
    PAGERDUTY_CRITICAL_KEY = var.pagerduty_critical_key
    PAGERDUTY_HIGH_KEY = var.pagerduty_high_key
    PAGERDUTY_SECURITY_KEY = var.pagerduty_security_key
    PAGERDUTY_DATABASE_KEY = var.pagerduty_database_key
    SMTP_PASSWORD = random_password.alertmanager_smtp_password.result
  })
}

# IAM role for AlertManager to access AWS resources
resource "aws_iam_role" "alertmanager_role" {
  name = "${var.project_name}-${var.environment}-alertmanager-role"
  
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
            "${aws_iam_openid_connect_provider.eks_oidc.url}:sub": "system:serviceaccount:${local.alertmanager_config.namespace}:${local.alertmanager_config.service_account_name}"
          }
        }
      }
    ]
  })
}

# IAM policy for AlertManager to access AWS resources
resource "aws_iam_policy" "alertmanager_policy" {
  name        = "${var.project_name}-${var.environment}-alertmanager-policy"
  description = "Policy for AlertManager to access AWS resources"
  
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          "${aws_secretsmanager_secret.alertmanager_credentials.arn}"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = "*"
      }
    ]
  })
}

# Attach the policy to the role
resource "aws_iam_role_policy_attachment" "alertmanager_policy_attachment" {
  role       = aws_iam_role.alertmanager_role.name
  policy_arn = aws_iam_policy.alertmanager_policy.arn
}

# Kubernetes service account for AlertManager
resource "kubernetes_service_account" "alertmanager" {
  metadata {
    name      = local.alertmanager_config.service_account_name
    namespace = local.alertmanager_config.namespace
    annotations = {
      "eks.amazonaws.com/role-arn" = aws_iam_role.alertmanager_role.arn
    }
  }
}

# Kubernetes secret for AlertManager credentials
resource "kubernetes_secret" "alertmanager_credentials" {
  metadata {
    name      = "alertmanager-credentials"
    namespace = local.alertmanager_config.namespace
  }
  
  data = {
    SLACK_API_URL = var.slack_webhook_url
    PAGERDUTY_CRITICAL_KEY = var.pagerduty_critical_key
    PAGERDUTY_HIGH_KEY = var.pagerduty_high_key
    PAGERDUTY_SECURITY_KEY = var.pagerduty_security_key
    PAGERDUTY_DATABASE_KEY = var.pagerduty_database_key
    SMTP_PASSWORD = random_password.alertmanager_smtp_password.result
  }
  
  type = "Opaque"
}

# ConfigMap for AlertManager configuration
resource "kubernetes_config_map" "alertmanager_config" {
  metadata {
    name      = "alertmanager-config"
    namespace = local.alertmanager_config.namespace
  }
  
  data = {
    "alertmanager.yml" = file("${path.module}/../alertmanager/alertmanager.yml")
    "default.tmpl"     = file("${path.module}/../alertmanager/templates/default.tmpl")
  }
}

# PersistentVolumeClaim for AlertManager data
resource "kubernetes_persistent_volume_claim" "alertmanager_storage" {
  metadata {
    name      = "alertmanager-storage"
    namespace = local.alertmanager_config.namespace
  }
  
  spec {
    access_modes = ["ReadWriteOnce"]
    storage_class_name = local.alertmanager_config.storage_class
    
    resources {
      requests = {
        storage = local.alertmanager_config.storage_size
      }
    }
  }
}

# AlertManager deployment
resource "kubernetes_deployment" "alertmanager" {
  metadata {
    name      = "alertmanager"
    namespace = local.alertmanager_config.namespace
    labels = {
      app       = "alertmanager"
      component = "monitoring"
    }
  }
  
  spec {
    replicas = local.alertmanager_config.replicas
    
    selector {
      match_labels = {
        app = "alertmanager"
      }
    }
    
    strategy {
      type = "Recreate"
    }
    
    template {
      metadata {
        labels = {
          app = "alertmanager"
        }
        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/port"   = "9093"
        }
      }
      
      spec {
        service_account_name = local.alertmanager_config.service_account_name
        
        security_context {
          fs_group        = 65534
          run_as_non_root = true
          run_as_user     = 65534
        }
        
        container {
          name  = "alertmanager"
          image = "prom/alertmanager:v${local.alertmanager_config.version}"
          
          args = [
            "--config.file=/etc/alertmanager/alertmanager.yml",
            "--storage.path=/alertmanager",
            "--web.external-url=http://alertmanager.${local.alertmanager_config.namespace}.svc:9093",
            "--web.route-prefix=/",
            "--cluster.listen-address=0.0.0.0:9094"
          ]
          
          port {
            container_port = 9093
            name           = "http"
            protocol       = "TCP"
          }
          
          port {
            container_port = 9094
            name           = "cluster"
            protocol       = "TCP"
          }
          
          resources {
            limits = {
              cpu    = "200m"
              memory = "256Mi"
            }
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }
          }
          
          volume_mount {
            name       = "config-volume"
            mount_path = "/etc/alertmanager"
          }
          
          volume_mount {
            name       = "templates-volume"
            mount_path = "/etc/alertmanager/templates"
          }
          
          volume_mount {
            name       = "storage-volume"
            mount_path = "/alertmanager"
          }
          
          volume_mount {
            name       = "alertmanager-secrets"
            mount_path = "/etc/alertmanager/secrets"
            read_only  = true
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
          name = "config-volume"
          config_map {
            name = "alertmanager-config"
            items {
              key  = "alertmanager.yml"
              path = "alertmanager.yml"
            }
          }
        }
        
        volume {
          name = "templates-volume"
          config_map {
            name = "alertmanager-config"
            items {
              key  = "default.tmpl"
              path = "default.tmpl"
            }
          }
        }
        
        volume {
          name = "storage-volume"
          persistent_volume_claim {
            claim_name = "alertmanager-storage"
          }
        }
        
        volume {
          name = "alertmanager-secrets"
          secret {
            secret_name = "alertmanager-credentials"
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

# AlertManager service
resource "kubernetes_service" "alertmanager" {
  metadata {
    name      = "alertmanager"
    namespace = local.alertmanager_config.namespace
    labels = {
      app       = "alertmanager"
      component = "monitoring"
    }
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "9093"
    }
  }
  
  spec {
    selector = {
      app = "alertmanager"
    }
    
    port {
      port        = 9093
      target_port = 9093
      protocol    = "TCP"
      name        = "http"
    }
    
    port {
      port        = 9094
      target_port = 9094
      protocol    = "TCP"
      name        = "cluster"
    }
    
    type = "ClusterIP"
  }
}

# Variables
variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "document-mgmt"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "monitoring_namespace" {
  description = "Kubernetes namespace for monitoring resources"
  type        = string
  default     = "monitoring"
}

variable "slack_webhook_url" {
  description = "Slack webhook URL for AlertManager notifications"
  type        = string
  default     = ""
  sensitive   = true
}

variable "pagerduty_critical_key" {
  description = "PagerDuty service key for critical alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "pagerduty_high_key" {
  description = "PagerDuty service key for high severity alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "pagerduty_security_key" {
  description = "PagerDuty service key for security alerts"
  type        = string
  default     = ""
  sensitive   = true
}

variable "pagerduty_database_key" {
  description = "PagerDuty service key for database alerts"
  type        = string
  default     = ""
  sensitive   = true
}

# Outputs
output "alertmanager_endpoint" {
  description = "Endpoint URL for AlertManager service"
  value       = "${kubernetes_service.alertmanager.metadata[0].name}.${kubernetes_service.alertmanager.metadata[0].namespace}.svc.cluster.local:9093"
}

output "alertmanager_service_account" {
  description = "Service account used by AlertManager"
  value       = kubernetes_service_account.alertmanager.metadata[0].name
}