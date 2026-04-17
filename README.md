# UFM Facade API

A clean, consistent REST facade for [NVIDIA Unified Fabric Manager (UFM) Enterprise](https://www.nvidia.com/en-us/networking/infiniband/ufm/).

## Why

UFM's REST API grew organically and has inconsistencies that make client
development and automation harder than it needs to be:

- `POST /ufmRest/actions` is a single endpoint that does reboot, firmware
  upgrade, port enable/disable, and CLI execution — differentiated only by
  request body.
- Three URL prefixes (`/ufmRest/`, `/ufmRestV2/`, `/ufmRestV3/`) for the
  same endpoints, selected by authentication method.
- Inconsistent resource creation patterns (two different POST endpoints for
  partition creation).
- No standard error envelope, no pagination, no consistent filtering.

This project provides:

1. **A clean OpenAPI 3.1 spec** (`api/openapi.yaml`) — the API UFM should have
2. **A facade server** that proxies clean requests to UFM's actual endpoints
3. **A generated Go client** (`pkg/client/`) for consumers

## Architecture

```
    Clean clients ──► ufm-api (facade) ──► UFM Enterprise
                      /api/v1/...          /ufmRest/...
```

Each operation in the spec includes an `x-ufm-upstream` extension documenting
which UFM endpoint(s) it maps to.

## API coverage

- **98 operations** covering partitions, systems, ports, jobs, monitoring,
  telemetry, topology, mirroring, templates, fabric validation, reports,
  cable images, discovery, and configuration.
- **Consistent patterns**: cursor-based pagination, standard error envelope,
  `include=` for optional nested data, proper HTTP status codes.

## Quick start

```bash
# Generate server stubs and client from the spec
make generate

# Run tests
make test

# Build the facade server
go build -o bin/ufm-api ./cmd/ufm-api

# Run (configure UFM connection via environment)
UFM_URL=https://ufm.example.com UFM_USERNAME=admin UFM_PASSWORD=secret ./bin/ufm-api
```

## Project structure

```
ufm-api/
├── api/
│   └── openapi.yaml          # Clean OpenAPI 3.1 spec (source of truth)
├── cmd/
│   └── ufm-api/              # Facade server entry point
├── internal/
│   ├── handler/              # Generated server stubs + implementation
│   ├── translator/           # Clean request ↔ UFM request translation
│   └── ufmclient/            # Raw HTTP client for UFM's actual API
├── pkg/
│   └── client/               # Generated Go client from clean spec
├── Makefile
└── README.md
```

## Versioning

This project tracks UFM Enterprise releases. The OpenAPI spec `info.version`
reflects the facade API version, while the `x-ufm-upstream` extensions and
README note the target UFM version (currently 6.24.2).

## UFM upstream reference

- [UFM REST API Guide v6.24.2](https://docs.nvidia.com/networking/display/ufmenterpriserestapiv6242)
- [UFM SDK (Python)](https://github.com/Mellanox/ufm_sdk_3.0)
