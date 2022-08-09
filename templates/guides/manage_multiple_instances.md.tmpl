---
subcategory: "Guides"
page_title: "Manage Multiple Instances"
description: |-
    Howto manage multiple instances.
---

# Manage Multiple Instances

When it comes to managing multiple NSO instances, one can create multiple provider configurations and distinguish them by `alias` ([documentation](https://www.terraform.io/language/providers/configuration#alias-multiple-provider-configurations)).

```terraform
provider "nso" {
  alias    = "NSO-DEV"
  username = "user1"
  password = "Cisco123"
  url      = "https://10.1.1.1"
}

provider "nso" {
  alias    = "NSO-PROD"
  username = "user1"
  password = "Cisco123"
  url      = "https://10.1.1.2"
}
```

The disadvantages here is that the `provider` attribute of resources cannot be dynamic and therefore cannot be used in combination with `for_each` as an example. The issue is being tracked [here](https://github.com/hashicorp/terraform/issues/24476).

This provider offers an alternative approach where mutliple instances can be managed by a single provider configuration and the optional `instance` attribute, which is available in every resource and data source, can then be used to select the respective instance. This assumes that every instances uses the same credentials.

```terraform
locals {
  instances = [
    {
      name = "NSO-DEV"
      url  = "https://10.1.1.1"
    },
    {
      name = "NSO-PROD"
      url  = "https://10.1.1.2"
    },
  ]
}

provider "nso" {
  username  = "admin"
  password  = "Cisco123"
  instances = local.instances
}

resource "nso_device" "example" {
  for_each   = toset([for inst in local.instances : instance.name])
  instance     = each.key
  name        = "test-device01"
  address     = "10.1.1.1"
  port        = 22
  authgroup   = "default"
  admin_state = "locked"
  cli_ned_id  = "cisco-ios-cli-3.0:cisco-ios-cli-3.0"
}
```
