variable "project_id" {
  type = string
}
variable "region" {
  type    = string
  default = "southamerica-east1"
}
variable "image" {
  type        = string
  description = "Artifact Registry image pinned by immutable digest, for example .../flashcart@sha256:..."
  validation {
    condition     = can(regex("@sha256:[0-9a-f]{64}$", var.image))
    error_message = "image must use an immutable sha256 digest, not a mutable tag."
  }
}
variable "app_origin" {
  type        = string
  description = "Public HTTPS origin of the Cloud Run web service. Set after first service creation."
  validation {
    condition     = startswith(var.app_origin, "https://")
    error_message = "app_origin must be HTTPS."
  }
}
variable "database_tier" {
  type    = string
  default = "db-f1-micro"
}
variable "database_name" {
  type    = string
  default = "flashcart"
}
variable "database_url_secret" {
  type    = string
  default = "flashcart-database-url"
}
variable "gemini_secret" {
  type    = string
  default = "flashcart-gemini-api-key"
}
variable "enable_gemini" {
  type    = bool
  default = false
}
variable "deletion_protection" {
  type    = bool
  default = true
}

