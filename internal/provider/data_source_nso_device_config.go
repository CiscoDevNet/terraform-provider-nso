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
	"fmt"

	"github.com/CiscoDevNet/terraform-provider-nso/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-restconf"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DeviceConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &DeviceConfigDataSource{}
)

func NewDeviceConfigDataSource() datasource.DataSource {
	return &DeviceConfigDataSource{}
}

type DeviceConfigDataSource struct {
	data *NsoProviderData
}

func (d *DeviceConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_config"
}

func (d *DeviceConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Retrieves a config part of an NSO device.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The RESTCONF path of the retrieved configuration.",
				Computed:            true,
			},
			"device": schema.StringAttribute{
				MarkdownDescription: "An NSO device name.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "A RESTCONF/YANG config path, e.g. `tailf-ned-cisco-ios:access-list/access-list=1`.",
				Optional:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "Map of key-value pairs which represents the YANG leafs and its values.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *DeviceConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.data = req.ProviderData.(*NsoProviderData)
}

func (d *DeviceConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config, state DeviceConfigData

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, ok := d.data.Instances[config.Instance.ValueString()]
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", config.Instance.ValueString()))
		return
	}

	restconfPath := "tailf-ncs:devices/device=" + config.Device.ValueString() + "/config"
	if config.Path.ValueString() != "" {
		restconfPath = "tailf-ncs:devices/device=" + config.Device.ValueString() + "/config/" + config.Path.ValueString()
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", restconfPath))

	if d.data.Transport == "netconf" {
		locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, false)
		if locked {
			defer instance.NetconfOpMutex.Unlock()
		}
		defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

		xpath := helpers.ConvertRestconfPathToXPath(restconfPath)
		filter := helpers.GetXpathFilter(xpath)
		res, err := instance.NetconfClient.GetConfig(ctx, "running", filter)
		if helpers.IsGetConfigResponseEmpty(&res) {
			state.Attributes = types.MapValueMust(types.StringType, map[string]attr.Value{})
		} else {
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to retrieve object (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
				return
			}

			state.Path = types.StringValue(restconfPath)
			state.Id = types.StringValue(restconfPath)

			attributes := make(map[string]attr.Value)
			dataPrefix := "data" + xpath
			for key, val := range res.Res.Get(dataPrefix).Map() {
				if !val.Exists() || val.String() == "" {
					attributes[key] = types.StringValue("")
				} else {
					attributes[key] = types.StringValue(val.String())
				}
			}
			state.Attributes = types.MapValueMust(types.StringType, attributes)
		}
	} else {
		res, err := instance.RestconfClient.GetData(restconfPath, restconf.Query("content", "config"))
		if res.StatusCode == 404 {
			state.Attributes = types.MapValueMust(types.StringType, map[string]attr.Value{})
		} else {
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to retrieve object, got error: %s", err))
				return
			}

			state.Path = types.StringValue(restconfPath)
			state.Id = types.StringValue(restconfPath)

			attributes := make(map[string]attr.Value)

			for attr, value := range res.Res.Get(helpers.LastElement(restconfPath)).Map() {
				if value.IsObject() && len(value.Map()) == 0 {
					attributes[attr] = types.StringValue("")
				} else if value.Raw == "[null]" {
					attributes[attr] = types.StringValue("")
				} else {
					attributes[attr] = types.StringValue(value.String())
				}
			}
			state.Attributes = types.MapValueMust(types.StringType, attributes)
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", restconfPath))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
