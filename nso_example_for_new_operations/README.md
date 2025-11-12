# NSO Terraform Provider - Advanced Workflow Implementation

## Overview

This project implements a comprehensive Terraform-based workflow for Cisco NSO (Network Services Orchestrator) operations, providing fine-grained control over commit and rollback operations with dry-run capabilities. This implementation extends the official NSO Terraform provider by introducing custom resources for advanced NSO operations.

## Motivation

The official [Terraform NSO provider](https://github.com/CiscoDevNet/terraform-provider-nso) lacks comprehensive support for NSO's advanced operational workflows, specifically:

- **Commit dry-run operations** - Preview changes before applying
- **Rollback dry-run operations** - Preview rollback effects before execution  
- **Granular operation control** - Selective execution of commit/rollback workflows
- **State consistency management** - Handling NSO's dynamic ID generation
- **Configuration diff visualization** - Clear visibility of changes being applied

This implementation provides these advanced capabilities through a complete workflow management solution.

## Features

### Operation Modes

1. **`full_workflow`** (Default): Complete pipeline with dry-run → commit → rollback preview
2. **`commit_only`**: Dry-run → commit operations (captures rollback ID)
3. **`rollback_only`**: Rollback preview → rollback execution (uses specified rollback ID)

### Enhanced Dry-Run Display

- **CLI-C Format**: Human-readable configuration diffs
- **Change Indicators**: Clear visualization of additions (+), deletions (-), and modifications (!)
- **Device-Specific**: Organized by device with full configuration context

### Rollback ID Management

- **Automatic Capture**: Rollback IDs are automatically captured from NSO commits
- **Sequential Tracking**: Support for multiple successive commits with rollback chains
- **Flexible Rollback**: Ability to rollback to any previous rollback point

### YAML-Based Configuration

- **Separation of Concerns**: Configuration data separate from Terraform logic
- **Version Control Friendly**: Easy to diff and track changes
- **Network Engineer Friendly**: Familiar YAML syntax for device configurations

## Prerequisites & Requirements

### System Requirements

1. **Terraform** (>= 1.0)
2. **Go** (>= 1.19) - For building the provider
3. **NSO** (Network Services Orchestrator) - Running and accessible
4. **Git** - For version control

### NSO Environment Setup

#### 1. NSO Configuration
```bash
# Ensure NSO is running and accessible
# Default NSO management interface runs on port 8080
ncs --status
```

#### 2. Protocol Configuration
**Important**: This implementation requires **HTTP** instead of HTTPS for NSO communication.

**Provider Configuration:**
```hcl
provider "nso" {
  url      = "http://127.0.0.1:8080"  # HTTP, not HTTPS
  username = "admin"
  password = "admin"
  insecure = true
}
```

#### 3. Network Proxy Handling
If you're behind a corporate proxy, bypass it for NSO communication:

```bash
# Method 1: Environment variable
export NO_PROXY=127.0.0.1,localhost

# Method 2: Run commands with proxy bypass
NO_PROXY=127.0.0.1 terraform apply
```

#### 4. NSO API Endpoints
The implementation uses these NSO RESTCONF endpoints:
- **Commit Operations**: `/restconf/data/tailf-ncs:devices`
- **Rollback Operations**: `/restconf/data/tailf-rollback:rollback-files/apply-rollback-file`
- **Dry-Run Format**: Uses `cli-c` format for enhanced diff display

## File Structure

```
nso_example-dry-run+commit+rollback/
├── main.tf                 # Main Terraform configuration
├── nso_config.yaml         # Device configuration (YAML)
├── rollback_config.yaml    # Rollback configuration (YAML)
├── terraform.tfvars        # Terraform variables (auto-loaded)
├── config.tfvars          # Alternative variables file
└── README.md              # This documentation
```

## Configuration Files

### 1. Device Configuration (`nso_config.yaml`)
```yaml
# Device configuration for NSO
tailf-ncs:devices:
  device:
    - name: ce0
      config:
        tailf-ned-cisco-ios-xr:hostname: your-hostname
    - name: ce1
      config:
        tailf-ned-cisco-ios-xr:interface:
          GigabitEthernet:
            - id: "0/1"
              description: "Your interface description"
    - name: ce2
      config:
        tailf-ned-cisco-ios-xr:vlan:
          vlan-list:
            - id: "100"
```

### 2. Rollback Configuration (`rollback_config.yaml`)
```yaml
# Rollback configuration for specific rollback operations
rollback:
  id: 10291  # Specify the rollback ID to use
  description: "Rollback to previous configuration"
  test_name: "rollback_test"
```

### 3. Terraform Variables (`terraform.tfvars`)
```hcl
# Main configuration variables (auto-loaded)
config_yaml_file = "nso_config.yaml"
operation_mode = "full_workflow"  # full_workflow, commit_only, or rollback_only
```

## Usage Guide

### Initial Setup

1. **Clone and Build Provider**
```bash
git clone <repository-url>
cd terraform-provider-nso-main
go build -o terraform-provider-nso
```

2. **Initialize Terraform**
```bash
cd nso_example-dry-run+commit+rollback
terraform init
```

3. **Verify NSO Connectivity**
```bash
# Test NSO API accessibility
curl -k -u admin:admin http://127.0.0.1:8080/restconf/data/tailf-ncs:devices
```

### Basic Operations

#### 1. Full Workflow (Recommended)
**Method 1: Using terraform.tfvars (Primary)**
```bash
# Update terraform.tfvars
cat > terraform.tfvars << EOF
config_yaml_file = "nso_config.yaml"
operation_mode = "full_workflow"
EOF

# Execute complete workflow: dry-run → commit → rollback preview
terraform plan
terraform apply
```

**Method 2: Using CLI variables (Alternative)**
```bash
# Complete workflow: dry-run → commit → rollback preview
terraform apply -var="operation_mode=full_workflow"
```

#### 2. Commit Only
**Method 1: Using terraform.tfvars (Primary)**
```bash
# Update terraform.tfvars for commit-only operation
cat > terraform.tfvars << EOF
config_yaml_file = "nso_config.yaml"
operation_mode = "commit_only"
EOF

# Execute commit operations and capture rollback ID
terraform plan
terraform apply

# Check captured rollback ID
terraform output captured_rollback_id
```

**Method 2: Using CLI variables (Alternative)**
```bash
# Commit configuration and capture rollback ID
terraform apply -var="operation_mode=commit_only"
```

#### 3. Rollback Operations
**Method 1: Using terraform.tfvars (Primary)**
```bash
# First, update rollback_config.yaml with desired rollback ID
# Then update terraform.tfvars for rollback operation
cat > terraform.tfvars << EOF
config_yaml_file = "rollback_config.yaml"
operation_mode = "rollback_only"
EOF

# Execute rollback operations
terraform plan
terraform apply
```

**Method 2: Using CLI variables (Alternative)**
```bash
# First, update rollback_config.yaml with desired rollback ID
# Then apply rollback
terraform apply -var="operation_mode=rollback_only" -var="config_yaml_file=rollback_config.yaml"
```

### Advanced Workflows

#### Successive Commits with Rollback Chain
**Method 1: Using terraform.tfvars (Primary)**
```bash
# Step 1: First commit
cat > terraform.tfvars << EOF
config_yaml_file = "nso_config.yaml"
operation_mode = "commit_only"
EOF
terraform plan && terraform apply
# → Captures rollback ID: 10290

# Step 2: Modify configuration and commit again
# (terraform.tfvars remains the same for another commit)
terraform plan && terraform apply
# → Captures rollback ID: 10291

# Step 3: Rollback to any previous point
# Update rollback_config.yaml with desired rollback ID, then:
cat > terraform.tfvars << EOF
config_yaml_file = "rollback_config.yaml"
operation_mode = "rollback_only"
EOF
terraform plan && terraform apply
```

**Method 2: Using CLI variables (Alternative)**
```bash
# Step 1: First commit
terraform apply -var="operation_mode=commit_only"
# → Captures rollback ID: 10290

# Step 2: Modify configuration and commit again
terraform apply -var="operation_mode=commit_only"
# → Captures rollback ID: 10291

# Step 3: Rollback to any previous point
# Update rollback_config.yaml with desired rollback ID
terraform apply -var="operation_mode=rollback_only" -var="config_yaml_file=rollback_config.yaml"
```

#### Environment-Specific Configurations
**Method 1: Using terraform.tfvars (Primary)**
```bash
# Development environment
cat > terraform.tfvars << EOF
config_yaml_file = "dev_config.yaml"
operation_mode = "full_workflow"
EOF
terraform plan && terraform apply

# Production environment
cat > terraform.tfvars << EOF
config_yaml_file = "prod_config.yaml"
operation_mode = "full_workflow"
EOF
terraform plan && terraform apply
```

**Method 2: Using CLI variables (Alternative)**
```bash
# Development environment
terraform apply -var="config_yaml_file=dev_config.yaml"

# Production environment
terraform apply -var="config_yaml_file=prod_config.yaml"
```

#### Device Configuration Overrides
**Method 1: Using terraform.tfvars (Primary)**
```bash
# Create terraform.tfvars with device overrides
cat > terraform.tfvars << EOF
config_yaml_file = "nso_config.yaml"
operation_mode = "commit_only"
device_overrides = {
  "ce0" = {
    "hostname" = "emergency-override-hostname"
  }
}
EOF
terraform plan && terraform apply
```

**Method 2: Using CLI variables (Alternative)**
```bash
# Override specific device configurations
terraform apply -var='device_overrides={
  "ce0" = {
    "hostname" = "emergency-override-hostname"
  }
}'
```

## Monitoring & Troubleshooting

### Output Inspection
```bash
# View workflow summary
terraform output workflow_summary

# Check dry-run results
terraform output dry_run_result

# Check rollback preview
terraform output rollback_preview_result

# View captured rollback ID
terraform output captured_rollback_id
```

### Common Issues & Solutions

#### 1. EOF Error
**Problem**: `Failed to perform dry-run operation, got error: EOF`
**Solution**: 
- Ensure NSO is running on HTTP (not HTTPS)
- Check proxy settings: `export NO_PROXY=127.0.0.1`
- Verify NSO API accessibility

#### 2. Rollback ID Not Found
**Problem**: `no such file` error during rollback
**Solution**:
- Verify rollback ID exists in NSO
- Check rollback_config.yaml has correct rollback ID
- Ensure rollback ID was captured from previous commit

#### 3. YAML Structure Errors
**Problem**: `Invalid index` errors in locals
**Solution**:
- Verify YAML syntax is correct
- Ensure device configs use `tailf-ncs:devices` structure
- Ensure rollback configs use `rollback:` structure

#### 4. Provider Development Overrides
**Problem**: Warning about provider development overrides
**Solution**: This is expected during development - the provider is loaded from local build

### Debug Commands
```bash
# Validate Terraform configuration
terraform validate

# Check parsed YAML configuration
terraform output parsed_config

# View JSON sent to NSO
terraform output json_config

# Test NSO connectivity
NO_PROXY=127.0.0.1 curl -k -u admin:admin http://127.0.0.1:8080/restconf/data/tailf-ncs:devices
```

## Output Examples

### Dry-Run Diff Display
```
=== NSO Dry-Run Diff - Configuration Changes Preview ===
The following shows what would be applied to NSO devices:
(+ indicates additions, - indicates deletions, ! indicates modifications)

devices device ce0
 config
  hostname ce0-new
 !
!
devices device ce1
 config
  interface GigabitEthernet 0/1
   description Test interface
   no shutdown
  exit
 !
!
devices device ce2
 config
  vlan 30
  exit
 !
!
```

### Rollback Preview
```
The following shows what changes would be rolled back from rollback ID 10291:

devices device ce0
 config
  hostname ce0-host
 !
!
devices device ce1
 config
  interface GigabitEthernet 0/0
   description Test interface
  exit
 !
!
devices device ce2
 config
  no vlan 2
 !
!
```

## Best Practices

### 1. Configuration Management
- **Version Control**: Always commit YAML configurations to Git
- **Validation**: Run `terraform validate` before applying
- **Testing**: Test configurations in development environment first
- **Documentation**: Document configuration changes in YAML comments

### 2. Operational Workflow
- **Dry-Run First**: Always review dry-run output before committing
- **Incremental Changes**: Make small, incremental configuration changes
- **Rollback Planning**: Keep track of rollback IDs for critical changes
- **Monitoring**: Monitor NSO and device logs during operations

### 3. Environment Management
- **Separate Configs**: Use different YAML files for different environments
- **Proxy Handling**: Consistently use `NO_PROXY` for NSO communication
- **Credential Management**: Secure NSO credentials appropriately

### 4. Troubleshooting
- **Logs**: Check both Terraform and NSO logs for issues
- **Connectivity**: Verify NSO API accessibility before operations
- **State Management**: Keep Terraform state files secure and backed up

## Security Considerations

1. **Credentials**: Store NSO credentials securely
2. **Network Access**: Limit NSO API access to authorized systems
3. **State Files**: Protect Terraform state files containing sensitive data
4. **Audit Trail**: Maintain audit logs of all configuration changes

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes and test thoroughly
4. Submit pull request with detailed description

## License

This project is licensed under the Mozilla Public License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
1. Check the troubleshooting section above
2. Review NSO and Terraform logs
3. Verify system requirements and setup
4. Create issue with detailed error information

## Acknowledgments

- **Cisco DevNet** for the NSO Terraform provider
- **Terraform Community** for lifecycle management patterns
- **NSO Documentation** for operational best practices

---

**Note**: This implementation is designed for NSO environments and requires proper NSO setup and configuration. Ensure all prerequisites are met before attempting to use this workflow.
