data "google_project" "current" {}

locals {
  services = toset([
    "artifactregistry.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "pubsub.googleapis.com",
    "cloudscheduler.googleapis.com",
    "secretmanager.googleapis.com",
    "iam.googleapis.com",
  ])
}

resource "google_project_service" "enabled" {
  for_each                   = local.services
  project                    = var.project_id
  service                    = each.value
  disable_on_destroy         = false
  disable_dependent_services = false
}

resource "google_artifact_registry_repository" "containers" {
  project       = var.project_id
  location      = var.region
  repository_id = "flashcart"
  description   = "Container images for FlashCart."
  format        = "DOCKER"
  depends_on    = [google_project_service.enabled]
}

resource "google_sql_database_instance" "postgres" {
  project             = var.project_id
  name                = "flashcart-postgres"
  region              = var.region
  database_version    = "POSTGRES_17"
  deletion_protection = var.deletion_protection

  settings {
    tier              = var.database_tier
    availability_type = "ZONAL"
    disk_type         = "PD_SSD"
    disk_size         = 10
    disk_autoresize   = true
    backup_configuration {
      enabled                        = true
      start_time                     = "08:00"
      point_in_time_recovery_enabled = true
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 7
        retention_unit   = "COUNT"
      }
    }
    ip_configuration {
      ipv4_enabled = true
    }
  }
  depends_on = [google_project_service.enabled]
}

resource "google_sql_database" "application" {
  project  = var.project_id
  name     = var.database_name
  instance = google_sql_database_instance.postgres.name
}

resource "google_secret_manager_secret" "database_url" {
  project   = var.project_id
  secret_id = var.database_url_secret
  replication {
    auto {}
  }
  depends_on = [google_project_service.enabled]
}

resource "google_secret_manager_secret" "gemini" {
  count     = var.enable_gemini ? 1 : 0
  project   = var.project_id
  secret_id = var.gemini_secret
  replication {
    auto {}
  }
  depends_on = [google_project_service.enabled]
}

resource "google_pubsub_topic" "events" {
  project                    = var.project_id
  name                       = "flashcart-events"
  message_retention_duration = "604800s"
  depends_on                 = [google_project_service.enabled]
}

resource "google_pubsub_topic" "dead_letters" {
  project                    = var.project_id
  name                       = "flashcart-events-dead-letter"
  message_retention_duration = "604800s"
  depends_on                 = [google_project_service.enabled]
}

resource "google_service_account" "web" {
  project      = var.project_id
  account_id   = "flashcart-web"
  display_name = "FlashCart public API"
}
resource "google_service_account" "publisher" {
  project      = var.project_id
  account_id   = "flashcart-publisher"
  display_name = "FlashCart outbox publisher"
}
resource "google_service_account" "consumer" {
  project      = var.project_id
  account_id   = "flashcart-consumer"
  display_name = "FlashCart event consumer"
}
resource "google_service_account" "maintenance" {
  project      = var.project_id
  account_id   = "flashcart-maintenance"
  display_name = "FlashCart reservation maintenance"
}
resource "google_service_account" "scheduler" {
  project      = var.project_id
  account_id   = "flashcart-scheduler"
  display_name = "FlashCart scheduled jobs invoker"
}
resource "google_service_account" "push" {
  project      = var.project_id
  account_id   = "flashcart-pubsub-push"
  display_name = "Pub/Sub push identity for FlashCart"
}

resource "google_project_iam_member" "cloudsql" {
  for_each = toset([
    google_service_account.web.email,
    google_service_account.publisher.email,
    google_service_account.consumer.email,
    google_service_account.maintenance.email,
  ])
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${each.value}"
}

resource "google_secret_manager_secret_iam_member" "database_url" {
  for_each  = toset([google_service_account.web.email, google_service_account.publisher.email, google_service_account.consumer.email, google_service_account.maintenance.email])
  project   = var.project_id
  secret_id = google_secret_manager_secret.database_url.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${each.value}"
}

resource "google_secret_manager_secret_iam_member" "gemini" {
  count     = var.enable_gemini ? 1 : 0
  project   = var.project_id
  secret_id = google_secret_manager_secret.gemini[0].secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.web.email}"
}

resource "google_pubsub_topic_iam_member" "publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.events.name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.publisher.email}"
}

