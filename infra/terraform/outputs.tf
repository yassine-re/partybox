output "public_ip" {
  description = "Adresse IPv4 publique et statique de la VM."
  value       = azurerm_public_ip.partybox.ip_address
}

output "ssh_command" {
  description = "Commande de connexion depuis une adresse autorisée par ssh_source_cidr."
  value       = "ssh ${var.admin_username}@${azurerm_public_ip.partybox.ip_address}"
}

output "resource_group_name" {
  description = "Groupe de ressources créé par Terraform."
  value       = azurerm_resource_group.partybox.name
}
