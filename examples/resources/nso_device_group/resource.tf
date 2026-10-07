resource "nso_device_group" "example" {
  name         = "test-group1"
  device_names = ["tf-acc-dg-device"]
}
