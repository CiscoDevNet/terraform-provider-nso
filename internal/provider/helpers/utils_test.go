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
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConfigured(t *testing.T) {
	cases := map[string]struct {
		value attr.Value
		want  bool
	}{
		"null int":       {types.Int64Null(), false},
		"unknown int":    {types.Int64Unknown(), false},
		"set int":        {types.Int64Value(0), true},
		"empty leaf on":  {types.BoolValue(true), true},
		"empty leaf off": {types.BoolValue(false), false},
		"null bool":      {types.BoolNull(), false},
		"set string":     {types.StringValue(""), true},
	}
	for name, c := range cases {
		if got := Configured(c.value); got != c.want {
			t.Errorf("%s: Configured = %v, want %v", name, got, c.want)
		}
	}
}

func TestDecimalEqual(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"-121.886330", "-121.88633", true},
		{"37.338207", "37.338207", true},
		{"1.0", "1", true},
		{"0.10", ".1", true},
		{"-121.88633", "-121.88634", false},
		{"1", "-1", false},
		{"", "0", false},
		{"abc", "abc", false},
	}
	for _, c := range cases {
		if got := DecimalEqual(c.a, c.b); got != c.want {
			t.Errorf("DecimalEqual(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
