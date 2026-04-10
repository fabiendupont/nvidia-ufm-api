// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/fabiendupont/nvidia-ufm-api/internal/handler"
	"github.com/fabiendupont/nvidia-ufm-api/pkg/ufmclient"
)

func main() {
	ufmURL := envOrDefault("UFM_URL", "https://localhost:443")
	ufmUser := envOrDefault("UFM_USERNAME", "admin")
	ufmPass := envOrDefault("UFM_PASSWORD", "")
	listenAddr := envOrDefault("LISTEN_ADDR", ":8080")

	var opts []ufmclient.Option
	if os.Getenv("UFM_TLS_SKIP_VERIFY") == "true" || os.Getenv("UFM_TLS_SKIP_VERIFY") == "1" {
		opts = append(opts, ufmclient.WithTLSSkipVerify())
	}

	client := ufmclient.New(ufmURL, ufmUser, ufmPass, opts...)
	srv := handler.NewServer(client)

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.RequestID())

	handler.RegisterHandlers(e, srv)

	log.Printf("Starting UFM API facade on %s (upstream: %s)", listenAddr, ufmURL)
	if err := e.Start(listenAddr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
