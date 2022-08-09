resource "nso_device_group" "example" {
  name         = "test-group1"
  device_names = ["c1"]
}
