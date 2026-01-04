# api2spec

## Code-First OpenAPI Generation for Modern APIs

---

# The Problem

**Keeping API documentation in sync with code is painful.**

- Manual spec writing is tedious and error-prone
- Runtime reflection requires full environment setup
- CI/CD pipelines shouldn't need your entire runtime
- Documentation drift leads to broken integrations

*Developers spend hours maintaining specs instead of building features.*

---

# The Solution

**api2spec: Static analysis that just works.**

```
Your Code  →  api2spec  →  openapi.yaml
   (Any Framework)    (Parse + Infer)     (Auto-synced)
```

- **Tree-sitter powered** - No runtime, no dependencies
- **Framework-aware** - Understands your routing patterns
- **Schema extraction** - Zod, Pydantic, Go structs, and more
- **CI/CD ready** - Validate specs match code on every commit

---

# Massive Framework Support

**36 frameworks across 15 languages**

| Language | Frameworks |
|----------|------------|
| Go | chi, gin, echo, fiber |
| TypeScript | Hono, Express, Fastify, NestJS, Koa, Elysia |
| Python | FastAPI, Flask, Django REST Framework |
| Rust | Axum, Actix-web, Rocket |
| C# | ASP.NET Core, FastEndpoints, Nancy |
| PHP | Laravel, Symfony, Slim |
| Java/Kotlin | Spring Boot, Micronaut, Ktor |
| C++ | Drogon, Oat++, Crow |
| Scala | Play, Tapir |
| Swift | Vapor |
| Haskell | Servant |
| + Ruby, Elixir, Gleam | Rails, Sinatra, Phoenix, Wisp |

---

# Get Started in 30 Seconds

```bash
# Install
brew install api2spec/tap/api2spec

# Auto-detect framework and generate config
api2spec init

# Generate your OpenAPI spec
api2spec generate

# Add to CI - fail if spec drifts
api2spec check --ci
```

**Your API docs, always in sync.**

github.com/api2spec/api2spec
