// Copyright © 2025 Cisco Systems, Inc. and its affiliates.
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

package helpers

import (
	"fmt"
	"strings"
)

// ConvertXPathToRestconfPath converts an XPath like
// "/tailf-ncs:devices/device[name='%s']"
// to a RESTCONF-style path like
// "tailf-ncs:devices/device=%s".
func ConvertXPathToRestconfPath(xpath string) string {
	xpath = strings.TrimPrefix(xpath, "/")
	segments := splitXPathSegmentsForConversion(xpath)
	var result []string

	for _, seg := range segments {
		if idx := strings.Index(seg, "["); idx != -1 {
			element := seg[:idx]
			predicates := seg[idx:]
			var keys []string
			for predicates != "" {
				start := strings.Index(predicates, "[")
				end := strings.Index(predicates, "]")
				if start == -1 || end == -1 {
					break
				}
				pred := predicates[start+1 : end]
				if eqIdx := strings.Index(pred, "="); eqIdx != -1 {
					val := pred[eqIdx+1:]
					val = strings.Trim(val, "'\"")
					keys = append(keys, val)
				}
				predicates = predicates[end+1:]
			}
			if len(keys) > 0 {
				result = append(result, fmt.Sprintf("%s=%s", element, strings.Join(keys, ",")))
			} else {
				result = append(result, element)
			}
		} else {
			result = append(result, seg)
		}
	}

	return strings.Join(result, "/")
}

// nsoListKeyNames maps well-known NSO YANG list element names to their key leaf names.
// Used by ConvertRestconfPathToXPath when the RESTCONF path uses =value syntax
// and we need to produce [keyname='value'] predicates.
var nsoListKeyNames = map[string]string{
	"device":       "name",
	"device-group": "name",
	"authgroup":    "name",
	"service":      "name",
	"template":     "name",
	"package":      "name",
	"user":         "name",
	"nacm":         "name",
	"rule-list":    "name",
	"rule":         "name",
	"group":        "name",
}

// ConvertRestconfPathToXPath converts a RESTCONF-style path like
// "tailf-ncs:devices/device=test-device01/config/interface"
// to an XPath like
// "/tailf-ncs:devices/device[name='test-device01']/config/interface".
//
// For list elements where the key name is not in the lookup table,
// the key name defaults to "name". To discover key names from resource
// attributes, use ConvertRestconfPathToXPathWithAttrs instead.
func ConvertRestconfPathToXPath(restconfPath string) string {
	return ConvertRestconfPathToXPathWithAttrs(restconfPath, nil)
}

// ConvertRestconfPathToXPathWithAttrs converts a RESTCONF-style path to XPath,
// using the provided attributes map to discover list key names. When a segment
// has "element=value" and the value matches an attribute value, that attribute
// name is used as the key leaf name. Falls back to nsoListKeyNames then "name".
func ConvertRestconfPathToXPathWithAttrs(restconfPath string, attributes map[string]string) string {
	restconfPath = strings.TrimPrefix(restconfPath, "/")
	segments := strings.Split(restconfPath, "/")
	var result []string

	for _, seg := range segments {
		if eqIdx := strings.Index(seg, "="); eqIdx != -1 {
			element := seg[:eqIdx]
			keyValues := seg[eqIdx+1:]

			// Strip namespace prefix to look up the key name
			cleanElement := element
			if colonIdx := strings.Index(element, ":"); colonIdx != -1 {
				cleanElement = element[colonIdx+1:]
			}

			keyName, ok := nsoListKeyNames[cleanElement]
			if !ok && attributes != nil {
				// Try to discover key name from attributes by matching the value
				values := strings.Split(keyValues, ",")
				if len(values) == 1 {
					for attrName, attrVal := range attributes {
						if attrVal == values[0] && !strings.Contains(attrName, "/") {
							keyName = attrName
							ok = true
							break
						}
					}
				}
			}
			if !ok {
				keyName = "name"
			}

			// Handle composite keys (comma-separated)
			values := strings.Split(keyValues, ",")
			predicates := ""
			for _, v := range values {
				predicates += fmt.Sprintf("[%s='%s']", keyName, v)
			}
			result = append(result, element+predicates)
		} else {
			result = append(result, seg)
		}
	}

	return "/" + strings.Join(result, "/")
}

func splitXPathSegmentsForConversion(xpath string) []string {
	var segments []string
	depth := 0
	start := 0
	for i, ch := range xpath {
		switch ch {
		case '[':
			depth++
		case ']':
			depth--
		case '/':
			if depth == 0 {
				if i > start {
					segments = append(segments, xpath[start:i])
				}
				start = i + 1
			}
		}
	}
	if start < len(xpath) {
		segments = append(segments, xpath[start:])
	}
	return segments
}
