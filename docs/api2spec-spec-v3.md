# api2spec

**Code-first OpenAPI specification generator for frameworks that don't hold your hand.**

---

## Overview

api2spec is an open source CLI tool that analyzes your API code and generates/maintains OpenAPI specifications automatically. It bridges the gap between code-first frameworks (Hono, Express, Go chi/gin/echo, etc.) and the OpenAPI ecosystem (Scalar, Redoc, Swagger UI).

### The Problem

Some frameworks auto-generate OpenAPI specs:

- FastAPI (Python)
- NestJS (TypeScript)
- ASP.NET Core

Most don't:

- **JavaScript/TypeScript**: Hono, Express, Fastify, Koa, Elysia
- **Go**: chi, gin, echo, fiber, gorilla/mux
- **Rust**: axum, actix-web, rocket

For these frameworks, developers either manually write YAML/JSON specs (tedious, error-prone, always out of sync) or skip documentation entirely.

### The Solution

api2spec parses your route definitions, extracts type information, and generates a complete OpenAPI 3.1 specification. When used with Claude Code, developers get intelligent assistance for descriptions, documentation quality, and edge case handling.

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

---

## Goals

1. **Zero-config start**: Run `api2spec init` and get a working spec immediately
2. **Framework-agnostic core**: Plugin architecture for any framework
3. **Type-aware**: Extract schemas from TypeScript types, Go structs, Zod schemas
4. **Offline-first**: Works fully without any external services
5. **CI/CD friendly**: Fail builds when spec drifts from implementation
6. **Single binary**: Distribute via Homebrew, no runtime dependencies
7. **Claude Code optimized**: Designed to work seamlessly with Claude Code for enhanced documentation

---

## Non-Goals

- Replacing framework-native OpenAPI generation (FastAPI, NestJS)
- Full API gateway functionality
- Runtime request validation (use other tools for that)
- Hosting documentation (use Scalar, Redoc, etc.)
- LLM API integrations (use Claude Code interactively instead)
- Full TypeScript type inference (see Parsing Limitations)

---

## Installation

### Homebrew (recommended)

```bash
brew install api2spec/tap/api2spec
```

### Go Install

```bash
go install github.com/api2spec/api2spec@latest
```

### GitHub Releases

Download binaries directly from [GitHub Releases](https://github.com/api2spec/api2spec/releases).

### Docker

```bash
docker run --rm -v $(pwd):/app ghcr.io/api2spec/api2spec generate
```

---

## Core Concepts

### Route Discovery

api2spec discovers routes by parsing framework-specific patterns:

```typescript
// Hono
app.get('/users/:id', handler)
app.post('/users', handler)

// Express
router.get('/users/:id', handler)
router.post('/users', handler)
```

```go
// chi
r.Get("/users/{id}", handler)
r.Post("/users", handler)

// gin
r.GET("/users/:id", handler)
r.POST("/users", handler)
```

### Schema Extraction

Schemas are extracted from type definitions in the codebase:

```typescript
// TypeScript interface → OpenAPI schema
interface User {
  id: string
  name: string
  email: string
  createdAt: Date
}

// Zod schema → OpenAPI schema (preferred for TS)
const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email(),
  createdAt: z.date()
})
```

```go
// Go struct → OpenAPI schema
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"createdAt"`
}
```

### Inference Modes

| Mode     | Description                      | Use Case                    |
|----------|----------------------------------|-----------------------------|
| `static` | Parse code only, no inference    | CI/CD, automated pipelines  |
| `hybrid` | Static + heuristics for gaps     | Default mode                |

**Note:** For AI-enhanced documentation (descriptions, examples, response inference), use api2spec with Claude Code. This provides interactive, high-quality documentation without API costs or configuration.

---

## CLI Interface

### Commands

```bash
# Initialize api2spec in a project
api2spec init

# Generate/update OpenAPI spec
api2spec generate

# Watch for changes and regenerate
api2spec watch

# Validate spec matches implementation
api2spec check

# Show diff between current spec and generated
api2spec diff

