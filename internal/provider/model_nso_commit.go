package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidwall/gjson"
)

type Commit struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	ConfigData types.String `tfsdk:"config_data"`
	Result     types.String `tfsdk:"result"`
	RollbackId types.Int64  `tfsdk:"rollback_id"`
}

func (data Commit) getPath() string {
	return "tailf-ncs:devices"
}

func (data Commit) toBody(ctx context.Context) string {
	if !data.ConfigData.IsNull() {
		configStr := data.ConfigData.ValueString()

		parsed := gjson.Parse(configStr)

		if devicesData := parsed.Get("tailf-ncs:devices"); devicesData.Exists() {
			return fmt.Sprintf(`{"tailf-ncs:devices":%s}`, devicesData.Raw)
		}

		return fmt.Sprintf(`{"tailf-ncs:devices":%s}`, configStr)
	}
	return "{}"
}

func (data *Commit) fromBody(ctx context.Context, res gjson.Result) {
	result := extractCommitResult(res.String())
	data.Result = types.StringValue(result)

	rollbackId := extractRollbackIdAsInt(res.String())
	data.RollbackId = types.Int64Value(rollbackId)
}

func extractCommitResult(responseBody string) string {
	parsed := gjson.Parse(responseBody)

	if parsed.Exists() {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(responseBody), &jsonData); err == nil {
			if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				return string(formatted)
			}
		}
	}

	if responseBody == "" || responseBody == "{}" {
		return "Configuration successfully applied to NSO devices"
	}

	return responseBody
}

func extractRollbackIdAsInt(responseBody string) int64 {
	parsed := gjson.Parse(responseBody)

	return parsed.Get("tailf-restconf:result.rollback.id").Int()
}

func (data Commit) getQueryParams() map[string]string {
	params := make(map[string]string)
	params["rollback-id"] = "true"
	return params
}
