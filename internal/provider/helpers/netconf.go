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
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-netconf"
	"github.com/netascode/xmldot"
)

var (
	xpathPredicateRegex = regexp.MustCompile(`\[[^\]]*\]`)
	namespaceRegex      = regexp.MustCompile(`[a-zA-Z0-9\-]+:`)
	predicatePattern    = regexp.MustCompile(`\[([^\]]+)\]`)
)

// coreNamespaces maps well-known NSO namespace prefixes to their URIs.
// NED module namespaces are discovered dynamically from NETCONF capabilities.
var coreNamespaces = map[string]string{
	"tailf-ncs":         "http://tail-f.com/ns/ncs",
	"tailf-common":      "http://tail-f.com/yang/common",
	"tailf-ncs-devices": "http://tail-f.com/ns/ncs",
}

// namespaceBaseURL is the fallback namespace URL pattern.
const namespaceBaseURL = "http://tail-f.com/ns/ncs/"

// namespaceExceptions is the resolved namespace map, populated at startup
// with coreNamespaces and augmented by BuildNamespaceMap from NETCONF capabilities.
var namespaceExceptions = make(map[string]string)

func init() {
	for k, v := range coreNamespaces {
		namespaceExceptions[k] = v
	}
}

// BuildNamespaceMap parses NETCONF server capabilities to discover NED module
// namespaces dynamically. NSO advertises NED meta modules with capabilities
// like "urn:ios-meta?module=tailf-ned-cisco-ios-meta&revision=...". The config
// module namespace is derived by stripping the "-meta" suffix from both the
// namespace URI and the module name.
func BuildNamespaceMap(capabilities []string) map[string]string {
	discovered := make(map[string]string)

	for _, cap := range capabilities {
		moduleName := ""
		namespace := cap

		if idx := strings.Index(cap, "?"); idx >= 0 {
			namespace = cap[:idx]
			params := cap[idx+1:]
			for _, param := range strings.Split(params, "&") {
				if strings.HasPrefix(param, "module=") {
					moduleName = strings.TrimPrefix(param, "module=")
					break
				}
			}
		}

		if moduleName == "" {
			continue
		}

		// NED meta modules: derive config module name and namespace
		if strings.HasSuffix(moduleName, "-meta") {
			configModule := strings.TrimSuffix(moduleName, "-meta")
			configNamespace := strings.TrimSuffix(namespace, "-meta")
			if strings.TrimSuffix(namespace, "/meta") != namespace {
				configNamespace = strings.TrimSuffix(namespace, "/meta")
			}
			discovered[configModule] = configNamespace
		}

		// Also store the module itself
		discovered[moduleName] = namespace
	}

	// Merge into the global map
	for k, v := range discovered {
		namespaceExceptions[k] = v
	}

	return discovered
}

// AcquireNetconfLock acquires the appropriate lock for a NETCONF operation.
//
// Returns true if lock was acquired, false if not acquired.
func AcquireNetconfLock(opMutex *sync.Mutex, reuseConnection bool, isWrite bool) bool {
	if !reuseConnection {
		opMutex.Lock()
		return true
	}
	if isWrite {
		opMutex.Lock()
		return true
	}
	return false
}

// CloseNetconfConnection safely closes a NETCONF connection if reuse is disabled.
func CloseNetconfConnection(ctx context.Context, client *netconf.Client, reuseConnection bool) {
	if reuseConnection {
		return
	}
	if err := client.Close(); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to close NETCONF connection: %s", err))
	}
}

// FormatNetconfError extracts detailed error information from a NETCONF error.
func FormatNetconfError(err error) string {
	if netconfErr, ok := err.(*netconf.NetconfError); ok {
		var details strings.Builder
		details.WriteString(netconfErr.Message)

		for i, e := range netconfErr.Errors {
			if i == 0 {
				details.WriteString("\n\nError Details:")
			}
			details.WriteString(fmt.Sprintf("\n  [%d] ", i+1))

			if e.ErrorMessage != "" {
				details.WriteString(e.ErrorMessage)
			}

			if e.ErrorPath != "" {
				details.WriteString(fmt.Sprintf(" (path: %s)", e.ErrorPath))
			}

			if e.ErrorType != "" || e.ErrorTag != "" {
				details.WriteString(fmt.Sprintf(" [type=%s, tag=%s]", e.ErrorType, e.ErrorTag))
			}

			if e.ErrorInfo != "" {
				details.WriteString(fmt.Sprintf("\n      Info: %s", e.ErrorInfo))
			}
		}

		return details.String()
	}
	return err.Error()
}

