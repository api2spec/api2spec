# api2spec

Code-first OpenAPI specification generator for frameworks that don't hold your hand.

## Overview

api2spec analyzes your API code and generates OpenAPI 3.1 specifications automatically. It bridges the gap between code-first frameworks (Hono, Express, Go chi/gin/echo, etc.) and the OpenAPI ecosystem (Scalar, Redoc, Swagger UI).

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Your Code      │────▶│    api2spec      │────▶│  openapi.yaml   │
│  (Hono/Go/etc)  │     │  Parse + Infer   │     │  (auto-synced)  │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                                 ┌─────────────────┐
                                                 │  Scalar/Redoc   │
                                                 │  (your docs)    │
                                                 └─────────────────┘
```

## Installation

### Homebrew (recommended)

```bash
brew install api2spec/tap/api2spec
```

### Go Install

```bash
go install github.com/api2spec/api2spec@latest
```

### From Source

```bash
git clone https://github.com/api2spec/api2spec
cd api2spec
go build ./cmd/api2spec
```

## Quick Start

```bash
# Initialize config in your project
api2spec init --framework chi

# Generate OpenAPI spec
api2spec generate

# Watch for changes
api2spec watch

# Validate spec matches code
api2spec check --ci
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize api2spec configuration |
| `generate` | Generate/update OpenAPI specification |
| `watch` | Watch for changes and regenerate |
| `check` | Validate spec matches implementation |
| `diff` | Show diff between spec and generated |
| `print` | Output spec to stdout |

## Configuration

api2spec looks for configuration in this order:
1. `api2spec.yaml`
2. `api2spec.json`
3. `.api2spec.yaml`
4. `.api2spec.json`

Example configuration:

```yaml
framework: chi

entry:
  - ./internal/api/**/*.go

exclude:
  - "**/*_test.go"

output:
  path: openapi.yaml
  format: yaml

openapi:
  info:
    title: My API
    version: 1.0.0
  servers:
    - url: http://localhost:8080
      description: Development
```

## Supported Frameworks

### Go
| Framework | Status |
|-----------|--------|
| chi | Priority |
| gin | Planned |
| echo | Planned |
| fiber | Planned |
| gorilla/mux | Planned |
| net/http | Planned |

### TypeScript/JavaScript
| Framework | Status |
|-----------|--------|
| Hono | Priority |
| Express | Planned |
| Fastify | Planned |

## Development

```bash
# Run tests
go test ./...

# Build
go build ./cmd/api2spec

# Run
./api2spec --help
```

## License

FSL-1.1-MIT (Functional Source License)

You may use api2spec freely in your projects. The only restriction is offering it as a hosted/managed service. After 2 years from each release, that version converts to MIT.

See [LICENSE.md](LICENSE.md) for details.
