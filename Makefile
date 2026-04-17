# SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
# SPDX-License-Identifier: Apache-2.0

OAPI_CODEGEN ?= $(shell go env GOPATH)/bin/oapi-codegen
SPEC         := api/openapi.yaml

.PHONY: generate test lint clean validate

# Generate Go server stubs and client from the clean OpenAPI spec.
generate: $(SPEC)
	@mkdir -p internal/handler pkg/client
	$(OAPI_CODEGEN) -generate types    -package handler -o internal/handler/types.gen.go $(SPEC)
	$(OAPI_CODEGEN) -generate server   -package handler -o internal/handler/server.gen.go $(SPEC)
	$(OAPI_CODEGEN) -generate types    -package client  -o pkg/client/types.gen.go $(SPEC)
	$(OAPI_CODEGEN) -generate client   -package client  -o pkg/client/client.gen.go $(SPEC)

test:
	go test -race ./...

lint:
	go vet ./...

# Validate the OpenAPI spec (requires redocly or swagger-cli).
validate:
	@command -v redocly >/dev/null 2>&1 && redocly lint $(SPEC) || \
	 command -v swagger-cli >/dev/null 2>&1 && swagger-cli validate $(SPEC) || \
	 echo "Install redocly or swagger-cli for spec validation"

clean:
	rm -f internal/handler/*.gen.go pkg/client/*.gen.go
