variable "location" {
  type        = string
  description = "Azure region."
  default     = "France Central"
}

variable "resource_group_name" {
  type        = string
  description = "Resource group name."
  default     = "rg-whatsapp-g4"
}

variable "container_apps_env_name" {
  type        = string
  description = "Azure Container Apps environment name."
  default     = "cae-whatsapp-g4"
}

variable "log_analytics_workspace_name" {
  type        = string
  description = "Log Analytics workspace name."
  default     = "law-whatsapp-g4"
}

variable "acr_name" {
  type        = string
  description = "Azure Container Registry name (must be globally unique, 5-50 alphanum)."
  default     = "acrwhatsappg4"
}

variable "postgres_server_name" {
  type        = string
  description = "Azure PostgreSQL Flexible Server name."
  default     = "pg-whatsapp-g4"
}

variable "postgres_admin_username" {
  type        = string
  description = "PostgreSQL admin username."
  default     = "whatsappadmin"
}

variable "postgres_admin_password" {
  type        = string
  description = "PostgreSQL admin password. Prefer alphanum/underscore to avoid URL encoding issues."
  sensitive   = true
}

variable "jwt_secret" {
  type        = string
  description = "JWT secret shared by auth-service and api-gateway."
  sensitive   = true
}

variable "images" {
  type = object({
    auth_service    = string
    chat_service    = string
    message_service = string
    api_gateway     = string
  })
  description = "Container image references (including tag) hosted in ACR."
}

