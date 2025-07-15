package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/go-restconf"
)

var _ resource.Resource = &RollbackResource{}

func NewRollbackResource() resource.Resource {
	return &RollbackResource{}
}

type RollbackResource struct {
	clients map[string]*restconf.Client
}

func (r *RollbackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rollback"
}

func (r *RollbackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{

		MarkdownDescription: "Performs a rollback operation using RESTCONF POST on `/restconf/data/tailf-rollback:rollback-files/apply-rollback-file` to apply rollback changes to NSO devices.",

		Attributes: map[string]schema.Attribute{
			"instance": schema.StringAttribute{
				MarkdownDescription: "An instance name from the provider configuration.",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The resource identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rollback_id": schema.Int64Attribute{
				MarkdownDescription: "The rollback ID to apply (integer).",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"result": schema.StringAttribute{
				MarkdownDescription: "The result of the rollback operation.",
				Computed:            true,
			},
		},
	}
}

func (r *RollbackResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(map[string]*restconf.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected map[string]*restconf.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clients = clients
}

func (r *RollbackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data Rollback

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var client *restconf.Client
	if data.Instance.IsNull() {
		client = r.clients[""]
	} else {
		client = r.clients[data.Instance.ValueString()]
	}
	if client == nil {
		resp.Diagnostics.AddError("Client Error", "Failed to retrieve client")
		return
	}

	body := data.toBody(ctx)
	pathWithParams := data.getPath()

	res, err := client.PostData(pathWithParams, body)
	if err != nil {

		if strings.Contains(err.Error(), "rollback") && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist")) {
			resp.Diagnostics.AddError("Rollback ID Not Found", fmt.Sprintf("Rollback ID %d was not found. Please check that the rollback ID exists in NSO.", data.RollbackId.ValueInt64()))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to perform rollback operation, got error: %s", err))
		return
	}

	data.fromBody(ctx, res.Res)
	data.Id = types.StringValue(fmt.Sprintf("rollback-%d-%d", data.RollbackId.ValueInt64(), time.Now().Unix()))

	if !data.Result.IsNull() && data.Result.ValueString() != "" {
		resp.Diagnostics.AddWarning(
			"NSO Rollback Result - Configuration Rolled Back",
			fmt.Sprintf("Rollback ID %d has been successfully applied to NSO devices:\n\n%s", data.RollbackId.ValueInt64(), data.Result.ValueString()),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RollbackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data Rollback

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RollbackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data Rollback

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var client *restconf.Client
	if data.Instance.IsNull() {
		client = r.clients[""]
	} else {
		client = r.clients[data.Instance.ValueString()]
	}
	if client == nil {
		resp.Diagnostics.AddError("Client Error", "Failed to retrieve client")
		return
	}

	body := data.toBody(ctx)
	pathWithParams := data.getPath()

	res, err := client.PostData(pathWithParams, body)
	if err != nil {

		if strings.Contains(err.Error(), "rollback") && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist")) {
			resp.Diagnostics.AddError("Rollback ID Not Found", fmt.Sprintf("Rollback ID %d was not found. Please check that the rollback ID exists in NSO.", data.RollbackId.ValueInt64()))
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to perform rollback operation, got error: %s", err))
		return
	}

	data.fromBody(ctx, res.Res)
	data.Id = types.StringValue(fmt.Sprintf("rollback-%d-%d", data.RollbackId.ValueInt64(), time.Now().Unix()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RollbackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data Rollback

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"NSO Rollback Resource Deleted",
		fmt.Sprintf("The rollback resource has been removed from Terraform state. Rollback ID %d has already been applied to NSO devices and cannot be undone through this resource.", data.RollbackId.ValueInt64()),
	)
}

func (r *RollbackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
