# Schema Extraction: Python

This document describes how api2spec extracts schemas from Python code patterns and their limitations.

## Frameworks Supported

- **FastAPI** - Modern async framework with automatic OpenAPI
- **Flask** - Lightweight WSGI framework

---

## Pydantic Models (FastAPI)

**Source pattern:**
```python
from pydantic import BaseModel, Field
from typing import Optional
from datetime import datetime

class Tea(BaseModel):
    id: str
    name: str = Field(..., min_length=1, max_length=100)
    description: Optional[str] = None
    steep_temp_celsius: int = Field(..., ge=60, le=100)
    created_at: datetime
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
    created_at:
      type: string
      format: date-time
  required:
    - id
    - name
    - steep_temp_celsius
    - created_at
```

**Capabilities:**
- Extracts classes inheriting from `BaseModel`
- Detects `Optional[T]` as nullable
- Maps Python type hints to OpenAPI types
- Generates `required` array for non-optional fields
- Handles `datetime`, `date`, `UUID` types

**Limitations:**
- `Field()` constraints (min/max, pattern) not extracted
- `validator` decorators not analyzed
- Union types (`str | int`) map to first type only
- Nested model references not resolved to `$ref`
- `Config` class settings not applied
- Generic models (`List[T]`) have limited support

---

## Dataclasses (Flask)

**Source pattern:**
```python
from dataclasses import dataclass
from typing import Optional
from datetime import datetime

@dataclass
class Tea:
    id: str
    name: str
    description: Optional[str]
    steep_temp_celsius: int
    created_at: datetime
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
    created_at:
      type: string
      format: date-time
  required:
    - id
    - name
    - steep_temp_celsius
    - created_at
```

**Capabilities:**
- Extracts `@dataclass` decorated classes
- Type hints mapped to OpenAPI types
- `Optional[T]` detected as nullable

**Limitations:**
- `field(default=...)` not analyzed for required detection
- No validation metadata available
- `@dataclass_json` decorators not processed

---

## Type Mapping Reference

| Python Type | OpenAPI Type | Format |
|-------------|--------------|--------|
| `str` | `string` | - |
| `int` | `integer` | - |
| `float` | `number` | - |
| `bool` | `boolean` | - |
| `list`, `List[T]` | `array` | items: T |
| `dict`, `Dict[K, V]` | `object` | - |
| `Optional[T]` | T | nullable: true |
| `datetime` | `string` | `date-time` |
| `date` | `string` | `date` |
| `time` | `string` | `time` |
| `UUID` | `string` | `uuid` |
| `bytes` | `string` | `byte` |
| `Decimal` | `number` | - |
| `Any` | `object` | - |
| Custom class | `object` | - |
| `Enum` | `string` | - |

---

## Known Issues & Future Improvements

### Not Yet Supported
- [ ] Pydantic `Field()` constraints (min, max, pattern, etc.)
- [ ] Pydantic validators and root validators
- [ ] Python `Enum` value extraction
- [ ] Union types (`T | None` syntax in 3.10+)
- [ ] `Literal` types for enum-like constraints
- [ ] Nested model `$ref` resolution
- [ ] `@dataclass` field defaults for required detection
- [ ] TypedDict support
- [ ] Attrs library support

### Framework-Specific Notes

**FastAPI:**
- Routes from `@app.get()`, `@router.post()`, etc.
- Path parameters from `{param}` in route path
- `Depends()` not analyzed for schema
- Response model from `response_model=` parameter (not extracted)

**Flask:**
- Routes from `@app.route()` and `@bp.route()`
- Blueprint prefixes tracked
- No built-in schema system (relies on dataclasses/Pydantic)