// EditConfig edits the configuration on the NSO instance.
// If the server supports the candidate capability, it edits the candidate datastore
// and commits to running when commit is true.
func EditConfig(ctx context.Context, client *netconf.Client, body string, commit bool) error {
	if err := client.Open(); err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}

	candidate := client.ServerHasCapability("urn:ietf:params:netconf:capability:candidate:1.0")

	if candidate {
		if commit {
			if _, err := client.Lock(ctx, "running"); err != nil {
				return fmt.Errorf("failed to lock running datastore: %s", FormatNetconfError(err))
			}
			defer client.Unlock(ctx, "running")

			if _, err := client.Lock(ctx, "candidate"); err != nil {
				return fmt.Errorf("failed to lock candidate datastore: %s", FormatNetconfError(err))
			}
			defer client.Unlock(ctx, "candidate")
		}

		if _, err := client.EditConfig(ctx, "candidate", body); err != nil {
			return fmt.Errorf("failed to edit config: %s", FormatNetconfError(err))
		}

		if commit {
			if _, err := client.Commit(ctx); err != nil {
				return fmt.Errorf("failed to commit config: %s", FormatNetconfError(err))
			}
		}
	} else {
		if _, err := client.Lock(ctx, "running"); err != nil {
			return fmt.Errorf("failed to lock running datastore: %s", FormatNetconfError(err))
		}
		defer client.Unlock(ctx, "running")

		if _, err := client.EditConfig(ctx, "running", body); err != nil {
			return fmt.Errorf("failed to edit config: %s", FormatNetconfError(err))
		}
	}
	return nil
}

// Commit commits the candidate datastore to the running datastore.
func Commit(ctx context.Context, client *netconf.Client) error {
	if err := client.Open(); err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}

	candidate := client.ServerHasCapability("urn:ietf:params:netconf:capability:candidate:1.0")

	if candidate {
		if _, err := client.Lock(ctx, "running"); err != nil {
			return fmt.Errorf("failed to lock running datastore: %s", FormatNetconfError(err))
		}
		defer client.Unlock(ctx, "running")

		if _, err := client.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit config: %s", FormatNetconfError(err))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// XPath filter helpers
// ---------------------------------------------------------------------------

// GetXpathFilter creates a NETCONF XPath filter with namespace prefixes removed.
func GetXpathFilter(xPath string) netconf.Filter {
	xPath = strings.TrimPrefix(xPath, "/")
	segments := splitXPathSegments(xPath)

	processedSegments := make([]string, 0, len(segments))
	for _, segment := range segments {
		elementName, keys := parseXPathSegment(segment)
		elementName = removeNamespacePrefix(elementName)

		if len(keys) > 0 {
			predicates := make([]string, 0, len(keys))
			for _, kv := range keys {
				keyName := removeNamespacePrefix(kv.Key)
				predicates = append(predicates, fmt.Sprintf("%s='%s'", keyName, kv.Value))
			}
			reconstructed := elementName
			for _, pred := range predicates {
				reconstructed += "[" + pred + "]"
			}
			processedSegments = append(processedSegments, reconstructed)
		} else {
			processedSegments = append(processedSegments, elementName)
		}
	}

	cleanedPath := "/" + strings.Join(processedSegments, "/")
	return netconf.XPathFilter(cleanedPath)
}

// IsGetConfigResponseEmpty checks if a GetConfig response has an empty <data> element.
func IsGetConfigResponseEmpty(res *netconf.Res) bool {
	if res == nil {
		return true
	}
	dataResult := res.Res.Get("data")
	if !dataResult.Exists() {
		return true
	}
	children := dataResult.Map()
	for key := range children {
		if key != "%" {
			return false
		}
	}
	return true
}

// IsListPath checks if an XPath represents a list item (ends with a predicate).
func IsListPath(xPath string) bool {
	return strings.HasSuffix(strings.TrimSpace(xPath), "]")
}

// ---------------------------------------------------------------------------
// XPath segment parsing
// ---------------------------------------------------------------------------

// KeyValue represents a key-value pair with preserved order.
type KeyValue struct {
	Key   string
	Value string
}

func splitXPathSegments(xPath string) []string {
	segments := []string{}
	var currentSegment strings.Builder
	bracketDepth := 0

	for _, char := range xPath {
		switch char {
		case '[':
			bracketDepth++
			currentSegment.WriteRune(char)
		case ']':
			bracketDepth--
			currentSegment.WriteRune(char)
		case '/':
			if bracketDepth == 0 {
				if currentSegment.Len() > 0 {
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
			} else {
				currentSegment.WriteRune(char)
			}
		default:
			currentSegment.WriteRune(char)
		}
	}
	if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}
	return segments
}

func splitDotSegments(path string) []string {
	segments := []string{}
	var currentSegment strings.Builder
	bracketDepth := 0

	for _, char := range path {
		switch char {
		case '[':
			bracketDepth++
			currentSegment.WriteRune(char)
		case ']':
			bracketDepth--
			currentSegment.WriteRune(char)
		case '.':
			if bracketDepth == 0 {
				if currentSegment.Len() > 0 {
					segments = append(segments, currentSegment.String())
					currentSegment.Reset()
				}
			} else {
				currentSegment.WriteRune(char)
			}
		default:
			currentSegment.WriteRune(char)
		}
	}
	if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}
	return segments
}

