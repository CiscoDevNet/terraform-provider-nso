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

func TestAccNsoNetconf(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoNetconfConfig_empty(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.test", "id", "/tailf-ncs:ssh"),
				),
			},
			{
				Config: testAccNsoNetconfConfig_ssh("reject-mismatch"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.test", "id", "/tailf-ncs:ssh"),
					resource.TestCheckResourceAttr("nso_netconf.test", "attributes.host-key-verification", "reject-mismatch"),
				),
			},
			{
				ResourceName:  "nso_netconf.test",
				ImportState:   true,
				ImportStateId: "/tailf-ncs:ssh",
				// attributes and lists are not discoverable from a path alone
				ImportStateVerifyIgnore: []string{"attributes"},
			},
			{
				Config: testAccNsoNetconfConfig_ssh("reject-unknown"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.test", "attributes.host-key-verification", "reject-unknown"),
				),
			},
			{
				Config: testAccNsoNetconfConfig_list([]string{"tf-acc-1", "tf-acc-2"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.list", "lists.0.name", "umap"),
					resource.TestCheckResourceAttr("nso_netconf.list", "lists.0.items.#", "2"),
					resource.TestCheckResourceAttr("nso_netconf.list", "lists.0.items.1.local-user", "tf-acc-2"),
					resource.TestCheckResourceAttr("nso_netconf.list", "lists.0.items.1.remote-name", "remote-2"),
				),
			},
			{
				// Removing an item from the plan must delete it on NSO
				Config: testAccNsoNetconfConfig_list([]string{"tf-acc-1"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.list", "lists.0.items.#", "1"),
				),
			},
			{
				Config: testAccNsoNetconfConfig_leafList([]string{"tf-acc-c1", "tf-acc-c2"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.leaf_list", "id", "/tailf-ncs:devices/device-group[name='tf-acc-group']"),
					resource.TestCheckResourceAttr("nso_netconf.leaf_list", "lists.0.values.#", "2"),
				),
			},
			{
				Config: testAccNsoNetconfConfig_leafList([]string{"tf-acc-c2"}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.leaf_list", "lists.0.values.#", "1"),
					resource.TestCheckResourceAttr("nso_netconf.leaf_list", "lists.0.values.0", "tf-acc-c2"),
				),
			},
			{
				Config: testAccNsoNetconfConfig_nested_attribute(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_netconf.nested_attribute", "attributes.default-map/remote-name", "tf-acc"),
				),
			},
		},
	})
}

func testAccNsoNetconfConfig_empty() string {
	return `
	resource "nso_netconf" "test" {
		path = "/tailf-ncs:ssh"
	}
	`
}

func testAccNsoNetconfConfig_ssh(hkv string) string {
	return fmt.Sprintf(`
	resource "nso_netconf" "test" {
		path = "/tailf-ncs:ssh"
		attributes = {
			host-key-verification = "%s"
		}
	}
	`, hkv)
}

func testAccNsoNetconfConfig_list(users []string) string {
	items := ""
	for i, u := range users {
		items += fmt.Sprintf(`{ local-user = "%s", remote-name = "remote-%d", same-pass = "" },`, u, i+1)
	}
	return fmt.Sprintf(`
	resource "nso_netconf" "list" {
		path = "/tailf-ncs:devices/authgroups/group[name='tf-acc-list']"
		lists = [
			{
				name = "umap"
				key = "local-user"
				items = [%s]
			}
		]
	}
	`, items)
}

func testAccNsoNetconfConfig_leafList(devices []string) string {
	values := ""
	for _, d := range devices {
		values += fmt.Sprintf(`"%s",`, d)
	}
	return fmt.Sprintf(`
	resource "nso_device" "leaf_list_c1" {
		name        = "tf-acc-c1"
		address     = "192.0.2.1"
		authgroup   = "default"
		admin_state = "southbound-locked"
		cli_ned_id  = "cisco-ios-cli-6.115:cisco-ios-cli-6.115"
	}

	resource "nso_device" "leaf_list_c2" {
		name        = "tf-acc-c2"
		address     = "192.0.2.2"
		authgroup   = "default"
		admin_state = "southbound-locked"
		cli_ned_id  = "cisco-ios-cli-6.115:cisco-ios-cli-6.115"
	}

	resource "nso_netconf" "leaf_list" {
		path = "/tailf-ncs:devices/device-group[name='tf-acc-group']"
		lists = [
			{
				name = "device-name"
				values = [%s]
			}
		]
		depends_on = [nso_device.leaf_list_c1, nso_device.leaf_list_c2]
	}
	`, values)
}

func testAccNsoNetconfConfig_nested_attribute() string {
	return `
	resource "nso_netconf" "nested_attribute" {
		path = "/tailf-ncs:devices/authgroups/group[name='tf-acc-nested']"
		attributes = {
			"default-map/remote-name" = "tf-acc"
			"default-map/same-pass"   = ""
		}
	}
	`
}
