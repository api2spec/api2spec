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
go test ./internal/plugins/chi/... -v

# Check for linter issues
go vet ./...
```

## Architecture

api2spec is a CLI tool that parses source code using tree-sitter to generate OpenAPI specifications. It uses a plugin architecture for framework-specific parsing.

### Package Structure

- `cmd/api2spec/` - CLI entry point
- `internal/cli/` - Cobra commands (init, generate, check, diff, watch, print, version)
- `internal/config/` - Viper-based configuration loading
- `internal/scanner/` - File discovery with glob patterns
- `internal/parser/` - Language parsers:
  - `golang.go` - Go AST parser (go/ast)
  - `typescript.go` - TypeScript/JavaScript parser (tree-sitter)
  - `python.go` - Python parser (tree-sitter)
  - `rust.go` - Rust parser (tree-sitter)
- `internal/schema/` - Schema extractors:
  - `golang.go` - Go structs → JSON Schema
  - `zod.go` - Zod schemas → JSON Schema
  - `registry.go` - Schema registry
- `internal/plugins/` - Framework plugins (14 total):
  - **Go**: chi, gin, echo, fiber
  - **JS/TS**: hono, express, fastify, koa, elysia, nestjs
  - **Python**: flask, fastapi
  - **Rust**: axum, actix
- `internal/openapi/` - OpenAPI spec builder, merger, differ, writer
- `pkg/types/` - Shared types: Route, Schema, OpenAPI document

### Plugin Interface

All framework plugins implement:
```go
type FrameworkPlugin interface {
    Name() string
    Extensions() []string
    Detect(projectRoot string) (bool, error)
    ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error)
    ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error)
}
```

### Key Dependencies

- **Cobra** - CLI framework
- **Viper** - Configuration management
- **tree-sitter** - Source code parsing (TypeScript, Python, Rust)
- **go/ast** - Go source parsing (native)
- **testify** - Test assertions
- **fsnotify** - File watching
- **doublestar** - Glob patterns

### Configuration

Config files are loaded in priority order:
`api2spec.yaml` → `api2spec.json` → `.api2spec.yaml` → `.api2spec.json`

### Adding a New Framework Plugin

1. Create `internal/plugins/<name>/<name>.go`
2. Implement `FrameworkPlugin` interface
3. Add `init()` function to auto-register: `plugins.MustRegister(&Plugin{})`
4. Import in `internal/cli/generate.go`: `_ "github.com/api2spec/api2spec/internal/plugins/<name>"`
5. Add tests in `<name>_test.go`