func parseXPathSegment(segment string) (string, []KeyValue) {
	if idx := strings.Index(segment, "["); idx != -1 {
		elementName := segment[:idx]
		keys := make([]KeyValue, 0)
		remainingPredicates := segment[idx:]

		predicates := predicatePattern.FindAllStringSubmatch(remainingPredicates, -1)
		for _, match := range predicates {
			if len(match) > 1 {
				predicate := match[1]
				conditions := strings.Split(predicate, " and ")
				for _, condition := range conditions {
					if eqIdx := strings.Index(condition, "="); eqIdx != -1 {
						keyName := strings.TrimSpace(condition[:eqIdx])
						value := condition[eqIdx+1:]
						keyValue := strings.Trim(value, `'"`)
						keys = append(keys, KeyValue{Key: keyName, Value: keyValue})
					}
				}
			}
		}
		return elementName, keys
	}
	return segment, nil
}

func dotPath(path string) string {
	path = xpathPredicateRegex.ReplaceAllString(path, "")
	path = strings.ReplaceAll(path, "/", ".")
	return namespaceRegex.ReplaceAllString(path, "")
}

func removeNamespacePrefix(name string) string {
	if idx := strings.Index(name, ":"); idx != -1 {
		return name[idx+1:]
	}
	return name
}

// ---------------------------------------------------------------------------
// XML body building via XPath
// ---------------------------------------------------------------------------

func setWithNamespaces(body netconf.Body, fullPath string, value any) netconf.Body {
	body = body.Set(dotPath(fullPath), value)
	body = augmentNamespaces(body, fullPath)
	return body
}

func augmentNamespaces(body netconf.Body, path string) netconf.Body {
	segments := splitDotSegments(path)
	pathWithoutPrefix := make([]string, 0, len(segments))

	for _, segment := range segments {
		cleanSegment := removeNamespacePrefix(segment)
		if idx := strings.IndexByte(cleanSegment, '['); idx != -1 {
			cleanSegment = cleanSegment[:idx]
		}
		pathWithoutPrefix = append(pathWithoutPrefix, cleanSegment)

		segmentBeforePredicate := segment
		if bracketIdx := strings.IndexByte(segment, '['); bracketIdx != -1 {
			segmentBeforePredicate = segment[:bracketIdx]
		}
		if idx := strings.Index(segmentBeforePredicate, ":"); idx != -1 {
			prefix := segmentBeforePredicate[:idx]
			currentPath := strings.Join(pathWithoutPrefix, ".")

			namespace, ok := namespaceExceptions[prefix]
			if !ok {
				namespace = namespaceBaseURL + prefix
			}

			countPath := currentPath + ".#"
			count := xmldot.Get(body.Res(), countPath).Int()

			if count > 1 {
				for i := 0; i < int(count); i++ {
					indexedXmlnsPath := fmt.Sprintf("%s.%d.@xmlns", currentPath, i)
					if !xmldot.Get(body.Res(), indexedXmlnsPath).Exists() {
						body = body.Set(indexedXmlnsPath, namespace)
					}
				}
			} else {
				xmlnsPath := currentPath + ".@xmlns"
				if !xmldot.Get(body.Res(), xmlnsPath).Exists() {
					body = body.Set(xmlnsPath, namespace)
				}
			}
		}
	}
	return body
}

