// Copyright © 2023 Cisco Systems, Inc. and its affiliates.
// All rights reserved.
//
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://mozilla.org/MPL/2.0/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/CiscoDevNet/terraform-provider-nso/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/netascode/go-netconf"
	"github.com/netascode/xmldot"
)

// YangList is a YANG list or leaf-list managed by nso_netconf and nso_device_config.
type YangList struct {
	Name   types.String `tfsdk:"name"`
	Key    types.String `tfsdk:"key"`
	Items  []types.Map  `tfsdk:"items"`
	Values types.List   `tfsdk:"values"`
}

type Netconf struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	Path       types.String `tfsdk:"path"`
	Delete     types.Bool   `tfsdk:"delete"`
	Attributes types.Map    `tfsdk:"attributes"`
	Lists      []YangList   `tfsdk:"lists"`
}

type NetconfDataSourceModel struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	Path       types.String `tfsdk:"path"`
	Attributes types.Map    `tfsdk:"attributes"`
}

func (data Netconf) getXPath() string {
	return data.Path.ValueString()
}

type DeviceConfig struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	Device     types.String `tfsdk:"device"`
	Path       types.String `tfsdk:"path"`
	Delete     types.Bool   `tfsdk:"delete"`
	Attributes types.Map    `tfsdk:"attributes"`
	Lists      []YangList   `tfsdk:"lists"`
}

type DeviceConfigDataSourceModel struct {
	Instance   types.String `tfsdk:"instance"`
	Id         types.String `tfsdk:"id"`
	Device     types.String `tfsdk:"device"`
	Path       types.String `tfsdk:"path"`
	Attributes types.Map    `tfsdk:"attributes"`
}

func deviceConfigXPath(device, path string) string {
	xpath := fmt.Sprintf("/tailf-ncs:devices/device[name='%s']/config", device)
	if path != "" {
		xpath += "/" + strings.TrimPrefix(path, "/")
	}
	return xpath
}

func (data DeviceConfig) getXPath() string {
	return deviceConfigXPath(data.Device.ValueString(), data.Path.ValueString())
}

// splitImportInstance splits an optional trailing ",<instance>" from an import
// identifier. XPath predicate values may contain commas, so a trailing segment
// only counts as an instance name if it cannot be part of an XPath.
func splitImportInstance(id string) (string, string) {
	i := strings.LastIndex(id, ",")
	if i < 0 {
		return id, ""
	}
	instance := id[i+1:]
	if instance == "" || strings.ContainsAny(instance, "/[]'\"=") {
		return id, ""
	}
	return id[:i], instance
}

// validateYangLists checks that every list item contains its key leaf(s).
func validateYangLists(ctx context.Context, lists []YangList) diag.Diagnostics {
	var diags diag.Diagnostics
	for l := range lists {
		if len(lists[l].Items) == 0 {
			continue
		}
		keys := listKeys(lists[l])
		if len(keys) == 0 {
			diags.AddAttributeError(path.Root("lists").AtListIndex(l).AtName("key"), "Invalid List Configuration",
				fmt.Sprintf("List '%s' has items and therefore requires a key.", lists[l].Name.ValueString()))
			continue
		}
		for i := range lists[l].Items {
			var item map[string]string
			lists[l].Items[i].ElementsAs(ctx, &item, false)
			for _, key := range keys {
				if _, ok := item[key]; !ok {
					diags.AddAttributeError(path.Root("lists").AtListIndex(l).AtName("items").AtListIndex(i), "Invalid List Configuration",
						fmt.Sprintf("Key '%s' is missing in list item.", key))
				}
			}
		}
	}
	return diags
}

// listItemPredicates returns the XPath key predicates (e.g. `[seq='10']`) of a list item.
func listItemPredicates(keys []string, item map[string]string) string {
	predicates := ""
	for _, key := range keys {
		predicates += fmt.Sprintf("[%s='%s']", key, item[key])
	}
	return predicates
}

func listKeys(list YangList) []string {
	return helpers.RemoveEmptyStrings(strings.Split(list.Key.ValueString(), ","))
}

