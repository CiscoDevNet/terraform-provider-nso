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

package helpers

import (
	"os"
	"testing"
)

// The XPath/XML helper tests are shared with the IOS-XE provider and use
// Cisco-IOS-XE module prefixes. NSO resolves namespaces from NETCONF
// capabilities at runtime, so register the URIs those tests expect here.
func TestMain(m *testing.M) {
	for _, module := range []string{
		"Cisco-IOS-XE-aaa", "Cisco-IOS-XE-acl", "Cisco-IOS-XE-bgp", "Cisco-IOS-XE-cdp",
		"Cisco-IOS-XE-dhcp", "Cisco-IOS-XE-ethernet", "Cisco-IOS-XE-icmp", "Cisco-IOS-XE-native",
		"Cisco-IOS-XE-policy", "Cisco-IOS-XE-spanning-tree",
	} {
		namespaceExceptions[module] = "http://cisco.com/ns/yang/" + module
	}
	namespaceExceptions["Cisco-IOS-XE-template"] = "http://cisco.com/ns/yang/ios-xe/template"
	os.Exit(m.Run())
}
