# Terraform configuration for Jaeger distributed tracing
# Provides end-to-end tracing capabilities for the Document Management Platform

# Local variables for Jaeger configuration
locals {
  jaeger_config = {
    version              = "1.38.0"
    collector_replicas   = "${var.environment == "prod" ? 3 : var.environment == "staging" ? 2 : 1}"
    query_replicas       = "${var.environment == "prod" ? 2 : var.environment == "staging" ? 2 : 1}"
    sampling_rate        = 0.1  # 10% sampling rate as per requirements
    service_account_name = "jaeger"
    config_map_name      = "jaeger-config"
    secret_name          = "jaeger-elasticsearch-credentials"
  }
}

# Service account for Jaeger components
resource "kubernetes_service_account" "jaeger" {
  metadata {
    name      = local.jaeger_config.service_account_name
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "monitoring"
    }
  }
}

# ConfigMap containing Jaeger configuration
resource "kubernetes_config_map" "jaeger_config" {
  metadata {
    name      = local.jaeger_config.config_map_name
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "monitoring"
    }
  }

  data = {
    "sampling.json" = <<-EOT
{
  "default_strategy": {
    "type": "probabilistic",
    "param": ${local.jaeger_config.sampling_rate}
  },
  "service_strategies": [
    {
      "service": "document-service",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    },
    {
      "service": "storage-service",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    },
    {
      "service": "search-service",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    },
    {
      "service": "folder-service",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    },
    {
      "service": "virus-scanning-service",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    },
    {
      "service": "api-gateway",
      "type": "probabilistic",
      "param": ${local.jaeger_config.sampling_rate}
    }
  ]
}
EOT

    "ui-config.json" = <<-EOT
{
  "tracking": {
    "gaID": "",
    "trackErrors": true
  },
  "menu": [
    {
      "label": "Document Management Platform",
      "url": "/"
    }
  ],
  "dependencies": {
    "menuEnabled": true
  },
  "search": {
    "maxLookback": {
      "label": "2 Days",
      "value": "2d"
    }
  }
}
EOT
  }
}

# Secret containing Elasticsearch credentials for Jaeger
resource "kubernetes_secret" "jaeger_elasticsearch_credentials" {
  metadata {
    name      = local.jaeger_config.secret_name
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "monitoring"
    }
  }

  data = {
    username = base64encode(elasticsearch_credentials.username)
    password = base64encode(elasticsearch_credentials.password)
  }

  type = "Opaque"
}

# Deployment for Jaeger Collector component
resource "kubernetes_deployment" "jaeger_collector" {
  metadata {
    name      = "jaeger-collector"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "collector"
    }
  }

  spec {
    replicas = local.jaeger_config.collector_replicas

    selector {
      match_labels = {
        app       = "jaeger"
        component = "collector"
      }
    }

    template {
      metadata {
        labels = {
          app       = "jaeger"
          component = "collector"
        }
      }

      spec {
        service_account_name = local.jaeger_config.service_account_name

        containers {
          name  = "jaeger-collector"
          image = "jaegertracing/jaeger-collector:${local.jaeger_config.version}"

          ports {
            container_port = 14250
            name           = "grpc"
          }

          ports {
            container_port = 14268
            name           = "http"
          }

          ports {
            container_port = 9411
            name           = "zipkin"
          }

          env {
            name  = "SPAN_STORAGE_TYPE"
            value = "elasticsearch"
          }

          env {
            name  = "ES_SERVER_URLS"
            value = elasticsearch_endpoint
          }

          env {
            name = "ES_USERNAME"
            value_from {
              secret_key_ref {
                name = local.jaeger_config.secret_name
                key  = "username"
              }
            }
          }

          env {
            name = "ES_PASSWORD"
            value_from {
              secret_key_ref {
                name = local.jaeger_config.secret_name
                key  = "password"
              }
            }
          }

          env {
            name  = "COLLECTOR_ZIPKIN_HOST_PORT"
            value = ":9411"
          }

          resources {
            limits = {
              cpu    = "${var.environment == "prod" ? "1" : "500m"}"
              memory = "${var.environment == "prod" ? "1Gi" : "512Mi"}"
            }
            requests = {
              cpu    = "${var.environment == "prod" ? "500m" : "200m"}"
              memory = "${var.environment == "prod" ? "512Mi" : "256Mi"}"
            }
          }

          volume_mounts {
            name       = "jaeger-config-volume"
            mount_path = "/etc/jaeger"
            read_only  = true
          }

          liveness_probe {
            http_get {
              path = "/"
              port = 14268
            }
            initial_delay_seconds = 60
            period_seconds        = 30
          }

          readiness_probe {
            http_get {
              path = "/"
              port = 14268
            }
            initial_delay_seconds = 30
            period_seconds        = 10
          }
        }

        volumes {
          name = "jaeger-config-volume"
          config_map {
            name = local.jaeger_config.config_map_name
          }
        }
      }
    }
  }
}

