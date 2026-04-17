// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

// Package client is an auto-generated Go client for the UFM Facade API.
//
// The types and request helpers in this package are produced by oapi-codegen
// from the OpenAPI specification. They provide a typed, ergonomic interface to
// every endpoint exposed by the facade server.
//
// # Creating a client
//
//	c, err := client.NewClientWithResponses("https://ufm-facade:8080")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Listing systems
//
//	resp, err := c.ListSystemsWithResponse(ctx, &client.ListSystemsParams{
//	    Type: &client.ListSystemsParamsType("switch"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, sys := range *resp.JSON200.Items {
//	    fmt.Println(*sys.Guid, *sys.Name)
//	}
//
// Do not edit the generated files (*.gen.go) directly; regenerate them from
// the OpenAPI spec instead.
package client
