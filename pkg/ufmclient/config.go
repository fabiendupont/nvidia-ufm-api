// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
)

// UFMConfig represents the UFM server configuration.
type UFMConfig struct {
	DefaultSessionInterval int      `json:"default_session_interval,omitempty"`
	DisabledFeatures       []string `json:"disabled_features,omitempty"`
	HAMode                 string   `json:"ha_mode,omitempty"`
	IsLocalUser            bool     `json:"is_local_user,omitempty"`
	SiteName               string   `json:"site_name,omitempty"`
}

// UFMVersion represents UFM version information.
type UFMVersion struct {
	UFMReleaseVersion string            `json:"ufm_release_version"`
	OpenSMVersion     string            `json:"opensm_version,omitempty"`
	SHARPVersion      string            `json:"sharp_version,omitempty"`
	IBDiagnetVersion  string            `json:"ibdiagnet_version,omitempty"`
	TelemetryVersion  string            `json:"telemetry_version,omitempty"`
	MFTVersion        string            `json:"mft_version,omitempty"`
	WebUIVersion      string            `json:"webui_version,omitempty"`
	Plugins           map[string]string `json:"plugins,omitempty"`
}

// TokenResponse is returned when generating an access token.
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// GetConfig returns UFM server configuration.
func (c *Client) GetConfig(ctx context.Context) (*UFMConfig, error) {
	var result UFMConfig
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/ufm_config", nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateConfig applies partial configuration changes.
func (c *Client) UpdateConfig(ctx context.Context, update map[string]interface{}) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/app/ufm_config", nil, update, nil)
}

// GetVersion returns UFM version information.
func (c *Client) GetVersion(ctx context.Context) (*UFMVersion, error) {
	var result UFMVersion
	err := c.do(ctx, "GET", c.apiPrefix()+"/app/ufm_version", nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateToken generates an API access token using basic auth credentials.
func (c *Client) CreateToken(ctx context.Context) (*TokenResponse, error) {
	var result TokenResponse
	err := c.do(ctx, "POST", c.apiPrefix()+"/app/tokens", nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ConfigureDumpStorage sets external storage path for system dumps.
func (c *Client) ConfigureDumpStorage(ctx context.Context, storagePath string) error {
	req := map[string]interface{}{"storage_path": storagePath}
	return c.do(ctx, "PUT", c.apiPrefix()+"/app/profile/system_dump", nil, req, nil)
}
