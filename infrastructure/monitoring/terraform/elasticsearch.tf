# Terraform configuration for Elasticsearch deployment for monitoring infrastructure
# Provides the logging storage backend for the Document Management Platform

# Random password for Elasticsearch
resource "random_password" "elasticsearch_password" {
  length  = 16
  special = false
}

# Local variables for Elasticsearch configuration
locals {
  elasticsearch_config = {
    chart_version         = "8.6.0"
    replica_count         = var.environment == "prod" ? 3 : var.environment == "staging" ? 2 : 1
    heap_size             = var.environment == "prod" ? "2g" : var.environment == "staging" ? "1g" : "512m"
    storage_size          = var.environment == "prod" ? "100Gi" : var.environment == "staging" ? "50Gi" : "20Gi"
    service_name          = "elasticsearch"
    headless_service_name = "elasticsearch-headless"
    config_map_name       = "elasticsearch-config"
    secret_name           = "elasticsearch-credentials"
    storage_class_name    = "elasticsearch-storage"
  }
}

# ConfigMap containing Elasticsearch configuration
resource "kubernetes_config_map" "elasticsearch_config" {
  metadata {
    name      = local.elasticsearch_config.config_map_name
    namespace = var.monitoring_namespace
  }

  data = {
    "elasticsearch.yml" = <<-EOT
cluster.name: document-mgmt-es-cluster
cluster.initial_master_nodes: ["elasticsearch-0", "elasticsearch-1", "elasticsearch-2"]
node.name: ${HOSTNAME}
node.master: true
node.data: true
node.ingest: true
path.data: /usr/share/elasticsearch/data
path.logs: /usr/share/elasticsearch/logs
network.host: 0.0.0.0
http.port: 9200
transport.port: 9300
http.cors.enabled: true
http.cors.allow-origin: "*"
discovery.seed_hosts: elasticsearch-headless.${var.monitoring_namespace}.svc.cluster.local
discovery.zen.minimum_master_nodes: 2
gateway.recover_after_nodes: 2
gateway.expected_nodes: 3
gateway.recover_after_time: 5m
action.destructive_requires_name: true
xpack.security.enabled: true
xpack.security.transport.ssl.enabled: true
xpack.security.transport.ssl.verification_mode: certificate
xpack.monitoring.collection.enabled: true
indices.recovery.max_bytes_per_sec: 50mb
indices.fielddata.cache.size: 20%
indices.memory.index_buffer_size: 10%
indices.query.bool.max_clause_count: 1024
thread_pool.search.size: 5
thread_pool.search.queue_size: 1000
thread_pool.write.size: 5
thread_pool.write.queue_size: 1000
bootstrap.memory_lock: true
EOT

    "index-templates.json" = <<-EOT
{
  "index_patterns": ["logs-*"],
  "template": {
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": ${local.elasticsearch_config.replica_count - 1},
      "index.refresh_interval": "5s"
    },
    "mappings": {
      "properties": {
        "@timestamp": { "type": "date" },
        "tenant_id": { "type": "keyword" },
        "level": { "type": "keyword" },
        "service": { "type": "keyword" },
        "message": { "type": "text" },
        "trace_id": { "type": "keyword" },
        "span_id": { "type": "keyword" }
      }
    }
  }
}
EOT
  }
}

# Secret containing Elasticsearch credentials
resource "kubernetes_secret" "elasticsearch_credentials" {
  metadata {
    name      = local.elasticsearch_config.secret_name
    namespace = var.monitoring_namespace
  }

  data = {
    username = "ZWxhc3RpYw==" # elastic in base64
    password = base64encode(random_password.elasticsearch_password.result)
  }

  type = "Opaque"
}

# Storage class for Elasticsearch persistent volumes
resource "kubernetes_storage_class" "elasticsearch_storage" {
  metadata {
    name = local.elasticsearch_config.storage_class_name
  }

  storage_provisioner    = "kubernetes.io/aws-ebs"
  reclaim_policy         = "Retain"
  parameters = {
    type   = "gp3"
    fsType = "ext4"
  }
  volume_binding_mode    = "WaitForFirstConsumer"
  allowed_topologies {
    match_label_expressions {
      key    = "failure-domain.beta.kubernetes.io/zone"
      values = var.availability_zones
    }
  }
}

# Service account for Elasticsearch
resource "kubernetes_service_account" "elasticsearch" {
  metadata {
    name      = "elasticsearch"
    namespace = var.monitoring_namespace
  }
}

# Headless service for Elasticsearch cluster communication
resource "kubernetes_service" "elasticsearch_headless" {
  metadata {
    name      = local.elasticsearch_config.headless_service_name
    namespace = var.monitoring_namespace
    labels = {
      app       = "elasticsearch"
      component = "monitoring"
    }
  }

  spec {
    selector = {
      app = "elasticsearch"
    }
    cluster_ip = "None"
    port {
      name        = "transport"
      port        = 9300
      protocol    = "TCP"
      target_port = 9300
    }
  }
}

# Service for Elasticsearch API access
resource "kubernetes_service" "elasticsearch" {
  metadata {
    name      = local.elasticsearch_config.service_name
    namespace = var.monitoring_namespace
    labels = {
      app       = "elasticsearch"
      component = "monitoring"
    }
  }

  spec {
    selector = {
      app = "elasticsearch"
    }
    port {
      name        = "http"
      port        = 9200
      protocol    = "TCP"
      target_port = 9200
    }
    port {
      name        = "transport"
      port        = 9300
      protocol    = "TCP"
      target_port = 9300
    }
    type = "ClusterIP"
  }
}

