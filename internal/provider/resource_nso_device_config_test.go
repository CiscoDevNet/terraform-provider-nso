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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The nso_device_config tests create their own southbound-locked device, so
// changes are only written to the NSO CDB and never pushed to a real device.
// NSO_TEST_CLI_NED_ID selects the Cisco IOS CLI NED to use for that device.
const testAccDevice = "tf-acc-ce0"

func testAccDeviceConfig() string {
	nedId := os.Getenv("NSO_TEST_CLI_NED_ID")
	if nedId == "" {
		nedId = "cisco-ios-cli-6.115:cisco-ios-cli-6.115"
	}
	return fmt.Sprintf(`
	resource "nso_device" "tf_acc" {
		name        = "%s"
		address     = "192.0.2.1"
		authgroup   = "default"
		admin_state = "southbound-locked"
		cli_ned_id  = "%s"
	}
	`, testAccDevice, nedId)
}

const testAccLoopbackPath = "tailf-ned-cisco-ios:interface/Loopback[name='4091']"

func TestAccNsoDeviceConfig(t *testing.T) {
	device := testAccDevice
	xpath := fmt.Sprintf("/tailf-ncs:devices/device[name='%s']/config/%s", device, testAccLoopbackPath)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoDeviceConfigConfig_loopback("tf-acc-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.test", "id", xpath),
					resource.TestCheckResourceAttr("nso_device_config.test", "attributes.description", "tf-acc-1"),
				),
			},
			{
				ResourceName:            "nso_device_config.test",
				ImportState:             true,
				ImportStateId:           device + "," + testAccLoopbackPath,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"attributes"},
			},
			{
				Config: testAccNsoDeviceConfigConfig_loopback("tf-acc-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.test", "attributes.description", "tf-acc-2"),
				),
			},
			{
				Config: testAccNsoDeviceConfigConfig_list([]string{"permit 10.0.0.0 0.0.0.255", "deny any"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.list", "lists.0.items.#", "2"),
					resource.TestCheckResourceAttr("nso_device_config.list", "lists.0.items.1.rule", "deny any"),
				),
			},
			{
				Config: testAccNsoDeviceConfigConfig_list([]string{"permit 10.0.0.0 0.0.0.255"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_device_config.list", "lists.0.items.#", "1"),
				),
			},
		},
	})
}

func testAccNsoDeviceConfigConfig_loopback(description string) string {
	return testAccDeviceConfig() + fmt.Sprintf(`
	resource "nso_device_config" "test" {
		device = nso_device.tf_acc.name
		path   = "%s"
		attributes = {
			description = "%s"
		}
	}
	`, testAccLoopbackPath, description)
}

func testAccNsoDeviceConfigConfig_list(rules []string) string {
	items := ""
	for _, r := range rules {
		items += fmt.Sprintf(`{ rule = "%s" },`, r)
	}
	return testAccDeviceConfig() + fmt.Sprintf(`
	resource "nso_device_config" "list" {
		device = nso_device.tf_acc.name
		path   = "tailf-ned-cisco-ios:ip/access-list/standard/std-named-acl[name='TF-ACC-TEST']"
		lists = [
			{
				name  = "std-access-list-rule"
				key   = "rule"
				items = [%s]
			}
		]
	}
	`, items)
}
