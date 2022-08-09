package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-restconf"
	"github.com/netascode/terraform-provider-nso/internal/provider/helpers"
)

type dataSourceDeviceConfigType struct{}

func (t dataSourceDeviceConfigType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Retrieves a config part of an NSO device.",

		Attributes: map[string]tfsdk.Attribute{
			"instance": {
				MarkdownDescription: "An instance name from the provider configuration.",
				Type:                types.StringType,
				Optional:            true,
			},
			"id": {
				MarkdownDescription: "The RESTCONF path of the retrieved configuration.",
				Type:                types.StringType,
				Computed:            true,
			},
			"device": {
				MarkdownDescription: "An NSO device name.",
				Type:                types.StringType,
				Required:            true,
			},
			"path": {
				MarkdownDescription: "A RESTCONF/YANG config path, e.g. `tailf-ned-cisco-ios:access-list/access-list=1`.",
				Type:                types.StringType,
				Optional:            true,
			},
			"attributes": {
				MarkdownDescription: "Map of key-value pairs which represents the attributes and its values.",
				Type:                types.MapType{ElemType: types.StringType},
				Computed:            true,
			},
		},
	}, nil
}

func (t dataSourceDeviceConfigType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return dataSourceDeviceConfig{
		provider: provider,
	}, diags
}

type dataSourceDeviceConfig struct {
	provider provider
}

func (d dataSourceDeviceConfig) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var config, state DeviceConfigDataSource

	// Read config
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := "tailf-ncs:devices/device=" + config.Device.Value + "/config"
	if config.Path.Value != "" {
		path = "tailf-ncs:devices/device=" + config.Device.Value + "/config/" + config.Path.Value
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Beginning Read", path))

	res, err := d.provider.clients[config.Instance.Value].GetData(path, restconf.Query("content", "config"))
	if res.StatusCode == 404 {
		state.Attributes.Elems = map[string]attr.Value{}
	} else {
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to retrieve object, got error: %s", err))
			return
		}

		state.Path.Value = path
		state.Id.Value = path

		attributes := make(map[string]attr.Value)

		for attr, value := range res.Res.Get(helpers.LastElement(path)).Map() {
			// handle empty maps
			if value.IsObject() && len(value.Map()) == 0 {
				attributes[attr] = types.String{Value: ""}
			} else if value.Raw == "[null]" {
				attributes[attr] = types.String{Value: ""}
			} else {
				attributes[attr] = types.String{Value: value.String()}
			}
		}
		state.Attributes.Elems = attributes
		state.Attributes.ElemType = types.StringType
	}

	tflog.Debug(ctx, fmt.Sprintf("%s: Read finished successfully", path))

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
