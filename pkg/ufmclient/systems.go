// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"net/url"
)

// System represents a UFM fabric system (switch, host, gateway, router).
type System struct {
	SystemGUID      string `json:"system_guid"`
	SystemName      string `json:"system_name,omitempty"`
	SystemIP        string `json:"ip,omitempty"`
	Type            string `json:"type,omitempty"`     // switch, host, gateway, router
	Model           string `json:"model,omitempty"`
	Role            string `json:"role,omitempty"`     // core, tor, endpoint
	Vendor          string `json:"vendor,omitempty"`
	FirmwareVersion string `json:"fw_version,omitempty"`
	Description     string `json:"description,omitempty"`
	URL             string `json:"url,omitempty"`
	Script          string `json:"script,omitempty"`
}

// SystemPower represents power consumption data for a switch.
type SystemPower struct {
	SystemID   string  `json:"system_id"`
	PowerWatts float64 `json:"power_watts"`
}

// ListSystems returns all systems with optional filtering.
func (c *Client) ListSystems(ctx context.Context, opts *ListSystemsOptions) ([]System, error) {
	q := url.Values{}
	if opts != nil {
		if opts.IP != "" {
			q.Set("ip", opts.IP)
		}
		if opts.Type != "" {
			q.Set("type", opts.Type)
		}
		if opts.Model != "" {
			q.Set("model", opts.Model)
		}
		if opts.Role != "" {
			q.Set("role", opts.Role)
		}
		if opts.Rack != "" {
			q.Set("in_rack", opts.Rack)
		}
		if opts.Computes != "" {
			q.Set("computes", opts.Computes)
		}
		if opts.Chassis {
			q.Set("chassis", "true")
		}
		if opts.Ports {
			q.Set("ports", "true")
		}
		if opts.Brief {
			q.Set("brief", "true")
		}
	}

	var result []System
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/systems", q, nil, &result)
	return result, err
}

// ListSystemsOptions holds optional filters for ListSystems.
type ListSystemsOptions struct {
	IP       string
	Type     string // switch, host, gateway, router
	Model    string
	Role     string // core, tor, endpoint
	Rack     string
	Computes string // allocated, free
	Chassis  bool
	Ports    bool
	Brief    bool
}

// GetSystem returns a specific system by name or GUID.
func (c *Client) GetSystem(ctx context.Context, systemID string) (*System, error) {
	var result System
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/systems/"+systemID, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateSystem updates manual IP or name for a system.
func (c *Client) UpdateSystem(ctx context.Context, systemID string, update map[string]interface{}) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/systems/"+systemID, nil, update, nil)
}

// UpdateSystemProperties sets URL and script attributes.
func (c *Client) UpdateSystemProperties(ctx context.Context, systemID string, update map[string]interface{}) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/systems/"+systemID+"/properties", nil, update, nil)
}

// GetSystemsPower returns power consumption for all managed switches.
func (c *Client) GetSystemsPower(ctx context.Context) ([]SystemPower, error) {
	q := url.Values{"csv_format": {"true"}}
	var result []SystemPower
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/systems/power", q, nil, &result)
	return result, err
}
