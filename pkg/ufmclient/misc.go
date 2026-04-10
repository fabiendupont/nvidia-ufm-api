// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"fmt"
	"net/url"
)

// MirroringTemplate represents a port mirroring configuration.
type MirroringTemplate struct {
	SystemID     string `json:"system_id"`
	TargetPort   string `json:"target_port"`
	PacketSize   int    `json:"packet_size,omitempty"`
	ServiceLevel int    `json:"service_level,omitempty"`
}

// MirroringAction represents an enable/disable mirroring request.
type MirroringAction struct {
	PortID string `json:"port_id"`
	Action string `json:"action"` // "enable" or "disable"
	RX     *bool  `json:"rx,omitempty"`
	TX     *bool  `json:"tx,omitempty"`
}

// ProvisioningTemplate represents a UFM provisioning template.
type ProvisioningTemplate struct {
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	SystemType  string        `json:"systemType,omitempty"`
	Owner       string        `json:"owner,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Content     []interface{} `json:"content,omitempty"`
}

// ReportRequest creates a new report.
type ReportRequest struct {
	Type       string                 `json:"type"` // Fabric_Health, UFM_Health, Topology_Compare
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// -- Mirroring --

// CreateMirroringTemplate creates a mirroring template.
func (c *Client) CreateMirroringTemplate(ctx context.Context, tmpl *MirroringTemplate) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/app/mirrorings", nil, tmpl, nil)
}

// GetMirroringTemplate returns a mirroring template by system ID.
func (c *Client) GetMirroringTemplate(ctx context.Context, systemID string) (*MirroringTemplate, error) {
	var result MirroringTemplate
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/mirrorings/"+systemID, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateMirroringTemplate updates a mirroring template.
func (c *Client) UpdateMirroringTemplate(ctx context.Context, tmpl *MirroringTemplate) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/app/mirrorings", nil, tmpl, nil)
}

// DeleteMirroringTemplate removes a mirroring template.
func (c *Client) DeleteMirroringTemplate(ctx context.Context, systemID string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/app/mirrorings/"+systemID, nil, nil, nil)
}

// ExecuteMirroringAction enables or disables mirroring on a port.
func (c *Client) ExecuteMirroringAction(ctx context.Context, action *MirroringAction) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/app/mirrorings/action", nil, action, nil)
}

// -- Templates --

// ListTemplates returns all provisioning templates.
func (c *Client) ListTemplates(ctx context.Context, tags, profile, systemType string) ([]ProvisioningTemplate, error) {
	q := url.Values{}
	if tags != "" {
		q.Set("tags", tags)
	}
	if profile != "" {
		q.Set("profile", profile)
	}
	if systemType != "" {
		q.Set("system_type", systemType)
	}
	var result []ProvisioningTemplate
	err := c.do(ctx, "GET", c.apiPrefix()+"/templates", q, nil, &result)
	return result, err
}

// GetTemplate returns a specific provisioning template.
func (c *Client) GetTemplate(ctx context.Context, name string) (*ProvisioningTemplate, error) {
	var result ProvisioningTemplate
	err := c.do(ctx, "GET", c.apiPrefix()+"/templates/"+name, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTemplate creates a provisioning template.
func (c *Client) CreateTemplate(ctx context.Context, tmpl *ProvisioningTemplate) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/templates", nil, tmpl, nil)
}

// DeleteTemplate removes a provisioning template.
func (c *Client) DeleteTemplate(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/templates/"+name, nil, nil, nil)
}

// RefreshTemplates refreshes the templates list after changes.
func (c *Client) RefreshTemplates(ctx context.Context) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/templates/refresh", nil, nil, nil)
}

// -- Fabric Validation --

// ListFabricValidationTests returns available test names.
func (c *Client) ListFabricValidationTests(ctx context.Context) ([]string, error) {
	var result []string
	err := c.do(ctx, "GET", c.apiPrefix()+"/fabricValidation/tests", nil, nil, &result)
	return result, err
}

// RunFabricValidationTest executes a specific test. Returns job ID via Location header.
func (c *Client) RunFabricValidationTest(ctx context.Context, testName string) (*Job, error) {
	resp, err := c.request(ctx, "POST", c.apiPrefix()+"/fabricValidation/tests/"+testName, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		return nil, &APIError{StatusCode: 409, Method: "POST", Path: testName, Body: "test already running"}
	}

	job := &Job{ID: resp.Header.Get("Location")}
	return job, nil
}

// -- Reports --

// CreateReport initiates a new report.
func (c *Client) CreateReport(ctx context.Context, req *ReportRequest) (*Job, error) {
	var result Job
	err := c.do(ctx, "POST", c.apiPrefix()+"/reports/"+req.Type, nil, req.Parameters, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetReport returns report results.
func (c *Client) GetReport(ctx context.Context, reportID string) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/reports/"+reportID, nil, nil, &result)
	return result, err
}

// DeleteReport removes a report.
func (c *Client) DeleteReport(ctx context.Context, reportID string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/reports/"+reportID, nil, nil, nil)
}

// GetLatestReport returns the most recent report of a given type.
func (c *Client) GetLatestReport(ctx context.Context, reportType string) (interface{}, error) {
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/reports/last_report/"+reportType, nil, nil, &result)
	return result, err
}

// -- Telemetry --

// GetTopTelemetry returns Top-X telemetry data.
func (c *Client) GetTopTelemetry(ctx context.Context, membersType, pickBy string, limit int, attributes string) (interface{}, error) {
	q := url.Values{
		"type":        {"topX"},
		"membersType": {membersType},
		"PickBy":      {pickBy},
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if attributes != "" {
		q.Set("attributes", attributes)
	}
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/telemetry", q, nil, &result)
	return result, err
}

// GetHistoryTelemetry returns historical telemetry data.
func (c *Client) GetHistoryTelemetry(ctx context.Context, membersType, attributes, members, startTime, endTime string) (interface{}, error) {
	q := url.Values{
		"type":        {"history"},
		"membersType": {membersType},
		"attributes":  {attributes},
		"members":     {members},
		"start_time":  {startTime},
		"end_time":    {endTime},
	}
	var result interface{}
	err := c.do(ctx, "GET", c.apiPrefix()+"/telemetry", q, nil, &result)
	return result, err
}

// -- Cable Images --

// ListCableImages returns available cable transceiver images.
func (c *Client) ListCableImages(ctx context.Context) ([]string, error) {
	var result []string
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/images/cables", nil, nil, &result)
	return result, err
}

// DeleteCableImage removes a cable image.
func (c *Client) DeleteCableImage(ctx context.Context, imageName string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/app/images/cables/"+imageName, nil, nil, nil)
}
