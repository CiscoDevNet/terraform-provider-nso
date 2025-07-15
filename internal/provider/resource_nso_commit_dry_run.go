package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/go-restconf"
)

var _ resource.Resource = &CommitDryRunResource{}

func NewCommitDryRunResource() resource.Resource {
	return &CommitDryRunResource{}
}

type CommitDryRunResource struct {
	clients map[string]*restconf.Client
}

func (r *CommitDryRunResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_commit_dry_run"
}

func (r *CommitDryRunResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{

		MarkdownDescription: "Performs a commit dry-run operation using RESTCONF PATCH on `/restconf/data/tailf-ncs:devices?dry-run` to preview changes without applying them to NSO devices.",

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
			"config_data": schema.StringAttribute{
				MarkdownDescription: "The configuration data to apply (as JSON string).",
				Required:            true,
			},
			"result": schema.StringAttribute{
				MarkdownDescription: "The result of the dry-run operation.",
				Computed:            true,
			},
		},
	}
}

func (r *CommitDryRunResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

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

func (r *CommitDryRunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CommitDryRun

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
	queryParams := data.getQueryParams()
	if len(queryParams) > 0 {
		pathWithParams += "?"
		first := true
		for k, v := range queryParams {
			if !first {
				pathWithParams += "&"
			}
			if v == "" {
				pathWithParams += k
			} else {
				pathWithParams += k + "=" + v
			}
			first = false
		}
	}

	res, err := client.PatchData(pathWithParams, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to perform dry-run operation, got error: %s", err))
		return
	}

	data.fromBody(ctx, res.Res)
	data.Id = types.StringValue(fmt.Sprintf("dry-run-%d", time.Now().Unix()))

	if !data.Result.IsNull() && data.Result.ValueString() != "" {
		resp.Diagnostics.AddWarning(
			"NSO Dry-Run Result - Configuration Changes Preview",
			data.Result.ValueString(),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitDryRunResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CommitDryRun

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitDryRunResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CommitDryRun

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
	queryParams := data.getQueryParams()
	if len(queryParams) > 0 {
		pathWithParams += "?"
		first := true
		for k, v := range queryParams {
			if !first {
				pathWithParams += "&"
			}
			if v == "" {
				pathWithParams += k
			} else {
				pathWithParams += k + "=" + v
			}
			first = false
		}
	}

	res, err := client.PatchData(pathWithParams, body)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to perform dry-run operation, got error: %s", err))
		return
	}

	data.fromBody(ctx, res.Res)
	data.Id = types.StringValue(fmt.Sprintf("dry-run-%d", time.Now().Unix()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitDryRunResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CommitDryRun

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *CommitDryRunResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