# Output spec to stdout (for piping)
api2spec print
```

### Global Options

```bash
--config, -c      Path to config file (default: api2spec.yaml)
--output, -o      Output file path (default: openapi.yaml)
--format, -f      Output format: yaml | json (default: yaml)
--framework       Override auto-detected framework
--verbose, -v     Verbose output
--quiet, -q       Suppress non-error output
```

### Init Command

```bash
api2spec init [options]

Options:
  --framework     Framework to configure (auto-detected if omitted)
```

Creates configuration file and initial spec scaffold.

### Generate Command

```bash
api2spec generate [options]

Options:
  --mode          Inference mode: static | hybrid (default: hybrid)
  --merge         Merge with existing spec instead of overwriting
  --dry-run       Show what would be generated without writing
  --include       Glob pattern for files to include
  --exclude       Glob pattern for files to exclude
```

### Watch Command

```bash
api2spec watch [options]

Options:
  --mode          Inference mode: static | hybrid (default: hybrid)
  --debounce      Debounce time in ms (default: 500)
  --on-change     Command to run after regeneration
```

### Check Command

```bash
api2spec check [options]

Options:
  --strict        Fail on any difference (default: true)
  --ignore        Patterns to ignore in comparison
  --ci            CI mode: exit code 1 on failure

Exit Codes:
  0 - Spec matches implementation
  1 - Spec differs from implementation
  2 - Error during analysis
```

### Diff Command

```bash
api2spec diff [options]

Options:
  --color         Colorize output (default: true if TTY)
  --unified       Unified diff format
  --side-by-side  Side-by-side diff format
```

---

## Configuration

### Configuration File

api2spec looks for configuration in this order:

1. `api2spec.yaml`
2. `api2spec.json`
3. `.api2spec.yaml`
4. `.api2spec.json`

### YAML Configuration

```yaml
# api2spec.yaml

# Framework detection (auto-detected if omitted)
framework: hono

# Entry points to scan
entry:
  - src/routes/**/*.ts
  - src/api/**/*.ts

# Files to exclude
exclude:
  - "**/*.test.ts"
  - "**/*.spec.ts"

# Output configuration
output:
  path: openapi.yaml
  format: yaml  # yaml | json

# OpenAPI document base
openapi:
  info:
    title: My API
    version: 1.0.0
    description: API description
  servers:
    - url: http://localhost:3000
      description: Development
    - url: https://api.example.com
      description: Production

# Schema extraction options
schemas:
  # Where to look for type definitions
  include:
    - src/schemas/**/*.ts
    - src/types/**/*.ts
  
  # Schema libraries to parse
  libraries:
    - zod
    - typescript

# Framework-specific options
frameworkOptions:
  hono:
    # Custom app variable names to look for
    appNames:
      - app
      - api
      - router
```

### JSON Configuration

```json
{
  "framework": "hono",
  "entry": ["src/routes/**/*.ts"],
  "exclude": ["**/*.test.ts"],
  "output": {
    "path": "openapi.yaml",
    "format": "yaml"
  },
  "openapi": {
    "info": {
      "title": "My API",
      "version": "1.0.0"
    }
  },
  "schemas": {
    "include": ["src/schemas/**/*.ts"],
    "libraries": ["zod"]
  }
}
```

---

## Parsing Architecture

### Overview

api2spec is written in Go and uses different parsing strategies depending on the language:

| Language | Parser | Capabilities |
|----------|--------|--------------|
| TypeScript/JavaScript | tree-sitter | Syntax extraction, pattern matching |
| Go | go/ast (native) | Full AST with type info |

### How tree-sitter Works

tree-sitter is a parser generator that produces fast, accurate parsers in C. api2spec uses tree-sitter bindings for Go to parse TypeScript/JavaScript code.

Given this code:

```typescript
app.get('/users/:id', handler)
```

tree-sitter produces an AST (Abstract Syntax Tree):

```
CallExpression
├── MemberExpression
│   ├── Identifier: "app"
│   └── Property: "get"
└── Arguments
    ├── StringLiteral: "/users/:id"
    └── Identifier: "handler"
