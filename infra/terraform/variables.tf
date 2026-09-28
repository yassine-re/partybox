variable "location" {
  description = "Région Azure de toutes les ressources."
  type        = string
  default     = "spaincentral"
}

variable "name_prefix" {
  description = "Préfixe des noms de ressources Azure."
  type        = string
  default     = "partybox"

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{2,20}$", var.name_prefix))
    error_message = "Utilisez 3 à 21 caractères minuscules, chiffres ou tirets, en commençant par une lettre."
  }
}

variable "vm_size" {
  description = "Taille de VM disponible pour cet abonnement et cette région ; vérifier son coût avant apply."
  type        = string
  default     = "Standard_B2als_v2"
}

variable "admin_username" {
  description = "Utilisateur SSH de la VM."
  type        = string
  default     = "partybox"
}

variable "ssh_public_key_path" {
  description = "Chemin local de la clé SSH publique autorisée sur la VM."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_source_cidr" {
  description = "Adresse ou plage IPv4 autorisée à accéder à SSH, par exemple 203.0.113.4/32."
  type        = string

  validation {
    condition     = can(cidrhost(var.ssh_source_cidr, 0)) && can(regex("^[0-9.]+/[0-9]+$", var.ssh_source_cidr)) && var.ssh_source_cidr != "0.0.0.0/0"
    error_message = "Indiquez un CIDR IPv4 restreint ; 0.0.0.0/0 est interdit pour SSH."
  }
}
