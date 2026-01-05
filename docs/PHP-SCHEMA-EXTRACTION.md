# Schema Extraction: Language & Framework Support

This document describes how api2spec extracts schemas from different code patterns and their limitations.

## PHP / Laravel

### Plain PHP 8 Classes (Constructor Promotion)

**Source pattern:**
```php
class Tea
{
    public function __construct(
        public string $id,
        public string $name,
        public ?string $description,
        public int $steep_temp_celsius,
    ) {}
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
  required:
    - id
    - name
    - steep_temp_celsius
```

**Capabilities:**
- Extracts all public properties
- Detects nullable types (`?string` → `nullable: true`)
- Generates `required` array for non-nullable fields
- Maps PHP types to OpenAPI types

**Limitations:**
- Only extracts `public` properties (private/protected ignored)
- Custom classes/enums map to `type: object` (no enum value extraction)
- No validation rules extracted (min/max, patterns, etc.)

---

### Eloquent Models ($fillable + $casts)

**Source pattern:**
```php
class TeaEloquent extends Model
{
    protected $fillable = [
        'name',
        'type',
        'steep_temp_celsius',
    ];

    protected $casts = [
        'type' => TeaType::class,
        'steep_temp_celsius' => 'integer',
        'created_at' => 'datetime',
    ];
}
```

**Extracted schema:**
```yaml
TeaEloquent:
  type: object
  properties:
    name:
      type: string
    type:
      type: string
    steep_temp_celsius:
      type: integer
    created_at:
      type: string
      format: date-time
```

**Capabilities:**
- Extracts fields from `$fillable` array
- Extracts additional fields from `$casts` (like timestamps)
- Maps cast types to OpenAPI: `integer`, `boolean`, `datetime`, `array`, etc.

**Limitations:**
- No `id` field (typically not in `$fillable`)
- No `nullable` detection (not expressed in these arrays)
- No `required` fields (can't determine from `$fillable` alone)
- Enum casts map to `string` (no enum values extracted)
- Laravel 11's `casts()` method not supported (only `$casts` property)
- `$hidden` fields still included (no filtering)

---

## Comparison: Plain PHP vs Eloquent

| Feature | Plain PHP 8 | Eloquent |
|---------|-------------|----------|
| Property detection | Constructor params | `$fillable` + `$casts` |
| `id` field | ✅ Included | ❌ Not included |
| Nullable types | ✅ From `?Type` | ❌ Not available |
| Required fields | ✅ Generated | ❌ Not available |
| Type mapping | ✅ From PHP types | ✅ From `$casts` |
| datetime format | ✅ From `Carbon` type | ✅ From `datetime` cast |
| Enum values | ❌ Maps to object | ❌ Maps to string |

**Recommendation:** For the richest OpenAPI schemas, prefer plain PHP 8 classes with typed constructor properties over Eloquent models.

---

## Known Issues & Future Improvements

### Not Yet Supported
- [ ] PHP Enum extraction (get actual enum values)
- [ ] Laravel Form Request validation rules → schema constraints
- [ ] Laravel API Resources → response schemas
- [ ] `$hidden` / `$visible` filtering for Eloquent
- [ ] Laravel 11 `casts()` method syntax
- [ ] PHPDoc `@property` annotations as fallback

### Type Mapping Reference

| PHP Type | OpenAPI Type | Format |
|----------|--------------|--------|
| `string` | `string` | - |
| `int`, `integer` | `integer` | - |
| `float`, `double` | `number` | - |
| `bool`, `boolean` | `boolean` | - |
| `array` | `array` | - |
| `DateTime`, `Carbon` | `string` | `date-time` |
| `DateTimeImmutable` | `string` | `date-time` |
| Custom class | `object` | - |
| Enum class | `object` or `string` | - |

| Laravel Cast | OpenAPI Type | Format |
|--------------|--------------|--------|
| `string` | `string` | - |
| `integer`, `int` | `integer` | - |
| `real`, `float`, `double` | `number` | - |
| `boolean`, `bool` | `boolean` | - |
| `array`, `collection` | `array` | - |
| `object` | `object` | - |
| `date` | `string` | `date` |
| `datetime`, `timestamp` | `string` | `date-time` |
| `CustomCast::class` | `string` | - |
