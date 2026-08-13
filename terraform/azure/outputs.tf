output "resource_group" { value = azurerm_resource_group.this.name }
output "aks_name" { value = azurerm_kubernetes_cluster.this.name }
output "acr_login_server" { value = azurerm_container_registry.this.login_server }
output "postgres_fqdn" { value = azurerm_postgresql_flexible_server.this.fqdn }
output "workload_client_id" { value = azurerm_user_assigned_identity.workload.client_id }
