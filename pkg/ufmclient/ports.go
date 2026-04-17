// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"net/url"
	"strings"
)

// Port represents a physical port in the fabric.
type Port struct {
	Name               string `json:"name"`
	GUID               string `json:"guid,omitempty"`
	SystemName         string `json:"system_name,omitempty"`
	SystemGUID         string `json:"system_guid,omitempty"`
	PhysicalPortNumber int    `json:"physical_port_number,omitempty"`
	LogicalPortState   string `json:"logical_state,omitempty"`
	PhysicalPortState  string `json:"physical_state,omitempty"`
	Speed              string `json:"speed,omitempty"`
	Width              string `json:"width,omitempty"`
	External           bool   `json:"external,omitempty"`
}

// VirtualPort represents a virtual port in the fabric.
type VirtualPort struct {
	VirtualPortGUID    string `json:"virtual_port_guid"`
	VirtualPortState   string `json:"virtual_port_state"`
	VirtualPortLID     int    `json:"virtual_port_lid"`
	SystemGUID         string `json:"system_guid"`
	SystemName         string `json:"system_name"`
	SystemIP           string `json:"system_ip"`
	PortGUID           string `json:"port_guid"`
	PortName           string `json:"port_name"`
	PhysicalPortNumber int    `json:"physical_port_number"`
}

// Link represents a connection between two ports.
type Link struct {
	SourcePort      string `json:"source_port"`
	DestinationPort string `json:"destination_port"`
	Speed           string `json:"speed,omitempty"`
	Width           string `json:"width,omitempty"`
}

// ListPorts returns ports with optional filtering.
func (c *Client) ListPorts(ctx context.Context, opts *ListPortsOptions) ([]Port, error) {
	q := url.Values{}
	if opts != nil {
		if opts.System != "" {
			q.Set("system", opts.System)
		}
		if opts.SystemType != "" {
			q.Set("sys_type", opts.SystemType)
		}
		if opts.Active {
			q.Set("active", "true")
		}
		if opts.External {
			q.Set("external", "true")
		}
		if opts.HighBER {
			q.Set("high_ber_only", "true")
		}
		if opts.HighBERSeverity != "" {
			q.Set("high_ber_severity", opts.HighBERSeverity)
		}
		if opts.CableInfo {
			q.Set("cable_info", "true")
		}
	}

	var result []Port
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/ports", q, nil, &result)
	return result, err
}

// ListPortsOptions holds optional filters for ListPorts.
type ListPortsOptions struct {
	System         string
	SystemType     string
	Active         bool
	External       bool
	HighBER        bool
	HighBERSeverity string // warning, error
	CableInfo      bool
}

// GetPorts returns specific ports by name.
func (c *Client) GetPorts(ctx context.Context, portNames []string) ([]Port, error) {
	path := c.apiPrefix() + "/resources/ports/" + strings.Join(portNames, ",")
	var result []Port
	err := c.do(ctx, "GET", path, nil, nil, &result)
	return result, err
}

// ListVirtualPorts returns virtual ports with optional filtering.
func (c *Client) ListVirtualPorts(ctx context.Context, system, port string) ([]VirtualPort, error) {
	q := url.Values{}
	if system != "" {
		q.Set("system", system)
	}
	if port != "" {
		q.Set("port", port)
	}

	var result []VirtualPort
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/vports", q, nil, &result)
	return result, err
}

// ListLinks returns all links in the fabric.
func (c *Client) ListLinks(ctx context.Context) ([]Link, error) {
	var result []Link
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/links", nil, nil, &result)
	return result, err
}
