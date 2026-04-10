// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// UFM API Facade server.
//
// This command starts an HTTP server that exposes a clean, resource-oriented
// REST API and proxies requests to a UFM Enterprise instance.
//
// # Environment variables
//
//	UFM_URL              Base URL of the UFM server (default: https://localhost:443)
//	UFM_USERNAME         UFM basic-auth username (default: admin)
//	UFM_PASSWORD         UFM basic-auth password (default: empty)
//	UFM_TLS_SKIP_VERIFY  Set to "true" or "1" to disable TLS cert verification
//	LISTEN_ADDR          Address to listen on (default: :8080)
//
// # Running
//
//	UFM_URL=https://ufm.example.com UFM_USERNAME=admin UFM_PASSWORD=secret ./ufm-api
package main
