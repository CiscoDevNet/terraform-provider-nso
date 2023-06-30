package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceNsoDeviceConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNsoDeviceConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nso_device_config.test", "id", "tailf-ncs:devices/device=c1/config"),
					resource.TestCheckResourceAttr("data.nso_device_config.test", "attributes.tailf-ned-cisco-ios:hostname", "R1"),
				),
			},
		},
	})
}

const testAccDataSourceNsoDeviceConfigConfig = `
resource "nso_device_config" "test" {
	device = "c1"
	attributes = {
		"tailf-ned-cisco-ios:hostname" = "R1"
	}
}

data "nso_device_config" "test" {
	device = "c1"
	depends_on = [nso_device_config.test]
}
`
