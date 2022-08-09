package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/go-restconf"
	"github.com/netascode/terraform-provider-nso/internal/provider/helpers"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DeviceConfig struct {
	Instance   types.String       `tfsdk:"instance"`
	Id         types.String       `tfsdk:"id"`
	Device     types.String       `tfsdk:"device"`
	Path       types.String       `tfsdk:"path"`
	Delete     types.Bool         `tfsdk:"delete"`
	Attributes types.Map          `tfsdk:"attributes"`
	Lists      []DeviceConfigList `tfsdk:"lists"`
}

type DeviceConfigList struct {
	Name   types.String           `tfsdk:"name"`
	Key    types.String           `tfsdk:"key"`
	Items  []DeviceConfigListItem `tfsdk:"items"`
	Values types.List             `tfsdk:"values"`
}

type DeviceConfigListItem struct {
	Attributes types.Map `tfsdk:"attributes"`
}

type DeviceConfigDataSource struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	Device     types.String `tfsdk:"device"`
	Path       types.String `tfsdk:"path"`
	Attributes types.Map    `tfsdk:"attributes"`
}

func (data DeviceConfig) getPath() string {
	if data.Path.Value != "" {
		return "tailf-ncs:devices/device=" + data.Device.Value + "/config/" + data.Path.Value
	} else {
		return "tailf-ncs:devices/device=" + data.Device.Value + "/config"
	}
}

// if last path element has a key -> remove it
func (data DeviceConfig) getPathShort() string {
	path := data.getPath()
	re := regexp.MustCompile(`(.*)=[^\/]*$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) <= 1 {
		return path
	}
	return matches[1]
}

func (data DeviceConfig) toBody(ctx context.Context) string {
	root := helpers.LastElement(data.getPath())
	if root == "tailf-ncs:config" {
		root = "config"
	}
	body := `{"` + root + `":{}}`

	var attributes map[string]string
	data.Attributes.ElementsAs(ctx, &attributes, false)

	for attr, value := range attributes {
		body, _ = sjson.Set(body, root+"."+attr, value)
	}
	for i := range data.Lists {
		if len(data.Lists[i].Items) > 0 {
			body, _ = sjson.Set(body, root+"."+data.Lists[i].Name.Value, []interface{}{})
			for ii := range data.Lists[i].Items {
				var listAttributes map[string]string
				data.Lists[i].Items[ii].Attributes.ElementsAs(ctx, &listAttributes, false)
				attrs := restconf.Body{}
				for attr, value := range listAttributes {
					attrs = attrs.Set(attr, value)
				}
				body, _ = sjson.SetRaw(body, root+"."+data.Lists[i].Name.Value+".-1", attrs.Str)
			}
		} else if len(data.Lists[i].Values.Elems) > 0 {
			var values []string
			data.Lists[i].Values.ElementsAs(ctx, &values, false)
			body, _ = sjson.Set(body, root+"."+data.Lists[i].Name.Value, values)
		}
	}

	return body
}

func (data *DeviceConfig) fromBody(ctx context.Context, res gjson.Result) {
	prefix := helpers.LastElement(data.getPath()) + "."
	if res.Get(helpers.LastElement(data.getPath())).IsArray() {
		prefix += "0."
	}
	for attr := range data.Attributes.Elems {
		value := res.Get(prefix + attr)
		if !value.Exists() ||
			(value.IsObject() && len(value.Map()) == 0) ||
			value.Raw == "[null]" {

			data.Attributes.Elems[attr] = types.String{Value: ""}
		} else {
			data.Attributes.Elems[attr] = types.String{Value: value.String()}
		}
	}

	for i := range data.Lists {
		if len(data.Lists[i].Items) > 0 {
			for ii := range data.Lists[i].Items {
				for attr := range data.Lists[i].Items[ii].Attributes.Elems {
					key := data.Lists[i].Key.Value
					v, _ := data.Lists[i].Items[ii].Attributes.Elems[key].ToTerraformValue(ctx)
					var keyValue string
					v.As(&keyValue)
					jsonPath := fmt.Sprintf(`%s%s.#(%s=="%s").%s`, prefix, data.Lists[i].Name.Value, key, keyValue, attr)
					value := res.Get(jsonPath)
					if !value.Exists() ||
						(value.IsObject() && len(value.Map()) == 0) ||
						value.Raw == "[null]" {

						data.Lists[i].Items[ii].Attributes.Elems[attr] = types.String{Value: ""}
					} else {
						data.Lists[i].Items[ii].Attributes.Elems[attr] = types.String{Value: value.String()}
					}
				}
			}
		} else if len(data.Lists[i].Values.Elems) > 0 {
			values := res.Get(prefix + data.Lists[i].Name.Value)
			if values.IsArray() {
				data.Lists[i].Values.Elems = helpers.GetValueSlice(values.Array())
			}
		}
	}
}

func (data *DeviceConfig) getDeletedListItems(ctx context.Context, state DeviceConfig) []string {
	deletedListItems := make([]string, 0)
	for l := range state.Lists {
		name := state.Lists[l].Name.Value
		key := state.Lists[l].Key.Value
		var dataList DeviceConfigList
		for _, dl := range data.Lists {
			if dl.Name.Value == name {
				dataList = dl
			}
		}
		if len(state.Lists[l].Items) > 0 {
			// check if state item is also included in plan, if not delete item
			for i := range state.Lists[l].Items {
				var slia map[string]string
				state.Lists[l].Items[i].Attributes.ElementsAs(ctx, &slia, false)
				if slia[key] == "" {
					continue
				}
				found := false
				for dli := range dataList.Items {
					var dlia map[string]string
					dataList.Items[dli].Attributes.ElementsAs(ctx, &dlia, false)
					if dlia[key] == slia[key] {
						found = true
						break
					}
				}
				if !found {
					deletedListItems = append(deletedListItems, state.getPath()+"/"+name+"="+slia[key])
				}
			}
		} else if len(state.Lists[l].Values.Elems) > 0 {
			var slv []string
			state.Lists[l].Values.ElementsAs(ctx, &slv, false)
			// check if state value is also included in plan, if not delete value from list
			for _, stateValue := range slv {
				found := false
				var dlv []string
				dataList.Values.ElementsAs(ctx, &dlv, false)
				for _, dataValue := range dlv {
					if stateValue == dataValue {
						found = true
						break
					}
				}
				if !found {
					deletedListItems = append(deletedListItems, state.getPath()+"/"+name+"="+stateValue)
				}
			}
		}
	}
	return deletedListItems
}
