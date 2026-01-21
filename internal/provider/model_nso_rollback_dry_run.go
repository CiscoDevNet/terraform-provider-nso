package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidwall/gjson"
)

type RollbackDryRun struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	RollbackId types.Int64  `tfsdk:"rollback_id"`
	Result     types.String `tfsdk:"result"`
}

func (data RollbackDryRun) getPath() string {
	return "tailf-rollback:rollback-files/apply-rollback-file"
}

func (data RollbackDryRun) toBody(ctx context.Context) string {
	payload := map[string]interface{}{
		"tailf-rollback:input": map[string]interface{}{
			"fixed-number": fmt.Sprintf("%d", data.RollbackId.ValueInt64()),
			"tailf-ncs-rollback:dry-run": map[string]interface{}{
				"outformat": "cli-c",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}

	return string(jsonData)
}

func (data *RollbackDryRun) fromBody(ctx context.Context, res gjson.Result) {
	result := extractRollbackDryRunResult(res.String())
	data.Result = types.StringValue(result)
}

func extractRollbackDryRunResult(responseBody string) string {
	parsed := gjson.Parse(responseBody)

	output := parsed.Get("tailf-rollback:output")
	if output.Exists() {
		cliResult := output.Get("cli-c")
		if cliResult.Exists() {
			localNode := cliResult.Get("local-node")
			if localNode.Exists() {
				data := localNode.Get("data")
				if data.Exists() {
					return formatRollbackDryRunOutput(data.String())
				}
			}
		}
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(responseBody), &jsonData); err == nil {
		if formatted, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
			return string(formatted)
		}
	}

	return responseBody
}

func formatRollbackDryRunOutput(data string) string {
	if data == "" {
		return "No rollback changes detected - rollback would not modify any configuration"
	}

	header := "=== NSO Rollback Dry-Run Result ===\n"
	header += "The following configuration would be applied during rollback:\n"
	header += "(This shows the commands that would revert the changes)\n\n"

	return header + data
}

func (data RollbackDryRun) isPostMode() bool {
	return true
}

func (data RollbackDryRun) getQueryParams() map[string]string {
	return make(map[string]string)
}