```

api2spec walks this tree to extract:
- HTTP method: `get`
- Path: `/users/:id`
- Path parameters: `id`

### What tree-sitter Parses Well

**Route definitions** ✅
```typescript
app.get('/users/:id', handler)
app.post('/users', validate(schema), handler)
router.route('/api/v1', subrouter)
```

**Zod schemas** ✅
```typescript
const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email(),
})
```

Zod schemas are self-describing chains of method calls. api2spec pattern matches:
- `z.string()` → `{ type: "string" }`
- `.uuid()` → `{ format: "uuid" }`
- `.min(1)` → `{ minLength: 1 }`
- `.optional()` → not in `required` array

**TypeScript interfaces** ✅ (when in scope)
```typescript
interface CreateUserRequest {
  name: string
  email: string
  age?: number
}
```

**Go structs** ✅ (native go/ast)
```go
type User struct {
    ID    string `json:"id" validate:"required,uuid"`
    Name  string `json:"name" validate:"required,min=1,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

### Parsing Limitations

tree-sitter parses syntax, not semantics. It cannot:

**Follow imports** ❌
```typescript
import { User } from './types'
// tree-sitter sees "User" but can't resolve what it contains
```

**Evaluate type inference** ❌
```typescript
type UserType = z.infer<typeof UserSchema>
// tree-sitter sees this as text, doesn't evaluate it
```

**Resolve complex generics** ❌
```typescript
const handler: Handler<User, Error> = ...
// tree-sitter sees text, not resolved types
```

### Working Around Limitations

api2spec uses these strategies to handle limitations:

**1. Schema discovery**

Before parsing routes, api2spec scans all files matching `schemas.include` patterns and builds a schema registry:

```yaml
schemas:
  include:
    - src/schemas/**/*.ts
    - src/types/**/*.ts
```

When a route references `User`, api2spec looks up `UserSchema` or `User` interface in the registry.

**2. Naming conventions**

api2spec follows conventions to match routes with schemas:
- Response type `User` → look for `UserSchema` or `User` interface
- Request body → look for schema passed to validation middleware
- Path like `/users` → tag as "Users"

**3. Co-located schemas (recommended)**

For best results, define schemas near routes:

```typescript
// routes/users.ts
import { z } from 'zod'

const CreateUserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
})

const UserSchema = z.object({
  id: z.string().uuid(),
  ...CreateUserSchema.shape,
  createdAt: z.string().datetime(),
})

app.post('/users', zValidator('json', CreateUserSchema), async (c) => {
  // ...
  return c.json(user, 201)
})
```

**4. JSDoc annotations (escape hatch)**

For cases api2spec can't infer, use JSDoc:

```typescript
/**
 * @api2spec
 * operationId: getUser
 * summary: Get user by ID
 * responses:
 *   200:
 *     schema: User
 *   404:
 *     schema: Error
 */
app.get('/users/:id', handler)
```

**5. Claude Code for gaps**

Run `api2spec generate`, then use Claude Code to fill in descriptions, examples, and complex response types interactively.

---

## Framework Plugins

### Plugin Architecture

Each framework has a dedicated parser plugin:

```go
type FrameworkPlugin interface {
    // Plugin identifier
    Name() string
    
    // File extensions this plugin handles
    Extensions() []string
    
    // Detect if this framework is used in the project
    Detect(projectRoot string) (bool, error)
    
    // Extract routes from source files
    ExtractRoutes(files []SourceFile) ([]Route, error)
    
    // Extract schemas from source files
    ExtractSchemas(files []SourceFile) ([]Schema, error)
}

type Route struct {
    Method      string
    Path        string
    OperationID string
    Summary     string
    Description string
    Tags        []string
    Parameters  []Parameter
    RequestBody *RequestBody
    Responses   map[string]Response
    Source      SourceLocation
}

