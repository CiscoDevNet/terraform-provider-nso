# Configuration Variables
# This file defines all configurable parameters for the NSO Terraform provider

# YAML configuration file path
config_yaml_file = "rollback_config.yaml"

# Operation mode - controls which resources are created
# Options: "full_workflow", "commit_only", "rollback_only"
operation_mode = "commit_only"

