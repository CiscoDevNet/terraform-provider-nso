package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-restconf"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &DeviceConfigResource{}
var _ resource.ResourceWithImportState = &DeviceConfigResource{}

func NewDeviceConfigResource() resource.Resource {
	return &DeviceConfigResource{}
}

type DeviceConfigResource struct {
	clients map[string]*restconf.Client
}

func (r *DeviceConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_config"
}

func (r *DeviceConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Manages a config part of an NSO device.",

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
			"device": schema.StringAttribute{
				MarkdownDescription: "An NSO device name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "A RESTCONF path.",
				Optional:            true,
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

func (r *DeviceConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(map[string]*restconf.Client)
}

func (r *DeviceConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeviceConfig

	// Read plan
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, ok := r.clients[plan.Instance.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", plan.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Create", plan.getPath()))

	body := plan.toBody(ctx)
	res, err := r.clients[plan.Instance.ValueString()].PatchData(plan.getPathShort(), body)
	if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
		_, err = r.clients[plan.Instance.ValueString()].PutData(plan.getPath(), body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
		return
	}

	plan.Id = types.StringValue(plan.getPath())

	if plan.Attributes.IsUnknown() {
		plan.Attributes = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Create finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *DeviceConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeviceConfig

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, ok := r.clients[state.Instance.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", state.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", state.getPath()))

	res, err := r.clients[state.Instance.ValueString()].GetData(state.getPath(), restconf.Query("content", "config"))
	if res.StatusCode == 404 {
		state.Attributes = types.MapNull(types.StringType)
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

func (r *DeviceConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	if _, ok := r.clients[plan.Instance.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", plan.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Update", plan.getPath()))

	body := plan.toBody(ctx)
	res, err := r.clients[plan.Instance.ValueString()].PatchData(plan.getPathShort(), body)
	if len(res.Errors.Error) > 0 && res.Errors.Error[0].ErrorMessage == "patch to a nonexistent resource" {
		_, err = r.clients[plan.Instance.ValueString()].PutData(plan.getPath(), body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to configure object (PATCH), got error: %s", err))
		return
	}

	deletedListItems := plan.getDeletedListItems(ctx, state)
	tflog.Debug(ctx, fmt.Sprintf("List items to delete: %+v", deletedListItems))

	for _, i := range deletedListItems {
		res, err := r.clients[state.Instance.ValueString()].DeleteData(i)
		if err != nil && res.StatusCode != 404 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Update finished successfully", plan.getPath()))

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *DeviceConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeviceConfig

	// Read state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, ok := r.clients[state.Instance.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(path.Root("instance"), "Invalid instance", fmt.Sprintf("Instance '%s' does not exist in provider configuration.", state.Instance.ValueString()))
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Delete", state.getPath()))

	if state.Delete.ValueBool() {
		res, err := r.clients[state.Instance.ValueString()].DeleteData(state.getPath())
		if err != nil && res.StatusCode != 404 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete object, got error: %s", err))
			return
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Delete finished successfully", state.getPath()))

	resp.State.RemoveResource(ctx)
}

func (r *DeviceConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Import", req.ID))

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	tflog.Debug(ctx, fmt.Sprintf("%s: Import finished successfully", req.ID))
}