type Schema struct {
    Name   string
    Schema JSONSchema
    Source SourceLocation
}
```

### Supported Frameworks

#### JavaScript/TypeScript

| Framework | Status      | Parser | Notes |
|-----------|-------------|--------|-------|
| Hono      | 🎯 Priority | tree-sitter | Full support planned |
| Express   | Planned     | tree-sitter | Most popular, essential |
| Fastify   | Planned     | tree-sitter | Has partial OpenAPI support |
| Koa       | Planned     | tree-sitter | |
| Elysia    | Planned     | tree-sitter | Has built-in support, lower priority |

#### Go

| Framework   | Status      | Parser | Notes |
|-------------|-------------|--------|-------|
| chi         | 🎯 Priority | go/ast | Clean router patterns |
| gin         | Planned     | go/ast | Most popular Go framework |
| echo        | Planned     | go/ast | |
| fiber       | Planned     | go/ast | |
| gorilla/mux | Planned     | go/ast | Legacy but widely used |
| net/http    | Planned     | go/ast | Standard library |

#### Rust

| Framework | Status | Parser | Notes |
|-----------|--------|--------|-------|
| axum      | Future | tree-sitter | Growing popularity |
| actix-web | Future | tree-sitter | |
| rocket    | Future | tree-sitter | Has macro-based routing |

---

## Schema Extraction

### Zod Schemas (TypeScript)

```typescript
// Input
const CreateUserSchema = z.object({
  name: z.string().min(1).max(100),
  email: z.string().email(),
  age: z.number().int().positive().optional(),
})

// Output (OpenAPI Schema)
{
  "type": "object",
  "required": ["name", "email"],
  "properties": {
    "name": { 
      "type": "string",
      "minLength": 1,
      "maxLength": 100
    },
    "email": { 
      "type": "string",
      "format": "email"
    },
    "age": { 
      "type": "integer",
      "minimum": 1
    }
  }
}
```

### Zod Method Mapping

| Zod Method | OpenAPI Schema |
|------------|---------------|
| `z.string()` | `{ type: "string" }` |
| `z.number()` | `{ type: "number" }` |
| `z.boolean()` | `{ type: "boolean" }` |
| `z.array(T)` | `{ type: "array", items: T }` |
| `z.object({})` | `{ type: "object", properties: {} }` |
| `.optional()` | Remove from `required` |
| `.nullable()` | Add `null` to type |
| `.default(v)` | `{ default: v }` |
| `.min(n)` | `{ minLength: n }` (string) or `{ minimum: n }` (number) |
| `.max(n)` | `{ maxLength: n }` (string) or `{ maximum: n }` (number) |
| `.email()` | `{ format: "email" }` |
| `.uuid()` | `{ format: "uuid" }` |
| `.url()` | `{ format: "uri" }` |
| `.datetime()` | `{ format: "date-time" }` |
| `.int()` | `{ type: "integer" }` |
| `.positive()` | `{ minimum: 1 }` |
| `.nonnegative()` | `{ minimum: 0 }` |
| `z.enum([...])` | `{ enum: [...] }` |

### TypeScript Interfaces

```typescript
// Input
interface CreateUserRequest {
  name: string
  email: string
  age?: number
}

// Output (OpenAPI Schema)
{
  "type": "object",
  "required": ["name", "email"],
  "properties": {
    "name": { "type": "string" },
    "email": { "type": "string" },
    "age": { "type": "integer" }
  }
}
```

### Go Structs

```go
// Input
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=1,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   *int   `json:"age,omitempty" validate:"omitempty,gt=0"`
}

// Output (OpenAPI Schema)
{
  "type": "object",
  "required": ["name", "email"],
  "properties": {
    "name": { 
      "type": "string",
      "minLength": 1,
      "maxLength": 100
    },
    "email": { 
      "type": "string",
      "format": "email"
    },
    "age": { 
      "type": "integer",
      "minimum": 1
    }
  }
}
```

### Go Tag Mapping

| Validate Tag | OpenAPI Schema |
|--------------|---------------|
| `required` | Add to `required` array |
| `min=n` | `{ minLength: n }` (string) or `{ minimum: n }` (number) |
| `max=n` | `{ maxLength: n }` (string) or `{ maximum: n }` (number) |
| `email` | `{ format: "email" }` |
| `uuid` | `{ format: "uuid" }` |
| `url` | `{ format: "uri" }` |
| `gt=n` | `{ minimum: n, exclusiveMinimum: true }` |
| `gte=n` | `{ minimum: n }` |
| `lt=n` | `{ maximum: n, exclusiveMaximum: true }` |
| `lte=n` | `{ maximum: n }` |
| `oneof=a b c` | `{ enum: ["a", "b", "c"] }` |

---

## Claude Code Integration

### Recommended Workflow

api2spec is designed to work seamlessly with Claude Code for enhanced documentation:

```bash
# 1. Generate base spec (static analysis)
api2spec generate

