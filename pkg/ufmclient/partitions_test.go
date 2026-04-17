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
)

// newTestClient creates a Client pointing at the given httptest server.
func newTestClient(t *testing.T, ts *httptest.Server) *Client {
	t.Helper()
	return New(ts.URL, "admin", "secret")
}

func TestListPKeys(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys") {
			t.Errorf("path = %q, want suffix /resources/pkeys", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("guids_data") != "" {
			t.Errorf("guids_data should not be set, got %q", q.Get("guids_data"))
		}
		if q.Get("qos_conf") != "" {
			t.Errorf("qos_conf should not be set, got %q", q.Get("qos_conf"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"0x7fff", "0x8001"})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	pkeys, err := c.ListPKeys(context.Background(), false, false, false, false, 0)
	if err != nil {
		t.Fatalf("ListPKeys() error = %v", err)
	}
	if len(pkeys) != 2 {
		t.Fatalf("len(pkeys) = %d, want 2", len(pkeys))
	}
	if pkeys[0] != "0x7fff" {
		t.Errorf("pkeys[0] = %q, want %q", pkeys[0], "0x7fff")
	}
}

func TestListPKeysWithQueryParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("guids_data") != "true" {
			t.Errorf("guids_data = %q, want true", q.Get("guids_data"))
		}
		if q.Get("qos_conf") != "true" {
			t.Errorf("qos_conf = %q, want true", q.Get("qos_conf"))
		}
		if q.Get("port_info") != "true" {
			t.Errorf("port_info = %q, want true", q.Get("port_info"))
		}
		if q.Get("sharp_state") != "true" {
			t.Errorf("sharp_state = %q, want true", q.Get("sharp_state"))
		}
		if q.Get("max_ports") != "50" {
			t.Errorf("max_ports = %q, want 50", q.Get("max_ports"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	_, err := c.ListPKeys(context.Background(), true, true, true, true, 50)
	if err != nil {
		t.Fatalf("ListPKeys() error = %v", err)
	}
}

func TestListPKeysDetailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("guids_data") != "true" {
			t.Errorf("guids_data = %q, want true", q.Get("guids_data"))
		}
		if q.Get("qos_conf") != "true" {
			t.Errorf("qos_conf = %q, want true", q.Get("qos_conf"))
		}
		resp := map[string]PKey{
			"0x8001": {
				Partition: "part1",
				IPOverIB:  true,
				GUIDs:     []PKeyMember{{GUID: "0xaabb", Membership: "full"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	result, err := c.ListPKeysDetailed(context.Background())
	if err != nil {
		t.Fatalf("ListPKeysDetailed() error = %v", err)
	}
	pk, ok := result["0x8001"]
	if !ok {
		t.Fatal("missing key 0x8001")
	}
	if pk.Partition != "part1" {
		t.Errorf("Partition = %q, want %q", pk.Partition, "part1")
	}
	if len(pk.GUIDs) != 1 {
		t.Fatalf("len(GUIDs) = %d, want 1", len(pk.GUIDs))
	}
	if pk.GUIDs[0].GUID != "0xaabb" {
		t.Errorf("GUID = %q, want %q", pk.GUIDs[0].GUID, "0xaabb")
	}
}

func TestGetPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/0x8001") {
			t.Errorf("path = %q, want suffix /resources/pkeys/0x8001", r.URL.Path)
		}
		if r.URL.Query().Get("guids_data") != "true" {
			t.Errorf("guids_data = %q, want true", r.URL.Query().Get("guids_data"))
		}
		resp := PKey{
			Partition:    "test-part",
			SharpEnabled: true,
			GUIDs:        []PKeyMember{{GUID: "0xaa", Membership: "full", Index0: true}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	pk, err := c.GetPKey(context.Background(), "0x8001", true)
	if err != nil {
		t.Fatalf("GetPKey() error = %v", err)
	}
	if pk.Partition != "test-part" {
		t.Errorf("Partition = %q, want %q", pk.Partition, "test-part")
	}
	if !pk.SharpEnabled {
		t.Error("SharpEnabled = false, want true")
	}
	if len(pk.GUIDs) != 1 || pk.GUIDs[0].GUID != "0xaa" {
		t.Errorf("GUIDs = %+v, want [{GUID: 0xaa}]", pk.GUIDs)
	}
}

func TestCreateEmptyPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/add") {
			t.Errorf("path = %q, want suffix /resources/pkeys/add", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeyCreateRequest
		json.Unmarshal(body, &req)
		if req.PKey != "0x8002" {
			t.Errorf("PKey = %q, want %q", req.PKey, "0x8002")
		}
		if !req.IPOverIB {
			t.Error("IPOverIB = false, want true")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.CreateEmptyPKey(context.Background(), &PKeyCreateRequest{
		PKey:     "0x8002",
		IPOverIB: true,
	})
	if err != nil {
		t.Fatalf("CreateEmptyPKey() error = %v", err)
	}
}

func TestAddGUIDsToPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/") {
			t.Errorf("path = %q, want suffix /resources/pkeys/", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeyAddGUIDsRequest
		json.Unmarshal(body, &req)
		if len(req.GUIDs) != 2 {
			t.Errorf("len(GUIDs) = %d, want 2", len(req.GUIDs))
		}
		if req.PKey != "0x8001" {
			t.Errorf("PKey = %q, want %q", req.PKey, "0x8001")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.AddGUIDsToPKey(context.Background(), &PKeyAddGUIDsRequest{
		GUIDs:      []string{"0xaa", "0xbb"},
		PKey:       "0x8001",
		Membership: "full",
	})
	if err != nil {
		t.Fatalf("AddGUIDsToPKey() error = %v", err)
	}
}

func TestSetPKeyGUIDs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/") {
			t.Errorf("path = %q, want suffix /resources/pkeys/", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeySetGUIDsRequest
		json.Unmarshal(body, &req)
		if req.PKey != "0x8001" {
			t.Errorf("PKey = %q, want %q", req.PKey, "0x8001")
		}
		if len(req.GUIDs) != 1 || req.GUIDs[0] != "0xcc" {
			t.Errorf("GUIDs = %v, want [0xcc]", req.GUIDs)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.SetPKeyGUIDs(context.Background(), &PKeySetGUIDsRequest{
		GUIDs: []string{"0xcc"},
		PKey:  "0x8001",
	})
	if err != nil {
		t.Fatalf("SetPKeyGUIDs() error = %v", err)
	}
}

func TestRemoveGUIDsFromPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		// Expect comma-separated GUIDs in the URL path.
		wantSuffix := "/resources/pkeys/0x8001/guids/0xaa,0xbb"
		if !strings.HasSuffix(r.URL.Path, wantSuffix) {
			t.Errorf("path = %q, want suffix %q", r.URL.Path, wantSuffix)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.RemoveGUIDsFromPKey(context.Background(), "0x8001", []string{"0xaa", "0xbb"})
	if err != nil {
		t.Fatalf("RemoveGUIDsFromPKey() error = %v", err)
	}
}

func TestDeletePKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/0x8001") {
			t.Errorf("path = %q, want suffix /resources/pkeys/0x8001", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.DeletePKey(context.Background(), "0x8001")
	if err != nil {
		t.Fatalf("DeletePKey() error = %v", err)
	}
}

func TestAddHostsToPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/hosts") {
			t.Errorf("path = %q, want suffix /resources/pkeys/hosts", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeyAddHostsRequest
		json.Unmarshal(body, &req)
		if req.HostsNames != "host1,host2" {
			t.Errorf("HostsNames = %q, want %q", req.HostsNames, "host1,host2")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Job{ID: "job-123", Status: "running"})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	job, err := c.AddHostsToPKey(context.Background(), &PKeyAddHostsRequest{
		HostsNames: "host1,host2",
		PKey:       "0x8001",
		Membership: "full",
	})
	if err != nil {
		t.Fatalf("AddHostsToPKey() error = %v", err)
	}
	if job.ID != "job-123" {
		t.Errorf("Job.ID = %q, want %q", job.ID, "job-123")
	}
}

func TestRemoveHostsFromPKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		wantSuffix := "/resources/pkeys/0x8001/hosts/host1,host2"
		if !strings.HasSuffix(r.URL.Path, wantSuffix) {
			t.Errorf("path = %q, want suffix %q", r.URL.Path, wantSuffix)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.RemoveHostsFromPKey(context.Background(), "0x8001", []string{"host1", "host2"})
	if err != nil {
		t.Fatalf("RemoveHostsFromPKey() error = %v", err)
	}
}

func TestUpdatePKeyQoS(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/qos_conf") {
			t.Errorf("path = %q, want suffix /resources/pkeys/qos_conf", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req PKeyQoSRequest
		json.Unmarshal(body, &req)
		if req.PKey != "0x8001" {
			t.Errorf("PKey = %q, want %q", req.PKey, "0x8001")
		}
		if req.MTULimit != 4 {
			t.Errorf("MTULimit = %d, want 4", req.MTULimit)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.UpdatePKeyQoS(context.Background(), &PKeyQoSRequest{
		PKey:     "0x8001",
		MTULimit: 4,
	})
	if err != nil {
		t.Fatalf("UpdatePKeyQoS() error = %v", err)
	}
}

func TestSetPKeySHARP(t *testing.T) {
	tests := []struct {
		name   string
		enable bool
		want   string
	}{
		{"enable", true, "enable"},
		{"disable", false, "disable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("method = %q, want PUT", r.Method)
				}
				if !strings.HasSuffix(r.URL.Path, "/resources/pkeys/0x8001/sharp-reservation") {
					t.Errorf("path = %q, want suffix /resources/pkeys/0x8001/sharp-reservation", r.URL.Path)
				}
				body, _ := io.ReadAll(r.Body)
				var req PKeySHARPRequest
				json.Unmarshal(body, &req)
				if req.Action != tt.want {
					t.Errorf("Action = %q, want %q", req.Action, tt.want)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer ts.Close()

			c := newTestClient(t, ts)
			err := c.SetPKeySHARP(context.Background(), "0x8001", tt.enable)
			if err != nil {
				t.Fatalf("SetPKeySHARP() error = %v", err)
			}
		})
	}
}

func TestGetPKeyNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("pkey not found"))
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	_, err := c.GetPKey(context.Background(), "0xffff", false)
	if err == nil {
		t.Fatal("GetPKey() error = nil, want error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}
