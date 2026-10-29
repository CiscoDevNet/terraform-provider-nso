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
	"strings"

	"github.com/CiscoDevNet/terraform-provider-nso/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-netconf"
)

var _ resource.Resource = &NetconfResource{}
var _ resource.ResourceWithImportState = &NetconfResource{}
var _ resource.ResourceWithValidateConfig = &NetconfResource{}

func NewNetconfResource() resource.Resource {
	return &NetconfResource{}
}

type NetconfResource struct {
	data *NsoProviderData
}

func (r *NetconfResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_netconf"
}

func (r *NetconfResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Manages NSO configuration via NETCONF. This resource manages part of a YANG model, addressed by an XPath. It is able to read the state and therefore reconcile configuration drift.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The XPath of the object.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "An absolute XPath, e.g. `/tailf-ncs:devices/device-group[name='GROUP1']`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"delete": schema.BoolAttribute{
				MarkdownDescription: "Delete object during destroy operation. Default value is `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "Map of key-value pairs which represents the YANG leafs and its values. Keys are XPaths relative to `path`.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
			},
			"lists": schema.ListNestedAttribute{
				MarkdownDescription: "YANG lists.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "YANG list name, relative to `path`.",
							Required:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "YANG list key attribute. In case of multiple keys, those should be separated by a comma (`,`). Required when `items` is used.",
							Optional:            true,
						},
						"items": schema.ListAttribute{
							MarkdownDescription: "List of maps of key-value pairs which represents the YANG leafs and its values.",
							Optional:            true,
							ElementType:         types.MapType{ElemType: types.StringType},
						},
						"values": schema.ListAttribute{
							MarkdownDescription: "YANG leaf-list values.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *NetconfResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data Netconf
	if diags := req.Config.Get(ctx, &data); diags.HasError() {
		return
	}
	resp.Diagnostics.Append(validateYangLists(ctx, data.Lists)...)
}

func (r *NetconfResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.data = req.ProviderData.(*NsoProviderData)
}

func (r *NetconfResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Netconf

	// Read plan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, ok := r.data.Instances[plan.Instance.ValueString()]
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", plan.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", plan.getXPath()))

	locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
	if locked {
		defer instance.NetconfOpMutex.Unlock()
	}
	defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

	if err := instance.Prepare(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	body := yangToBodyXML(ctx, plan.getXPath(), plan.Attributes, plan.Lists)
	if err := helpers.EditConfig(ctx, instance.NetconfClient, body, instance.AutoCommit); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object, got error: %s", err))
		return
	}

	plan.Id = types.StringValue(plan.getXPath())
	if plan.Attributes.IsUnknown() {
		plan.Attributes = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Create finished successfully", plan.getXPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetconfResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Netconf

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, ok := r.data.Instances[state.Instance.ValueString()]
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", state.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", state.getXPath()))

	locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, false)
	if locked {
		defer instance.NetconfOpMutex.Unlock()
	}
	defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

	if err := instance.Prepare(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	res, err := instance.NetconfClient.GetConfig(ctx, "running", helpers.GetXpathFilter(state.getXPath()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to read object, got error: %s", helpers.FormatNetconfError(err)))
		return
	}

	if helpers.IsGetConfigResponseEmpty(&res) && helpers.IsListPath(state.getXPath()) {
		tflog.Debug(ctx, fmt.Sprintf("%s: Object does not exist", state.getXPath()))
		resp.State.RemoveResource(ctx)
		return
	}

	state.Attributes, state.Lists = yangFromBodyXML(ctx, res.Res, state.getXPath(), state.Attributes, state.Lists)

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", state.getXPath()))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *NetconfResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state Netconf

	// Read plan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read state
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, ok := r.data.Instances[plan.Instance.ValueString()]
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", plan.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Update", plan.getXPath()))

	locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
	if locked {
		defer instance.NetconfOpMutex.Unlock()
	}
	defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

	if err := instance.Prepare(); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	body := yangToBodyXML(ctx, plan.getXPath(), plan.Attributes, plan.Lists)
	body = yangAddDeletedItemsXML(ctx, body, plan.getXPath(), plan.Lists, state.Lists)
	if err := helpers.EditConfig(ctx, instance.NetconfClient, body, instance.AutoCommit); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to update object, got error: %s", err))
		return
	}

	if plan.Attributes.IsUnknown() {
		plan.Attributes = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Update finished successfully", plan.getXPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetconfResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Netconf

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instance, ok := r.data.Instances[state.Instance.ValueString()]
	if !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", state.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Delete", state.getXPath()))

	if state.Delete.ValueBool() {
		locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
		if locked {
			defer instance.NetconfOpMutex.Unlock()
		}
		defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

		if err := instance.Prepare(); err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}

		body := helpers.RemoveFromXPath(netconf.Body{}, state.getXPath()).Res()
		if err := helpers.EditConfig(ctx, instance.NetconfClient, body, instance.AutoCommit); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", state.getXPath()))

	resp.State.RemoveResource(ctx)
}

func (r *NetconfResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	xpath, instance := splitImportInstance(req.ID)
	if !strings.HasPrefix(xpath, "/") {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: '<path>' or '<path>,<instance>', where <path> is an absolute XPath. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), xpath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), xpath)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("delete"), true)...)
	if instance != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance"), instance)...)
	}
}