// ---------------------------------------------------------------------------
// Sibling handling
// ---------------------------------------------------------------------------

type SiblingAction int

const (
	SiblingActionNew SiblingAction = iota
	SiblingActionUpdate
	SiblingActionAppend
)

type SiblingResult struct {
	Action SiblingAction
	Index  int
}

func buildXPathStructure(body netconf.Body, xPath string, ensureStructure bool) (netconf.Body, []string) {
	xPath = strings.TrimPrefix(xPath, "/")
	segments := splitXPathSegments(xPath)

	pathSegments := make([]string, 0, len(segments))
	originalSegments := make([]string, 0, len(segments))

	for segIdx, segment := range segments {
		elementName, keys := parseXPathSegment(segment)
		cleanElementName := removeNamespacePrefix(elementName)

		if len(keys) > 0 {
			parentPath := ""
			if len(pathSegments) > 0 {
				parentPath = strings.Join(pathSegments, ".")
			}

			result := findSiblingInfo(body, parentPath, cleanElementName, keys)

			switch result.Action {
			case SiblingActionNew:
				pathSegments = append(pathSegments, cleanElementName)
				originalSegments = append(originalSegments, elementName)
				fullPath := strings.Join(pathSegments, ".")
				originalFullPath := strings.Join(originalSegments, ".")

				for _, kv := range keys {
					if kv.Key == "." {
						body = body.Set(dotPath(fullPath), kv.Value)
						body = augmentNamespaces(body, originalFullPath)
					} else {
						keyPath := fullPath + "." + kv.Key
						originalKeyPath := originalFullPath + "." + kv.Key
						body = body.Set(dotPath(keyPath), kv.Value)
						body = augmentNamespaces(body, originalKeyPath)
					}
				}

			case SiblingActionUpdate:
				pathSegments = append(pathSegments, fmt.Sprintf("%s.%d", cleanElementName, result.Index))
				originalSegments = append(originalSegments, elementName)

			case SiblingActionAppend:
				remainingSegments := segments[segIdx:]
				body, pathSegments = appendSiblingElement(body, pathSegments, remainingSegments, ensureStructure)
				return body, pathSegments
			}
		} else {
			pathSegments = append(pathSegments, cleanElementName)
			originalSegments = append(originalSegments, elementName)
		}
	}

	if ensureStructure && len(pathSegments) > 0 {
		fullPath := strings.Join(pathSegments, ".")
		originalFullPath := strings.Join(originalSegments, ".")
		existingContent := xmldot.Get(body.Res(), dotPath(fullPath)).String()
		if existingContent == "" {
			body = body.Set(dotPath(fullPath), "")
			body = augmentNamespaces(body, originalFullPath)
		}
	}

	return body, pathSegments
}

func findSiblingInfo(body netconf.Body, parentPath, elementName string, keys []KeyValue) SiblingResult {
	basePath := elementName
	if parentPath != "" {
		basePath = parentPath + "." + elementName
	}

	countPath := basePath + ".#"
	count := xmldot.Get(body.Res(), countPath).Int()

	isLeafList := len(keys) == 1 && keys[0].Key == "."

	if count == 0 {
		if len(keys) > 0 {
			var checkExists string
			if isLeafList {
				checkExists = basePath
			} else {
				checkExists = basePath + "." + keys[0].Key
			}
			if xmldot.Get(body.Res(), checkExists).Exists() {
				allMatch := true
				for _, kv := range keys {
					var checkPath string
					if kv.Key == "." {
						checkPath = basePath
					} else {
						checkPath = basePath + "." + kv.Key
					}
					if xmldot.Get(body.Res(), checkPath).String() != kv.Value {
						allMatch = false
						break
					}
				}
				if allMatch {
					return SiblingResult{Action: SiblingActionUpdate, Index: 0}
				}
				return SiblingResult{Action: SiblingActionAppend, Index: -1}
			}
		}
		return SiblingResult{Action: SiblingActionNew, Index: -1}
	}

	for i := 0; i < int(count); i++ {
		allKeysMatch := true
		for _, kv := range keys {
			var keyPath string
			if kv.Key == "." {
				keyPath = fmt.Sprintf("%s.%d", basePath, i)
			} else {
				keyPath = fmt.Sprintf("%s.%d.%s", basePath, i, kv.Key)
			}
			existingValue := xmldot.Get(body.Res(), keyPath).String()
			if existingValue != kv.Value {
				allKeysMatch = false
				break
			}
		}
		if allKeysMatch {
			return SiblingResult{Action: SiblingActionUpdate, Index: i}
		}
	}

	return SiblingResult{Action: SiblingActionAppend, Index: -1}
}

