// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"
)

// setupTestServer creates a mock UFM backend and returns an Echo instance
// with the facade handler registered, plus the mock server for deferred cleanup.
func setupTestServer(t *testing.T, handler http.Handler) (*echo.Echo, *httptest.Server) {
	t.Helper()
	mock := httptest.NewServer(handler)
	ufm := ufmclient.New(mock.URL, "admin", "secret")
	srv := NewServer(ufm)
	e := echo.New()
	RegisterHandlers(e, srv)
	return e, mock
}

// doRequest performs an HTTP request against the Echo instance and returns the recorder.
func doRequest(t *testing.T, e *echo.Echo, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// decodeJSON decodes the response body into the given value.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v\nbody: %s", err, rec.Body.String())
	}
}

// --------------------------------------------------------------------------
// Partitions
// --------------------------------------------------------------------------

func TestListPartitions(t *testing.T) {
	// Mock UFM returns 5 pkeys; we request with limit=2 to test pagination.
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"0x7fff", "0x8001", "0x8002", "0x8003", "0x8004"})
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/partitions?limit=2", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result PartitionList
	decodeJSON(t, rec, &result)

	if result.Items == nil || len(*result.Items) != 2 {
		t.Fatalf("len(items) = %v, want 2", result.Items)
	}
	if result.PageInfo == nil || result.PageInfo.Total == nil || *result.PageInfo.Total != 5 {
		t.Errorf("total = %v, want 5", result.PageInfo)
	}
	if result.PageInfo.NextCursor == nil {
		t.Error("NextCursor = nil, want non-nil (more pages)")
	}
}

func TestListPartitionsEmpty(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/partitions", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result PartitionList
	decodeJSON(t, rec, &result)
	if result.PageInfo != nil && result.PageInfo.Total != nil && *result.PageInfo.Total != 0 {
		t.Errorf("total = %d, want 0", *result.PageInfo.Total)
	}
}

func TestCreatePartitionEmpty(t *testing.T) {
	var gotPath, gotMethod string
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	body := `{"pkey":"0x8001"}`
	rec := doRequest(t, e, "POST", "/partitions", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if gotMethod != "POST" {
		t.Errorf("upstream method = %q, want POST", gotMethod)
	}
	if !strings.HasSuffix(gotPath, "/resources/pkeys/add") {
		t.Errorf("upstream path = %q, want suffix /resources/pkeys/add", gotPath)
	}

	var result Partition
	decodeJSON(t, rec, &result)
	if result.Pkey == nil || *result.Pkey != "0x8001" {
		t.Errorf("Pkey = %v, want 0x8001", result.Pkey)
	}
}

func TestCreatePartitionWithMembers(t *testing.T) {
	var gotPath string
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	body := `{"pkey":"0x8001","members":[{"guid":"0xaa","membership":"full"}]}`
	rec := doRequest(t, e, "POST", "/partitions", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	// With members, it should POST to /resources/pkeys/ (not /add).
	if !strings.HasSuffix(gotPath, "/resources/pkeys/") {
		t.Errorf("upstream path = %q, want suffix /resources/pkeys/", gotPath)
	}
}

func TestGetPartition(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify include=members causes guids_data=true.
		if r.URL.Query().Get("guids_data") != "true" {
			t.Errorf("guids_data = %q, want true", r.URL.Query().Get("guids_data"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ufmclient.PKey{
			Partition: "test",
			IPOverIB:  true,
			GUIDs: []ufmclient.PKeyMember{
				{GUID: "0xaa", Membership: "full", Index0: true},
			},
		})
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/partitions/0x8001?include=members", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result Partition
	decodeJSON(t, rec, &result)
	if result.Name == nil || *result.Name != "test" {
		t.Errorf("Name = %v, want %q", result.Name, "test")
	}
	if result.Members == nil || len(*result.Members) != 1 {
		t.Fatalf("Members = %v, want 1 member", result.Members)
	}
	if (*result.Members)[0].Guid == nil || *(*result.Members)[0].Guid != "0xaa" {
		t.Errorf("Member GUID = %v, want 0xaa", (*result.Members)[0].Guid)
	}
}

func TestDeletePartition(t *testing.T) {
	var gotMethod string
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	rec := doRequest(t, e, "DELETE", "/partitions/0x8001", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if gotMethod != "DELETE" {
		t.Errorf("upstream method = %q, want DELETE", gotMethod)
	}
}

// --------------------------------------------------------------------------
// Systems
// --------------------------------------------------------------------------

func TestListSystems(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("type") != "switch" {
			t.Errorf("upstream type = %q, want switch", q.Get("type"))
		}
		if q.Get("role") != "core" {
			t.Errorf("upstream role = %q, want core", q.Get("role"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]ufmclient.System{
			{SystemGUID: "0x1", SystemName: "sw1", Type: "switch", Role: "core"},
			{SystemGUID: "0x2", SystemName: "sw2", Type: "switch", Role: "core"},
		})
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/systems?type=switch&role=core", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result SystemList
	decodeJSON(t, rec, &result)
	if result.Items == nil || len(*result.Items) != 2 {
		t.Fatalf("len(items) = %v, want 2", result.Items)
	}
	if (*result.Items)[0].Guid == nil || *(*result.Items)[0].Guid != "0x1" {
		t.Errorf("items[0].Guid = %v, want 0x1", (*result.Items)[0].Guid)
	}
}

// --------------------------------------------------------------------------
// Jobs
// --------------------------------------------------------------------------

func TestGetJob(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ufmclient.Job{
			ID:        "j42",
			Status:    "running",
			Operation: "add_hosts",
		})
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/jobs/j42", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result Job
	decodeJSON(t, rec, &result)
	if result.Id == nil || *result.Id != "j42" {
		t.Errorf("Id = %v, want j42", result.Id)
	}
	if result.Status == nil || *result.Status != "running" {
		t.Errorf("Status = %v, want running", result.Status)
	}
}

// --------------------------------------------------------------------------
// Actions (reboot)
// --------------------------------------------------------------------------

func TestRebootSystem(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath string
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	rec := doRequest(t, e, "POST", "/systems/sys-123/reboot", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	// Should fan out to POST /ufmRest/actions.
	if !strings.HasSuffix(gotPath, "/actions") {
		t.Errorf("upstream path = %q, want suffix /actions", gotPath)
	}
	if gotBody != nil {
		if action, ok := gotBody["action"]; ok {
			if action != "reboot" {
				t.Errorf("action = %v, want reboot", action)
			}
		}
	}
}

func TestRebootSystemInBand(t *testing.T) {
	var gotPath string
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	body := `{"in_band":true}`
	rec := doRequest(t, e, "POST", "/systems/sys-123/reboot", body)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if !strings.HasSuffix(gotPath, "/actions/inband_reboot") {
		t.Errorf("upstream path = %q, want suffix /actions/inband_reboot", gotPath)
	}
}

// --------------------------------------------------------------------------
// Error translation
// --------------------------------------------------------------------------

func TestErrorTranslation(t *testing.T) {
	tests := []struct {
		name           string
		ufmStatus      int
		wantStatus     int
		wantErrorCode  string
	}{
		{"UFM 404 to facade 404", 404, 404, "not_found"},
		{"UFM 409 to facade 409", 409, 409, "conflict"},
		{"UFM 400 to facade 400", 400, 400, "bad_request"},
		{"UFM 500 to facade 502", 500, 502, "upstream_error"},
		{"UFM 503 to facade 502", 503, 502, "upstream_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.ufmStatus)
				w.Write([]byte("upstream error details"))
			}))
			defer mock.Close()

			rec := doRequest(t, e, "GET", "/partitions/0xdead", "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var errResp Error
			decodeJSON(t, rec, &errResp)
			if errResp.Error.Code != tt.wantErrorCode {
				t.Errorf("error code = %q, want %q", errResp.Error.Code, tt.wantErrorCode)
			}
		})
	}
}

func TestErrorTranslation401(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("bad creds"))
	}))
	defer mock.Close()

	rec := doRequest(t, e, "GET", "/systems", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var errResp Error
	decodeJSON(t, rec, &errResp)
	if errResp.Error.Code != "unauthorized" {
		t.Errorf("error code = %q, want %q", errResp.Error.Code, "unauthorized")
	}
}

