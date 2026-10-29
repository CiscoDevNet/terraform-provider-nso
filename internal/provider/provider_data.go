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
	"fmt"
	"sync"

	"github.com/CiscoDevNet/terraform-provider-nso/internal/provider/helpers"
	"github.com/netascode/go-netconf"
)

type NsoProviderData struct {
	Instances map[string]*NsoInstanceData
}

type NsoInstanceData struct {
	NetconfClient   *netconf.Client
	AutoCommit      bool
	ReuseConnection bool
	NetconfOpMutex  sync.Mutex

	namespacesMutex sync.Mutex
	namespacesReady bool
}

// Prepare opens the NETCONF session and, on first use, registers the XML
// namespaces of the NED modules advertised in the server capabilities.
// Request bodies reference NED modules by prefix (e.g. tailf-ned-cisco-ios),
// so this must run before building any edit-config body.
func (i *NsoInstanceData) Prepare() error {
	if err := i.NetconfClient.Open(); err != nil {
		return fmt.Errorf("failed to open NETCONF connection: %w", err)
	}
	i.namespacesMutex.Lock()
	defer i.namespacesMutex.Unlock()
	if !i.namespacesReady {
		helpers.BuildNamespaceMap(i.NetconfClient.ServerCapabilities())
		i.namespacesReady = true
	}
	return nil
}
