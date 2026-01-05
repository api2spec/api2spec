# Schema Extraction: Go

This document describes how api2spec extracts schemas from Go code patterns and their limitations.

## Frameworks Supported

- **Chi** - Lightweight router
- **Gin** - High-performance framework
- **Echo** - Minimalist framework
- **Fiber** - Express-inspired framework
- **Gorilla Mux** - Powerful URL router
- **stdlib** - `net/http` standard library

---

## Go Structs with JSON Tags

**Source pattern:**
```go
type Tea struct {
    ID               string     `json:"id"`
    Name             string     `json:"name"`
    Description      *string    `json:"description,omitempty"`
    SteepTempCelsius int        `json:"steepTempCelsius"`
    CaffeineLevel    string     `json:"caffeineLevel"`
    CreatedAt        time.Time  `json:"createdAt"`
}
```

**Extracted schema:**
```yaml
Tea:
  type: object
  properties:
    id:
      type: string
    name:
      type: string
    description:
      type: string
      nullable: true
    steepTempCelsius:
      type: integer
    caffeineLevel:
      type: string
    createdAt:
      type: string
      format: date-time
  required:
    - id
    - name
    - steepTempCelsius
    - caffeineLevel
    - createdAt
```

**Capabilities:**
- Extracts structs with `json` tags
- Uses `json` tag name for property names
- Detects pointer types (`*string`) as nullable
- `omitempty` fields marked as not required
- Maps Go types to OpenAPI types

**Limitations:**
- Only structs with `json` tags are extracted
- Embedded structs not flattened
- Interface fields map to `object`
- Custom types require explicit mapping
- `json:"-"` fields still sometimes included

---

## Filtering Implementation Structs

api2spec filters out structs that appear to be implementation details rather than API schemas:

**Excluded suffixes:**
- `Handler`, `Controller`, `Service`
- `Repository`, `Store`, `Manager`
- `Middleware`, `Config`, `Options`
- `Server`, `Client`, `Pool`
- `Router`, `Engine`

**Example - excluded:**
```go
type TeaHandler struct {  // Excluded - has "Handler" suffix
    store TeaStore
}

type TeaStore struct {    // Excluded - has "Store" suffix
    db *sql.DB
}
```

**Example - included:**
```go
type Tea struct {         // Included - data model
    ID   string `json:"id"`
    Name string `json:"name"`
}

type CreateTeaRequest struct {  // Included - request DTO
    Name string `json:"name"`
}
```

---

## Type Mapping Reference

| Go Type | OpenAPI Type | Format |
|---------|--------------|--------|
| `string` | `string` | - |
| `int`, `int8`, `int16`, `int32`, `int64` | `integer` | - |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | `integer` | - |
| `float32`, `float64` | `number` | - |
| `bool` | `boolean` | - |
| `[]T` | `array` | items: T |
| `map[K]V` | `object` | - |
| `*T` | T | nullable: true |
| `time.Time` | `string` | `date-time` |
| `uuid.UUID` | `string` | `uuid` |
| `json.RawMessage` | `object` | - |
| `interface{}`, `any` | `object` | - |
| Custom struct | `object` | - |

---

## Known Issues & Future Improvements

### Not Yet Supported
- [ ] Embedded struct flattening
- [ ] `json:"-"` field exclusion
- [ ] Custom type resolution (type aliases)
- [ ] Struct tag validation (`validate` tags)
- [ ] Go 1.18+ generics
- [ ] `encoding/xml` tag support
- [ ] Enum-like `const` blocks
- [ ] Interface implementations

### Framework-Specific Notes

**Chi:**
- Routes from `r.Get()`, `r.Post()`, etc.
- Route groups via `r.Route()` and `r.Group()`
- URL parameters from `{param}` syntax

**Gin:**
- Routes from `r.GET()`, `r.POST()`, etc.
- Route groups via `r.Group()`
- Binding tags (`binding:"required"`) not extracted

**Echo:**
- Routes from `e.GET()`, `e.POST()`, etc.
- Groups via `e.Group()`
- Validation tags not extracted

**Fiber:**
- Routes from `app.Get()`, `app.Post()`, etc.
- Groups via `app.Group()`
- Path parameters from `:param` syntax

**Gorilla Mux:**
- Routes from `r.HandleFunc()` with `.Methods()`
- Subrouters via `r.PathPrefix().Subrouter()`
- Path variables from `{param}` syntax

**stdlib (net/http):**
- Routes from `http.HandleFunc()` patterns
- `http.ServeMux` routing
- Limited route detection capability