# StatefulSet for Elasticsearch cluster
resource "kubernetes_stateful_set" "elasticsearch" {
  metadata {
    name      = "elasticsearch"
    namespace = var.monitoring_namespace
    labels = {
      app       = "elasticsearch"
      component = "monitoring"
    }
  }

  spec {
    replicas     = local.elasticsearch_config.replica_count
    service_name = local.elasticsearch_config.headless_service_name

    selector {
      match_labels = {
        app = "elasticsearch"
      }
    }

    update_strategy {
      type = "RollingUpdate"
    }

    pod_management_policy = "Parallel"

    template {
      metadata {
        labels = {
          app       = "elasticsearch"
          component = "monitoring"
        }
      }

      spec {
        service_account_name = "elasticsearch"
        
        security_context {
          fs_group    = 1000
          run_as_user = 1000
        }

        # Init containers to prepare for Elasticsearch
        init_container {
          name  = "fix-permissions"
          image = "busybox:1.35.0"
          command = ["sh", "-c", "chown -R 1000:1000 /usr/share/elasticsearch/data"]
          security_context {
            run_as_user = 0
          }
          volume_mount {
            name       = "elasticsearch-data"
            mount_path = "/usr/share/elasticsearch/data"
          }
        }

        init_container {
          name  = "increase-vm-max-map"
          image = "busybox:1.35.0"
          command = ["sysctl", "-w", "vm.max_map_count=262144"]
          security_context {
            privileged = true
          }
        }

        init_container {
          name  = "increase-fd-ulimit"
          image = "busybox:1.35.0"
          command = ["sh", "-c", "ulimit -n 65536"]
          security_context {
            privileged = true
          }
        }

        # Main Elasticsearch container
        container {
          name  = "elasticsearch"
          image = "docker.elastic.co/elasticsearch/elasticsearch:${local.elasticsearch_config.chart_version}"

          resources {
            limits = {
              cpu    = var.environment == "prod" ? "2" : var.environment == "staging" ? "1" : "0.5"
              memory = var.environment == "prod" ? "4Gi" : var.environment == "staging" ? "2Gi" : "1Gi"
            }
            requests = {
              cpu    = var.environment == "prod" ? "1" : var.environment == "staging" ? "0.5" : "0.2"
              memory = var.environment == "prod" ? "2Gi" : var.environment == "staging" ? "1Gi" : "512Mi"
            }
          }

          port {
            name           = "http"
            container_port = 9200
            protocol       = "TCP"
          }

          port {
            name           = "transport"
            container_port = 9300
            protocol       = "TCP"
          }

          env {
            name = "NAMESPACE"
            value_from {
              field_ref {
                field_path = "metadata.namespace"
              }
            }
          }

          env {
            name = "HOSTNAME"
            value_from {
              field_ref {
                field_path = "metadata.name"
              }
            }
          }

          env {
            name  = "CLUSTER_NAME"
            value = "document-mgmt-es-cluster"
          }

          env {
            name  = "ES_JAVA_OPTS"
            value = "-Xms${local.elasticsearch_config.heap_size} -Xmx${local.elasticsearch_config.heap_size}"
          }

          env {
            name = "ELASTIC_PASSWORD"
            value_from {
              secret_key_ref {
                name = local.elasticsearch_config.secret_name
                key  = "password"
              }
            }
          }

          volume_mount {
            name       = "elasticsearch-data"
            mount_path = "/usr/share/elasticsearch/data"
          }

          volume_mount {
            name       = "elasticsearch-config"
            mount_path = "/usr/share/elasticsearch/config/elasticsearch.yml"
            sub_path   = "elasticsearch.yml"
          }

          volume_mount {
            name       = "elasticsearch-config"
            mount_path = "/usr/share/elasticsearch/config/index-templates.json"
            sub_path   = "index-templates.json"
          }

          readiness_probe {
            http_get {
              path   = "/_cluster/health"
              port   = 9200
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
              path   = "/_cluster/health?local=true"
              port   = 9200
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
          name = "elasticsearch-config"
          config_map {
            name = local.elasticsearch_config.config_map_name
          }
        }

        # Pod anti-affinity to spread across nodes
        affinity {
          pod_anti_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              pod_affinity_term {
                label_selector {
                  match_expressions {
                    key      = "app"
                    operator = "In"
                    values   = ["elasticsearch"]
                  }
                }
                topology_key = "kubernetes.io/hostname"
              }
            }
          }
        }

        termination_grace_period_seconds = 120
      }
    }

    # PVC template for Elasticsearch data
    volume_claim_template {
      metadata {
        name = "elasticsearch-data"
      }
      spec {
        access_modes       = ["ReadWriteOnce"]
        storage_class_name = local.elasticsearch_config.storage_class_name
        resources {
          requests = {
            storage = local.elasticsearch_config.storage_size
          }
        }
      }
    }
  }
}

# Output the Elasticsearch service name and endpoint for other components to use
output "elasticsearch_service_name" {
  description = "Service name for Elasticsearch"
  value       = local.elasticsearch_config.service_name
}

output "elasticsearch_endpoint" {
  description = "Endpoint URL for the Elasticsearch service"
  value       = "http://${local.elasticsearch_config.service_name}.${var.monitoring_namespace}.svc.cluster.local:9200"
}

output "elasticsearch_credentials" {
  description = "Credentials for accessing Elasticsearch"
  value = {
    username = "elastic"
    password = random_password.elasticsearch_password.result
  }
  sensitive = true
}