// yangToBodyXML builds an edit-config body that merges the attributes and lists below xpath.
func yangToBodyXML(ctx context.Context, xpath string, attributes types.Map, lists []YangList) string {
	body := netconf.Body{}

	var attrs map[string]string
	attributes.ElementsAs(ctx, &attrs, false)

	if len(attrs) == 0 && len(lists) == 0 {
		return helpers.SetFromXPath(body, xpath, nil).Res()
	}

	for attr, value := range attrs {
		body = helpers.SetFromXPath(body, xpath+"/"+attr, value)
	}
	for i := range lists {
		listXPath := xpath + "/" + lists[i].Name.ValueString()
		if len(lists[i].Items) > 0 {
			keys := listKeys(lists[i])
			for ii := range lists[i].Items {
				var item map[string]string
				lists[i].Items[ii].ElementsAs(ctx, &item, false)
				itemXPath := listXPath + listItemPredicates(keys, item)
				body = helpers.SetFromXPath(body, itemXPath, nil)
				for attr, value := range item {
					if helpers.Contains(keys, attr) {
						continue
					}
					body = helpers.SetFromXPath(body, itemXPath+"/"+attr, value)
				}
			}
		} else if len(lists[i].Values.Elements()) > 0 {
			var values []string
			lists[i].Values.ElementsAs(ctx, &values, false)
			for _, value := range values {
				body = helpers.AppendFromXPath(body, listXPath, value)
			}
		}
	}

	return body.Res()
}

// yangFromBodyXML refreshes the configured attributes and lists from a get-config reply.
func yangFromBodyXML(ctx context.Context, res xmldot.Result, xpath string, attributes types.Map, lists []YangList) (types.Map, []YangList) {
	root := "data" + xpath

	if !attributes.IsNull() && !attributes.IsUnknown() {
		attrs := attributes.Elements()
		for attr := range attrs {
			attrs[attr] = types.StringValue(helpers.GetFromXPath(res, root+"/"+attr).String())
		}
		attributes = types.MapValueMust(types.StringType, attrs)
	}

	for i := range lists {
		listXPath := root + "/" + lists[i].Name.ValueString()
		if len(lists[i].Items) > 0 {
			keys := listKeys(lists[i])
			for ii := range lists[i].Items {
				var item map[string]string
				lists[i].Items[ii].ElementsAs(ctx, &item, false)
				r := helpers.GetFromXPath(res, listXPath+listItemPredicates(keys, item))

				itemAttrs := lists[i].Items[ii].Elements()
				for attr := range itemAttrs {
					if helpers.Contains(keys, attr) && !r.Exists() {
						continue
					}
					itemAttrs[attr] = types.StringValue(helpers.GetFromXPath(r, attr).String())
				}
				lists[i].Items[ii] = types.MapValueMust(types.StringType, itemAttrs)
			}
		} else if len(lists[i].Values.Elements()) > 0 {
			values := helpers.GetFromXPath(res, listXPath)
			lists[i].Values = types.ListValueMust(types.StringType, helpers.GetXmlValueSlice(values.Array()))
		}
	}

	return attributes, lists
}

// yangAddDeletedItemsXML adds remove operations for list items and leaf-list values
// which are in state but no longer in the plan.
func yangAddDeletedItemsXML(ctx context.Context, body string, xpath string, plan, state []YangList) string {
	b := netconf.NewBody(body)

	for _, stateList := range state {
		var planList YangList
		for _, pl := range plan {
			if pl.Name.ValueString() == stateList.Name.ValueString() {
				planList = pl
			}
		}
		listXPath := xpath + "/" + stateList.Name.ValueString()

		if len(stateList.Items) > 0 {
			keys := listKeys(stateList)
			for _, stateItem := range stateList.Items {
				var si map[string]string
				stateItem.ElementsAs(ctx, &si, false)

				found := false
				for _, planItem := range planList.Items {
					var pi map[string]string
					planItem.ElementsAs(ctx, &pi, false)
					found = true
					for _, key := range keys {
						if pi[key] != si[key] {
							found = false
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					b = helpers.RemoveFromXPath(b, listXPath+listItemPredicates(keys, si))
				}
			}
		} else if len(stateList.Values.Elements()) > 0 {
			var stateValues, planValues []string
			stateList.Values.ElementsAs(ctx, &stateValues, false)
			planList.Values.ElementsAs(ctx, &planValues, false)
			for _, v := range stateValues {
				if !helpers.Contains(planValues, v) {
					b = helpers.RemoveFromXPath(b, fmt.Sprintf("%s[.='%s']", listXPath, v))
				}
			}
		}
	}

	return b.Res()
}

// yangDataSourceAttributes returns all leafs directly below xpath from a get-config reply.
func yangDataSourceAttributes(res xmldot.Result, xpath string) types.Map {
	attributes := make(map[string]attr.Value)
	for name, value := range helpers.GetFromXPath(res, "data"+xpath).Map() {
		// Skip text nodes, XML attributes, leaf-lists and containers
		if value.IsArray() || name == "%" || strings.HasPrefix(name, "@") || strings.Contains(value.Raw, "<") {
			continue
		}
		attributes[name] = types.StringValue(value.String())
	}
	return types.MapValueMust(types.StringType, attributes)
}
