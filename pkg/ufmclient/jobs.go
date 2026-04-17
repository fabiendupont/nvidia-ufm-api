// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package ufmclient

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// Job represents a UFM async operation.
type Job struct {
	ID        string `json:"id"`
	Operation string `json:"operation,omitempty"`
	Status    string `json:"status,omitempty"`
}

// ListJobs returns jobs with optional filtering.
func (c *Client) ListJobs(ctx context.Context, opts *ListJobsOptions) ([]Job, error) {
	q := url.Values{}
	if opts != nil {
		if opts.Status != "" {
			q.Set("status", opts.Status)
		}
		if opts.Operation != "" {
			q.Set("operation", opts.Operation)
		}
		if opts.ParentOnly {
			q.Set("parent_id", "null")
		}
		if opts.SystemIDs != "" {
			q.Set("object_ids", opts.SystemIDs)
		}
	}

	var result []Job
	err := c.do(ctx, "GET", c.apiPrefix()+"/jobs", q, nil, &result)
	return result, err
}

// ListJobsOptions holds optional filters for ListJobs.
type ListJobsOptions struct {
	Status     string // running, completed, failed, aborted
	Operation  string
	ParentOnly bool
	SystemIDs  string // comma-separated
}

// GetJob returns details of a specific job.
func (c *Client) GetJob(ctx context.Context, jobID string, advanced bool) (*Job, error) {
	q := url.Values{}
	if advanced {
		q.Set("advanced_information", "true")
	}
	var result Job
	err := c.do(ctx, "GET", c.apiPrefix()+"/jobs/"+jobID, q, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteJob removes a job record.
func (c *Client) DeleteJob(ctx context.Context, jobID string) error {
	return c.do(ctx, "DELETE", c.apiPrefix()+"/jobs/"+jobID, nil, nil, nil)
}

// AbortAllJobs terminates all active jobs.
func (c *Client) AbortAllJobs(ctx context.Context) error {
	return c.do(ctx, "POST", c.apiPrefix()+"/jobs/abortall", nil, nil, nil)
}

// WaitForJob polls a job until it completes or the context is cancelled.
func (c *Client) WaitForJob(ctx context.Context, jobID string, interval time.Duration) (*Job, error) {
	for {
		job, err := c.GetJob(ctx, jobID, false)
		if err != nil {
			return nil, err
		}

		switch job.Status {
		case "completed", "Completed":
			return job, nil
		case "failed", "Failed":
			return job, fmt.Errorf("job %s failed", jobID)
		case "aborted", "Aborted":
			return job, fmt.Errorf("job %s aborted", jobID)
		}

		select {
		case <-ctx.Done():
			return job, ctx.Err()
		case <-time.After(interval):
		}
	}
}
