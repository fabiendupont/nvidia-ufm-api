// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestListJobs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/jobs") {
			t.Errorf("path = %q, want suffix /jobs", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("status") != "running" {
			t.Errorf("status = %q, want running", q.Get("status"))
		}
		if q.Get("operation") != "add_hosts" {
			t.Errorf("operation = %q, want add_hosts", q.Get("operation"))
		}
		if q.Get("parent_id") != "null" {
			t.Errorf("parent_id = %q, want null", q.Get("parent_id"))
		}
		if q.Get("object_ids") != "sys1,sys2" {
			t.Errorf("object_ids = %q, want sys1,sys2", q.Get("object_ids"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Job{
			{ID: "j1", Status: "running", Operation: "add_hosts"},
			{ID: "j2", Status: "running", Operation: "add_hosts"},
		})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	jobs, err := c.ListJobs(context.Background(), &ListJobsOptions{
		Status:     "running",
		Operation:  "add_hosts",
		ParentOnly: true,
		SystemIDs:  "sys1,sys2",
	})
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}
	if jobs[0].ID != "j1" {
		t.Errorf("jobs[0].ID = %q, want %q", jobs[0].ID, "j1")
	}
}

func TestListJobsNoFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if len(q) != 0 {
			t.Errorf("expected no query params, got %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Job{})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	jobs, err := c.ListJobs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListJobs() error = %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("len(jobs) = %d, want 0", len(jobs))
	}
}

func TestGetJob(t *testing.T) {
	tests := []struct {
		name     string
		advanced bool
		wantQ    string
	}{
		{"basic", false, ""},
		{"advanced", true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/jobs/job-42") {
					t.Errorf("path = %q, want suffix /jobs/job-42", r.URL.Path)
				}
				got := r.URL.Query().Get("advanced_information")
				if got != tt.wantQ {
					t.Errorf("advanced_information = %q, want %q", got, tt.wantQ)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(Job{
					ID:        "job-42",
					Status:    "completed",
					Operation: "firmware_upgrade",
				})
			}))
			defer ts.Close()

			c := newTestClient(t, ts)
			job, err := c.GetJob(context.Background(), "job-42", tt.advanced)
			if err != nil {
				t.Fatalf("GetJob() error = %v", err)
			}
			if job.ID != "job-42" {
				t.Errorf("ID = %q, want %q", job.ID, "job-42")
			}
			if job.Status != "completed" {
				t.Errorf("Status = %q, want %q", job.Status, "completed")
			}
		})
	}
}

func TestDeleteJob(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/jobs/job-99") {
			t.Errorf("path = %q, want suffix /jobs/job-99", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	err := c.DeleteJob(context.Background(), "job-99")
	if err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}
}

func TestWaitForJob(t *testing.T) {
	var callCount atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			json.NewEncoder(w).Encode(Job{ID: "j1", Status: "running"})
		} else {
			json.NewEncoder(w).Encode(Job{ID: "j1", Status: "completed"})
		}
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	job, err := c.WaitForJob(context.Background(), "j1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForJob() error = %v", err)
	}
	if job.Status != "completed" {
		t.Errorf("Status = %q, want %q", job.Status, "completed")
	}
	if callCount.Load() < 3 {
		t.Errorf("poll count = %d, want >= 3", callCount.Load())
	}
}

func TestWaitForJobFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Job{ID: "j1", Status: "failed"})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	job, err := c.WaitForJob(context.Background(), "j1", 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForJob() error = nil, want error for failed job")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("error = %q, want it to contain 'failed'", err.Error())
	}
	if job.ID != "j1" {
		t.Errorf("job.ID = %q, want %q", job.ID, "j1")
	}
}

func TestWaitForJobCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Job{ID: "j1", Status: "running"})
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(t, ts)
	_, err := c.WaitForJob(ctx, "j1", 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForJob() error = nil, want context error")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want context-related error", err.Error())
	}
}

func TestWaitForJobAborted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Job{ID: "j1", Status: "Aborted"})
	}))
	defer ts.Close()

	c := newTestClient(t, ts)
	_, err := c.WaitForJob(context.Background(), "j1", 10*time.Millisecond)
	if err == nil {
		t.Fatal("WaitForJob() error = nil, want error for aborted job")
	}
	if !strings.Contains(err.Error(), "aborted") {
		t.Errorf("error = %q, want it to contain 'aborted'", err.Error())
	}
}
