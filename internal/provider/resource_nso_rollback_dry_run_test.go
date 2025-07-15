
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRollbackDryRunResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			
			{
				Config: testAccRollbackDryRunResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_rollback_dry_run.test", "rollback_id", "0"),
					resource.TestCheckResourceAttrSet("nso_rollback_dry_run.test", "id"),
					resource.TestCheckResourceAttrSet("nso_rollback_dry_run.test", "result"),
				),
			},
			
			{
				ResourceName:      "nso_rollback_dry_run.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			
			{
				Config: testAccRollbackDryRunResourceConfigUpdate(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nso_rollback_dry_run.test", "rollback_id", "1"),
					resource.TestCheckResourceAttrSet("nso_rollback_dry_run.test", "id"),
					resource.TestCheckResourceAttrSet("nso_rollback_dry_run.test", "result"),
				),
			},
		},
	})
}

func testAccRollbackDryRunResourceConfig() string {
	return `
resource "nso_rollback_dry_run" "test" {
  rollback_id = 0
}
`
}

func testAccRollbackDryRunResourceConfigUpdate() string {
	return `
resource "nso_rollback_dry_run" "test" {
  rollback_id = 1
}
`
}
