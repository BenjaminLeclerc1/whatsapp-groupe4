output "container_apps" {
  value = {
    auth_service_fqdn    = azurerm_container_app.auth.ingress[0].fqdn
    chat_service_fqdn    = azurerm_container_app.chat.ingress[0].fqdn
    message_service_fqdn = azurerm_container_app.message.ingress[0].fqdn
    api_gateway_fqdn     = azurerm_container_app.gateway.ingress[0].fqdn
  }
}

output "postgres_fqdn" {
  value = azurerm_postgresql_flexible_server.pg.fqdn
}

output "acr_login_server" {
  value = azurerm_container_registry.acr.login_server
}