resource "google_cloud_run_v2_service" "web" {
  project             = var.project_id
  name                = "flashcart-web"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = var.deletion_protection
  template {
    service_account = google_service_account.web.email
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }
    containers {
      image = var.image
      ports { container_port = 8080 }
      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
        cpu_idle = true
      }
      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
      env {
        name  = "APP_ENV"
        value = "production"
      }
      env {
        name  = "APP_ORIGIN"
        value = var.app_origin
      }
      env {
        name  = "FRONTEND_DIST"
        value = "/app/static"
      }
      env {
        name  = "ASSISTANT_MODE"
        value = var.enable_gemini ? "gemini" : "template"
      }
      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
      dynamic "env" {
        for_each = var.enable_gemini ? [1] : []
        content {
          name = "GEMINI_API_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.gemini[0].secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.database_url]
}

resource "google_cloud_run_v2_service" "consumer" {
  project             = var.project_id
  name                = "flashcart-consumer"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = var.deletion_protection
  template {
    service_account = google_service_account.consumer.email
    scaling {
      min_instance_count = 0
      max_instance_count = 1
    }
    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.postgres.connection_name]
      }
    }
    containers {
      image = var.image
      args  = ["consume-push"]
      ports { container_port = 8080 }
      resources {
        limits = {
          cpu    = "1"
          memory = "256Mi"
        }
        cpu_idle = true
      }
      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }
      env {
        name  = "APP_ENV"
        value = "production"
      }
      env {
        name = "DATABASE_URL"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.database_url.secret_id
            version = "latest"
          }
        }
      }
    }
  }
  depends_on = [google_secret_manager_secret_iam_member.database_url]
}

resource "google_cloud_run_v2_job" "maintenance" {
  project  = var.project_id
  name     = "flashcart-maintenance"
  location = var.region
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = google_service_account.maintenance.email
      timeout         = "60s"
      max_retries     = 1
      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.postgres.connection_name]
        }
      }
      containers {
        image = var.image
        args  = ["maintenance"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
        env {
          name  = "APP_ENV"
          value = "production"
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.database_url.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }
}

resource "google_cloud_run_v2_job" "publisher" {
  project  = var.project_id
  name     = "flashcart-publish-outbox"
  location = var.region
  template {
    task_count  = 1
    parallelism = 1
    template {
      service_account = google_service_account.publisher.email
      timeout         = "60s"
      max_retries     = 1
      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.postgres.connection_name]
        }
      }
      containers {
        image = var.image
        args  = ["publish-outbox"]
        resources { limits = { cpu = "1", memory = "512Mi" } }
        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }
        env {
          name  = "APP_ENV"
          value = "production"
        }
        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.database_url.secret_id
              version = "latest"
            }
          }
        }
        env {
          name  = "GOOGLE_CLOUD_PROJECT"
          value = var.project_id
        }
        env {
          name  = "PUBSUB_TOPIC"
          value = google_pubsub_topic.events.name
        }
      }
    }
  }
  depends_on = [google_pubsub_topic_iam_member.publisher]
}

resource "google_cloud_run_v2_job_iam_member" "maintenance_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_job.maintenance.location
  name     = google_cloud_run_v2_job.maintenance.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}

resource "google_cloud_run_v2_job_iam_member" "publisher_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_job.publisher.location
  name     = google_cloud_run_v2_job.publisher.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler.email}"
}

resource "google_cloud_scheduler_job" "maintenance" {
  project     = var.project_id
  region      = var.region
  name        = "flashcart-maintenance"
  description = "Expire overdue FlashCart reservations."
  schedule    = "* * * * *"
  time_zone   = "Etc/UTC"
  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/${google_cloud_run_v2_job.maintenance.id}:run"
    oauth_token {
      service_account_email = google_service_account.scheduler.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }
  depends_on = [google_cloud_run_v2_job_iam_member.maintenance_invoker]
}

resource "google_cloud_scheduler_job" "publisher" {
  project     = var.project_id
  region      = var.region
  name        = "flashcart-publish-outbox"
  description = "Publish committed FlashCart outbox events."
  schedule    = "* * * * *"
  time_zone   = "Etc/UTC"
  http_target {
    http_method = "POST"
    uri         = "https://run.googleapis.com/v2/${google_cloud_run_v2_job.publisher.id}:run"
    oauth_token {
      service_account_email = google_service_account.scheduler.email
      scope                 = "https://www.googleapis.com/auth/cloud-platform"
    }
  }
  depends_on = [google_cloud_run_v2_job_iam_member.publisher_invoker]
}

resource "google_cloud_run_v2_service_iam_member" "web_public" {
  project  = var.project_id
  location = google_cloud_run_v2_service.web.location
  name     = google_cloud_run_v2_service.web.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "push_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.consumer.location
  name     = google_cloud_run_v2_service.consumer.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.push.email}"
}

resource "google_pubsub_subscription" "push" {
  project                    = var.project_id
  name                       = "flashcart-events-push"
  topic                      = google_pubsub_topic.events.id
  ack_deadline_seconds       = 30
  message_retention_duration = "604800s"
  push_config {
    push_endpoint = "${google_cloud_run_v2_service.consumer.uri}/"
    oidc_token {
      service_account_email = google_service_account.push.email
      audience              = google_cloud_run_v2_service.consumer.uri
    }
  }
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "60s"
  }
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letters.id
    max_delivery_attempts = 5
  }
  depends_on = [google_cloud_run_v2_service_iam_member.push_invoker]
}

resource "google_pubsub_topic_iam_member" "dead_letter_publisher" {
  project = var.project_id
  topic   = google_pubsub_topic.dead_letters.name
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

resource "google_pubsub_subscription_iam_member" "dead_letter_consumer" {
  project      = var.project_id
  subscription = google_pubsub_subscription.push.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

resource "google_service_account_iam_member" "push_token_creator" {
  service_account_id = google_service_account.push.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.current.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

