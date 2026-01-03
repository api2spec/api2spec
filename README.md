# api2spec

Code-first OpenAPI specification generator for frameworks that don't hold your hand.

## Overview

api2spec analyzes your API code and generates OpenAPI 3.1 specifications automatically. It uses **tree-sitter** for static analysis - no runtime dependencies needed, perfect for CI/CD pipelines.

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Your Code      │────▶│    api2spec      │────▶│  openapi.yaml   │
│  (Any Framework)│     │  Parse + Infer   │     │  (auto-synced)  │
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
# Initialize config (auto-detects framework)
api2spec init

# Generate OpenAPI spec
api2spec generate

# Watch for changes
api2spec watch

# Validate spec matches code (for CI)
api2spec check --ci

# Show differences
api2spec diff
```

## Supported Frameworks

**23 frameworks across 11 languages** - with more being added regularly.

### Go

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **chi** | `go-chi/chi` in go.mod | Go structs + validate tags |
| **gin** | `gin-gonic/gin` in go.mod | Go structs + binding tags |
| **echo** | `labstack/echo` in go.mod | Go structs + validate tags |
| **fiber** | `gofiber/fiber` in go.mod | Go structs + validate tags |

### TypeScript/JavaScript

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Hono** | `hono` in package.json | Zod schemas |
| **Express** | `express` in package.json | express-validator, Zod |
| **Fastify** | `fastify` in package.json | Built-in JSON Schema, Zod |
| **Koa** | `koa` in package.json | Zod schemas |
| **Elysia** | `elysia` in package.json | TypeBox, Zod |
| **NestJS** | `@nestjs/core` in package.json | class-validator DTOs |

### Python

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **FastAPI** | `fastapi` in requirements.txt/pyproject.toml | Pydantic models |
| **Flask** | `flask` in requirements.txt/pyproject.toml | Type hints |

### Rust

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Axum** | `axum` in Cargo.toml | Rust structs + serde |
| **Actix-web** | `actix-web` in Cargo.toml | Rust structs + serde |
| **Rocket** | `rocket` in Cargo.toml | Rust structs + serde |

### C#

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **ASP.NET Core** | `Microsoft.AspNetCore` in *.csproj | DTOs, records |

### PHP

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Laravel** | `laravel/framework` in composer.json | Request classes |

### Java

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Spring Boot** | `spring-boot` in pom.xml/build.gradle | DTOs, records |

### Kotlin

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Ktor** | `io.ktor` in build.gradle.kts | Data classes |

### Elixir

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Phoenix** | `:phoenix` in mix.exs | Ecto schemas |

### Ruby

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Rails** | `rails` in Gemfile | ActiveRecord models |
| **Sinatra** | `sinatra` in Gemfile | Plain Ruby |

### Gleam

| Framework | Detection | Schema Support |
|-----------|-----------|----------------|
| **Wisp** | `wisp` in gleam.toml | Gleam types |

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
framework: chi  # or auto-detect

source:
  paths:
    - ./internal/api
    - ./cmd/server
  include:
    - "**/*.go"
  exclude:
    - "**/*_test.go"
    - "**/mocks/**"

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

## CI/CD Integration

### GitHub Actions

```yaml
- name: Check API spec
  run: |
    go install github.com/api2spec/api2spec@latest
    api2spec check --ci
```

### Pre-commit Hook

```bash
#!/bin/sh
api2spec check --strict || exit 1
```

## Why Tree-sitter?

api2spec uses tree-sitter for static source code analysis instead of runtime reflection:

| Feature | Tree-sitter (api2spec) | Runtime Reflection |
|---------|------------------------|-------------------|
| Dependencies | None needed | Full environment |
| Speed | Fast - just parses | Slower - runs code |
| Safety | No code execution | Executes imports |
| CI/CD | Perfect fit | Needs setup |

This makes api2spec ideal for CI pipelines where you don't want to install framework dependencies just to generate documentation.

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
