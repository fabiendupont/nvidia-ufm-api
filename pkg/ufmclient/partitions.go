// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// PKey represents a UFM partition key with its members.
type PKey struct {
	Partition    string       `json:"partition"`
	IPOverIB     bool         `json:"ip_over_ib"`
	SharpEnabled bool         `json:"sharp_enabled"`
	GUIDs        []PKeyMember `json:"guids,omitempty"`
	// QoS fields (present when qos_conf=true).
	MTULimit     *int     `json:"mtu_limit,omitempty"`
	ServiceLevel *int     `json:"service_level,omitempty"`
	RateLimit    *float64 `json:"rate_limit,omitempty"`
}

// PKeyMember represents a GUID's membership in a partition.
type PKeyMember struct {
	GUID       string `json:"guid"`
	Membership string `json:"membership"` // "full" or "limited"
	Index0     bool   `json:"index0"`
}

// PKeyCreateRequest creates an empty partition.
type PKeyCreateRequest struct {
	PKey          string  `json:"pkey"`
	PartitionName string  `json:"partition_name,omitempty"`
	Index0        bool    `json:"index0"`
	IPOverIB      bool    `json:"ip_over_ib"`
	MTULimit      *int    `json:"mtu_limit,omitempty"`
	ServiceLevel  *int    `json:"service_level,omitempty"`
	RateLimit     *float64 `json:"rate_limit,omitempty"`
}

// PKeyAddGUIDsRequest adds GUIDs to a partition.
type PKeyAddGUIDsRequest struct {
	GUIDs         []string `json:"guids"`
	PKey          string   `json:"pkey"`
	PartitionName string   `json:"partition_name,omitempty"`
	IPOverIB      bool     `json:"ip_over_ib"`
	Index0        bool     `json:"index0"`
	Membership    string   `json:"membership,omitempty"`    // "full" or "limited"
	Memberships   []string `json:"memberships,omitempty"`   // per-GUID membership
}

// PKeySetGUIDsRequest replaces all GUIDs in a partition (PUT).
type PKeySetGUIDsRequest struct {
	GUIDs         []string `json:"guids"`
	PKey          string   `json:"pkey"`
	PartitionName string   `json:"partition_name,omitempty"`
	IPOverIB      bool     `json:"ip_over_ib"`
	Index0        bool     `json:"index0"`
	Membership    string   `json:"membership,omitempty"`
	Memberships   []string `json:"memberships,omitempty"`
	MTULimit      *int     `json:"mtu_limit,omitempty"`
	ServiceLevel  *int     `json:"service_level,omitempty"`
	RateLimit     *float64 `json:"rate_limit,omitempty"`
}

// PKeyAddHostsRequest adds hosts to a partition.
type PKeyAddHostsRequest struct {
	HostsNames string `json:"hosts_names"` // comma-separated
	PKey       string `json:"pkey"`
	IPOverIB   bool   `json:"ip_over_ib"`
	Index0     bool   `json:"index0"`
	Membership string `json:"membership,omitempty"`
}

// PKeyQoSRequest updates QoS settings for a partition.
type PKeyQoSRequest struct {
	PKey         string  `json:"pkey"`
	MTULimit     int     `json:"mtu_limit"`
	ServiceLevel int     `json:"service_level"`
	RateLimit    float64 `json:"rate_limit"`
}

// PKeySHARPRequest enables or disables SHARP reservation.
type PKeySHARPRequest struct {
	Action string `json:"action"` // "enable" or "disable"
}

// PKeyLastUpdated is the response from the last_updated endpoint.
type PKeyLastUpdated struct {
	LastUpdated *string `json:"last_updated"`
}

// ListPKeys returns all partition keys.
// With guidsData=true, each PKey includes its GUID members.
func (c *Client) ListPKeys(ctx context.Context, guidsData, qosConf, portInfo, sharpState bool, maxPorts int) ([]string, error) {
	q := url.Values{}
	if guidsData {
		q.Set("guids_data", "true")
	}
	if qosConf {
		q.Set("qos_conf", "true")
	}
	if portInfo {
		q.Set("port_info", "true")
	}
	if sharpState {
		q.Set("sharp_state", "true")
	}
	if maxPorts > 0 {
		q.Set("max_ports", fmt.Sprintf("%d", maxPorts))
	}

	var result []string
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/pkeys", q, nil, &result)
	return result, err
}

