output "web_url" { value = google_cloud_run_v2_service.web.uri }
output "consumer_url" { value = google_cloud_run_v2_service.consumer.uri }
output "artifact_repository" { value = google_artifact_registry_repository.containers.name }
output "database_connection_name" { value = google_sql_database_instance.postgres.connection_name }
output "database_url_secret" { value = google_secret_manager_secret.database_url.secret_id }
output "gemini_secret" { value = var.enable_gemini ? google_secret_manager_secret.gemini[0].secret_id : null }