func appendSiblingElement(body netconf.Body, parentPathSegments []string, remainingSegments []string, ensureStructure bool) (netconf.Body, []string) {
	if len(remainingSegments) == 0 {
		return body, parentPathSegments
	}

	firstSegment := remainingSegments[0]
	elementName, keys := parseXPathSegment(firstSegment)
	cleanElementName := removeNamespacePrefix(elementName)

	resultPathSegments := make([]string, len(parentPathSegments))
	copy(resultPathSegments, parentPathSegments)

	var innerXML strings.Builder

	for _, kv := range keys {
		if kv.Key == "." {
			innerXML.WriteString(html.EscapeString(kv.Value))
		} else {
			innerXML.WriteString(fmt.Sprintf("<%s>%s</%s>", kv.Key, html.EscapeString(kv.Value), kv.Key))
		}
	}

	skipNestedKeyElement := false
	if len(remainingSegments) == 2 {
		nestedElementName, nestedKeys := parseXPathSegment(remainingSegments[1])
		cleanNestedName := removeNamespacePrefix(nestedElementName)
		if len(nestedKeys) == 0 {
			for _, kv := range keys {
				if kv.Key != "." && removeNamespacePrefix(kv.Key) == cleanNestedName {
					skipNestedKeyElement = true
					break
				}
			}
		}
	}

	nestedPathSegments := []string{cleanElementName}
	if len(remainingSegments) > 1 {
		for _, nestedSegment := range remainingSegments[1:] {
			nestedElementName, nestedKeys := parseXPathSegment(nestedSegment)
			cleanNestedName := removeNamespacePrefix(nestedElementName)
			nestedPathSegments = append(nestedPathSegments, cleanNestedName)

			if skipNestedKeyElement {
				continue
			}

			innerXML.WriteString(fmt.Sprintf("<%s>", cleanNestedName))
			for _, kv := range nestedKeys {
				innerXML.WriteString(fmt.Sprintf("<%s>%s</%s>", kv.Key, html.EscapeString(kv.Value), kv.Key))
			}
		}

		if !skipNestedKeyElement {
			for i := len(remainingSegments) - 1; i > 0; i-- {
				nestedElementName, _ := parseXPathSegment(remainingSegments[i])
				cleanNestedName := removeNamespacePrefix(nestedElementName)
				innerXML.WriteString(fmt.Sprintf("</%s>", cleanNestedName))
			}
		}
	}

	elementXML := fmt.Sprintf("<%s>%s</%s>", cleanElementName, innerXML.String(), cleanElementName)

	parentPath := ""
	if len(parentPathSegments) > 0 {
		parentPath = strings.Join(parentPathSegments, ".")
	}

	if parentPath != "" {
		existingContent := xmldot.Get(body.Res(), parentPath).Raw
		newContent := existingContent + elementXML
		body = body.SetRaw(parentPath, newContent)
	} else {
		existingXML := body.Res()
		if existingXML != "" {
			body = netconf.NewBody(existingXML + elementXML)
		} else {
			body = netconf.NewBody(elementXML)
		}
	}

	basePath := cleanElementName
	if parentPath != "" {
		basePath = parentPath + "." + cleanElementName
	}
	newCount := xmldot.Get(body.Res(), basePath+".#").Int()
	newIndex := int(newCount) - 1
	if newIndex < 0 {
		newIndex = 0
	}

	resultPathSegments = append(resultPathSegments, fmt.Sprintf("%s.%d", cleanElementName, newIndex))

	if len(nestedPathSegments) > 1 {
		resultPathSegments = append(resultPathSegments, nestedPathSegments[1:]...)
	}

	fullXPath := strings.Join(remainingSegments, "/")
	if parentPath != "" {
		fullXPath = parentPath + "/" + fullXPath
	}
	body = augmentNamespaces(body, strings.ReplaceAll(fullXPath, "/", "."))

	return body, resultPathSegments
}