# Deployment for Jaeger Query component
resource "kubernetes_deployment" "jaeger_query" {
  metadata {
    name      = "jaeger-query"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "query"
    }
  }

  spec {
    replicas = local.jaeger_config.query_replicas

    selector {
      match_labels = {
        app       = "jaeger"
        component = "query"
      }
    }

    template {
      metadata {
        labels = {
          app       = "jaeger"
          component = "query"
        }
      }

      spec {
        service_account_name = local.jaeger_config.service_account_name

        containers {
          name  = "jaeger-query"
          image = "jaegertracing/jaeger-query:${local.jaeger_config.version}"

          ports {
            container_port = 16686
            name           = "http"
          }

          env {
            name  = "SPAN_STORAGE_TYPE"
            value = "elasticsearch"
          }

          env {
            name  = "ES_SERVER_URLS"
            value = elasticsearch_endpoint
          }

          env {
            name = "ES_USERNAME"
            value_from {
              secret_key_ref {
                name = local.jaeger_config.secret_name
                key  = "username"
              }
            }
          }

          env {
            name = "ES_PASSWORD"
            value_from {
              secret_key_ref {
                name = local.jaeger_config.secret_name
                key  = "password"
              }
            }
          }

          env {
            name  = "QUERY_BASE_PATH"
            value = "/jaeger"
          }

          resources {
            limits = {
              cpu    = "${var.environment == "prod" ? "1" : "500m"}"
              memory = "${var.environment == "prod" ? "1Gi" : "512Mi"}"
            }
            requests = {
              cpu    = "${var.environment == "prod" ? "500m" : "200m"}"
              memory = "${var.environment == "prod" ? "512Mi" : "256Mi"}"
            }
          }

          volume_mounts {
            name       = "jaeger-config-volume"
            mount_path = "/etc/jaeger"
            read_only  = true
          }

          readiness_probe {
            http_get {
              path = "/"
              port = 16686
            }
            initial_delay_seconds = 30
            period_seconds        = 10
          }

          liveness_probe {
            http_get {
              path = "/"
              port = 16686
            }
            initial_delay_seconds = 60
            period_seconds        = 30
          }
        }

        volumes {
          name = "jaeger-config-volume"
          config_map {
            name = local.jaeger_config.config_map_name
          }
        }
      }
    }
  }
}

# DaemonSet for Jaeger Agent component
resource "kubernetes_daemon_set" "jaeger_agent" {
  metadata {
    name      = "jaeger-agent"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "agent"
    }
  }

  spec {
    selector {
      match_labels = {
        app       = "jaeger"
        component = "agent"
      }
    }

    template {
      metadata {
        labels = {
          app       = "jaeger"
          component = "agent"
        }
      }

      spec {
        service_account_name = local.jaeger_config.service_account_name

        containers {
          name  = "jaeger-agent"
          image = "jaegertracing/jaeger-agent:${local.jaeger_config.version}"

          ports {
            container_port = 5775
            protocol       = "UDP"
            name           = "zipkin-compact"
          }

          ports {
            container_port = 6831
            protocol       = "UDP"
            name           = "thrift-compact"
          }

          ports {
            container_port = 6832
            protocol       = "UDP"
            name           = "thrift-binary"
          }

          ports {
            container_port = 5778
            name           = "config-rest"
          }

          env {
            name  = "REPORTER_GRPC_HOST_PORT"
            value = "jaeger-collector:14250"
          }

          env {
            name  = "REPORTER_TYPE"
            value = "grpc"
          }

          args = [
            "--reporter.grpc.host-port=jaeger-collector:14250",
            "--sampling.strategies-file=/etc/jaeger/sampling.json"
          ]

          resources {
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }
          }

          volume_mounts {
            name       = "jaeger-config-volume"
            mount_path = "/etc/jaeger"
            read_only  = true
          }
        }

        volumes {
          name = "jaeger-config-volume"
          config_map {
            name = local.jaeger_config.config_map_name
          }
        }
      }
    }
  }
}

