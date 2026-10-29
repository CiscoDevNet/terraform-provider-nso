resource "nso_device_config" "example" {
  device = "c1"
  attributes = {
    "tailf-ned-cisco-ios:hostname" = "R1"
  }
}

resource "nso_device_config" "access_list" {
  device = "c1"
  path   = "tailf-ned-cisco-ios:access-list/access-list[id='1']"
  lists = [
    {
      name = "rule"
      key  = "rule"
      items = [
        {
          rule = "permit any"
        }
      ]
    }
  ]
}
