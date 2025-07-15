terraform {
  required_providers {
    nso = {
      source = "ciscodevnet/nso"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

provider "nso" {
  url      = "http://127.0.0.1:8080"
  username = "admin"
  password = "admin"
  insecure = true
}

# Variables
variable "operation_mode" {
  description = "Operation mode: full_workflow, commit_only, or rollback_only"
  type        = string
  default     = "full_workflow"
  validation {
    condition     = contains(["full_workflow", "commit_only", "rollback_only"], var.operation_mode)
    error_message = "Operation mode must be one of: full_workflow, commit_only, rollback_only"
  }
}

variable "config_yaml_file" {
  description = "Path to the YAML configuration file"
  type        = string
  default     = "nso_config.yaml"
}

####Can be planned for future implementation####
variable "device_overrides" {
  description = "Optional device configuration overrides"
  type        = map(map(string))
  default     = {}
}

# Local values - Load and parse YAML configuration
locals {
  # Load YAML configuration
  yaml_content = file(var.config_yaml_file)
  config_data  = yamldecode(local.yaml_content)
  
  # Handle different YAML structures based on operation mode
  has_device_config = can(local.config_data["tailf-ncs:devices"])
  has_rollback_config = can(local.config_data["rollback"])
  
  # Convert to JSON format expected by NSO (only for device configs)
  json_config = local.has_device_config ? jsonencode({
    "tailf-ncs:devices" = local.config_data["tailf-ncs:devices"]
  }) : "{}"
  
  # Get rollback ID from config if available
  rollback_id_from_config = local.has_rollback_config ? local.config_data["rollback"]["id"] : 0
  
  # Operation mode flags
  commit_enabled   = contains(["full_workflow", "commit_only"], var.operation_mode)
  rollback_enabled = contains(["full_workflow", "rollback_only"], var.operation_mode)
  
  # Create a hash of the configuration for versioning
  config_hash = sha256(local.json_config)
}

# Random ID for unique resource identification
resource "random_id" "config_version" {
  byte_length = 8
  keepers = {
    config_hash = local.config_hash
  }
}

# Dry-run resource (always created for preview)
resource "nso_commit_dry_run" "test_preview" {
  count       = local.commit_enabled ? 1 : 0
  config_data = local.json_config
  
  lifecycle {
    replace_triggered_by = [
      random_id.config_version
    ]
  }
}

# Main commit resource
resource "nso_commit" "test_commit" {
  count       = local.commit_enabled ? 1 : 0
  config_data = local.json_config
  
  depends_on = [nso_commit_dry_run.test_preview]
  
  lifecycle {
    replace_triggered_by = [
      random_id.config_version
    ]
  }
}

# Rollback preview resource
resource "nso_rollback_dry_run" "rollback_preview" {
  count       = local.rollback_enabled ? 1 : 0
  rollback_id = local.rollback_enabled && local.commit_enabled ? one(nso_commit.test_commit[*].rollback_id) : local.rollback_id_from_config
  
  depends_on = [nso_commit.test_commit]
  
  lifecycle {
    replace_triggered_by = [
      random_id.config_version
    ]
  }
}

# Rollback execution resource
resource "nso_rollback" "test_rollback" {
  count       = var.operation_mode == "rollback_only" ? 1 : 0
  rollback_id = var.operation_mode == "rollback_only" && local.commit_enabled ? one(nso_commit.test_commit[*].rollback_id) : local.rollback_id_from_config
  
  depends_on = [nso_rollback_dry_run.rollback_preview]
  
  lifecycle {
    replace_triggered_by = [
      random_id.config_version
    ]
  }
}

# Outputs
# Optional outputs which can be displayed.
/**
  output "operation_mode" {
    description = "The operation mode used"
    value       = var.operation_mode
  }
  
  output "config_version" {
    description = "Version identifier for the configuration"
    value       = random_id.config_version.hex
  }
  
  output "yaml_config_file" {
    description = "YAML configuration file used"
    value       = var.config_yaml_file
  }
  
  output "parsed_config" {
    description = "Parsed configuration from YAML"
    value       = local.config_data
    sensitive   = true
  }
*/

output "json_config" {
  description = "JSON configuration sent to NSO"
  value       = local.json_config
  sensitive   = true
}

output "dry_run_result" {
  description = "Result of the dry-run operation"
  value       = local.commit_enabled ? try(one(nso_commit_dry_run.test_preview[*].result), "No dry-run performed") : "Dry-run disabled"
}

output "commit_result" {
  description = "Result of the commit operation"
  value       = local.commit_enabled ? try(one(nso_commit.test_commit[*].result), "No commit performed") : "Commit disabled"
}

output "captured_rollback_id" {
  description = "Rollback ID captured from the commit operation"
  value       = local.commit_enabled ? try(tostring(one(nso_commit.test_commit[*].rollback_id)), "No rollback ID captured") : "Commit disabled"
}

output "rollback_preview_result" {
  description = "Result of the rollback preview operation"
  value       = local.rollback_enabled ? try(one(nso_rollback_dry_run.rollback_preview[*].result), "No rollback preview performed") : "Rollback preview disabled"
}

output "rollback_result" {
  description = "Result of the rollback operation"
  value       = var.operation_mode == "rollback_only" ? try(one(nso_rollback.test_rollback[*].result), "No rollback performed") : "Rollback not executed"
}

output "workflow_summary" {
  description = "Summary of the workflow execution"
  value = {
    operation_mode                 = var.operation_mode
    config_version                 = random_id.config_version.hex
    yaml_file_used                 = var.config_yaml_file
    commit_enabled                 = local.commit_enabled
    rollback_enabled               = local.rollback_enabled
    rollback_id_captured           = local.commit_enabled ? try(tostring(one(nso_commit.test_commit[*].rollback_id)), "No rollback ID captured") : "Commit disabled"
    rollback_id_source             = local.commit_enabled ? "Captured from NSO commit" : (local.has_rollback_config ? "From rollback config file" : "N/A")
    rollback_id_used_in_preview    = local.rollback_enabled && local.commit_enabled ? try(tostring(one(nso_rollback_dry_run.rollback_preview[*].rollback_id)), "N/A") : (local.has_rollback_config ? tostring(local.rollback_id_from_config) : "N/A")
    rollback_id_used_in_rollback   = var.operation_mode == "rollback_only" ? try(tostring(one(nso_rollback.test_rollback[*].rollback_id)), tostring(local.rollback_id_from_config)) : "N/A"
    workflow_status                = "✓ Operation mode: ${var.operation_mode}"
    config_hash                    = local.config_hash
    config_type                    = local.has_device_config ? "Device Configuration" : (local.has_rollback_config ? "Rollback Configuration" : "Unknown")
  }
}
