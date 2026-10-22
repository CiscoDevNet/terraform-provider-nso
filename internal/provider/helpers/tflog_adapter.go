// Copyright © 2025 Cisco Systems, Inc. and its affiliates.
// All rights reserved.
//
// Licensed under the Mozilla Public License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://mozilla.org/MPL/2.0/
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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netascode/go-netconf"
)

// TflogAdapter adapts go-netconf's Logger interface to Terraform's tflog package.
type TflogAdapter struct {
	deviceID string
}

var _ netconf.Logger = (*TflogAdapter)(nil)

// NewTflogAdapter creates a new Terraform logging adapter with device identification.
func NewTflogAdapter(deviceID string) *TflogAdapter {
	return &TflogAdapter{
		deviceID: deviceID,
	}
}

func (t *TflogAdapter) Debug(ctx context.Context, msg string, keysAndValues ...any) {
	if ctx == nil {
		return
	}
	ctx = tflog.NewSubsystem(ctx, "netconf")
	fields := keysAndValuesToMap(keysAndValues)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["device"] = t.deviceID
	tflog.SubsystemDebug(ctx, "netconf", msg, fields)
}

func (t *TflogAdapter) Info(ctx context.Context, msg string, keysAndValues ...any) {
	if ctx == nil {
		return
	}
	ctx = tflog.NewSubsystem(ctx, "netconf")
	fields := keysAndValuesToMap(keysAndValues)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["device"] = t.deviceID
	tflog.SubsystemInfo(ctx, "netconf", msg, fields)
}

func (t *TflogAdapter) Warn(ctx context.Context, msg string, keysAndValues ...any) {
	if ctx == nil {
		return
	}
	ctx = tflog.NewSubsystem(ctx, "netconf")
	fields := keysAndValuesToMap(keysAndValues)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["device"] = t.deviceID
	tflog.SubsystemWarn(ctx, "netconf", msg, fields)
}

func (t *TflogAdapter) Error(ctx context.Context, msg string, keysAndValues ...any) {
	if ctx == nil {
		return
	}
	ctx = tflog.NewSubsystem(ctx, "netconf")
	fields := keysAndValuesToMap(keysAndValues)
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["device"] = t.deviceID
	tflog.SubsystemError(ctx, "netconf", msg, fields)
}

func keysAndValuesToMap(keysAndValues []any) map[string]any {
	if len(keysAndValues) == 0 {
		return nil
	}
	fields := make(map[string]any, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		var value any
		if i+1 < len(keysAndValues) {
			value = keysAndValues[i+1]
		}
		fields[key] = value
	}
	return fields
}
