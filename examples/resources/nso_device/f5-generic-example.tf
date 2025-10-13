# Example: F5 BIG-IP device using Generic NED
resource "nso_device" "f5_bigip" {
  name = "f5-bigip-01"
  address = "192.168.1.100"
  port = 443
  authgroup = "f5-auth"
  admin_state = "unlocked"
  generic_ned_id = "f5-bigip-nc-1.0:f5-bigip-nc-1.0"
}

# Example: Another F5 device with different NED version
resource "nso_device" "f5_bigip_v2" {
  name = "f5-bigip-02"
  address = "192.168.1.101"
  port = 443
  authgroup = "f5-auth"
  admin_state = "unlocked"
  generic_ned_id = "f5-bigip-nc-2.0:f5-bigip-nc-2.0"
}

# Example: Generic device with custom NED
resource "nso_device" "custom_device" {
  name = "custom-device-01"
  address = "10.0.0.50"
  port = 830
  authgroup = "default"
  admin_state = "locked"
  generic_ned_id = "custom-ned-1.0:custom-ned-1.0"
}
