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
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-netconf"
	"github.com/netascode/go-restconf"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &RestconfResource{}
var _ resource.ResourceWithImportState = &RestconfResource{}

func NewRestconfResource() resource.Resource {
	return &RestconfResource{}
}

type RestconfResource struct {
	data *NsoProviderData
}

func (r *RestconfResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restconf"
}

func (r *RestconfResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Manages NSO configuration via RESTCONF calls. This resource manages part of a YANG model. It is able to read the state and therefore reconcile configuration drift.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The RESTCONF path.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "A RESTCONF path.",
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
				MarkdownDescription: "Map of key-value pairs which represents the YANG leafs and its values.",
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
							MarkdownDescription: "YANG list name.",
							Required:            true,
						},
						"key": schema.StringAttribute{
							MarkdownDescription: "YANG list key attribute. In case of multiple keys, those should be separated by a comma (`,`).",
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

func (r *RestconfResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.data = req.ProviderData.(*NsoProviderData)
}

func (r *RestconfResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Restconf

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

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", plan.getPath()))

	if r.data.Transport == "netconf" {
		locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
		if locked {
			defer instance.NetconfOpMutex.Unlock()
		}
		defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

		body := plan.toBodyXML(ctx)
		if err := helpers.EditConfig(ctx, instance.NetconfClient, body, instance.AutoCommit); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
			return
		}
	} else {
		body := plan.toBody(ctx)
		res, err := instance.RestconfClient.PatchData(plan.getPathShort(), body)
		if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
			_, err = instance.RestconfClient.PutData(plan.getPath(), body)
		}
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
			return
		}
	}

	plan.Id = plan.Path

	if plan.Attributes.IsUnknown() {
		plan.Attributes = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Create finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RestconfResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Restconf

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

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", state.getPath()))

	if r.data.Transport == "netconf" {
		locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, false)
		if locked {
			defer instance.NetconfOpMutex.Unlock()
		}
		defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

		filter := helpers.GetXpathFilter(state.getXPath())
		res, err := instance.NetconfClient.GetConfig(ctx, "running", filter)
		if helpers.IsGetConfigResponseEmpty(&res) {
			state.Attributes = types.MapNull(types.StringType)
			state.Lists = make([]RestconfList, 0)
		} else {
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to read object (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
				return
			}
			state.fromBodyXML(ctx, res.Res)
		}
	} else {
		res, err := instance.RestconfClient.GetData(state.getPath(), restconf.Query("content", "config"))
		if res.StatusCode == 404 {
			state.Attributes = types.MapNull(types.StringType)
			state.Lists = make([]RestconfList, 0)
		} else {
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to read object, got error: %s", err))
				return
			}
			state.fromBody(ctx, res.Res)
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", state.getPath()))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *RestconfResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state Restconf

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Update", plan.getPath()))

	if r.data.Transport == "netconf" {
		locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
		if locked {
			defer instance.NetconfOpMutex.Unlock()
		}
		defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

		body := plan.toBodyXML(ctx)
		if err := helpers.EditConfig(ctx, instance.NetconfClient, body, instance.AutoCommit); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
			return
		}

		deletedListItems := plan.getDeletedListItems(ctx, state)
		tflog.Debug(ctx, fmt.Sprintf("List items to delete: %+v", deletedListItems))
		for _, item := range deletedListItems {
			deleteBody := netconf.Body{}
			deleteBody = helpers.RemoveFromXPath(deleteBody, helpers.ConvertRestconfPathToXPath(item))
			if err := helpers.EditConfig(ctx, instance.NetconfClient, deleteBody.Res(), instance.AutoCommit); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete list item (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
				return
			}
		}
	} else {
		body := plan.toBody(ctx)
		res, err := instance.RestconfClient.PatchData(plan.getPathShort(), body)
		if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
			_, err = instance.RestconfClient.PutData(plan.getPath(), body)
		}
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
			return
		}

		deletedListItems := plan.getDeletedListItems(ctx, state)
		tflog.Debug(ctx, fmt.Sprintf("List items to delete: %+v", deletedListItems))
		for _, i := range deletedListItems {
			res, err := instance.RestconfClient.DeleteData(i)
			if err != nil && res.StatusCode != 404 {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
				return
			}
		}
	}

	if plan.Attributes.IsUnknown() {
		plan.Attributes = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Update finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *RestconfResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Restconf

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

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Delete", state.getPath()))

	if state.Delete.ValueBool() {
		if r.data.Transport == "netconf" {
			locked := helpers.AcquireNetconfLock(&instance.NetconfOpMutex, instance.ReuseConnection, true)
			if locked {
				defer instance.NetconfOpMutex.Unlock()
			}
			defer helpers.CloseNetconfConnection(ctx, instance.NetconfClient, instance.ReuseConnection)

			body := netconf.Body{}
			body = helpers.RemoveFromXPath(body, state.getXPath())
			if err := helpers.EditConfig(ctx, instance.NetconfClient, body.Res(), instance.AutoCommit); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object (NETCONF), got error: %s", helpers.FormatNetconfError(err)))
				return
			}
		} else {
			res, err := instance.RestconfClient.DeleteData(state.getPath())
			if err != nil && res.StatusCode != 404 {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
				return
			}
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", state.getPath()))

	resp.State.RemoveResource(ctx)
}

func (r *RestconfResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Import", req.ID))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	tflog.Debug(ctx, fmt.Sprintf("%s: Import finished successfully", req.ID))
}
