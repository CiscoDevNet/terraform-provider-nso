resource "nso_device_group" "example" {
  name               = "test-group1"
  device_names       = ["tf-acc-dg-device"]
  location_name      = "Building-1"
  location_latitude  = "37.338207"
  location_longitude = "-121.88633"
  location_altitude  = 50
}
