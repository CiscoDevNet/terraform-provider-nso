package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNsoDeviceConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoDeviceConfigConfig_empty(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.test", "id", "tailf-ncs:devices/device=ce0/config"),
				),
			},
			{
				Config: testAccNsoDeviceConfigConfig_hostname("R1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.test", "id", "tailf-ncs:devices/device=ce0/config"),
					resource.TestCheckResourceAttr("nso_device_config.test", "attributes.tailf-ned-cisco-ios:hostname", "R1"),
				),
			},
			{
				ResourceName:  "nso_device_config.test",
				ImportState:   true,
				ImportStateId: "tailf-ncs:devices/device=ce0/config",
			},
			{
				Config: testAccNsoDeviceConfigConfig_hostname("R2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.test", "attributes.tailf-ned-cisco-ios:hostname", "R2"),
				),
			},
			{
				Config: testAccNsoDeviceConfigConfig_nested(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.nested", "lists.0.name", "std-access-list-rule"),
					resource.TestCheckResourceAttr("nso_device_config.nested", "lists.0.items.0.rule", "permit ip any"),
				),
			},
		},
	})
}

func testAccNsoDeviceConfigConfig_empty() string {
	return `
	resource "nso_device_config" "test" {
		device = "ce0"
	}
	`
}

func testAccNsoDeviceConfigConfig_hostname(hostname string) string {
	return fmt.Sprintf(`
	resource "nso_device_config" "test" {
		device = "ce0"
		attributes = {
			"tailf-ned-cisco-ios:hostname" = "%s"
		}
	}
	`, hostname)
}

func testAccNsoDeviceConfigConfig_nested() string {
	return `
	resource "nso_device_config" "nested" {
		device = "ce0"
		path = "tailf-ned-cisco-ios:access-list/access-list-standard-range=1"
		attributes = {
			listnumber = 1
		}
		lists = [
			{
				name = "std-access-list-rule"
				key = "rule"
				items = [
					{
						rule = "permit ip any"
					}
				]
			}
		]
	}
	`
}
