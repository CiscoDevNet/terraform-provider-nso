package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceNsoRestconf(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNsoRestconfConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nso_restconf.test", "id", "tailf-ncs:ssh"),
					resource.TestCheckResourceAttr("data.nso_restconf.test", "attributes.host-key-verification", "reject-unknown"),
				),
			},
		},
	})
}

const testAccDataSourceNsoRestconfConfig = `
resource "nso_restconf" "test" {
	path = "tailf-ncs:ssh"
	attributes = {
		host-key-verification = "reject-unknown"
	}
}

data "nso_restconf" "test" {
	path = "tailf-ncs:ssh"
	depends_on = [nso_restconf.test]
}
`
