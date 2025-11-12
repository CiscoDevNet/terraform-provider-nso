
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCommitResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			
			{
				Config: testAccCommitResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_commit.test", "config_data", testAccCommitResourceConfigData()),
					resource.TestCheckResourceAttrSet("nso_commit.test", "id"),
					resource.TestCheckResourceAttrSet("nso_commit.test", "result"),
				),
			},
			
			{
				ResourceName:      "nso_commit.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			
			{
				Config: testAccCommitResourceConfigUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_commit.test", "config_data", testAccCommitResourceConfigDataUpdate()),
					resource.TestCheckResourceAttrSet("nso_commit.test", "id"),
					resource.TestCheckResourceAttrSet("nso_commit.test", "result"),
				),
			},
		},
	})
}

func testAccCommitResourceConfig() string {
	return `
resource "nso_commit" "test" {
  config_data = jsonencode({
    "tailf-ncs:devices" = {
      "device" = [
        {
          "name" = "ce0"
          "config" = {
            "tailf-ned-cisco-ios-xr:interface" = {
              "GigabitEthernet" = [
                {
                  "id" = "0/1"
                  "description" = "Test interface via commit"
                }
              ]
            }
          }
        }
      ]
    }
  })
}
`
}

func testAccCommitResourceConfigData() string {
	return `{"tailf-ncs:devices":{"device":[{"config":{"tailf-ned-cisco-ios-xr:interface":{"GigabitEthernet":[{"description":"Test interface via commit","id":"0/1"}]}},"name":"ce0"}]}}`
}

func testAccCommitResourceConfigUpdate() string {
	return `
resource "nso_commit" "test" {
  config_data = jsonencode({
    "tailf-ncs:devices" = {
      "device" = [
        {
          "name" = "ce0"
          "config" = {
            "tailf-ned-cisco-ios-xr:interface" = {
              "GigabitEthernet" = [
                {
                  "id" = "0/1"
                  "description" = "Updated test interface via commit"
                }
              ]
            }
          }
        }
      ]
    }
  })
}
`
}

func testAccCommitResourceConfigDataUpdate() string {
	return `{"tailf-ncs:devices":{"device":[{"config":{"tailf-ned-cisco-ios-xr:interface":{"GigabitEthernet":[{"description":"Updated test interface via commit","id":"0/1"}]}},"name":"ce0"}]}}`
}
