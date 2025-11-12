
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRollbackResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			
			{
				Config: testAccRollbackResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_rollback.test", "rollback_id", "0"),
					resource.TestCheckResourceAttrSet("nso_rollback.test", "id"),
					resource.TestCheckResourceAttrSet("nso_rollback.test", "result"),
				),
			},
			
			{
				ResourceName:      "nso_rollback.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			
			{
				Config: testAccRollbackResourceConfigUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_rollback.test", "rollback_id", "1"),
					resource.TestCheckResourceAttrSet("nso_rollback.test", "id"),
					resource.TestCheckResourceAttrSet("nso_rollback.test", "result"),
				),
			},
		},
	})
}

func testAccRollbackResourceConfig() string {
	return `
resource "nso_rollback" "test" {
  rollback_id = 0
}
`
}

func testAccRollbackResourceConfigUpdate() string {
	return `
resource "nso_rollback" "test" {
  rollback_id = 1
}
`
}