// ---------------------------------------------------------------------------
// SetFromXPath / GetFromXPath / RemoveFromXPath
// ---------------------------------------------------------------------------

// SetFromXPath creates all elements in an XPath, including keys and namespaces,
// and optionally sets a value at the final path location.
func SetFromXPath(body netconf.Body, xPath string, value any) netconf.Body {
	hasValue := value != nil && value != ""
	ensureStructure := !hasValue

	body, pathSegments := buildXPathStructure(body, xPath, ensureStructure)

	if hasValue && len(pathSegments) > 0 {
		fullPath := strings.Join(pathSegments, ".")
		body = body.Set(dotPath(fullPath), value)
		dotXPath := strings.ReplaceAll(strings.TrimPrefix(xPath, "/"), "/", ".")
		body = augmentNamespaces(body, dotXPath)
	}

	return body
}

// RemoveFromXPath creates all elements in an XPath with an operation="remove" attribute
// on the last element for NETCONF delete operations.
func RemoveFromXPath(body netconf.Body, xPath string) netconf.Body {
	body, pathSegments := buildXPathStructure(body, xPath, false)

	if len(pathSegments) > 0 {
		targetPath := strings.Join(pathSegments, ".")
		operationPath := targetPath + ".@operation"
		body = body.Set(dotPath(operationPath), "remove")
		dotXPath := strings.ReplaceAll(strings.TrimPrefix(xPath, "/"), "/", ".")
		body = augmentNamespaces(body, dotXPath)
	}

	return body
}

// DeleteFromXPath builds a NETCONF body structure using operation="delete".
// Unlike RemoveFromXPath, this will fail if the element doesn't exist.
func DeleteFromXPath(body netconf.Body, xPath string) netconf.Body {
	body, pathSegments := buildXPathStructure(body, xPath, false)

	if len(pathSegments) > 0 {
		targetPath := strings.Join(pathSegments, ".")
		operationPath := targetPath + ".@operation"
		body = body.Set(dotPath(operationPath), "delete")
		dotXPath := strings.ReplaceAll(strings.TrimPrefix(xPath, "/"), "/", ".")
		body = augmentNamespaces(body, dotXPath)
	}

	return body
}