# Service for Jaeger Collector component
resource "kubernetes_service" "jaeger_collector" {
  metadata {
    name      = "jaeger-collector"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "collector"
    }
  }

  spec {
    selector = {
      app       = "jaeger"
      component = "collector"
    }

    ports {
      name        = "grpc"
      port        = 14250
      target_port = 14250
      protocol    = "TCP"
    }

    ports {
      name        = "http"
      port        = 14268
      target_port = 14268
      protocol    = "TCP"
    }

    ports {
      name        = "zipkin"
      port        = 9411
      target_port = 9411
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

# Service for Jaeger Query component
resource "kubernetes_service" "jaeger_query" {
  metadata {
    name      = "jaeger-query"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "query"
    }
  }

  spec {
    selector = {
      app       = "jaeger"
      component = "query"
    }

    ports {
      name        = "http"
      port        = 16686
      target_port = 16686
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

# Service for Jaeger Agent component
resource "kubernetes_service" "jaeger_agent" {
  metadata {
    name      = "jaeger-agent"
    namespace = var.monitoring_namespace
    labels = {
      app       = "jaeger"
      component = "agent"
    }
  }

  spec {
    selector = {
      app       = "jaeger"
      component = "agent"
    }

    ports {
      name        = "zipkin-compact"
      port        = 5775
      target_port = 5775
      protocol    = "UDP"
    }

    ports {
      name        = "thrift-compact"
      port        = 6831
      target_port = 6831
      protocol    = "UDP"
    }

    ports {
      name        = "thrift-binary"
      port        = 6832
      target_port = 6832
      protocol    = "UDP"
    }

    ports {
      name        = "config-rest"
      port        = 5778
      target_port = 5778
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

# Ingress for Jaeger Query UI
resource "kubernetes_ingress" "jaeger_query_ingress" {
  metadata {
    name      = "jaeger-query-ingress"
    namespace = var.monitoring_namespace
    annotations = {
      "kubernetes.io/ingress.class"                = "nginx"
      "nginx.ingress.kubernetes.io/rewrite-target" = "/$1"
      "nginx.ingress.kubernetes.io/ssl-redirect"   = "true"
    }
  }

  spec {
    rule {
      host = local.monitoring_domain
      http {
        path {
          path      = "/jaeger/(.*)"
          path_type = "Prefix"
          backend {
            service {
              name = "jaeger-query"
              port {
                number = 16686
              }
            }
          }
        }
      }
    }

    tls {
      hosts       = [local.monitoring_domain]
      secret_name = "monitoring-tls"
    }
  }
}

# Variables
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

# Outputs - exposing endpoints for other components to use
output "jaeger_collector_endpoint" {
  description = "Endpoint URL for the Jaeger collector service"
  value       = "jaeger-collector.${var.monitoring_namespace}.svc.cluster.local:14250"
}

output "jaeger_query_endpoint" {
  description = "Endpoint URL for the Jaeger query service"
  value       = "jaeger-query.${var.monitoring_namespace}.svc.cluster.local:16686"
}

output "jaeger_agent_endpoint" {
  description = "Endpoint URL for the Jaeger agent service"
  value       = "jaeger-agent.${var.monitoring_namespace}.svc.cluster.local:6831"
}

output "jaeger_ui_url" {
  description = "URL for accessing the Jaeger UI"
  value       = "https://${local.monitoring_domain}/jaeger"
}