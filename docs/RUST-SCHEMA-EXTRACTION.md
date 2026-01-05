# Schema Extraction: Rust

This document describes how api2spec extracts schemas from Rust code patterns and their limitations.

## Frameworks Supported

- **Axum** - Tokio-based web framework
- **Actix** - Actor-based web framework
- **Rocket** - Type-safe web framework

---

## Serde Structs

**Source pattern:**
```rust
#[derive(Serialize, Deserialize)]
pub struct Tea {
    pub id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub steep_temp_celsius: i32,
    #[serde(rename = "caffeineLevel")]
    pub caffeine_level: CaffeineLevel,
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
    steep_temp_celsius:
      type: integer
    caffeineLevel:
      type: string
  required:
    - id
    - name
    - steep_temp_celsius
    - caffeineLevel
```

**Capabilities:**
- Extracts structs with `#[derive(Serialize)]` or `#[derive(Deserialize)]`
- Detects `Option<T>` as nullable
- Honors `#[serde(rename = "...")]` for field names
- Maps Rust types to OpenAPI types
- Generates `required` array for non-Option fields

**Limitations:**
- Only extracts public structs with serde derives
- Enum variants not extracted (maps to `string` or `object`)
- `#[serde(flatten)]` not followed
- `#[serde(skip)]` fields still included
- Generic structs (`Vec<T>`, custom generics) have limited support

---

## Type Mapping Reference

| Rust Type | OpenAPI Type | Format |
|-----------|--------------|--------|
| `String`, `&str` | `string` | - |
| `i8`, `i16`, `i32`, `i64` | `integer` | - |
| `u8`, `u16`, `u32`, `u64` | `integer` | - |
| `f32`, `f64` | `number` | - |
| `bool` | `boolean` | - |
| `Vec<T>` | `array` | items: T |
| `Option<T>` | T | nullable: true |
| `HashMap<K, V>` | `object` | - |
| `DateTime<Utc>` (chrono) | `string` | `date-time` |
| `NaiveDate` (chrono) | `string` | `date` |
| `Uuid` | `string` | `uuid` |
| Custom struct | `object` | - |
| Enum | `string` | - |

---

## Known Issues & Future Improvements

### Not Yet Supported
- [ ] Rust enum variant extraction (unit, tuple, struct variants)
- [ ] `#[serde(flatten)]` field expansion
- [ ] `#[serde(skip)]` field filtering
- [ ] `#[serde(default)]` for optional detection
- [ ] Generic type parameter resolution
- [ ] Newtype pattern unwrapping (`struct Id(String)`)
- [ ] `#[serde(tag = "type")]` tagged enum representation

### Framework-Specific Notes

**Axum:**
- Routes extracted from `Router::new().route()` chains
- Path parameters from `:param` converted to `{param}`
- Handler function names used for operationId

**Actix:**
- Routes from `#[get]`, `#[post]`, etc. macros
- `web::Path<T>` and `web::Query<T>` for parameters
- `web::Json<T>` for request/response body schema linking

**Rocket:**
- Routes from `#[get]`, `#[post]`, etc. macros
- Path parameters from `<param>` syntax
- Request guards not analyzed for schema
