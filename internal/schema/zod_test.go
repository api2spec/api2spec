// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/parser"
)

func TestZodParser_ParseZodObject(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email(),
  age: z.number().int().positive().optional(),
});
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
	require.NoError(t, err)

	assert.Equal(t, "object", schema.Type)
	assert.NotNil(t, schema.Properties)
	assert.Len(t, schema.Properties, 4)

	// Check id property
	idProp := schema.Properties["id"]
	require.NotNil(t, idProp)
	assert.Equal(t, "string", idProp.Type)
	assert.Equal(t, "uuid", idProp.Format)

	// Check name property
	nameProp := schema.Properties["name"]
	require.NotNil(t, nameProp)
	assert.Equal(t, "string", nameProp.Type)

	// Check email property
	emailProp := schema.Properties["email"]
	require.NotNil(t, emailProp)
	assert.Equal(t, "string", emailProp.Type)
	assert.Equal(t, "email", emailProp.Format)
}

func TestZodParser_ParsePrimitiveTypes(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		wantType   string
		wantFormat string
	}{
		{
			name:     "z.string()",
			code:     `const s = z.string();`,
			wantType: "string",
		},
		{
			name:     "z.number()",
			code:     `const n = z.number();`,
			wantType: "number",
		},
		{
			name:     "z.boolean()",
			code:     `const b = z.boolean();`,
			wantType: "boolean",
		},
		{
			name:       "z.bigint()",
			code:       `const bi = z.bigint();`,
			wantType:   "integer",
			wantFormat: "int64",
		},
		{
			name:       "z.date()",
			code:       `const d = z.date();`,
			wantType:   "string",
			wantFormat: "date-time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullCode := `import { z } from 'zod';
` + tt.code

			tsParser := parser.NewTypeScriptParser()
			defer tsParser.Close()

			pf, err := tsParser.ParseSource("test.ts", fullCode)
			require.NoError(t, err)
			defer pf.Close()

			require.Len(t, pf.ZodSchemas, 1)

			zodParser := NewZodParser(tsParser)
			schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
			require.NoError(t, err)

			assert.Equal(t, tt.wantType, schema.Type)
			if tt.wantFormat != "" {
				assert.Equal(t, tt.wantFormat, schema.Format)
			}
		})
	}
}

func TestZodParser_ParseStringModifiers(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const EmailSchema = z.string().email();
const UrlSchema = z.string().url();
const UuidSchema = z.string().uuid();
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 3)

	zodParser := NewZodParser(tsParser)

	// Find EmailSchema
	for _, zs := range pf.ZodSchemas {
		schema, err := zodParser.ParseZodSchema(zs.Node, pf.Content)
		require.NoError(t, err)

		assert.Equal(t, "string", schema.Type)
		switch zs.Name {
		case "EmailSchema":
			assert.Equal(t, "email", schema.Format)
		case "UrlSchema":
			assert.Equal(t, "uri", schema.Format)
		case "UuidSchema":
			assert.Equal(t, "uuid", schema.Format)
		}
	}
}

func TestZodParser_ParseArray(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const StringArraySchema = z.array(z.string());
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
	require.NoError(t, err)

	assert.Equal(t, "array", schema.Type)
	require.NotNil(t, schema.Items)
	assert.Equal(t, "string", schema.Items.Type)
}

func TestZodParser_ParseEnum(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const StatusSchema = z.enum(['active', 'inactive', 'pending']);
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
	require.NoError(t, err)

	assert.Equal(t, "string", schema.Type)
	assert.Len(t, schema.Enum, 3)
	assert.Contains(t, schema.Enum, "active")
	assert.Contains(t, schema.Enum, "inactive")
	assert.Contains(t, schema.Enum, "pending")
}

func TestZodParser_ParseRecord(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const RecordSchema = z.record(z.string(), z.number());
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
	require.NoError(t, err)

	assert.Equal(t, "object", schema.Type)
	require.NotNil(t, schema.AdditionalProperties)
	assert.Equal(t, "number", schema.AdditionalProperties.Type)
}

func TestZodParser_ExtractAndRegister(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const UserSchema = z.object({
  name: z.string(),
  email: z.string().email(),
});
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema := zodParser.ExtractAndRegister("UserSchema", pf.ZodSchemas[0].Node, pf.Content)

	assert.Equal(t, "UserSchema", schema.Title)
	assert.Equal(t, "object", schema.Type)

	// Check registry
	assert.True(t, zodParser.Registry().Has("UserSchema"))
	registeredSchema, ok := zodParser.Registry().Get("UserSchema")
	assert.True(t, ok)
	assert.Equal(t, "UserSchema", registeredSchema.Title)
}

func TestZodParser_ParseNullable(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const NullableStringSchema = z.string().nullable();
`

	tsParser := parser.NewTypeScriptParser()
	defer tsParser.Close()

	pf, err := tsParser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	require.Len(t, pf.ZodSchemas, 1)

	zodParser := NewZodParser(tsParser)
	schema, err := zodParser.ParseZodSchema(pf.ZodSchemas[0].Node, pf.Content)
	require.NoError(t, err)

	assert.Equal(t, "string", schema.Type)
	assert.True(t, schema.Nullable)
}

func TestExtractZodMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"z.string", "string"},
		{"z.object", "object"},
		{"z.array", "array"},
		{"z.string().email", "email"},
		{"string", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractZodMethod(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
