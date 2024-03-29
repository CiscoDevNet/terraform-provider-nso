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

// Code generated by "gen/generator.go"; DO NOT EDIT.

package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/CiscoDevNet/terraform-provider-nso/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type Device struct {
	Instance     types.String `tfsdk:"instance"`
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Address      types.String `tfsdk:"address"`
	Port         types.Int64  `tfsdk:"port"`
	Authgroup    types.String `tfsdk:"authgroup"`
	AdminState   types.String `tfsdk:"admin_state"`
	NetconfNetId types.String `tfsdk:"netconf_net_id"`
	CliNedId     types.String `tfsdk:"cli_ned_id"`
}

type DeviceData struct {
	Instance     types.String `tfsdk:"instance"`
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Address      types.String `tfsdk:"address"`
	Port         types.Int64  `tfsdk:"port"`
	Authgroup    types.String `tfsdk:"authgroup"`
	AdminState   types.String `tfsdk:"admin_state"`
	NetconfNetId types.String `tfsdk:"netconf_net_id"`
	CliNedId     types.String `tfsdk:"cli_ned_id"`
}

func (data Device) getPath() string {
	return fmt.Sprintf("tailf-ncs:devices/device=%v", url.QueryEscape(fmt.Sprintf("%v", data.Name.ValueString())))
}

func (data DeviceData) getPath() string {
	return fmt.Sprintf("tailf-ncs:devices/device=%v", url.QueryEscape(fmt.Sprintf("%v", data.Name.ValueString())))
}

// if last path element has a key -> remove it
func (data Device) getPathShort() string {
	path := data.getPath()
	re := regexp.MustCompile(`(.*)=[^\/]*$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) <= 1 {
		return path
	}
	return matches[1]
}

func (data Device) toBody(ctx context.Context) string {
	body := `{"` + helpers.LastElement(data.getPath()) + `":{}}`
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"name", data.Name.ValueString())
	}
	if !data.Address.IsNull() && !data.Address.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"address", data.Address.ValueString())
	}
	if !data.Port.IsNull() && !data.Port.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"port", strconv.FormatInt(data.Port.ValueInt64(), 10))
	}
	if !data.Authgroup.IsNull() && !data.Authgroup.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"authgroup", data.Authgroup.ValueString())
	}
	if !data.AdminState.IsNull() && !data.AdminState.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"state.admin-state", data.AdminState.ValueString())
	}
	if !data.NetconfNetId.IsNull() && !data.NetconfNetId.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"device-type.netconf.ned-id", data.NetconfNetId.ValueString())
	}
	if !data.CliNedId.IsNull() && !data.CliNedId.IsUnknown() {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"device-type.cli.ned-id", data.CliNedId.ValueString())
	}
	return body
}

func (data *Device) updateFromBody(ctx context.Context, res gjson.Result) {
	prefix := helpers.LastElement(data.getPath()) + "."
	if res.Get(helpers.LastElement(data.getPath())).IsArray() {
		prefix += "0."
	}
	if value := res.Get(prefix + "name"); value.Exists() && !data.Name.IsNull() {
		data.Name = types.StringValue(value.String())
	} else {
		data.Name = types.StringNull()
	}
	if value := res.Get(prefix + "address"); value.Exists() && !data.Address.IsNull() {
		data.Address = types.StringValue(value.String())
	} else {
		data.Address = types.StringNull()
	}
	if value := res.Get(prefix + "port"); value.Exists() && !data.Port.IsNull() {
		data.Port = types.Int64Value(value.Int())
	} else {
		data.Port = types.Int64Null()
	}
	if value := res.Get(prefix + "authgroup"); value.Exists() && !data.Authgroup.IsNull() {
		data.Authgroup = types.StringValue(value.String())
	} else {
		data.Authgroup = types.StringNull()
	}
	if value := res.Get(prefix + "state.admin-state"); value.Exists() && !data.AdminState.IsNull() {
		data.AdminState = types.StringValue(value.String())
	} else {
		data.AdminState = types.StringNull()
	}
	if value := res.Get(prefix + "device-type.netconf.ned-id"); value.Exists() && !data.NetconfNetId.IsNull() {
		data.NetconfNetId = types.StringValue(value.String())
	} else {
		data.NetconfNetId = types.StringNull()
	}
	if value := res.Get(prefix + "device-type.cli.ned-id"); value.Exists() && !data.CliNedId.IsNull() {
		data.CliNedId = types.StringValue(value.String())
	} else {
		data.CliNedId = types.StringNull()
	}
}

func (data *DeviceData) fromBody(ctx context.Context, res gjson.Result) {
	prefix := helpers.LastElement(data.getPath()) + "."
	if res.Get(helpers.LastElement(data.getPath())).IsArray() {
		prefix += "0."
	}
	if value := res.Get(prefix + "address"); value.Exists() {
		data.Address = types.StringValue(value.String())
	}
	if value := res.Get(prefix + "port"); value.Exists() {
		data.Port = types.Int64Value(value.Int())
	}
	if value := res.Get(prefix + "authgroup"); value.Exists() {
		data.Authgroup = types.StringValue(value.String())
	}
	if value := res.Get(prefix + "state.admin-state"); value.Exists() {
		data.AdminState = types.StringValue(value.String())
	}
	if value := res.Get(prefix + "device-type.netconf.ned-id"); value.Exists() {
		data.NetconfNetId = types.StringValue(value.String())
	}
	if value := res.Get(prefix + "device-type.cli.ned-id"); value.Exists() {
		data.CliNedId = types.StringValue(value.String())
	}
}

func (data *Device) getDeletedListItems(ctx context.Context, state Device) []string {
	deletedListItems := make([]string, 0)
	return deletedListItems
}

func (data *Device) getEmptyLeafsDelete(ctx context.Context) []string {
	emptyLeafsDelete := make([]string, 0)
	return emptyLeafsDelete
}

func (data *Device) getDeletePaths(ctx context.Context) []string {
	var deletePaths []string
	if !data.Address.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/address", data.getPath()))
	}
	if !data.Port.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/port", data.getPath()))
	}
	if !data.Authgroup.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/authgroup", data.getPath()))
	}
	if !data.AdminState.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/state/admin-state", data.getPath()))
	}
	if !data.NetconfNetId.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/device-type/netconf/ned-id", data.getPath()))
	}
	if !data.CliNedId.IsNull() {
		deletePaths = append(deletePaths, fmt.Sprintf("%v/device-type/cli/ned-id", data.getPath()))
	}
	return deletePaths
}
