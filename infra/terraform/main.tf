locals {
  name = var.name_prefix
  tags = {
    project    = "partybox"
    managed_by = "terraform"
  }
}

resource "azurerm_resource_group" "partybox" {
  name     = "${local.name}-rg"
  location = var.location
  tags     = local.tags
}

resource "azurerm_virtual_network" "partybox" {
  name                = "${local.name}-vnet"
  location            = azurerm_resource_group.partybox.location
  resource_group_name = azurerm_resource_group.partybox.name
  address_space       = ["10.42.0.0/16"]
  tags                = local.tags
}

resource "azurerm_subnet" "partybox" {
  name                 = "${local.name}-subnet"
  resource_group_name  = azurerm_resource_group.partybox.name
  virtual_network_name = azurerm_virtual_network.partybox.name
  address_prefixes     = ["10.42.1.0/24"]
}

resource "azurerm_network_security_group" "partybox" {
  name                = "${local.name}-nsg"
  location            = azurerm_resource_group.partybox.location
  resource_group_name = azurerm_resource_group.partybox.name
  tags                = local.tags

  security_rule {
    name                       = "ssh-admin"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefix      = var.ssh_source_cidr
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "http"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "80"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }

  security_rule {
    name                       = "https"
    priority                   = 120
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "443"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}

resource "azurerm_public_ip" "partybox" {
  name                = "${local.name}-ip"
  location            = azurerm_resource_group.partybox.location
  resource_group_name = azurerm_resource_group.partybox.name
  allocation_method   = "Static"
  sku                 = "Standard"
  tags                = local.tags
}

resource "azurerm_network_interface" "partybox" {
  name                = "${local.name}-nic"
  location            = azurerm_resource_group.partybox.location
  resource_group_name = azurerm_resource_group.partybox.name
  tags                = local.tags

  ip_configuration {
    name                          = "primary"
    subnet_id                     = azurerm_subnet.partybox.id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = azurerm_public_ip.partybox.id
  }
}

resource "azurerm_network_interface_security_group_association" "partybox" {
  network_interface_id      = azurerm_network_interface.partybox.id
  network_security_group_id = azurerm_network_security_group.partybox.id
}

resource "azurerm_linux_virtual_machine" "partybox" {
  name                            = "${local.name}-vm"
  resource_group_name             = azurerm_resource_group.partybox.name
  location                        = azurerm_resource_group.partybox.location
  size                            = var.vm_size
  admin_username                  = var.admin_username
  disable_password_authentication = true
  network_interface_ids           = [azurerm_network_interface.partybox.id]
  tags                            = local.tags

  admin_ssh_key {
    username   = var.admin_username
    public_key = file(pathexpand(var.ssh_public_key_path))
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
    disk_size_gb         = 30
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "ubuntu-24_04-lts"
    sku       = "server"
    version   = "latest"
  }
}
