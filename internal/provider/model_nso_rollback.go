package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidwall/gjson"
)

type Rollback struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	RollbackId types.Int64  `tfsdk:"rollback_id"`
	Result     types.String `tfsdk:"result"`
}

func (data Rollback) getPath() string {
	return "tailf-rollback:rollback-files/apply-rollback-file"
}

func (data Rollback) toBody(ctx context.Context) string {
	payload := map[string]interface{}{
		"tailf-rollback:input": map[string]interface{}{
			"fixed-number": fmt.Sprintf("%d", data.RollbackId.ValueInt64()),
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}

	return string(jsonData)
}

func (data *Rollback) fromBody(ctx context.Context, res gjson.Result) {
	result := extractRollbackResult(res.String())
	data.Result = types.StringValue(result)
}

func extractRollbackResult(responseBody string) string {
	parsed := gjson.Parse(responseBody)

	output := parsed.Get("tailf-rollback:output")
	if output.Exists() {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(output.String()), &jsonData); err == nil {
			if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				return string(formatted)
			}
		}
		return output.String()
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(responseBody), &jsonData); err == nil {
		if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
			return string(formatted)
		}
	}

	if responseBody == "" || responseBody == "{}" {
		return "Rollback operation completed successfully"
	}

	return responseBody
}

func (data Rollback) isPostMode() bool {
	return true
}

func (data Rollback) getQueryParams() map[string]string {
	return make(map[string]string)
}
