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

// wait-for-lock is a YANG choice (no-wait-for-lock | wait-for-lock), and
// wait-for-lock holds a nested one (timeout | infinity). NSO rejects an edit
// that touches two cases at once, so each switch between cases must send only
// the new case. These steps move through every case of both choices.
func TestAccNsoGlobalSettingsWaitForLockChoice(t *testing.T) {
	const name = "nso_global_settings.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNsoGlobalSettingsWaitForLock("no_wait_for_lock = true"),
				Check:  resource.TestCheckResourceAttr(name, "no_wait_for_lock", "true"),
			},
			{
				Config: testAccNsoGlobalSettingsWaitForLock("wait_for_lock_timeout = 45"),
				Check:  resource.TestCheckResourceAttr(name, "wait_for_lock_timeout", "45"),
			},
			{
				Config: testAccNsoGlobalSettingsWaitForLock("wait_for_lock_infinity = true"),
				Check:  resource.TestCheckResourceAttr(name, "wait_for_lock_infinity", "true"),
			},
			{
				// Explicit false on the other cases, as a full configuration would write.
				Config: testAccNsoGlobalSettingsWaitForLock("no_wait_for_lock = false\n\t\twait_for_lock_timeout = 30\n\t\twait_for_lock_infinity = false"),
				Check:  resource.TestCheckResourceAttr(name, "wait_for_lock_timeout", "30"),
			},
			{
				Config: testAccNsoGlobalSettingsWaitForLock("no_wait_for_lock = true"),
				Check:  resource.TestCheckResourceAttr(name, "no_wait_for_lock", "true"),
			},
		},
	})
}

func testAccNsoGlobalSettingsWaitForLock(attributes string) string {
	return fmt.Sprintf(`
	resource "nso_global_settings" "test" {
		%s
	}
	`, attributes)
}
