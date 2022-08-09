package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccNsoRestconf(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoRestconfConfig_empty(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_restconf.test", "id", "tailf-ncs:ssh"),
				),
			},
			{
				Config: testAccNsoRestconfConfig_ssh("reject-mismatch"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_restconf.test", "id", "tailf-ncs:ssh"),
					resource.TestCheckResourceAttr("nso_restconf.test", "attributes.host-key-verification", "reject-mismatch"),
				),
			},
			{
				ResourceName:  "nso_restconf.test",
				ImportState:   true,
				ImportStateId: "tailf-ncs:ssh",
			},
			{
				Config: testAccNsoRestconfConfig_ssh("reject-unknown"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_restconf.test", "attributes.host-key-verification", "reject-unknown"),
				),
			},
			{
				Config: testAccNsoRestconfConfig_nested(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_restconf.nested", "lists.0.name", "customer"),
					resource.TestCheckResourceAttr("nso_restconf.nested", "lists.0.items.0.attributes.id", "123"),
				),
			},
		},
	})
}

func testAccNsoRestconfConfig_empty() string {
	return `
	resource "nso_restconf" "test" {
		path = "tailf-ncs:ssh"
	}
	`
}

func testAccNsoRestconfConfig_ssh(hkv string) string {
	return fmt.Sprintf(`
	resource "nso_restconf" "test" {
		path = "tailf-ncs:ssh"
		attributes = {
			host-key-verification = "%s"
		}
	}
	`, hkv)
}

func testAccNsoRestconfConfig_nested() string {
	return `
	resource "nso_restconf" "nested" {
		path = "tailf-ncs:customers"
		lists = [
			{
				name = "customer"
				key = "id"
				items = [
					{
						attributes = {
							id = 123
						}
					}
				]
			}
		]
	}
	`
}
