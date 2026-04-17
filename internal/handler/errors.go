// SPDX-FileCopyrightText: Copyright (c) 2026 Fabien Dupont <fdupont@redhat.com>
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"
)

// apiError returns a structured JSON error response.
func apiError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, Error{
		Error: struct {
			Code    string `json:"code"`
			Details *[]struct {
				Field  *string `json:"field,omitempty"`
				Reason *string `json:"reason,omitempty"`
			} `json:"details,omitempty"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
}

// apiErrorf returns a structured JSON error response with a formatted message.
func apiErrorf(ctx echo.Context, status int, code, format string, args ...interface{}) error {
	return apiError(ctx, status, code, fmt.Sprintf(format, args...))
}

// handleUFMError translates a ufmclient.APIError into an appropriate HTTP response.
// For non-APIError errors it returns a generic 502 Bad Gateway.
func handleUFMError(ctx echo.Context, err error) error {
	var apiErr *ufmclient.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusNotFound:
			return apiError(ctx, http.StatusNotFound, "not_found", apiErr.Body)
		case http.StatusConflict:
			return apiError(ctx, http.StatusConflict, "conflict", apiErr.Body)
		case http.StatusBadRequest:
			return apiError(ctx, http.StatusBadRequest, "bad_request", apiErr.Body)
		case http.StatusUnauthorized:
			return apiError(ctx, http.StatusUnauthorized, "unauthorized", apiErr.Body)
		case http.StatusForbidden:
			return apiError(ctx, http.StatusForbidden, "forbidden", apiErr.Body)
		case http.StatusUnprocessableEntity:
			return apiError(ctx, http.StatusUnprocessableEntity, "unprocessable_entity", apiErr.Body)
		default:
			return apiError(ctx, http.StatusBadGateway, "upstream_error",
				fmt.Sprintf("UFM returned %d: %s", apiErr.StatusCode, apiErr.Body))
		}
	}
	return apiError(ctx, http.StatusBadGateway, "upstream_error", err.Error())
}