// GetFromXPath converts an XPath expression to a xmldot path and retrieves the result.
func GetFromXPath(res xmldot.Result, xPath string) xmldot.Result {
	xPath = strings.TrimPrefix(xPath, "/")
	segments := splitXPathSegments(xPath)

	current := res
	pathSoFar := make([]string, 0, len(segments))

	for _, segment := range segments {
		elementName, keys := parseXPathSegment(segment)
		elementName = removeNamespacePrefix(elementName)
		pathSoFar = append(pathSoFar, elementName)

		currentPath := strings.Join(pathSoFar, ".")

		var count int64
		if len(pathSoFar) == 1 {
			count = current.Get(elementName + ".#").Int()
		} else {
			count = res.Get(currentPath + ".#").Int()
		}

		if len(keys) > 0 {
			found := false
			if count > 1 {
				for idx := 0; idx < int(count); idx++ {
					var item xmldot.Result
					if len(pathSoFar) == 1 {
						item = current.Get(fmt.Sprintf("%s.%d", elementName, idx))
					} else {
						indexedPath := fmt.Sprintf("%s.%d", currentPath, idx)
						item = res.Get(indexedPath)
					}

					allMatch := true
					for _, kv := range keys {
						keyName := removeNamespacePrefix(kv.Key)
						keyResult := item.Get(keyName)
						if !keyResult.Exists() || keyResult.String() != kv.Value {
							allMatch = false
							break
						}
					}
					if allMatch {
						pathSoFar[len(pathSoFar)-1] = fmt.Sprintf("%s.%d", elementName, idx)
						current = item
						found = true
						break
					}
				}
			} else {
				var currentResult xmldot.Result
				if len(pathSoFar) == 1 {
					currentResult = current.Get(elementName)
				} else {
					currentResult = res.Get(currentPath)
				}
				allMatch := true
				for _, kv := range keys {
					keyName := removeNamespacePrefix(kv.Key)
					keyResult := currentResult.Get(keyName)
					if !keyResult.Exists() || keyResult.String() != kv.Value {
						allMatch = false
						break
					}
				}
				found = allMatch
				if found {
					current = currentResult
				}
			}
			if !found {
				return xmldot.Result{}
			}
		} else {
			current = current.Get(elementName)
		}
	}

	lastElementName := pathSoFar[len(pathSoFar)-1]
	if dotIdx := strings.LastIndex(lastElementName, "."); dotIdx >= 0 {
		if _, err := fmt.Sscanf(lastElementName[dotIdx+1:], "%d", new(int)); err == nil {
			lastElementName = lastElementName[:dotIdx]
		}
	}

	var parentResult xmldot.Result
	if len(pathSoFar) == 1 {
		parentResult = res
	} else {
		parentPath := strings.Join(pathSoFar[:len(pathSoFar)-1], ".")
		parentResult = res.Get(parentPath)
	}

	count := parentResult.Get(lastElementName + ".#").Int()
	if count > 1 {
		return parentResult.Get("#." + lastElementName)
	}

	return current
}

// ListXPathKeys enumerates list entries named elementName below parentXPath
// and returns their key values.
func ListXPathKeys(res xmldot.Result, parentXPath, elementName string, keyNames []string) [][]string {
	parent := GetFromXPath(res, parentXPath)
	if !parent.Exists() {
		return nil
	}

	name := removeNamespacePrefix(elementName)

	var entries []xmldot.Result
	if count := parent.Get(name + ".#").Int(); count > 1 {
		for i := 0; i < int(count); i++ {
			entries = append(entries, parent.Get(fmt.Sprintf("%s.%d", name, i)))
		}
	} else if entry := parent.Get(name); entry.Exists() {
		entries = append(entries, entry)
	}

	keys := make([][]string, 0, len(entries))
	for _, entry := range entries {
		values := make([]string, 0, len(keyNames))
		complete := true
		for _, keyName := range keyNames {
			value := entry.Get(removeNamespacePrefix(keyName))
			if !value.Exists() {
				complete = false
				break
			}
			values = append(values, value.String())
		}
		if complete {
			keys = append(keys, values)
		}
	}
	return keys
}

// SetRawFromXPath creates all elements in an XPath and inserts raw XML content
// at the final path location.
func SetRawFromXPath(body netconf.Body, xPath string, value string) netconf.Body {
	if len(value) == 0 {
		return body
	}

	xPath = strings.TrimPrefix(xPath, "/")
	segments := splitXPathSegments(xPath)
	if len(segments) == 0 {
		return body
	}

	finalSegment := segments[len(segments)-1]
	finalElement, keys := parseXPathSegment(finalSegment)
	finalElementClean := removeNamespacePrefix(finalElement)

	if len(segments) > 1 {
		wrappedContent := "<" + finalElementClean + ">" + value + "</" + finalElementClean + ">"
		parentXPath := "/" + strings.Join(segments[:len(segments)-1], "/")
		body, _ = buildXPathStructure(body, parentXPath, false)

		parentPathSegments := make([]string, 0, len(segments)-1)
		for _, segment := range segments[:len(segments)-1] {
			elementName, _ := parseXPathSegment(segment)
			parentPathSegments = append(parentPathSegments, elementName)
		}
		parentPath := dotPath(strings.Join(parentPathSegments, "."))

		existingXML := xmldot.Get(body.Res(), parentPath).Raw
		if existingXML != "" {
			combinedXML := existingXML + wrappedContent
			body = body.SetRaw(parentPath, combinedXML)
		} else {
			body = body.SetRaw(parentPath, wrappedContent)
		}
	} else {
		innerContent := value
		if len(keys) > 0 {
			tempBody := netconf.Body{}
			for _, kv := range keys {
				tempBody = setWithNamespaces(tempBody, kv.Key, kv.Value)
			}
			innerContent = tempBody.Res() + value
		}

		countPath := finalElementClean + ".#"
		count := xmldot.Get(body.Res(), countPath).Int()

		if count > 0 {
			currentXML := body.Res()
			wrappedNew := "<" + finalElementClean + ">" + innerContent + "</" + finalElementClean + ">"

			var reconstructedElements string
			for i := 0; i < int(count); i++ {
				elementPath := fmt.Sprintf("%s.%d", finalElementClean, i)
				elementContent := xmldot.Get(currentXML, elementPath).Raw
				reconstructedElements += "<" + finalElementClean + ">" + elementContent + "</" + finalElementClean + ">"
			}
			reconstructedElements += wrappedNew

			finalXML := extractAndReplaceElements(currentXML, finalElementClean, reconstructedElements)
			body = netconf.NewBody(finalXML)
		} else {
			body = body.SetRaw(finalElementClean, innerContent)
		}
	}

	if len(segments) > 0 {
		dotPathForNamespaces := strings.Join(segments, ".")
		body = augmentNamespaces(body, dotPathForNamespaces)
	}

	return body
}

