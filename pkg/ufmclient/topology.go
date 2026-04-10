// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
)

// Environment represents a UFM logical model environment.
type Environment struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// LogicalServer represents a logical server within an environment.
type LogicalServer struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Compute represents a compute node assigned to a logical server.
type Compute struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"` // allocated, free
}

// Network represents a global or local IB network.
type Network struct {
	Name     string `json:"name"`
	PKey     string `json:"pkey,omitempty"`
	IPOverIB bool   `json:"ip_over_ib,omitempty"`
}

// ListEnvironments returns all environments.
func (c *Client) ListEnvironments(ctx context.Context) ([]Environment, error) {
	var result []Environment
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/environments", nil, nil, &result)
	return result, err
}

// GetEnvironment returns a specific environment.
func (c *Client) GetEnvironment(ctx context.Context, name string) (*Environment, error) {
	var result Environment
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/environments/"+name, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateEnvironment creates a new environment.
func (c *Client) CreateEnvironment(ctx context.Context, env *Environment) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/environments", nil, env, nil)
}

// UpdateEnvironment modifies an environment.
func (c *Client) UpdateEnvironment(ctx context.Context, name string, env *Environment) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/environments/"+name, nil, env, nil)
}

// DeleteEnvironment removes an environment.
func (c *Client) DeleteEnvironment(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/resources/environments/"+name, nil, nil, nil)
}

// ListLogicalServers returns logical servers in an environment.
func (c *Client) ListLogicalServers(ctx context.Context, envName string) ([]LogicalServer, error) {
	var result []LogicalServer
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/environments/"+envName+"/logical_servers", nil, nil, &result)
	return result, err
}

// GetLogicalServer returns a specific logical server.
func (c *Client) GetLogicalServer(ctx context.Context, envName, serverName string) (*LogicalServer, error) {
	var result LogicalServer
	path := c.apiPrefix() + "/resources/environments/" + envName + "/logical_servers/" + serverName
	err := c.do(ctx, "GET", path, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateLogicalServer creates a logical server in an environment.
func (c *Client) CreateLogicalServer(ctx context.Context, envName string, server *LogicalServer) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/environments/"+envName+"/logical_servers", nil, server, nil)
}

// DeleteLogicalServer removes a logical server.
func (c *Client) DeleteLogicalServer(ctx context.Context, envName, serverName string) error {
	path := c.apiPrefix() + "/resources/environments/" + envName + "/logical_servers/" + serverName
	return c.do(ctx, "DELETE", path, nil, nil, nil)
}

// ListComputes returns computes assigned to a logical server.
func (c *Client) ListComputes(ctx context.Context, envName, serverName string) ([]Compute, error) {
	var result []Compute
	path := c.apiPrefix() + "/resources/environments/" + envName + "/logical_servers/" + serverName + "/computes"
	err := c.do(ctx, "GET", path, nil, nil, &result)
	return result, err
}

// AllocateComputes assigns computes to a logical server.
func (c *Client) AllocateComputes(ctx context.Context, envName, serverName string, req map[string]interface{}) error {
	path := c.apiPrefix() + "/resources/environments/" + envName + "/logical_servers/" + serverName + "/allocate-computes"
	return c.do(ctx, "PUT", path, nil, req, nil)
}

// FreeComputes deallocates computes from a logical server.
func (c *Client) FreeComputes(ctx context.Context, envName, serverName string) error {
	path := c.apiPrefix() + "/resources/environments/" + envName + "/logical_servers/" + serverName + "/free-computes"
	return c.do(ctx, "PUT", path, nil, nil, nil)
}

// ListGlobalNetworks returns all global networks.
func (c *Client) ListGlobalNetworks(ctx context.Context) ([]Network, error) {
	var result []Network
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/networks", nil, nil, &result)
	return result, err
}

// GetGlobalNetwork returns a specific global network.
func (c *Client) GetGlobalNetwork(ctx context.Context, name string) (*Network, error) {
	var result Network
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/networks/"+name, nil, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateGlobalNetwork creates a global network.
func (c *Client) CreateGlobalNetwork(ctx context.Context, net *Network) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/networks", nil, net, nil)
}

// UpdateGlobalNetwork modifies a global network.
func (c *Client) UpdateGlobalNetwork(ctx context.Context, net *Network) error {
	return c.do(ctx, "PUT", c.apiPrefix()+"/resources/networks", nil, net, nil)
}

// DeleteGlobalNetwork removes a global network.
func (c *Client) DeleteGlobalNetwork(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/resources/networks/"+name, nil, nil, nil)
}

// ListLocalNetworks returns networks in an environment.
func (c *Client) ListLocalNetworks(ctx context.Context, envName string) ([]Network, error) {
	var result []Network
	err := c.do(ctx, "GET", c.apiPrefix()+"/resources/environments/"+envName+"/networks", nil, nil, &result)
	return result, err
}

// CreateLocalNetwork creates a network in an environment.
func (c *Client) CreateLocalNetwork(ctx context.Context, envName string, net *Network) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/resources/environments/"+envName+"/networks", nil, net, nil)
}
