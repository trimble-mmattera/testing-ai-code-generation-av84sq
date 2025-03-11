# Terraform configuration for provisioning and configuring Kibana resources
# for the Document Management Platform's monitoring infrastructure

# Generate a random encryption key for Kibana
resource "random_password" "kibana_encryption_key" {
  length  = 32
  special = true
}

# Local variables for Kibana configuration
locals {
  kibana_config = {
    chart_version  = "8.6.0"
    replica_count  = "${var.environment == "prod" ? 2 : 1}"
    service_name   = "kibana"
    config_map_name = "kibana-config"
    secret_name    = "kibana-secrets"
    storage_size   = "${var.environment == "prod" ? "20Gi" : "10Gi"}"
  }
}

# ConfigMap containing Kibana configuration
resource "kubernetes_config_map" "kibana_config" {
  metadata {
    name      = "${local.kibana_config.config_map_name}"
    namespace = "${var.monitoring_namespace}"
  }

  data = {
    "kibana.yml" = "server.name: document-mgmt-kibana\nserver.host: 0.0.0.0\nserver.port: 5601\nserver.publicBaseUrl: ${KIBANA_PUBLIC_URL}\nserver.maxPayloadBytes: 10485760\nserver.rewriteBasePath: true\n\nelasticsearch.hosts: [\"${elasticsearch_endpoint}\"] \nelasticsearch.username: ${ELASTICSEARCH_USERNAME}\nelasticsearch.password: ${ELASTICSEARCH_PASSWORD}\nelasticsearch.requestTimeout: 30000\nelasticsearch.shardTimeout: 30000\nelasticsearch.ssl.verificationMode: certificate\n\nkibana.index: .kibana\nkibana.defaultAppId: discover\n\nxpack.security.enabled: true\nxpack.security.encryptionKey: ${ENCRYPTION_KEY}\nxpack.security.session.idleTimeout: 1h\nxpack.security.session.lifespan: 24h\nxpack.security.audit.enabled: true\n\nxpack.reporting.enabled: true\nxpack.reporting.kibanaServer.hostname: localhost\nxpack.reporting.capture.timeout: 30s\nxpack.reporting.csv.maxSizeBytes: 10485760\n\nxpack.monitoring.enabled: true\nxpack.monitoring.kibana.collection.enabled: true\nxpack.monitoring.ui.container.elasticsearch.enabled: true\n\nlogging.root.level: info\nlogging.appenders.file.type: file\nlogging.appenders.file.fileName: /var/log/kibana/kibana.log\nlogging.appenders.file.layout.type: json\n\nsavedObjects.maxImportPayloadBytes: 26214400\nsavedObjects.maxImportExportSize: 10000\n\ntelemetry.enabled: false\nmap.includeElasticMapsService: false\n\nxpack.spaces.enabled: true\nxpack.spaces.maxSpaces: 50\n\nxpack.alerting.enabled: true\nxpack.actions.enabled: true\nxpack.actions.preconfiguredAlertHistoryEsIndex: true"
  }
}

