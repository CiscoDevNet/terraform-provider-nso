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
	_ datasource.DataSource              = &NetconfDataSource{}
	_ datasource.DataSourceWithConfigure = &NetconfDataSource{}
)

func NewNetconfDataSource() datasource.DataSource {
	return &NetconfDataSource{}
}

type NetconfDataSource struct {
	data *NsoProviderData
}

func (d *NetconfDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_netconf"
}

func (d *NetconfDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "This data source can read the configuration of an NSO object via NETCONF, addressed by an XPath.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The XPath of the retrieved object.",
				Computed:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "An absolute XPath, e.g. `/tailf-ncs:devices/device[name='c1']`.",
				Required:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "Map of key-value pairs which represents the YANG leafs directly below the object and their values.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *NetconfDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.data = req.ProviderData.(*NsoProviderData)
}

func (d *NetconfDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NetconfDataSourceModel

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

	xpath := config.Path.ValueString()

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
