package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-restconf"
	"github.com/netascode/terraform-provider-nso/internal/provider/helpers"
)

type resourceDeviceConfigType struct{}

func (t resourceDeviceConfigType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Manages a config part of an NSO device.",

		Attributes: map[string]tfsdk.Attribute{
			"instance": {
				MarkdownDescription: "An instance name from the provider configuration.",
				Type:                types.StringType,
				Optional:            true,
			},
			"id": {
				MarkdownDescription: "The RESTCONF path.",
				Type:                types.StringType,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"device": {
				MarkdownDescription: "An NSO device name.",
				Type:                types.StringType,
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"path": {
				MarkdownDescription: "A RESTCONF/YANG config path, e.g. `tailf-ned-cisco-ios:access-list/access-list=1`.",
				Type:                types.StringType,
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"delete": {
				MarkdownDescription: "Delete object during destroy operation. Default value is `true`.",
				Type:                types.BoolType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					helpers.BooleanDefaultModifier(true),
				},
			},
			"attributes": {
				Type:                types.MapType{ElemType: types.StringType},
				MarkdownDescription: "Map of key-value pairs which represents the attributes and its values. Nested attributes (in YANG containers) can be specified using `/` as delimiter.",
				Optional:            true,
				Computed:            true,
			},
			"lists": {
				MarkdownDescription: "YANG lists.",
				Optional:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"name": {
						MarkdownDescription: "YANG list name. Nested lists (in YANG containers) can be specified using `/` as delimiter.",
						Type:                types.StringType,
						Required:            true,
					},
					"key": {
						MarkdownDescription: "YANG list key attribute.",
						Type:                types.StringType,
						Optional:            true,
					},
					"items": {
						MarkdownDescription: "Items of YANG lists.",
						Optional:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"attributes": {
								Type:                types.MapType{ElemType: types.StringType},
								MarkdownDescription: "Map of key-value pairs which represents the attributes and its values. Nested attributes (in YANG containers) can be specified using `/` as delimiter.",
								Optional:            true,
								Computed:            true,
							},
						}),
					},
					"values": {
						MarkdownDescription: "YANG leaf-list values.",
						Type:                types.ListType{ElemType: types.StringType},
						Optional:            true,
					},
				}),
			},
		},
	}, nil
}

func (t resourceDeviceConfigType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return resourceDeviceConfig{
		provider: provider,
	}, diags
}

type resourceDeviceConfig struct {
	provider provider
}

func (r resourceDeviceConfig) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var plan DeviceConfig

	// Read plan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", plan.getPath()))

	body := plan.toBody(ctx)
	res, err := r.provider.clients[plan.Instance.Value].PatchData(plan.getPathShort(), body)
	if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
		_, err = r.provider.clients[plan.Instance.Value].PutData(plan.getPath(), body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
		return
	}

	plan.Id = types.String{Value: plan.getPath()}
	plan.Attributes.Unknown = false

	tflog.Debug(ctx, fmt.Sprintf("%s: Create finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r resourceDeviceConfig) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var state DeviceConfig

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", state.getPath()))

	res, err := r.provider.clients[state.Instance.Value].GetData(state.getPath(), restconf.Query("content", "config"))
	if res.StatusCode == 404 {
		state.Attributes.Elems = map[string]attr.Value{}
		state.Lists = make([]DeviceConfigList, 0)
	} else {
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to read object, got error: %s", err))
			return
		}

		state.fromBody(ctx, res.Res)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", state.getPath()))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r resourceDeviceConfig) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan, state DeviceConfig

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

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Update", plan.getPath()))

	body := plan.toBody(ctx)
	res, err := r.provider.clients[plan.Instance.Value].PatchData(plan.getPathShort(), body)
	if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
		_, err = r.provider.clients[plan.Instance.Value].PutData(plan.getPath(), body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
		return
	}

	deletedListItems := plan.getDeletedListItems(ctx, state)
	tflog.Debug(ctx, fmt.Sprintf("List items to delete: %+v", deletedListItems))

	for _, i := range deletedListItems {
		res, err := r.provider.clients[state.Instance.Value].DeleteData(i)
		if err != nil && res.StatusCode != 404 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Update finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r resourceDeviceConfig) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var state DeviceConfig

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Delete", state.getPath()))

	if state.Delete.Value {
		res, err := r.provider.clients[state.Instance.Value].DeleteData(state.getPath())
		if err != nil && res.StatusCode != 404 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", state.getPath()))

	resp.State.RemoveResource(ctx)
}

func (r resourceDeviceConfig) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Import", req.ID))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	tflog.Debug(ctx, fmt.Sprintf("%s: Import finished successfully", req.ID))
}