# 2. Use Claude Code to enhance
# Claude Code can:
# - Add meaningful descriptions to operations
# - Generate request/response examples
# - Fill in response schemas for complex handlers
# - Suggest tags and groupings
# - Fix inconsistencies
```

### Why Claude Code Instead of API Integration?

1. **No API costs** — Users don't need API keys or pay per request
2. **No configuration** — No provider setup, rate limits, or token management
3. **Interactive refinement** — Claude Code enables back-and-forth iteration
4. **Context-aware** — Claude Code sees your entire codebase
5. **Privacy** — Code stays local until you choose to share

### Future: Enterprise Edition

A future paid enterprise version may include:

- Programmatic LLM API integration for CI/CD pipelines
- Batch processing of large codebases
- Custom model selection
- Automated documentation refresh

---

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/api-spec.yml
name: API Spec Check

on:
  pull_request:
    paths:
      - 'src/**'
      - 'openapi.yaml'

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install api2spec
        run: |
          curl -fsSL https://github.com/api2spec/api2spec/releases/latest/download/api2spec_linux_amd64.tar.gz | tar xz
          sudo mv api2spec /usr/local/bin/
      
      - name: Check API spec is up to date
        run: api2spec check --ci
      
      - name: Upload diff on failure
        if: failure()
        run: |
          api2spec diff > spec-diff.txt
          cat spec-diff.txt
        
      - uses: actions/upload-artifact@v4
        if: failure()
        with:
          name: spec-diff
          path: spec-diff.txt
```

### Pre-commit Hook

```bash
#!/bin/sh
# .husky/pre-commit

api2spec check --strict

if [ $? -ne 0 ]; then
  echo "API spec is out of date. Run 'api2spec generate' to update."
  exit 1
fi
```

### GitLab CI

```yaml
# .gitlab-ci.yml
api-spec-check:
  stage: test
  image: golang:1.22
  before_script:
    - go install github.com/api2spec/api2spec@latest
  script:
    - api2spec check --ci
  rules:
    - changes:
        - src/**/*
        - openapi.yaml
```

---

## Repository Structure

### Multi-Repo Architecture

api2spec uses a multi-repo structure to keep concerns separated:

```
github.com/api2spec/
├── api2spec/                      # Main CLI tool (Go)
├── homebrew-tap/                  # Homebrew formula
├── api2spec-fixture-hono/         # Hono test fixture API
├── api2spec-fixture-express/      # Express test fixture API
├── api2spec-fixture-chi/          # Go chi test fixture API
├── api2spec-fixture-gin/          # Go gin test fixture API
└── api2spec-fixture-fastify/      # Fastify test fixture API
```

### Main Repository (api2spec)

```
api2spec/
├── cmd/
│   └── api2spec/
│       └── main.go               # CLI entry point
├── internal/
│   ├── cli/                      # Cobra commands
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── generate.go
│   │   ├── watch.go
│   │   ├── check.go
│   │   ├── diff.go
│   │   └── print.go
│   ├── config/                   # Viper-based config
│   │   └── config.go
│   ├── parser/                   # Language parsers
│   │   ├── typescript.go         # tree-sitter-typescript
│   │   ├── golang.go             # go/ast (native)
│   │   └── zod.go                # Zod pattern matcher
│   ├── plugins/                  # Framework plugins
│   │   ├── plugin.go             # Interface definition
│   │   ├── hono.go
│   │   ├── express.go
│   │   ├── chi.go
│   │   └── gin.go
│   ├── schema/                   # Schema extraction
│   │   ├── registry.go           # Schema registry
│   │   ├── zod.go                # Zod → JSON Schema
│   │   ├── typescript.go         # TS interface → JSON Schema
│   │   └── golang.go             # Go struct → JSON Schema
│   ├── openapi/                  # OpenAPI spec generation
│   │   ├── builder.go
│   │   ├── merger.go
│   │   └── differ.go
│   └── scanner/                  # File discovery
│       └── scanner.go
├── pkg/
│   └── types/                    # Shared types
│       ├── route.go
│       ├── schema.go
│       └── openapi.go
├── testdata/                     # Inline test fixtures
├── go.mod
├── go.sum
├── .goreleaser.yaml              # Release automation
└── README.md
```

