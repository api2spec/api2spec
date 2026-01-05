# Schema Extraction: TypeScript / JavaScript

This document describes how api2spec extracts schemas from TypeScript/JavaScript code patterns and their limitations.

## Frameworks Supported

- **Express** - Minimal Node.js framework
- **Fastify** - Fast and low overhead framework
- **Koa** - Next-generation Express
- **Hono** - Ultrafast web framework
- **Elysia** - Bun-first framework
- **NestJS** - Enterprise Angular-inspired framework

---

## Zod Schemas

**Source pattern:**
```typescript
import { z } from 'zod';

export const TeaSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  description: z.string().optional(),
  steepTempCelsius: z.number().int().min(60).max(100),
  caffeineLevel: z.enum(['none', 'low', 'medium', 'high']),
  createdAt: z.date(),
});

export type Tea = z.infer<typeof TeaSchema>;
```

**Extracted schema:**
```yaml
Tea:
  type: object
  properties:
    id:
      type: string
      format: uuid
    name:
      type: string
    description:
      type: string
      nullable: true
    steepTempCelsius:
      type: integer
    caffeineLevel:
      type: string
      enum:
        - none
        - low
        - medium
        - high
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
- Extracts `z.object()` definitions
- Detects `.optional()` and `.nullable()` modifiers
- Extracts `z.enum()` values
- Maps Zod types to OpenAPI types
- Handles `.uuid()`, `.email()`, `.url()` formats
- Schema name inferred from variable name

**Limitations:**
- Validation constraints (min, max, regex) not extracted
- `.refine()` and `.transform()` not analyzed
- `z.union()` maps to first type only
- `z.intersection()` not merged
- `z.lazy()` for recursive types not resolved
- Nested schema `$ref` not generated

---

## TypeScript Interfaces

**Source pattern:**
```typescript
export interface Tea {
  id: string;
  name: string;
  description?: string;
  steepTempCelsius: number;
  caffeineLevel: 'none' | 'low' | 'medium' | 'high';
  createdAt: Date;
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
      type: number
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
- Extracts `interface` and `type` definitions
- Optional properties (`?:`) marked as nullable
- Maps TypeScript types to OpenAPI types

**Limitations:**
- Literal union types not extracted as enums
- `extends` not resolved (no inheritance flattening)
- Mapped types not expanded
- Generics not resolved
- Index signatures map to `object`

---

## Type Mapping Reference

| TypeScript/Zod Type | OpenAPI Type | Format |
|---------------------|--------------|--------|
| `string`, `z.string()` | `string` | - |
| `number`, `z.number()` | `number` | - |
| `z.number().int()` | `integer` | - |
| `boolean`, `z.boolean()` | `boolean` | - |
| `Array<T>`, `T[]`, `z.array()` | `array` | items: T |
| `object`, `z.object()` | `object` | - |
| `Date`, `z.date()` | `string` | `date-time` |
| `z.string().uuid()` | `string` | `uuid` |
| `z.string().email()` | `string` | `email` |
| `z.string().url()` | `string` | `uri` |
| `z.enum([...])` | `string` | enum: [...] |
| `z.optional()`, `?:` | - | nullable: true |
| `null`, `z.null()` | - | nullable: true |
| `any`, `unknown` | `object` | - |

---

## Known Issues & Future Improvements

### Not Yet Supported
- [ ] Zod validation constraints (min, max, length, regex)
- [ ] Zod refinements and transforms
- [ ] TypeScript literal union → enum conversion
- [ ] Interface `extends` resolution
- [ ] Type aliases and mapped types
- [ ] Generic type resolution
- [ ] `z.discriminatedUnion()` support
- [ ] `z.record()` for dynamic keys
- [ ] Nested schema `$ref` generation
- [ ] io-ts, yup, joi schema libraries

### Framework-Specific Notes

**Express:**
- Routes from `app.get()`, `router.post()`, etc.
- Router mounting via `app.use('/prefix', router)`
- Cross-file router imports tracked

**Fastify:**
- Routes from `fastify.get()`, etc.
- Schema option in route config not extracted
- Plugins not followed for route discovery

**Koa:**
- Routes from `router.get()`, etc.
- Koa-router middleware chains
- Context types not analyzed

**Hono:**
- Routes from `app.get()`, etc.
- Route groups via `app.route()`
- RPC mode not supported

**Elysia:**
- Routes from `.get()`, `.post()` chains
- Type-safe schemas via `t.Object()`
- Elysia's `t` schema similar to Zod

**NestJS:**
- Routes from `@Get()`, `@Post()` decorators
- `@Controller()` prefix handling
- DTOs with class-validator decorators not extracted
