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
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourceNsoDeviceConfig(t *testing.T) {
	device := testAccDevice
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceNsoDeviceConfigConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.nso_device_config.test", "id", fmt.Sprintf("/tailf-ncs:devices/device[name='%s']/config/%s", device, testAccLoopbackPath)),
					resource.TestCheckResourceAttr("data.nso_device_config.test", "attributes.description", "tf-acc-ds"),
				),
			},
		},
	})
}

func testAccDataSourceNsoDeviceConfigConfig() string {
	return testAccDeviceConfig() + fmt.Sprintf(`
	resource "nso_device_config" "test" {
		device = nso_device.tf_acc.name
		path   = "%[1]s"
		attributes = {
			description = "tf-acc-ds"
		}
	}

	data "nso_device_config" "test" {
		device     = nso_device.tf_acc.name
		path       = "%[1]s"
		depends_on = [nso_device_config.test]
	}
	`, testAccLoopbackPath)
}
