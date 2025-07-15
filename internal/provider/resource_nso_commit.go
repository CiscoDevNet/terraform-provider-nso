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

var _ resource.Resource = &CommitResource{}

func NewCommitResource() resource.Resource {
	return &CommitResource{}
}

type CommitResource struct {
	clients map[string]*restconf.Client
}

func (r *CommitResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_commit"
}

func (r *CommitResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs a commit operation using RESTCONF PATCH on `/restconf/data/tailf-ncs:devices` to apply configuration changes to NSO devices.",

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
				MarkdownDescription: "The result of the commit operation.",
				Computed:            true,
			},
			"rollback_id": schema.Int64Attribute{
				MarkdownDescription: "The rollback ID returned by NSO for this commit operation.",
				Computed:            true,
			},
		},
	}
}

func (r *CommitResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CommitResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data Commit
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.performCommit(ctx, &data, resp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data Commit

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data Commit

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.performCommit(ctx, &data, resp)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CommitResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data Commit
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddWarning(
		"NSO Commit Resource Deleted",
		"The commit resource has been removed from Terraform state, but the configuration remains applied on NSO devices. To remove the configuration, you would need to apply a configuration that removes or replaces the committed changes.",
	)
}

func (r *CommitResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CommitResource) performCommit(ctx context.Context, data *Commit, resp interface{}) error {
	var client *restconf.Client
	if data.Instance.IsNull() {
		client = r.clients[""]
	} else {
		client = r.clients[data.Instance.ValueString()]
	}
	if client == nil {
		return fmt.Errorf("failed to retrieve client")
	}

	body := data.toBody(ctx)
	pathWithParams := r.buildPathWithParams(data)

	res, err := client.PatchData(pathWithParams, body)
	if err != nil {
		return fmt.Errorf("failed to perform commit operation, got error: %s", err)
	}

	data.fromBody(ctx, res.Res)
	data.Id = types.StringValue(fmt.Sprintf("commit-%d", time.Now().Unix()))

	if !data.Result.IsNull() && data.Result.ValueString() != "" {
		warningMsg := fmt.Sprintf("Configuration has been successfully applied to NSO devices:\n\n%s", data.Result.ValueString())

		if !data.RollbackId.IsNull() && data.RollbackId.ValueInt64() != 0 {
			warningMsg += fmt.Sprintf("\n\nRollback ID captured: %d", data.RollbackId.ValueInt64())
		}

		r.addCommitWarning(resp, warningMsg)
	}

	return nil
}

func (r *CommitResource) buildPathWithParams(data *Commit) string {
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
	return pathWithParams
}

func (r *CommitResource) addCommitWarning(resp interface{}, warningMsg string) {
	title := "NSO Commit Result - Configuration Applied"

	switch r := resp.(type) {
	case *resource.CreateResponse:
		r.Diagnostics.AddWarning(title, warningMsg)
	case *resource.UpdateResponse:
		r.Diagnostics.AddWarning(title, warningMsg)
	}
}
