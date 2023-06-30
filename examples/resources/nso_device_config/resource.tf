resource "nso_device_config" "example" {
  device = "c1"
  attributes = {
    "tailf-ned-cisco-ios:hostname" = "%s"
  }
}

resource "nso_device_config" "access_list" {
  device = "c1"
  path   = "tailf-ned-cisco-ios:access-list/access-list=1"
  attributes = {
    id = 1
  }
  lists = [
    {
      name = "rule"
      key  = "seq"
      items = [
        {
          seq  = 10
          rule = "permit ip any"
        }
      ]
    }
  ]
}
