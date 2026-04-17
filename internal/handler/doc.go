// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

// Package handler implements the UFM Facade API server.
//
// It translates clean, resource-oriented REST requests into calls against
// UFM Enterprise's actual API via the [ufmclient] package. Key responsibilities
// include:
//
//   - Cursor-based pagination over UFM's unpaginated list endpoints
//   - Error normalization from UFM status codes to structured JSON errors
//   - Action fan-out (UFM's monolithic POST /actions becomes discrete sub-resources)
//   - Type mapping between the OpenAPI-generated types and ufmclient types
//
// The generated files (server.gen.go, types.gen.go) define the ServerInterface
// and request/response types. [Server] in server.go implements that interface.
package handler