### Fixture Repository Structure

Each fixture repo follows the same structure:

```
api2spec-fixture-hono/
├── src/
│   ├── index.ts              # App entry point
│   ├── routes/               # Route definitions
│   │   ├── users.ts
│   │   ├── posts.ts
│   │   └── auth.ts
│   ├── schemas/              # Zod/type definitions
│   │   ├── user.ts
│   │   ├── post.ts
│   │   └── common.ts
│   └── middleware/           # Middleware (for testing detection)
├── expected/
│   └── openapi.yaml          # Expected output (gold standard)
├── api2spec.yaml             # Config for this fixture
├── package.json
├── tsconfig.json
└── README.md                 # Documents what patterns this tests
```

---

## Technical Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                         api2spec CLI                            │
│                           (Cobra)                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │   Config    │  │   Scanner   │  │    Spec Generator       │ │
│  │   (Viper)   │  │             │  │                         │ │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘ │
│         │                │                      │              │
│         ▼                ▼                      ▼              │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Plugin Manager                        │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │   │
│  │  │  Hono   │ │ Express │ │   chi   │ │   gin   │  ...   │   │
│  │  │ Plugin  │ │ Plugin  │ │ Plugin  │ │ Plugin  │        │   │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Language Parsers                       │   │
│  │  ┌───────────────────────┐ ┌───────────────────────┐    │   │
│  │  │  tree-sitter          │ │  go/ast               │    │   │
│  │  │  (TypeScript/JS)      │ │  (Go - native)        │    │   │
│  │  └───────────────────────┘ └───────────────────────┘    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Schema Extractors                       │   │
│  │  ┌───────────┐ ┌───────────┐ ┌───────────┐              │   │
│  │  │    Zod    │ │ TypeScript│ │ Go Struct │              │   │
│  │  │  Patterns │ │ Interface │ │  + Tags   │              │   │
│  │  └───────────┘ └───────────┘ └───────────┘              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Output Writer                         │   │
│  │              (YAML / JSON / stdout)                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Component      | Technology              | Rationale                                |
|----------------|-------------------------|------------------------------------------|
| Language       | Go                      | Single binary, cross-platform            |
| CLI Framework  | Cobra                   | Standard for Go CLIs                     |
| Config         | Viper                   | Flexible config loading                  |
| Parser (TS/JS) | tree-sitter             | Fast, accurate syntax parsing            |
| Parser (Go)    | go/ast                  | Native, full type information            |
| File watching  | fsnotify                | Cross-platform file watching             |
| Output         | gopkg.in/yaml.v3        | YAML 1.2 support                         |
| Testing        | go test + testify       | Standard Go testing                      |
| Release        | goreleaser              | Cross-platform builds, Homebrew          |

### Dependencies

```go
// go.mod (key dependencies)
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    github.com/smacker/go-tree-sitter v0.0.0-20240625050157-a31a98a7c0f6
    github.com/fsnotify/fsnotify v1.7.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/stretchr/testify v1.9.0
)
```

---

## Release & Distribution

### goreleaser Configuration

```yaml
# .goreleaser.yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=1  # Required for tree-sitter
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

brews:
  - repository:
      owner: api2spec
      name: homebrew-tap
    homepage: https://github.com/api2spec/api2spec
    description: Code-first OpenAPI specification generator
    license: MIT
    install: |
      bin.install "api2spec"

dockers:
  - image_templates:
      - "ghcr.io/api2spec/api2spec:{{ .Tag }}"
      - "ghcr.io/api2spec/api2spec:latest"
    dockerfile: Dockerfile
    build_flag_templates:
      - "--platform=linux/amd64"

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
```

### Homebrew Tap

