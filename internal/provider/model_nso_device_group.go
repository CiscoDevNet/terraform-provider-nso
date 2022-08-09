// Code generated by "gen/generator.go"; DO NOT EDIT.

package provider

import (
	"context"
	"regexp"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/terraform-provider-nso/internal/provider/helpers"
	"github.com/tidwall/sjson"
	"github.com/tidwall/gjson"
)
type DeviceGroup struct {
	Instance types.String `tfsdk:"instance"`
	Id       types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	DeviceNames types.List `tfsdk:"device_names"`
	DeviceGroups types.List `tfsdk:"device_groups"`
}

func (data DeviceGroup) getPath() string {
	return fmt.Sprintf("tailf-ncs:devices/device-group=%v", url.QueryEscape(fmt.Sprintf("%v", data.Name.Value)))
}

// if last path element has a key -> remove it
func (data DeviceGroup) getPathShort() string {
	path := data.getPath()
	re := regexp.MustCompile(`(.*)=[^\/]*$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) <= 1 {
		return path
	}
	return matches[1]
}

func (data DeviceGroup) toBody(ctx context.Context) string {
	body := `{"` + helpers.LastElement(data.getPath()) + `":{}}`
	if !data.Name.Null && !data.Name.Unknown {
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"name", data.Name.Value)
	}
	if !data.DeviceNames.Null && !data.DeviceNames.Unknown {
		var values []string
		data.DeviceNames.ElementsAs(ctx, &values, false)
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"device-name", values)
	}
	if !data.DeviceGroups.Null && !data.DeviceGroups.Unknown {
		var values []string
		data.DeviceGroups.ElementsAs(ctx, &values, false)
		body, _ = sjson.Set(body, helpers.LastElement(data.getPath())+"."+"device-group", values)
	}
	return body
}

func (data *DeviceGroup) updateFromBody(ctx context.Context, res gjson.Result) {
	prefix := helpers.LastElement(data.getPath()) + "."
	if res.Get(helpers.LastElement(data.getPath())).IsArray() {
		prefix += "0."
	}
	if value := res.Get(prefix+"name"); value.Exists() {
		data.Name.Value = value.String()
	} else {
		data.Name.Null = true
	}
	if value := res.Get(prefix+"device-name"); value.Exists() {
		data.DeviceNames.Elems = helpers.GetValueSlice(value.Array())
	} else {
		data.DeviceNames.Null = true
	}
	if value := res.Get(prefix+"device-group"); value.Exists() {
		data.DeviceGroups.Elems = helpers.GetValueSlice(value.Array())
	} else {
		data.DeviceGroups.Null = true
	}
}

func (data *DeviceGroup) fromBody(ctx context.Context, res gjson.Result) {
	prefix := helpers.LastElement(data.getPath()) + "."
	if res.Get(helpers.LastElement(data.getPath())).IsArray() {
		prefix += "0."
	}
	if value := res.Get(prefix+"device-name"); value.Exists() {
		data.DeviceNames.Elems = helpers.GetValueSlice(value.Array())
		data.DeviceNames.Null = false
	}
	if value := res.Get(prefix+"device-group"); value.Exists() {
		data.DeviceGroups.Elems = helpers.GetValueSlice(value.Array())
		data.DeviceGroups.Null = false
	}
}

func (data *DeviceGroup) setUnknownValues() {
	if data.Instance.Unknown {
		data.Instance.Unknown = false
		data.Instance.Null = true
	}
	if data.Id.Unknown {
		data.Id.Unknown = false
		data.Id.Null = true
	}
	if data.Name.Unknown {
		data.Name.Unknown = false
		data.Name.Null = true
	}
	if data.DeviceNames.Unknown {
		data.DeviceNames.Unknown = false
		data.DeviceNames.Null = true
	}
	if data.DeviceGroups.Unknown {
		data.DeviceGroups.Unknown = false
		data.DeviceGroups.Null = true
	}
}

func (data *DeviceGroup) getDeletedListItems(ctx context.Context, state DeviceGroup) []string {
	deletedListItems := make([]string, 0)
	var stateDeviceNames []string
	state.DeviceNames.ElementsAs(ctx, &stateDeviceNames, false)
	for _, stateValue := range stateDeviceNames {
		found := false
		var dataDeviceNames []string
		data.DeviceNames.ElementsAs(ctx, &dataDeviceNames, false)
		for _, dataValue := range dataDeviceNames {
			if stateValue == dataValue {
				found = true
				break
			}
		}
		if !found {
			deletedListItems = append(deletedListItems, state.getPath()+"/device-name="+stateValue)
		}
	}
	var stateDeviceGroups []string
	state.DeviceGroups.ElementsAs(ctx, &stateDeviceGroups, false)
	for _, stateValue := range stateDeviceGroups {
		found := false
		var dataDeviceGroups []string
		data.DeviceGroups.ElementsAs(ctx, &dataDeviceGroups, false)
		for _, dataValue := range dataDeviceGroups {
			if stateValue == dataValue {
				found = true
				break
			}
		}
		if !found {
			deletedListItems = append(deletedListItems, state.getPath()+"/device-group="+stateValue)
		}
	}
	return deletedListItems
}

func (data *DeviceGroup) getEmptyLeafsDelete(ctx context.Context) []string {
	emptyLeafsDelete := make([]string, 0)
	return emptyLeafsDelete
}
