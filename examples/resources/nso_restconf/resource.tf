resource "nso_restconf" "example" {
  path = "tailf-ncs:ssh"
  attributes = {
    host-key-verification = "reject-unknown"
  }
}

resource "nso_restconf" "customer" {
  path = "tailf-ncs:customers"
  lists = [
    {
      name = "customer"
      key  = "id"
      items = [
        {
          attributes = {
            id = 123
          }
        }
      ]
    }
  ]
}

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
