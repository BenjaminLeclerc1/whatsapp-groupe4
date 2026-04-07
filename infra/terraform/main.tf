resource "azurerm_resource_group" "rg" {
  name     = var.resource_group_name
  location = var.location
}

resource "azurerm_log_analytics_workspace" "law" {
  name                = var.log_analytics_workspace_name
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}

resource "azurerm_container_app_environment" "cae" {
  name                       = var.container_apps_env_name
  location                   = azurerm_resource_group.rg.location
  resource_group_name        = azurerm_resource_group.rg.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.law.id
}

resource "azurerm_container_registry" "acr" {
  name                = var.acr_name
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  sku                 = "Basic"
  admin_enabled       = true
}

resource "azurerm_postgresql_flexible_server" "pg" {
  name                   = var.postgres_server_name
  resource_group_name    = azurerm_resource_group.rg.name
  location               = azurerm_resource_group.rg.location
  version                = "16"
  sku_name               = "B_Standard_B1ms"
  administrator_login    = var.postgres_admin_username
  administrator_password = var.postgres_admin_password

  storage_mb = 32768

  backup_retention_days = 7

  lifecycle {
    ignore_changes = [zone]
  }

  depends_on = [azurerm_resource_group.rg]
}

resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_azure" {
  name             = "allow-azure-services"
  server_id        = azurerm_postgresql_flexible_server.pg.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

resource "azurerm_postgresql_flexible_server_database" "shard0" {
  name      = "whatsapp_shard_0"
  server_id = azurerm_postgresql_flexible_server.pg.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

resource "azurerm_postgresql_flexible_server_database" "shard1" {
  name      = "whatsapp_shard_1"
  server_id = azurerm_postgresql_flexible_server.pg.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

locals {
  pg_fqdn       = azurerm_postgresql_flexible_server.pg.fqdn
  db_url_shard0 = "postgres://${var.postgres_admin_username}:${var.postgres_admin_password}@${local.pg_fqdn}:5432/${azurerm_postgresql_flexible_server_database.shard0.name}?sslmode=require"
  db_url_shard1 = "postgres://${var.postgres_admin_username}:${var.postgres_admin_password}@${local.pg_fqdn}:5432/${azurerm_postgresql_flexible_server_database.shard1.name}?sslmode=require"
  shard_urls    = "${local.db_url_shard0},${local.db_url_shard1}"
}

resource "azurerm_container_app" "auth" {
  name                         = "auth-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }

  secret {
    name  = "db-url"
    value = local.db_url_shard0
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8084
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 10

    container {
      name   = "auth-service"
      image  = var.images.auth_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8084"
      }
      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "db-url"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "chat" {
  name                         = "chat-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  secret {
    name  = "db-url"
    value = local.db_url_shard0
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8088
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 2
    max_replicas = 10

    container {
      name   = "chat-service"
      image  = var.images.chat_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8088"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "db-url"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "message" {
  name                         = "message-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  secret {
    name  = "db-url"
    value = local.db_url_shard1
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8082
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 2
    max_replicas = 20

    container {
      name   = "message-service"
      image  = var.images.message_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8082"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "db-url"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "user" {
  name                         = "user-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }
  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }
  secret {
    name  = "shard-urls"
    value = local.shard_urls
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8081
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 10

    container {
      name   = "user-service"
      image  = var.images.user_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8081"
      }
      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }
      env {
        name        = "SHARD_URLS"
        secret_name = "shard-urls"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "notification" {
  name                         = "notification-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8085
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 5

    container {
      name   = "notification-service"
      image  = var.images.notification_service
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "PORT"
        value = "8085"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "presence" {
  name                         = "presence-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8086
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 5

    container {
      name   = "presence-service"
      image  = var.images.presence_service
      cpu    = 0.25
      memory = "0.5Gi"

      env {
        name  = "PORT"
        value = "8086"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "search" {
  name                         = "search-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }
  secret {
    name  = "db-url"
    value = local.db_url_shard1
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8087
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 10

    container {
      name   = "search-service"
      image  = var.images.search_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8087"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "db-url"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "channel" {
  name                         = "channel-service"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }
  secret {
    name  = "db-url"
    value = local.db_url_shard0
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8088
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 10

    container {
      name   = "channel-service"
      image  = var.images.channel_service
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "PORT"
        value = "8088"
      }
      env {
        name        = "DATABASE_URL"
        secret_name = "db-url"
      }
      env {
        name  = "GIN_MODE"
        value = "release"
      }
    }
  }
}

resource "azurerm_container_app" "gateway" {
  name                         = "api-gateway"
  resource_group_name          = azurerm_resource_group.rg.name
  container_app_environment_id = azurerm_container_app_environment.cae.id
  revision_mode                = "Single"

  secret {
    name  = "acr-password"
    value = azurerm_container_registry.acr.admin_password
  }

  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }

  registry {
    server               = azurerm_container_registry.acr.login_server
    username             = azurerm_container_registry.acr.admin_username
    password_secret_name = "acr-password"
  }

  ingress {
    external_enabled = true
    target_port      = 8080
    traffic_weight {
      percentage      = 100
      latest_revision = true
    }
  }

  template {
    min_replicas = 1
    max_replicas = 10

    container {
      name   = "api-gateway"
      image  = var.images.api_gateway
      cpu    = 0.5
      memory = "1Gi"

      env {
        name  = "API_GATEWAY_PORT"
        value = "8080"
      }
      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }
      env {
        name  = "AUTH_SERVICE_URL"
        value = "https://${azurerm_container_app.auth.ingress[0].fqdn}"
      }
      env {
        name  = "CHAT_SERVICE_URL"
        value = "https://${azurerm_container_app.chat.ingress[0].fqdn}"
      }
      env {
        name  = "MESSAGE_SERVICE_URL"
        value = "https://${azurerm_container_app.message.ingress[0].fqdn}"
      }
      env {
        name  = "USER_SERVICE_URL"
        value = "https://${azurerm_container_app.user.ingress[0].fqdn}"
      }
      env {
        name  = "NOTIFICATION_SERVICE_URL"
        value = "https://${azurerm_container_app.notification.ingress[0].fqdn}"
      }
      env {
        name  = "PRESENCE_SERVICE_URL"
        value = "https://${azurerm_container_app.presence.ingress[0].fqdn}"
      }
      env {
        name  = "SEARCH_SERVICE_URL"
        value = "https://${azurerm_container_app.search.ingress[0].fqdn}"
      }
      env {
        name  = "CHANNEL_SERVICE_URL"
        value = "https://${azurerm_container_app.channel.ingress[0].fqdn}"
      }
    }
  }
}