// ListPKeysDetailed returns all partition keys with full details.
func (c *Client) ListPKeysDetailed(ctx context.Context) (map[string]PKey, error) {
	q := url.Values{"guids_data": {"true"}, "qos_conf": {"true"}}
	var result map[string]PKey
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/pkeys", q, nil, &result)
	return result, err
}

// GetPKey returns details of a specific partition.
func (c *Client) GetPKey(ctx context.Context, pkey string, guidsData bool) (*PKey, error) {
	q := url.Values{}
	if guidsData {
		q.Set("guids_data", "true")
	}
	var result PKey
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/pkeys/"+pkey, q, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateEmptyPKey creates a partition with no GUID members.
func (c *Client) CreateEmptyPKey(ctx context.Context, req *PKeyCreateRequest) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/pkeys/add", nil, req, nil)
}

// AddGUIDsToPKey adds GUIDs to an existing (or new) partition.
func (c *Client) AddGUIDsToPKey(ctx context.Context, req *PKeyAddGUIDsRequest) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/pkeys/", nil, req, nil)
}

// SetPKeyGUIDs replaces all GUIDs in a partition.
func (c *Client) SetPKeyGUIDs(ctx context.Context, req *PKeySetGUIDsRequest) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/pkeys/", nil, req, nil)
}

// RemoveGUIDsFromPKey removes specific GUIDs from a partition.
// UFM encodes GUIDs as comma-separated values in the URL path.
func (c *Client) RemoveGUIDsFromPKey(ctx context.Context, pkey string, guids []string) error {
	path := fmt.Sprintf("%s/resources/pkeys/%s/guids/%s", c.apiPrefix(), pkey, strings.Join(guids, ","))
	return c.do(ctx, "DELETE", path, nil, nil, nil)
}

// AddHostsToPKey adds all ports of named hosts to a partition.
// Returns a Job for tracking the async operation.
func (c *Client) AddHostsToPKey(ctx context.Context, req *PKeyAddHostsRequest) (*Job, error) {
	var result Job
	err := c.do(ctx, "POST", c.apiPrefix()+"/resources/pkeys/hosts", nil, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RemoveHostsFromPKey removes hosts from a partition.
// UFM encodes host names as comma-separated values in the URL path.
func (c *Client) RemoveHostsFromPKey(ctx context.Context, pkey string, hosts []string) error {
	path := fmt.Sprintf("%s/resources/pkeys/%s/hosts/%s", c.apiPrefix(), pkey, strings.Join(hosts, ","))
	return c.do(ctx, "DELETE", path, nil, nil, nil)
}

// DeletePKey deletes a partition.
func (c *Client) DeletePKey(ctx context.Context, pkey string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/resources/pkeys/"+pkey, nil, nil, nil)
}

// UpdatePKeyQoS updates QoS settings for a partition.
// Note: UFM requires a restart for QoS changes to take effect.
func (c *Client) UpdatePKeyQoS(ctx context.Context, req *PKeyQoSRequest) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/pkeys/qos_conf", nil, req, nil)
}

// SetPKeySHARP enables or disables SHARP reservation for a partition.
func (c *Client) SetPKeySHARP(ctx context.Context, pkey string, enable bool) error {
	action := "disable"
	if enable {
		action = "enable"
	}
	req := &PKeySHARPRequest{Action: action}
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/pkeys/"+pkey+"/sharp-reservation", nil, req, nil)
}

// GetPKeyLastUpdated returns the last-updated timestamp for partition changes.
func (c *Client) GetPKeyLastUpdated(ctx context.Context) (*PKeyLastUpdated, error) {
	var result PKeyLastUpdated
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/pkeys/last_updated", nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
