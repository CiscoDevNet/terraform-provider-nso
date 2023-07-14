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
					resource.TestCheckResourceAttr("nso_restconf.nested", "lists.0.items.0.id", "123"),
				),
			},
			{
				Config: testAccNsoRestconfConfig_nested_attribute(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("nso_restconf.nested_attribute", "attributes.global-settings/connect-timeout", "25"),
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
						id = 123
					}
				]
			}
		]
	}
	`
}

func testAccNsoRestconfConfig_nested_attribute() string {
	return `
	resource "nso_restconf" "nested_attribute" {
		path = "tailf-ncs:devices"
		delete = false
		attributes = {
			"global-settings/connect-timeout" = 25
		}
	}
	`
}
