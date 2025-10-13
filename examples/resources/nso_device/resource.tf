resource "nso_device" "example" {
  name = "test-device01"
  address = "10.1.1.1"
  port = 22
  authgroup = "default"
  admin_state = "locked"
  cli_ned_id = "cisco-ios-cli-3.8:cisco-ios-cli-3.8"
  generic_ned_id = "f5-bigip-nc-1.0:f5-bigip-nc-1.0"
}
