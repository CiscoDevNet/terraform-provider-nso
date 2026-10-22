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
	"strings"
	"testing"

	"github.com/netascode/go-netconf"
	"github.com/netascode/xmldot"
)

// Capabilities as advertised by NSO 6.x with the IOS, NX-OS and IOS-XR CLI NEDs loaded.
var nsoNedCapabilities = []string{
	"http://tail-f.com/ns/ncs?module=tailf-ncs&revision=2024-06-03",
	"urn:ios-meta?module=tailf-ned-cisco-ios-meta&revision=2026-09-10",
	"http://tail-f.com/ned/cisco-nx/meta?module=tailf-ned-cisco-nx-meta&revision=2026-09-10",
	"http://tail-f.com/ned/cisco-iosxr/meta?module=tailf-ned-cisco-ios-xr-meta&revision=2026-08-04",
	"urn:ietf:params:netconf:base:1.1",
}

func TestBuildNamespaceMap(t *testing.T) {
	discovered := BuildNamespaceMap(nsoNedCapabilities)

	expected := map[string]string{
		"tailf-ncs":              "http://tail-f.com/ns/ncs",
		"tailf-ned-cisco-ios":    "urn:ios",
		"tailf-ned-cisco-nx":     "http://tail-f.com/ned/cisco-nx",
		"tailf-ned-cisco-ios-xr": "http://tail-f.com/ned/cisco-iosxr",
	}
	for module, namespace := range expected {
		if got := discovered[module]; got != namespace {
			t.Errorf("discovered[%q] = %q, want %q", module, got, namespace)
		}
		if got := namespaceExceptions[module]; got != namespace {
			t.Errorf("namespaceExceptions[%q] = %q, want %q", module, got, namespace)
		}
	}
	if _, ok := discovered["urn:ietf:params:netconf:base:1.1"]; ok {
		t.Error("capability without a module parameter must be ignored")
	}
}

func TestSetFromXPath_NsoDeviceConfig(t *testing.T) {
	BuildNamespaceMap(nsoNedCapabilities)

	body := netconf.Body{}
	body = SetFromXPath(body, "/tailf-ncs:devices/device[name='c1']/config/tailf-ned-cisco-ios:hostname", "R1")
	xml := body.Res()

	for _, want := range []string{
		`<devices xmlns="http://tail-f.com/ns/ncs">`,
		`<name>c1</name>`,
		`<hostname xmlns="urn:ios">R1</hostname>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("missing %s in %s", want, xml)
		}
	}
}

func TestSetFromXPath_NsoDeviceGroupLeafList(t *testing.T) {
	body := netconf.Body{}
	xpath := "/tailf-ncs:devices/device-group[name='G1']"
	body = SetFromXPath(body, xpath, nil)
	body = AppendFromXPath(body, xpath+"/device-name", "c1")
	body = AppendFromXPath(body, xpath+"/device-name", "c2")
	xml := body.Res()

	if got := strings.Count(xml, "<device-name>"); got != 2 {
		t.Errorf("expected 2 device-name entries, got %d: %s", got, xml)
	}
	if got := strings.Count(xml, "<name>G1</name>"); got != 1 {
		t.Errorf("expected key leaf exactly once, got %d: %s", got, xml)
	}
}

func TestRemoveFromXPath_NsoDevice(t *testing.T) {
	body := RemoveFromXPath(netconf.Body{}, "/tailf-ncs:devices/device[name='c1']")
	xml := body.Res()

	if !strings.Contains(xml, `<device operation="remove"><name>c1</name></device>`) {
		t.Errorf("unexpected remove body: %s", xml)
	}
}

func TestGetXpathFilter_Nso(t *testing.T) {
	filter := GetXpathFilter("/tailf-ncs:devices/device[name='c1']/config/tailf-ned-cisco-ios:interface/GigabitEthernet[name='1/0/1']")
	want := "/devices/device[name='c1']/config/interface/GigabitEthernet[name='1/0/1']"
	if filter.Content != want {
		t.Errorf("filter = %q, want %q", filter.Content, want)
	}
}

func TestGetFromXPath_NsoResponse(t *testing.T) {
	// Trimmed get-config reply from NSO for a device with the IOS CLI NED.
	reply := `<rpc-reply><data><devices xmlns="http://tail-f.com/ns/ncs"><device><name>c1</name>` +
		`<address>10.0.0.1</address><config><hostname xmlns="urn:ios">R1</hostname>` +
		`<interface xmlns="urn:ios"><Loopback><name>0</name><description>Router ID</description></Loopback>` +
		`<Loopback><name>100</name><description>Management</description></Loopback></interface>` +
		`</config></device></devices></data></rpc-reply>`
	res := xmldot.Get(reply, "rpc-reply")

	base := "data/tailf-ncs:devices/device[name='c1']"
	cases := map[string]string{
		base + "/address": "10.0.0.1",
		base + "/config/tailf-ned-cisco-ios:hostname":                                   "R1",
		base + "/config/tailf-ned-cisco-ios:interface/Loopback[name='100']/description": "Management",
	}
	for xpath, want := range cases {
		if got := GetFromXPath(res, xpath).String(); got != want {
			t.Errorf("GetFromXPath(%q) = %q, want %q", xpath, got, want)
		}
	}
}

func TestGetFromXPath_KeyedEntryAmongSiblings(t *testing.T) {
	reply := `<rpc-reply><data><devices xmlns="http://tail-f.com/ns/ncs"><authgroups><group><name>g</name>` +
		`<umap><local-user>u1</local-user><remote-name>r1</remote-name></umap>` +
		`<umap><local-user>u2</local-user><remote-name>r2</remote-name></umap>` +
		`</group></authgroups></devices></data></rpc-reply>`
	res := xmldot.Get(reply, "rpc-reply")

	entry := GetFromXPath(res, "data/tailf-ncs:devices/authgroups/group[name='g']/umap[local-user='u2']")
	if entry.IsArray() {
		t.Fatalf("expected a single list entry, got array: %s", entry.Raw)
	}
	if got := entry.Get("remote-name").String(); got != "r2" {
		t.Errorf("remote-name = %q, want r2", got)
	}
}
