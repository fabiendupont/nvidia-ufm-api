// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"net/url"
)

// MonitoringSession represents a UFM monitoring session.
type MonitoringSession struct {
	ID         string   `json:"id,omitempty"`
	Interval   int      `json:"interval,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
	Members    []string `json:"members,omitempty"`
	Status     string   `json:"status,omitempty"`
}

// MonitoringTemplate represents a reusable monitoring template.
type MonitoringTemplate struct {
	Name       string   `json:"name,omitempty"`
	Interval   int      `json:"interval,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}

// CreateMonitoringSession creates and starts a monitoring session.
func (c *Client) CreateMonitoringSession(ctx context.Context, req *MonitoringSession) (*MonitoringSession, error) {
	var result MonitoringSession
	err := c.do(ctx, "POST", c.apiPrefix()+"/monitoring/start", nil, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMonitoringSession returns session configuration.
func (c *Client) GetMonitoringSession(ctx context.Context, sessionID string) (*MonitoringSession, error) {
	var result MonitoringSession
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/session/"+sessionID, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetMonitoringSessionData returns collected metrics for a session.
func (c *Client) GetMonitoringSessionData(ctx context.Context, sessionID, pkey string) (interface{}, error) {
	q := url.Values{}
	if pkey != "" {
		q.Set("pkey", pkey)
	}
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/session/"+sessionID+"/data", q, nil, &result)
	return result, err
}

// DeleteMonitoringSession stops and removes a monitoring session.
func (c *Client) DeleteMonitoringSession(ctx context.Context, sessionID string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/monitoring/session/"+sessionID, nil, nil, nil)
}

// CreateMonitoringSnapshot creates a one-time snapshot.
func (c *Client) CreateMonitoringSnapshot(ctx context.Context, req *MonitoringSession) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "POST", c.apiPrefix()+"/monitoring/snapshot", nil, req, &result)
	return result, err
}

// GetMonitoringAttributes returns available counters, classes, and functions.
func (c *Client) GetMonitoringAttributes(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/attributes", nil, nil, &result)
	return result, err
}

// GetCongestion returns traffic and congestion information by tier.
func (c *Client) GetCongestion(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/congestion", nil, nil, &result)
	return result, err
}

// GetPortGroups returns port group traffic and congestion metrics.
func (c *Client) GetPortGroups(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/port_groups", nil, nil, &result)
	return result, err
}

// GetInventory returns fabric inventory summary.
func (c *Client) GetInventory(ctx context.Context, showPorts bool) (interface{}, error) {
	q := url.Values{}
	if showPorts {
		q.Set("show_ports", "true")
	}
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/inventory", q, nil, &result)
	return result, err
}

// GetInventoryCount returns inventory component counts.
func (c *Client) GetInventoryCount(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/monitoring/inventory/count", nil, nil, &result)
	return result, err
}

// GetPerformanceCounters extracts PM counters for specified hostnames.
func (c *Client) GetPerformanceCounters(ctx context.Context, hostnames []string) (interface{}, error) {
	req := map[string]interface{}{"hostnames": hostnames}
	var result interface{}
	err := c.do(ctx, "POST", c.apiPrefix()+"/monitoring/job/resources/pm_counters", nil, req, &result)
	return result, err
}

// ListMonitoringTemplates returns all monitoring templates.
func (c *Client) ListMonitoringTemplates(ctx context.Context) ([]MonitoringTemplate, error) {
	var result []MonitoringTemplate
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/monitoring", nil, nil, &result)
	return result, err
}

// GetMonitoringTemplate returns a specific monitoring template.
func (c *Client) GetMonitoringTemplate(ctx context.Context, name string) (*MonitoringTemplate, error) {
	var result MonitoringTemplate
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/monitoring/"+name, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateMonitoringTemplate creates a new monitoring template.
func (c *Client) CreateMonitoringTemplate(ctx context.Context, tmpl *MonitoringTemplate) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/app/monitoring", nil, tmpl, nil)
}

// UpdateMonitoringTemplate updates an existing monitoring template.
func (c *Client) UpdateMonitoringTemplate(ctx context.Context, tmpl *MonitoringTemplate) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/app/monitoring", nil, tmpl, nil)
}

// DeleteMonitoringTemplate deletes a monitoring template.
func (c *Client) DeleteMonitoringTemplate(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/app/monitoring/"+name, nil, nil, nil)
}