```ruby
# homebrew-tap/Formula/api2spec.rb
class Api2spec < Formula
  desc "Code-first OpenAPI specification generator"
  homepage "https://github.com/api2spec/api2spec"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/api2spec/api2spec/releases/download/v#{version}/api2spec_darwin_amd64.tar.gz"
      sha256 "..."
    end
    on_arm do
      url "https://github.com/api2spec/api2spec/releases/download/v#{version}/api2spec_darwin_arm64.tar.gz"
      sha256 "..."
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/api2spec/api2spec/releases/download/v#{version}/api2spec_linux_amd64.tar.gz"
      sha256 "..."
    end
    on_arm do
      url "https://github.com/api2spec/api2spec/releases/download/v#{version}/api2spec_linux_arm64.tar.gz"
      sha256 "..."
    end
  end

  def install
    bin.install "api2spec"
  end

  test do
    system "#{bin}/api2spec", "--version"
  end
end
```

---

## Roadmap

### Phase 1: Foundation (MVP)

- [ ] Core CLI structure (Cobra)
- [ ] Configuration loading (Viper)
- [ ] tree-sitter TypeScript parser
- [ ] Zod schema extraction
- [ ] Hono plugin
- [ ] YAML/JSON output
- [ ] Basic `generate` command
- [ ] Hono fixture repository
- [ ] goreleaser setup
- [ ] Homebrew tap

### Phase 2: Essential Features

- [ ] `watch` command (fsnotify)
- [ ] `check` command for CI
- [ ] `diff` command
- [ ] TypeScript interface extraction
- [ ] Express plugin + fixture
- [ ] chi plugin + fixture (Go, native go/ast)
- [ ] Go struct extraction

### Phase 3: Polish & Ecosystem

- [ ] gin plugin + fixture
- [ ] Fastify plugin + fixture
- [ ] `init` command with scaffolding
- [ ] Merge strategy for existing specs
- [ ] JSDoc annotation support
- [ ] GitHub Action

### Phase 4: Growth

- [ ] Plugin contribution guide
- [ ] More framework plugins (community)
- [ ] Documentation site
- [ ] VS Code extension (syntax highlighting for config)

---

## Contributing

### Development Setup

```bash
git clone https://github.com/api2spec/api2spec
cd api2spec
go mod download
go build ./cmd/api2spec
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/plugins/...
```

### Testing Against Fixtures

```bash
# Clone fixture repos
git clone https://github.com/api2spec/api2spec-fixture-hono ../api2spec-fixture-hono

# Run integration tests
go test ./tests/integration/...

# Test specific fixture
go test ./tests/integration/hono_test.go
```

### Adding a Framework Plugin

1. Create fixture repository first (`api2spec-fixture-[framework]`)
2. Define expected OpenAPI output in `expected/openapi.yaml`
3. Create plugin file in `internal/plugins/[framework].go`
4. Implement `FrameworkPlugin` interface
5. Write unit tests
6. Run against fixture to validate

---

## License

### FSL-1.1-MIT (Functional Source License)

```
Functional Source License, Version 1.1, MIT Future License

Licensor: llbbl
Licensed Work: api2spec

Additional Use Grant: You may use the Licensed Work for any purpose
except for providing a commercial hosted service that offers the
functionality of the Licensed Work to third parties as a managed
service or API.

Change Date: Two years from each release date
Change License: MIT
```

**What this means:**

✅ **Allowed:**
- Use api2spec in your projects (commercial or personal)
- Use api2spec in your company's internal tooling
- Modify the source code for your own use
- Contribute back to the project
- Fork for non-competing purposes

❌ **Not Allowed:**
- Offering api2spec as a hosted/managed service
- Building a commercial SaaS product around api2spec functionality
- Repackaging and selling api2spec as a service

After 2 years from each release, that version converts to MIT and all restrictions are removed.

For more information: https://fsl.software


---

## Related Projects

- [Scalar](https://github.com/scalar/scalar) - Beautiful API references from OpenAPI
- [Redoc](https://github.com/Redocly/redoc) - OpenAPI documentation generator
- [Spectral](https://github.com/stoplightio/spectral) - OpenAPI linter
- [openapi-generator](https://github.com/OpenAPITools/openapi-generator) - Generate clients/servers from OpenAPI
- [tree-sitter](https://tree-sitter.github.io/tree-sitter/) - Parser generator for syntax analysis