// AppendFromXPath creates all elements in an XPath and appends a value to a list.
func AppendFromXPath(body netconf.Body, xPath string, value any) netconf.Body {
	hasValue := value != nil && value != ""
	ensureStructure := !hasValue

	body, pathSegments := buildXPathStructure(body, xPath, ensureStructure)

	if hasValue && len(pathSegments) > 0 {
		fullPath := strings.Join(pathSegments, ".") + ".-1"
		body = setWithNamespaces(body, fullPath, value)
	}

	dotXPath := strings.ReplaceAll(strings.TrimPrefix(xPath, "/"), "/", ".")
	body = augmentNamespaces(body, dotXPath)

	return body
}

func extractAndReplaceElements(xml, targetElementName, replacementXML string) string {
	if xml == "" {
		return replacementXML
	}

	var result strings.Builder
	remaining := strings.TrimSpace(xml)
	foundTarget := false

	for len(remaining) > 0 {
		remaining = strings.TrimSpace(remaining)
		if len(remaining) == 0 {
			break
		}
		if !strings.HasPrefix(remaining, "<") {
			break
		}

		endOfOpenTag := strings.Index(remaining, ">")
		if endOfOpenTag == -1 {
			break
		}

		openTag := remaining[1:endOfOpenTag]
		isSelfClosing := strings.HasSuffix(openTag, "/")
		if isSelfClosing {
			openTag = strings.TrimSuffix(openTag, "/")
		}

		elementName := strings.TrimSpace(openTag)
		if spaceIdx := strings.IndexAny(elementName, " \t\n"); spaceIdx != -1 {
			elementName = elementName[:spaceIdx]
		}
		if colonIdx := strings.Index(elementName, ":"); colonIdx != -1 {
			elementName = elementName[colonIdx+1:]
		}

		if elementName == targetElementName {
			if !foundTarget {
				result.WriteString(replacementXML)
				foundTarget = true
			}
			if isSelfClosing {
				remaining = remaining[endOfOpenTag+1:]
			} else {
				closingTag := "</" + elementName + ">"
				closingIdx := strings.Index(remaining, closingTag)
				if closingIdx == -1 {
					closingTag = "</" + strings.Split(openTag, " ")[0] + ">"
					closingIdx = strings.Index(remaining, closingTag)
					if closingIdx == -1 {
						break
					}
				}
				remaining = remaining[closingIdx+len(closingTag):]
			}
		} else {
			if isSelfClosing {
				fullElement := remaining[:endOfOpenTag+1]
				result.WriteString(fullElement)
				remaining = remaining[endOfOpenTag+1:]
			} else {
				closingTag := "</" + elementName + ">"
				closingIdx := strings.Index(remaining, closingTag)
				if closingIdx == -1 {
					break
				}
				fullElement := remaining[:closingIdx+len(closingTag)]
				result.WriteString(fullElement)
				remaining = remaining[closingIdx+len(closingTag):]
			}
		}
	}

	if !foundTarget {
		result.WriteString(replacementXML)
	}

	return result.String()
}
