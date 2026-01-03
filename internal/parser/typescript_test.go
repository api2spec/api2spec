// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTypeScriptParser(t *testing.T) {
	parser := NewTypeScriptParser()
	require.NotNil(t, parser)
}

func TestTypeScriptParser_ParseSource(t *testing.T) {
	parser := NewTypeScriptParser()

	source := `
interface User {
  id: number;
  name: string;
  email?: string;
}

type UserResponse = {
  user: User;
  token: string;
};
`
	result, err := parser.ParseSource("test.ts", source)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test.ts", result.Path)

	// Stub implementation returns empty results
	assert.Empty(t, result.Interfaces)
	assert.Empty(t, result.TypeAliases)
}

func TestTypeScriptParser_ParseFile(t *testing.T) {
	parser := NewTypeScriptParser()

	result, err := parser.ParseFile("test.ts")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test.ts", result.Path)
}

func TestTypeScriptParser_IsSupported(t *testing.T) {
	parser := NewTypeScriptParser()

	// Stub implementation returns false
	assert.False(t, parser.IsSupported())
}

func TestTypeScriptParser_SupportedExtensions(t *testing.T) {
	parser := NewTypeScriptParser()
	extensions := parser.SupportedExtensions()

	assert.Contains(t, extensions, ".ts")
	assert.Contains(t, extensions, ".tsx")
	assert.Contains(t, extensions, ".js")
	assert.Contains(t, extensions, ".jsx")
}

func TestTypeScriptParser_ExtractInterfaces(t *testing.T) {
	parser := NewTypeScriptParser()
	pf := &ParsedTSFile{
		Path: "test.ts",
		Interfaces: []TSInterface{
			{Name: "User", IsExported: true},
		},
	}

	interfaces := parser.ExtractInterfaces(pf)
	assert.Len(t, interfaces, 1)
	assert.Equal(t, "User", interfaces[0].Name)
}

func TestTypeScriptParser_ExtractTypeAliases(t *testing.T) {
	parser := NewTypeScriptParser()
	pf := &ParsedTSFile{
		Path: "test.ts",
		TypeAliases: []TSTypeAlias{
			{Name: "UserId", Type: "number", IsExported: true},
		},
	}

	aliases := parser.ExtractTypeAliases(pf)
	assert.Len(t, aliases, 1)
	assert.Equal(t, "UserId", aliases[0].Name)
}

func TestTypeScriptTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		tsType         string
		expectedType   string
		expectedFormat string
	}{
		{"string", "string", ""},
		{"number", "number", ""},
		{"boolean", "boolean", ""},
		{"Date", "string", "date-time"},
		{"any", "object", ""},
		{"unknown", "object", ""},
		{"null", "null", ""},
		{"undefined", "", ""},
		{"void", "", ""},
		{"string[]", "array", ""},
		{"User", "object", ""},
	}

	for _, tt := range tests {
		t.Run(tt.tsType, func(t *testing.T) {
			openAPIType, format := TypeScriptTypeToOpenAPI(tt.tsType)
			assert.Equal(t, tt.expectedType, openAPIType)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

func TestClassifyType(t *testing.T) {
	tests := []struct {
		tsType   string
		expected TSTypeKind
	}{
		{"string", TSKindPrimitive},
		{"number", TSKindPrimitive},
		{"boolean", TSKindPrimitive},
		{"null", TSKindPrimitive},
		{"undefined", TSKindPrimitive},
		{"void", TSKindPrimitive},
		{"any", TSKindPrimitive},
		{"unknown", TSKindPrimitive},
		{"string[]", TSKindArray},
		{"User[]", TSKindArray},
		{"string | number", TSKindUnion},
		{"User | null", TSKindUnion},
		{"A & B", TSKindIntersection},
		{"Promise<User>", TSKindGeneric},
		{"Array<string>", TSKindGeneric},
		{`"active"`, TSKindLiteral},
		{"42", TSKindLiteral},
		{"User", TSKindUnknown},
		{"MyCustomType", TSKindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.tsType, func(t *testing.T) {
			result := ClassifyType(tt.tsType)
			assert.Equal(t, tt.expected, result, "ClassifyType(%q) = %v, want %v", tt.tsType, result, tt.expected)
		})
	}
}

func TestTSTypeKind_String(t *testing.T) {
	tests := []struct {
		kind     TSTypeKind
		expected string
	}{
		{TSKindPrimitive, "primitive"},
		{TSKindInterface, "interface"},
		{TSKindTypeAlias, "typeAlias"},
		{TSKindArray, "array"},
		{TSKindUnion, "union"},
		{TSKindIntersection, "intersection"},
		{TSKindLiteral, "literal"},
		{TSKindGeneric, "generic"},
		{TSKindUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.kind.String())
		})
	}
}

func TestValidateTypeScriptFile(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		expectErr bool
	}{
		{
			name:      "valid source",
			source:    "interface User { id: number; }",
			expectErr: false,
		},
		{
			name:      "empty source",
			source:    "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTypeScriptFile(tt.source)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTSInterface_Struct(t *testing.T) {
	iface := TSInterface{
		Name:        "User",
		Description: "A user in the system",
		IsExported:  true,
		Line:        10,
		Extends:     []string{"BaseEntity"},
		Properties: []TSProperty{
			{
				Name:        "id",
				Type:        "number",
				IsOptional:  false,
				IsReadonly:  true,
				Description: "Unique identifier",
			},
			{
				Name:       "email",
				Type:       "string",
				IsOptional: true,
			},
		},
	}

	assert.Equal(t, "User", iface.Name)
	assert.True(t, iface.IsExported)
	assert.Len(t, iface.Properties, 2)
	assert.True(t, iface.Properties[0].IsReadonly)
	assert.True(t, iface.Properties[1].IsOptional)
}

func TestTSTypeAlias_Struct(t *testing.T) {
	alias := TSTypeAlias{
		Name:        "UserId",
		Type:        "number",
		Description: "A unique user identifier",
		IsExported:  true,
		Line:        5,
	}

	assert.Equal(t, "UserId", alias.Name)
	assert.Equal(t, "number", alias.Type)
	assert.True(t, alias.IsExported)
}

func TestParsedTSFile_Struct(t *testing.T) {
	pf := ParsedTSFile{
		Path: "models/user.ts",
		Interfaces: []TSInterface{
			{Name: "User"},
			{Name: "UserProfile"},
		},
		TypeAliases: []TSTypeAlias{
			{Name: "UserId"},
		},
		Exports: []string{"User", "UserProfile", "UserId"},
	}

	assert.Equal(t, "models/user.ts", pf.Path)
	assert.Len(t, pf.Interfaces, 2)
	assert.Len(t, pf.TypeAliases, 1)
	assert.Len(t, pf.Exports, 3)
}
