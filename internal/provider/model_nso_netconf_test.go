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
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/xmldot"
)

func TestYangDataSourceAttributes(t *testing.T) {
	reply := `<rpc-reply><data><ssh xmlns="http://tail-f.com/ns/ncs">` +
		`<host-key-verification>reject-unknown</host-key-verification><empty/>` +
		`<container><leaf>1</leaf></container><leaf-list>a</leaf-list><leaf-list>b</leaf-list>` +
		`</ssh></data></rpc-reply>`
	got := yangDataSourceAttributes(xmldot.Get(reply, "rpc-reply"), "/tailf-ncs:ssh").Elements()

	want := map[string]string{"host-key-verification": "reject-unknown", "empty": ""}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != types.StringValue(v) {
			t.Errorf("attributes[%q] = %v, want %q", k, got[k], v)
		}
	}
}

func testYangList(name, key string, items []map[string]string, values []string) YangList {
	l := YangList{Name: types.StringValue(name), Key: types.StringNull(), Values: types.ListNull(types.StringType)}
	if key != "" {
		l.Key = types.StringValue(key)
	}
	for _, item := range items {
		m := map[string]attr.Value{}
		for k, v := range item {
			m[k] = types.StringValue(v)
		}
		l.Items = append(l.Items, types.MapValueMust(types.StringType, m))
	}
	if values != nil {
		v := []attr.Value{}
		for _, s := range values {
			v = append(v, types.StringValue(s))
		}
		l.Values = types.ListValueMust(types.StringType, v)
	}
	return l
}

func TestYangToBodyXML(t *testing.T) {
	ctx := context.Background()
	attrs := types.MapValueMust(types.StringType, map[string]attr.Value{"default-map/remote-name": types.StringValue("admin")})
	lists := []YangList{
		testYangList("umap", "local-user", []map[string]string{{"local-user": "u1", "remote-name": "r1"}}, nil),
	}
	body := yangToBodyXML(ctx, "/tailf-ncs:devices/authgroups/group[name='g1']", attrs, lists)

	for _, want := range []string{
		`<devices xmlns="http://tail-f.com/ns/ncs">`,
		`<name>g1</name>`,
		`<default-map><remote-name>admin</remote-name></default-map>`,
		`<umap><local-user>u1</local-user><remote-name>r1</remote-name></umap>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
}

func TestYangAddDeletedItemsXML(t *testing.T) {
	ctx := context.Background()
	xpath := "/tailf-ncs:devices/device-group[name='g1']"
	state := []YangList{
		testYangList("device-name", "", nil, []string{"c1", "c2"}),
		testYangList("umap", "local-user", []map[string]string{{"local-user": "u1"}, {"local-user": "u2"}}, nil),
	}
	plan := []YangList{
		testYangList("device-name", "", nil, []string{"c2"}),
		testYangList("umap", "local-user", []map[string]string{{"local-user": "u2"}}, nil),
	}
	body := yangAddDeletedItemsXML(ctx, "", xpath, plan, state)

	for _, want := range []string{
		`<device-name operation="remove">c1</device-name>`,
		`<umap operation="remove"><local-user>u1</local-user></umap>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %s in %s", want, body)
		}
	}
	for _, unwanted := range []string{">c2<", ">u2<"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("unexpected %s in %s", unwanted, body)
		}
	}
}

func TestSplitImportInstance(t *testing.T) {
	cases := []struct{ id, path, instance string }{
		{"/tailf-ncs:ssh", "/tailf-ncs:ssh", ""},
		{"/tailf-ncs:ssh,prod", "/tailf-ncs:ssh", "prod"},
		{"/tailf-ncs:devices/device[name='a,b']", "/tailf-ncs:devices/device[name='a,b']", ""},
		{"c1,tailf-ned-cisco-ios:interface/Loopback[name='0'],prod", "c1,tailf-ned-cisco-ios:interface/Loopback[name='0']", "prod"},
	}
	for _, c := range cases {
		path, instance := splitImportInstance(c.id)
		if path != c.path || instance != c.instance {
			t.Errorf("splitImportInstance(%q) = (%q, %q), want (%q, %q)", c.id, path, instance, c.path, c.instance)
		}
	}
}
