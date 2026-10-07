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
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		MarkdownDescription: "This data source can read a config part of an NSO device via NETCONF.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The XPath of the retrieved object.",
				Computed:            true,
			},
			"device": schema.StringAttribute{
				MarkdownDescription: "An NSO device name.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "An XPath relative to the device config container (`/tailf-ncs:devices/device[name='<device>']/config`), e.g. `tailf-ned-cisco-ios:interface/Loopback[name='0']`.",
				Optional:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "Map of key-value pairs which represents the YANG leafs directly below the object and their values.",
				Computed:            true,
				ElementType:         types.StringType,
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
	var config DeviceConfigDataSourceModel

	// Read config
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

	xpath := deviceConfigXPath(config.Device.ValueString(), config.Path.ValueString())

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", xpath))

	locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, false)
	if locked {
		defer instance.NetconfOpMutex.Unlock()
	}
	defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

	if err := instance.Prepare(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	res, err := instance.NetconfClient.GetConfig(ctx, "running", helpers.GetXpathFilter(xpath))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to retrieve object, got error: %s", helpers.FormatNetconfError(err)))
		return
	}

	config.Id = types.StringValue(xpath)
	config.Attributes = yangDataSourceAttributes(res.Res, xpath)

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", xpath))

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