func TestErrorTranslation403(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer mock.Close()

	rec := doRequest(t, e, "DELETE", "/partitions/0x1234", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	var errResp Error
	decodeJSON(t, rec, &errResp)
	if errResp.Error.Code != "forbidden" {
		t.Errorf("error code = %q, want %q", errResp.Error.Code, "forbidden")
	}
}

// --------------------------------------------------------------------------
// Pagination edge cases
// --------------------------------------------------------------------------

func TestListPartitionsPaginationCursor(t *testing.T) {
	// 3 items, limit=2, first page gets 2, cursor page gets 1.
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"0x8001", "0x8002", "0x8003"})
	}))
	defer mock.Close()

	// First page.
	rec1 := doRequest(t, e, "GET", "/partitions?limit=2", "")
	if rec1.Code != http.StatusOK {
		t.Fatalf("page1 status = %d, want %d", rec1.Code, http.StatusOK)
	}
	var page1 PartitionList
	decodeJSON(t, rec1, &page1)
	if page1.Items == nil || len(*page1.Items) != 2 {
		t.Fatalf("page1 len(items) = %v, want 2", page1.Items)
	}
	if page1.PageInfo == nil || page1.PageInfo.NextCursor == nil {
		t.Fatal("page1 NextCursor = nil, want cursor for next page")
	}

	// Second page using cursor.
	rec2 := doRequest(t, e, "GET", "/partitions?limit=2&cursor="+*page1.PageInfo.NextCursor, "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("page2 status = %d, want %d", rec2.Code, http.StatusOK)
	}
	var page2 PartitionList
	decodeJSON(t, rec2, &page2)
	if page2.Items == nil || len(*page2.Items) != 1 {
		t.Fatalf("page2 len(items) = %v, want 1", page2.Items)
	}
	if page2.PageInfo != nil && page2.PageInfo.NextCursor != nil {
		t.Error("page2 NextCursor should be nil (last page)")
	}
}

func TestDeleteJob(t *testing.T) {
	e, mock := setupTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer mock.Close()

	rec := doRequest(t, e, "DELETE", "/jobs/j99", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}
