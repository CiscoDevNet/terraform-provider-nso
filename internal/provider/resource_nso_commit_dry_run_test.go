
package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNsoCommitDryRun(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoCommitDryRunConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nso_commit_dry_run.test", "result"),
				),
			},
			{
				Config: testAccNsoCommitDryRunConfig_withOptions(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("nso_commit_dry_run.test", "result"),
				),
			},
		},
	})
}

func testAccNsoCommitDryRunConfig_basic() string {
	return `
	resource "nso_commit_dry_run" "test" {
	}
	`
}

func testAccNsoCommitDryRunConfig_withOptions() string {
	return `
	resource "nso_commit_dry_run" "test" {
	}
	`
}
