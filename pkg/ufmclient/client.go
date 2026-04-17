// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

// Package ufmclient provides a thin HTTP client for UFM Enterprise's actual
// REST API. It maps 1:1 to UFM's endpoints without normalization — the
// translation layer handles mapping clean facade requests to these calls.
package ufmclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a raw HTTP client for UFM Enterprise's REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	token      string
}

// Option configures a Client.
type Option func(*Client)

// WithTLSSkipVerify disables TLS certificate verification.
func WithTLSSkipVerify() Option {
	return func(c *Client) {
		transport := c.httpClient.Transport.(*http.Transport)
		transport.TLSClientConfig.InsecureSkipVerify = true
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithToken sets a pre-obtained API token for authentication.
// When set, token auth is used instead of basic auth.
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// New creates a UFM client with basic auth credentials.
func New(baseURL, username, password string, opts ...Option) *Client {
	c := &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// apiPrefix returns the URL prefix based on auth method.
// UFM uses different prefixes: /ufmRest (basic), /ufmRestV2 (session), /ufmRestV3 (token).
func (c *Client) apiPrefix() string {
	if c.token != "" {
		return "/ufmRestV3"
	}
	return "/ufmRest"
}

// request builds and executes an HTTP request against UFM.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body interface{}) (*http.Response, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Basic "+c.token)
	} else {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}

	return resp, nil
}

// do executes a request and decodes the JSON response into result.
// If result is nil, the response body is discarded.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, result interface{}) error {
	resp, err := c.request(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Method:     method,
			Path:       path,
			Body:       string(respBody),
		}
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response from %s %s: %w", method, path, err)
		}
	}

	return nil
}

// UploadMultipart sends a multipart file upload to the given path.
func (c *Client) UploadMultipart(ctx context.Context, path, fieldName, fileName string, fileReader io.Reader) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, fileReader); err != nil {
		return fmt.Errorf("copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	u := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, "POST", u, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Basic "+c.token)
	} else {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute upload %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Method: "POST", Path: path, Body: string(respBody)}
	}
	return nil
}

// UploadCableImage uploads a cable transceiver image to UFM.
func (c *Client) UploadCableImage(ctx context.Context, fileName string, fileReader io.Reader) error {
	return c.UploadMultipart(ctx, c.apiPrefix()+"/app/images/cables", "file", fileName, fileReader)
}

// APIError represents an error response from UFM.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("UFM API error: %s %s returned %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}
