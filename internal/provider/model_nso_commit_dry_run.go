package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidwall/gjson"
)

type CommitDryRun struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	ConfigData types.String `tfsdk:"config_data"`
	Result     types.String `tfsdk:"result"`
}

func (data CommitDryRun) getPath() string {
	return "tailf-ncs:devices"
}

func (data CommitDryRun) toBody(ctx context.Context) string {
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

func (data *CommitDryRun) fromBody(ctx context.Context, res gjson.Result) {
	result := extractDryRunResult(res.String())
	data.Result = types.StringValue(result)
}

func extractDryRunResult(responseBody string) string {
	parsed := gjson.Parse(responseBody)

	cliResult := parsed.Get("dry-run-result.cli.local-node.data")
	if cliResult.Exists() {
		diffData := cliResult.String()
		if diffData != "" {
			return formatDryRunDiff(diffData)
		}
	}

	cliCResult := parsed.Get("dry-run-result.cli-c.local-node.data")
	if cliCResult.Exists() {
		diffData := cliCResult.String()
		if diffData != "" {
			return formatDryRunDiff(diffData)
		}
	}

	dryRunResult := parsed.Get("dry-run-result")
	if dryRunResult.Exists() {
		resultXML := dryRunResult.Get("result-xml")
		if resultXML.Exists() {
			localNode := resultXML.Get("local-node")
			if localNode.Exists() {
				data := localNode.Get("data")
				if data.Exists() {
					diffData := data.String()
					if diffData != "" {
						return formatDryRunDiff(diffData)
					}
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

func formatDryRunDiff(diffData string) string {
	if diffData == "" {
		return "No changes detected - configuration is already in desired state"
	}

	header := "=== NSO Dry-Run Diff - Configuration Changes Preview ===\n"
	header += "The following shows what would be applied to NSO devices:\n"
	header += "(+ indicates additions, - indicates deletions, ! indicates modifications)\n\n"

	return header + diffData
}

func (data CommitDryRun) getQueryParams() map[string]string {
	params := make(map[string]string)

	params["dry-run"] = "cli-c"

	return params
}
