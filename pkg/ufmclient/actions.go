// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

// This file wraps UFM's monolithic POST /ufmRest/actions endpoint.
// UFM uses a single endpoint for all infrastructure actions, differentiated
// by the request body. The facade splits these into proper sub-resources.

package ufmclient

import (
	"context"
)

// ActionRequest is the generic request body for POST /ufmRest/actions.
// UFM discriminates action type by the combination of fields present.
type ActionRequest struct {
	Action      string   `json:"action,omitempty"`
	Identifier  string   `json:"identifier,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Description string   `json:"description,omitempty"`
	ObjectIDs   []string `json:"object_ids,omitempty"`
}

// RebootSystem reboots a system via POST /ufmRest/actions.
func (c *Client) RebootSystem(ctx context.Context, systemID string) error {
	req := &ActionRequest{
		Action:    "reboot",
		ObjectIDs: []string{systemID},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// InBandRebootSystem reboots an unmanaged switch via POST /ufmRest/actions/inband_reboot.
func (c *Client) InBandRebootSystem(ctx context.Context, systemID string) error {
	req := &ActionRequest{
		ObjectIDs: []string{systemID},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions/inband_reboot", nil, req, nil)
}

// UpgradeFirmware initiates firmware upgrade via POST /ufmRest/actions.
func (c *Client) UpgradeFirmware(ctx context.Context, systemID, image string, inBand bool) error {
	req := &ActionRequest{
		Action:    "firmware_upgrade",
		ObjectIDs: []string{systemID},
		Params: map[string]interface{}{
			"image":   image,
			"in_band": inBand,
		},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// UpgradeSoftware initiates software upgrade via POST /ufmRest/actions.
func (c *Client) UpgradeSoftware(ctx context.Context, systemID, image string) error {
	req := &ActionRequest{
		Action:    "software_upgrade",
		ObjectIDs: []string{systemID},
		Params: map[string]interface{}{
			"image": image,
		},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// EnablePort enables a port via POST /ufmRest/actions.
func (c *Client) EnablePort(ctx context.Context, portName string) error {
	req := &ActionRequest{
		Action:    "port_enable",
		ObjectIDs: []string{portName},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// DisablePort disables a port via POST /ufmRest/actions.
func (c *Client) DisablePort(ctx context.Context, portName string) error {
	req := &ActionRequest{
		Action:    "port_disable",
		ObjectIDs: []string{portName},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// ResetPort resets a port via POST /ufmRest/actions.
func (c *Client) ResetPort(ctx context.Context, portName string) error {
	req := &ActionRequest{
		Action:    "port_reset",
		ObjectIDs: []string{portName},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// CollectSystemDump initiates dump collection via POST /ufmRest/actions.
func (c *Client) CollectSystemDump(ctx context.Context, systemIDs []string) error {
	req := &ActionRequest{
		Action:    "system_dump",
		ObjectIDs: systemIDs,
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// MarkUnhealthy marks a device as unhealthy via POST /ufmRest/actions.
func (c *Client) MarkUnhealthy(ctx context.Context, systemID, isolationPolicy string) error {
	req := &ActionRequest{
		Action:    "mark_unhealthy",
		ObjectIDs: []string{systemID},
		Params: map[string]interface{}{
			"isolation_policy": isolationPolicy,
		},
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions", nil, req, nil)
}

// RestoreHealthy restores a device to healthy status via POST /ufmRestV2/actions.
// Note: this uses the V2 prefix regardless of auth method.
func (c *Client) RestoreHealthy(ctx context.Context, systemID string) error {
	req := &ActionRequest{
		Action:    "restore_healthy",
		ObjectIDs: []string{systemID},
	}
	return c.do(ctx, "POST", "/ufmRestV2/actions", nil, req, nil)
}

// RefreshFabricDiscovery triggers fabric discovery refresh.
func (c *Client) RefreshFabricDiscovery(ctx context.Context) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/actions/fabric_discovery_refresh", nil, nil, nil)
}

// ExecuteProvisioningTemplate runs a provisioning template on systems.
func (c *Client) ExecuteProvisioningTemplate(ctx context.Context, templateName string, systemIDs []string) error {
	req := &ActionRequest{
		ObjectIDs: systemIDs,
	}
	return c.do(ctx, "POST", c.apiPrefix()+"/actions/provisioning/"+templateName, nil, req, nil)
}
