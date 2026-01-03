# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## License Header

All Go source files must include this header:
```go
// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT
```

## Build & Test Commands

```bash
# Build
go build ./cmd/api2spec

# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run single test
go test ./internal/config -run TestLoad_YAMLConfigFile -v

# Run tests for a specific package
go test ./internal/cli/... -v
```

## Architecture

api2spec is a CLI tool that parses source code to generate OpenAPI specifications. It uses a plugin architecture for framework-specific parsing.

### Package Structure

- `cmd/api2spec/` - CLI entry point, calls `cli.Execute()`
- `internal/cli/` - Cobra commands (root, init, generate, check, diff, watch, print, version)
- `internal/config/` - Viper-based configuration loading and validation
- `pkg/types/` - Shared types: `Route`, `Schema`, `OpenAPI` document structures

### Key Dependencies

- **Cobra** - CLI framework
- **Viper** - Configuration management (supports yaml, json)
- **testify** - Test assertions

### Configuration

Config files are loaded in priority order: `api2spec.yaml` → `api2spec.json` → `.api2spec.yaml` → `.api2spec.json`

### Planned Components (from spec)

- `internal/parser/` - Language parsers (tree-sitter for TS/JS, go/ast for Go)
- `internal/plugins/` - Framework plugins implementing `FrameworkPlugin` interface
- `internal/schema/` - Schema extractors (Zod, TypeScript interfaces, Go structs)
- `internal/openapi/` - OpenAPI spec builder, merger, differ
- `internal/scanner/` - File discovery using glob patterns
