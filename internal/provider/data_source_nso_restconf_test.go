// Copyright © 2023 Cisco Systems, Inc. and its affiliates.
// All rights reserved.
//
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
