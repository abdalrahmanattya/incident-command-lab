resource "azurerm_resource_group" "this" {
  name     = "${var.prefix}-rg"
  location = var.location
}
resource "azurerm_virtual_network" "this" {
  name                = "${var.prefix}-vnet"
  location            = var.location
  resource_group_name = azurerm_resource_group.this.name
  address_space       = ["10.40.0.0/16"]
}
resource "azurerm_subnet" "aks" {
  name                 = "aks"
  resource_group_name  = azurerm_resource_group.this.name
  virtual_network_name = azurerm_virtual_network.this.name
  address_prefixes     = ["10.40.0.0/20"]
}
resource "azurerm_subnet" "postgres" {
  name                 = "postgres"
  resource_group_name  = azurerm_resource_group.this.name
  virtual_network_name = azurerm_virtual_network.this.name
  address_prefixes     = ["10.40.16.0/24"]
  delegation {
    name = "postgres"
    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}
resource "azurerm_private_dns_zone" "postgres" {
  name                = "private.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.this.name
}
resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${var.prefix}-postgres-dns"
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.this.id
  resource_group_name   = azurerm_resource_group.this.name
}
resource "azurerm_container_registry" "this" {
  name                = replace("${var.prefix}acr", "-", "")
  resource_group_name = azurerm_resource_group.this.name
  location            = var.location
  sku                 = "Basic"
  admin_enabled       = false
}
resource "azurerm_user_assigned_identity" "workload" {
  name                = "${var.prefix}-workload"
  location            = var.location
  resource_group_name = azurerm_resource_group.this.name
}
resource "azurerm_log_analytics_workspace" "this" {
  name                = "${var.prefix}-logs"
  location            = var.location
  resource_group_name = azurerm_resource_group.this.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}
resource "azurerm_kubernetes_cluster" "this" {
  name                              = "${var.prefix}-aks"
  location                          = var.location
  resource_group_name               = azurerm_resource_group.this.name
  dns_prefix                        = var.prefix
  oidc_issuer_enabled               = true
  workload_identity_enabled         = true
  role_based_access_control_enabled = true
  default_node_pool {
    name                         = "system"
    vm_size                      = "Standard_B2s"
    vnet_subnet_id               = azurerm_subnet.aks.id
    node_count                   = 2
    only_critical_addons_enabled = true
  }
  identity { type = "SystemAssigned" }
  linux_profile {
    admin_username = "azureuser"
    ssh_key { key_data = var.ssh_public_key }
  }
  oms_agent { log_analytics_workspace_id = azurerm_log_analytics_workspace.this.id }
  azure_policy_enabled = true
}
resource "azurerm_role_assignment" "acr_pull" {
  scope                = azurerm_container_registry.this.id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.this.kubelet_identity[0].object_id
}
resource "azurerm_role_assignment" "cluster_admin" {
  count                = var.admin_group_object_id == "" ? 0 : 1
  scope                = azurerm_kubernetes_cluster.this.id
  role_definition_name = "Azure Kubernetes Service RBAC Cluster Admin"
  principal_id         = var.admin_group_object_id
}
resource "azurerm_federated_identity_credential" "workload" {
  name                = "${var.prefix}-gateway"
  resource_group_name = azurerm_resource_group.this.name
  parent_id           = azurerm_user_assigned_identity.workload.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.this.oidc_issuer_url
  subject             = "system:serviceaccount:incidentlab:gateway"
}
resource "azurerm_postgresql_flexible_server" "this" {
  name                   = "${var.prefix}-pg"
  resource_group_name    = azurerm_resource_group.this.name
  location               = var.location
  version                = "16"
  administrator_login    = "incidentadmin"
  administrator_password = var.postgres_admin_password
  storage_mb             = 32768
  sku_name               = "B_Standard_B1ms"
  zone                   = "1"
  delegated_subnet_id    = azurerm_subnet.postgres.id
  private_dns_zone_id    = azurerm_private_dns_zone.postgres.id
}
data "azurerm_client_config" "current" {}
resource "azurerm_key_vault" "this" {
  name                       = "${var.prefix}-kv"
  location                   = var.location
  resource_group_name        = azurerm_resource_group.this.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  purge_protection_enabled   = false
  soft_delete_retention_days = 7
  rbac_authorization_enabled = true
}
resource "azurerm_monitor_diagnostic_setting" "aks" {
  name                       = "${var.prefix}-diagnostics"
  target_resource_id         = azurerm_kubernetes_cluster.this.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.this.id
  enabled_log { category = "kube-apiserver" }
  enabled_log { category = "kube-audit" }
  enabled_metric { category = "AllMetrics" }
}
