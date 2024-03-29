---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "nso_device Resource - terraform-provider-nso"
subcategory: "Device"
description: |-
  This resource can manage the Device configuration.
---

# nso_device (Resource)

This resource can manage the Device configuration.

## Example Usage

```terraform
resource "nso_device" "example" {
  name        = "test-device01"
  address     = "10.1.1.1"
  port        = 22
  authgroup   = "default"
  admin_state = "locked"
  cli_ned_id  = "cisco-ios-cli-3.8:cisco-ios-cli-3.8"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) A string uniquely identifying the managed device.

### Optional

- `address` (String) IP address or host name for the management interface on the device.
- `admin_state` (String) Administrative state.
  - Choices: `locked`, `unlocked`, `southbound-locked`, `config-locked`, `call-home`
- `authgroup` (String) The authentication credentials used when connecting to this managed device.
- `cli_ned_id` (String) CLI NED ID.
- `instance` (String) An instance name from the provider configuration.
- `netconf_net_id` (String) NETCONF NED ID.
- `port` (Number) Port for the management interface on the device. If this leaf is not configured, NCS will use a default value based on the type of device. For example, a NETCONF device uses port 830, a CLI device over SSH uses port 22, and an SNMP device uses port 161.
  - Range: `0`-`65535`

### Read-Only

- `id` (String) The RESTCONF path.

## Import

Import is supported using the following syntax:

```shell
terraform import nso_device.example "tailf-ncs:devices/device=test-device01"
```
