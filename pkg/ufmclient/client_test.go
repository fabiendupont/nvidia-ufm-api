// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("https://ufm.example.com/", "admin", "secret")
	if c.baseURL != "https://ufm.example.com" {
		t.Errorf("baseURL = %q, want trailing slash stripped", c.baseURL)
	}
	if c.username != "admin" {
		t.Errorf("username = %q, want %q", c.username, "admin")
	}
	if c.password != "secret" {
		t.Errorf("password = %q, want %q", c.password, "secret")
	}
	if c.token != "" {
		t.Errorf("token = %q, want empty", c.token)
	}
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", c.httpClient.Timeout)
	}
}

func TestNewWithOptions(t *testing.T) {
	c := New("https://ufm.example.com", "admin", "pass",
		WithTimeout(60*time.Second),
		WithToken("tok123"),
		WithTLSSkipVerify(),
	)
	if c.httpClient.Timeout != 60*time.Second {
		t.Errorf("timeout = %v, want 60s", c.httpClient.Timeout)
	}
	if c.token != "tok123" {
		t.Errorf("token = %q, want %q", c.token, "tok123")
	}
}

func TestAPIPrefix(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"basic auth", "", "/ufmRest"},
		{"token auth", "mytoken", "/ufmRestV3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New("https://x", "u", "p")
			if tt.token != "" {
				c.token = tt.token
			}
			got := c.apiPrefix()
			if got != tt.want {
				t.Errorf("apiPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	e := &APIError{
		StatusCode: 404,
		Method:     "GET",
		Path:       "/resources/pkeys/0x1234",
		Body:       "not found",
	}
	got := e.Error()
	if !strings.Contains(got, "404") {
		t.Errorf("Error() = %q, want it to contain 404", got)
	}
	if !strings.Contains(got, "GET") {
		t.Errorf("Error() = %q, want it to contain GET", got)
	}
	if !strings.Contains(got, "not found") {
		t.Errorf("Error() = %q, want it to contain body text", got)
	}
}

func TestDoSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/ufmRest/resources/systems" {
			t.Errorf("path = %q, want /ufmRest/resources/systems", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "switch" {
			t.Errorf("query type = %q, want switch", r.URL.Query().Get("type"))
		}
		// Check basic auth.
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("basic auth = (%q, %q, %v), want (admin, secret, true)", user, pass, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]System{
			{SystemGUID: "0x1234", SystemName: "sw1"},
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret")
	systems, err := c.ListSystems(context.Background(), &ListSystemsOptions{Type: "switch"})
	if err != nil {
		t.Fatalf("ListSystems() error = %v", err)
	}
	if len(systems) != 1 {
		t.Fatalf("len(systems) = %d, want 1", len(systems))
	}
	if systems[0].SystemGUID != "0x1234" {
		t.Errorf("SystemGUID = %q, want %q", systems[0].SystemGUID, "0x1234")
	}
}

func TestDoError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", 400},
		{"not found", 404},
		{"internal server error", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte("error details"))
			}))
			defer ts.Close()

			c := New(ts.URL, "admin", "secret")
			var result []string
			err := c.do(context.Background(), "GET", "/ufmRest/test", nil, nil, &result)
			if err == nil {
				t.Fatal("do() error = nil, want error")
			}
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("error type = %T, want *APIError", err)
			}
			if apiErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.statusCode)
			}
			if apiErr.Body != "error details" {
				t.Errorf("Body = %q, want %q", apiErr.Body, "error details")
			}
		})
	}
}

func TestDoTokenAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Basic mytoken" {
			t.Errorf("Authorization = %q, want %q", auth, "Basic mytoken")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret", WithToken("mytoken"))
	var result []string
	err := c.do(context.Background(), "GET", "/ufmRestV3/test", nil, nil, &result)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
}

func TestDoRequestBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeyCreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if req.PKey != "0x8001" {
			t.Errorf("PKey = %q, want %q", req.PKey, "0x8001")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret")
	err := c.do(context.Background(), "POST", "/ufmRest/resources/pkeys/add", nil,
		&PKeyCreateRequest{PKey: "0x8001"}, nil)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
}

func TestUploadMultipart(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Errorf("Content-Type = %q, want multipart/form-data prefix", ct)
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("FormFile error = %v", err)
		}
		defer file.Close()
		if header.Filename != "cable.bin" {
			t.Errorf("filename = %q, want %q", header.Filename, "cable.bin")
		}
		data, _ := io.ReadAll(file)
		if string(data) != "file-content" {
			t.Errorf("file data = %q, want %q", string(data), "file-content")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret")
	err := c.UploadMultipart(context.Background(), "/upload",
		"image", "cable.bin", strings.NewReader("file-content"))
	if err != nil {
		t.Fatalf("UploadMultipart() error = %v", err)
	}
}

func TestUploadMultipartError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad file"))
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret")
	err := c.UploadMultipart(context.Background(), "/upload",
		"file", "test.bin", strings.NewReader("data"))
	if err == nil {
		t.Fatal("UploadMultipart() error = nil, want error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", apiErr.StatusCode)
	}
}

func TestDoNoContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := New(ts.URL, "admin", "secret")
	var result map[string]string
	err := c.do(context.Background(), "DELETE", "/ufmRest/test", nil, nil, &result)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
}
