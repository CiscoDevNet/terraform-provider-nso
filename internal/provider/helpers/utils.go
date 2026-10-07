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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/xmldot"
)

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func GetXmlValueSlice(result []xmldot.Result) []attr.Value {
	v := make([]attr.Value, len(result))
	for r := range result {
		v[r] = types.StringValue(result[r].String())
	}
	return v
}

// GetStringListXML converts a slice of xmldot.Result to a Terraform types.List of strings.
func GetStringListXML(result []xmldot.Result) types.List {
	v := make([]attr.Value, len(result))
	for r := range result {
		v[r] = types.StringValue(result[r].String())
	}
	return types.ListValueMust(types.StringType, v)
}

// GetInt64ListXML converts a slice of xmldot.Result to a Terraform types.List of int64.
func GetInt64ListXML(result []xmldot.Result) types.List {
	v := make([]attr.Value, len(result))
	for r := range result {
		v[r] = types.Int64Value(result[r].Int())
	}
	return types.ListValueMust(types.Int64Type, v)
}

// RemoveEmptyStrings filters out empty strings from a slice.
func RemoveEmptyStrings(s []string) []string {
	var r []string
	for _, v := range s {
		if v != "" {
			r = append(r, v)
		}
	}
	return r
}
