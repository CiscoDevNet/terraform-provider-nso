# Simple example

resource "nso_restconf" "ssh" {
  path = "tailf-ncs:ssh"
  attributes = {
    host-key-verification = "reject-unknown"
  }
}

# Define YANG lists and its elements

resource "nso_restconf" "customer" {
  path = "tailf-ncs:customers"
  lists = [
    {
      name = "customer"
      key  = "id"
      items = [
        {
          id = 123
        }
      ]
    }
  ]
}

# Define YANG leaf-list values

resource "nso_restconf" "device_group" {
  path = "tailf-ncs:devices/device-group=GROUP1"
  attributes = {
    name = "GROUP1"
  }
  lists = [
    {
      name   = "device-name"
      values = ["c1"]
    }
  ]
}

# Define nested attributes in YANG containers

resource "nso_restconf" "nested_attribute" {
  path   = "tailf-ncs:devices"
  delete = false
  attributes = {
    "global-settings/connect-timeout" = 25
  }
}