# ConfigMap containing Kibana dashboards
resource "kubernetes_config_map" "kibana_dashboards" {
  metadata {
    name      = "kibana-dashboards"
    namespace = "${var.monitoring_namespace}"
  }

  data = {
    "logs-dashboard.json" = "{\n  \"version\": \"8.6.0\",\n  \"objects\": [\n    {\n      \"id\": \"document-mgmt-logs\",\n      \"type\": \"dashboard\",\n      \"attributes\": {\n        \"title\": \"Document Management Platform Logs\",\n        \"description\": \"Overview of logs from the Document Management Platform\",\n        \"hits\": 0,\n        \"timeRestore\": true,\n        \"timeFrom\": \"now-24h\",\n        \"timeTo\": \"now\",\n        \"refreshInterval\": {\n          \"pause\": false,\n          \"value\": 60000\n        },\n        \"panels\": [\n          {\n            \"id\": \"log-volume-panel\",\n            \"type\": \"visualization\",\n            \"panelIndex\": 1,\n            \"gridData\": {\n              \"x\": 0,\n              \"y\": 0,\n              \"w\": 24,\n              \"h\": 8,\n              \"i\": \"1\"\n            }\n          },\n          {\n            \"id\": \"error-logs-panel\",\n            \"type\": \"visualization\",\n            \"panelIndex\": 2,\n            \"gridData\": {\n              \"x\": 0,\n              \"y\": 8,\n              \"w\": 24,\n              \"h\": 8,\n              \"i\": \"2\"\n            }\n          },\n          {\n            \"id\": \"service-logs-panel\",\n            \"type\": \"visualization\",\n            \"panelIndex\": 3,\n            \"gridData\": {\n              \"x\": 0,\n              \"y\": 16,\n              \"w\": 24,\n              \"h\": 16,\n              \"i\": \"3\"\n            }\n          }\n        ]\n      }\n    },\n    {\n      \"id\": \"log-volume-panel\",\n      \"type\": \"visualization\",\n      \"attributes\": {\n        \"title\": \"Log Volume Over Time\",\n        \"visState\": \"{\\\"type\\\":\\\"histogram\\\",\\\"aggs\\\":[{\\\"id\\\":\\\"1\\\",\\\"enabled\\\":true,\\\"type\\\":\\\"count\\\",\\\"schema\\\":\\\"metric\\\",\\\"params\\\":{}},{\\\"id\\\":\\\"2\\\",\\\"enabled\\\":true,\\\"type\\\":\\\"date_histogram\\\",\\\"schema\\\":\\\"segment\\\",\\\"params\\\":{\\\"field\\\":\\\"@timestamp\\\",\\\"timeRange\\\":{\\\"from\\\":\\\"now-24h\\\",\\\"to\\\":\\\"now\\\"},\\\"useNormalizedEsInterval\\\":true,\\\"interval\\\":\\\"auto\\\",\\\"drop_partials\\\":false,\\\"min_doc_count\\\":1,\\\"extended_bounds\\\":{}}},{\\\"id\\\":\\\"3\\\",\\\"enabled\\\":true,\\\"type\\\":\\\"terms\\\",\\\"schema\\\":\\\"group\\\",\\\"params\\\":{\\\"field\\\":\\\"level\\\",\\\"size\\\":5,\\\"order\\\":\\\"desc\\\",\\\"orderBy\\\":\\\"1\\\"}}]}\"\n      }\n    },\n    {\n      \"id\": \"error-logs-panel\",\n      \"type\": \"visualization\",\n      \"attributes\": {\n        \"title\": \"Error Logs by Service\",\n        \"visState\": \"{\\\"type\\\":\\\"pie\\\",\\\"aggs\\\":[{\\\"id\\\":\\\"1\\\",\\\"enabled\\\":true,\\\"type\\\":\\\"count\\\",\\\"schema\\\":\\\"metric\\\",\\\"params\\\":{}},{\\\"id\\\":\\\"2\\\",\\\"enabled\\\":true,\\\"type\\\":\\\"terms\\\",\\\"schema\\\":\\\"segment\\\",\\\"params\\\":{\\\"field\\\":\\\"service\\\",\\\"size\\\":10,\\\"order\\\":\\\"desc\\\",\\\"orderBy\\\":\\\"1\\\"}}],\\\"params\\\":{\\\"type\\\":\\\"pie\\\",\\\"addTooltip\\\":true,\\\"addLegend\\\":true,\\\"legendPosition\\\":\\\"right\\\",\\\"isDonut\\\":true}}\"\n      }\n    },\n    {\n      \"id\": \"service-logs-panel\",\n      \"type\": \"search\",\n      \"attributes\": {\n        \"title\": \"Service Logs\",\n        \"columns\": [\"@timestamp\", \"service\", \"level\", \"message\"],\n        \"sort\": [[\"@timestamp\", \"desc\"]],\n        \"kibanaSavedObjectMeta\": {\n          \"searchSourceJSON\": \"{\\\"query\\\":{\\\"query_string\\\":{\\\"query\\\":\\\"*\\\",\\\"analyze_wildcard\\\":true}},\\\"filter\\\":[],\\\"highlightAll\\\":true,\\\"version\\\":true}\"\n        }\n      }\n    }\n  ]\n}"
  }
}

# Secret containing Kibana encryption key
resource "kubernetes_secret" "kibana_secrets" {
  metadata {
    name      = "${local.kibana_config.secret_name}"
    namespace = "${var.monitoring_namespace}"
  }

  data = {
    "encryption-key" = "${base64encode(random_password.kibana_encryption_key.result)}"
  }

  type = "Opaque"
}

# PVC for Kibana data storage
resource "kubernetes_persistent_volume_claim" "kibana_data" {
  metadata {
    name      = "kibana-data"
    namespace = "${var.monitoring_namespace}"
  }

  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "${local.kibana_config.storage_size}"
      }
    }
    storage_class_name = "gp2"
  }
}

# Service account for Kibana
resource "kubernetes_service_account" "kibana" {
  metadata {
    name      = "kibana"
    namespace = "${var.monitoring_namespace}"
  }
}

# Service for Kibana
resource "kubernetes_service" "kibana" {
  metadata {
    name      = "${local.kibana_config.service_name}"
    namespace = "${var.monitoring_namespace}"
    labels = {
      app       = "kibana"
      component = "monitoring"
    }
    annotations = {
      "prometheus.io/scrape" = "true"
      "prometheus.io/port"   = "5601"
      "prometheus.io/path"   = "/metrics"
    }
  }

  spec {
    selector = {
      app = "kibana"
    }
    port {
      name        = "http"
      port        = 5601
      protocol    = "TCP"
      target_port = 5601
    }
    type = "ClusterIP"
  }
}

# Deployment for Kibana
resource "kubernetes_deployment" "kibana" {
  metadata {
    name      = "kibana"
    namespace = "${var.monitoring_namespace}"
    labels = {
      app       = "kibana"
      component = "monitoring"
    }
  }

  spec {
    replicas = "${local.kibana_config.replica_count}"
    
    selector {
      match_labels = {
        app = "kibana"
      }
    }
    
    strategy {
      type = "RollingUpdate"
      rolling_update {
        max_surge       = 1
        max_unavailable = 0
      }
    }
    
    template {
      metadata {
        labels = {
          app       = "kibana"
          component = "monitoring"
        }
        annotations = {
          "prometheus.io/scrape" = "true"
          "prometheus.io/port"   = "5601"
        }
      }
      
      spec {
        service_account_name = "kibana"
        
        security_context {
          fs_group    = 1000
          run_as_user = 1000
        }
        
        container {
          name  = "kibana"
          image = "docker.elastic.co/kibana/kibana:${local.kibana_config.chart_version}"
          
          resources {
            limits = {
              cpu    = "${var.environment == "prod" ? "1" : "0.5"}"
              memory = "${var.environment == "prod" ? "2Gi" : "1Gi"}"
            }
            requests = {
              cpu    = "${var.environment == "prod" ? "500m" : "200m"}"
              memory = "${var.environment == "prod" ? "1Gi" : "512Mi"}"
            }
          }
          
          port {
            name           = "http"
            container_port = 5601
            protocol       = "TCP"
          }
          
          env {
            name  = "ELASTICSEARCH_HOSTS"
            value = "${elasticsearch_endpoint}"
          }
          
          env {
            name = "ELASTICSEARCH_USERNAME"
            value_from {
              secret_key_ref {
                name = "elasticsearch-credentials"
                key  = "username"
              }
            }
          }
          
          env {
            name = "ELASTICSEARCH_PASSWORD"
            value_from {
              secret_key_ref {
                name = "elasticsearch-credentials"
                key  = "password"
              }
            }
          }
          
          env {
            name  = "KIBANA_PUBLIC_URL"
            value = "https://kibana.${local.monitoring_domain}"
          }
          
          env {
            name = "ENCRYPTION_KEY"
            value_from {
              secret_key_ref {
                name = "${local.kibana_config.secret_name}"
                key  = "encryption-key"
              }
            }
          }
          
          env {
            name  = "NODE_OPTIONS"
            value = "--max-old-space-size=${var.environment == "prod" ? "1536" : "1024"}"
          }
          
          volume_mount {
            name       = "kibana-config"
            mount_path = "/usr/share/kibana/config/kibana.yml"
            sub_path   = "kibana.yml"
          }
          
          volume_mount {
            name       = "kibana-data"
            mount_path = "/usr/share/kibana/data"
          }
          
          volume_mount {
            name       = "kibana-dashboards"
            mount_path = "/usr/share/kibana/dashboards"
          }
          
          readiness_probe {
            http_get {
              path   = "/api/status"
              port   = 5601
              scheme = "HTTP"
            }
            initial_delay_seconds = 60
            timeout_seconds       = 30
            period_seconds        = 10
            success_threshold     = 1
            failure_threshold     = 3
          }
          
          liveness_probe {
            http_get {
              path   = "/api/status"
              port   = 5601
              scheme = "HTTP"
            }
            initial_delay_seconds = 120
            timeout_seconds       = 30
            period_seconds        = 30
            success_threshold     = 1
            failure_threshold     = 3
          }
        }
        
        volume {
          name = "kibana-config"
          config_map {
            name = "${local.kibana_config.config_map_name}"
          }
        }
        
        volume {
          name = "kibana-data"
          persistent_volume_claim {
            claim_name = "kibana-data"
          }
        }
        
        volume {
          name = "kibana-dashboards"
          config_map {
            name = "kibana-dashboards"
          }
        }
        
        affinity {
          pod_anti_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              pod_affinity_term {
                label_selector {
                  match_expressions {
                    key      = "app"
                    operator = "In"
                    values   = ["kibana"]
                  }
                }
                topology_key = "kubernetes.io/hostname"
              }
            }
          }
        }
        
        termination_grace_period_seconds = 60
      }
    }
  }
}

# Output the Kibana service name and endpoint for other components to use
output "kibana_service_name" {
  description = "Service name for Kibana"
  value       = "${local.kibana_config.service_name}"
}

output "kibana_endpoint" {
  description = "Endpoint URL for the Kibana service"
  value       = "http://${local.kibana_config.service_name}.${var.monitoring_namespace}.svc.cluster.local:5601"
}

output "kibana_url" {
  description = "Public URL for accessing Kibana"
  value       = "https://kibana.${local.monitoring_domain}"
}