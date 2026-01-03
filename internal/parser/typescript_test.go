// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package parser

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTypeScriptParser(t *testing.T) {
	parser := NewTypeScriptParser()
	require.NotNil(t, parser)
	defer parser.Close()
}

func TestTypeScriptParser_ParseSource_Basic(t *testing.T) {
	parser := NewTypeScriptParser()
	defer parser.Close()

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
	defer result.Close()

	assert.Equal(t, "test.ts", result.Path)
	assert.NotNil(t, result.Tree)
	assert.NotNil(t, result.RootNode)
}

func TestTypeScriptParser_ParseInterfaces(t *testing.T) {
	const testCode = `
interface User {
  id: string;
  name: string;
  email: string;
  age?: number;
}

export interface CreateUserRequest {
  name: string;
  email: string;
}

interface ExtendedUser extends User {
  role: string;
}
`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	assert.Len(t, pf.Interfaces, 3)

	// Check User interface
	user := findInterface(pf.Interfaces, "User")
	require.NotNil(t, user)
	assert.Len(t, user.Properties, 4)
	assert.False(t, user.IsExported)

	// Check properties
	idProp := findProperty(user.Properties, "id")
	require.NotNil(t, idProp)
	assert.Equal(t, "string", idProp.Type)
	assert.False(t, idProp.IsOptional)

	ageProp := findProperty(user.Properties, "age")
	require.NotNil(t, ageProp)
	assert.Equal(t, "number", ageProp.Type)
	assert.True(t, ageProp.IsOptional)

	// Check CreateUserRequest is exported
	createUser := findInterface(pf.Interfaces, "CreateUserRequest")
	require.NotNil(t, createUser)
	assert.True(t, createUser.IsExported)

	// Check ExtendedUser extends User
	extended := findInterface(pf.Interfaces, "ExtendedUser")
	require.NotNil(t, extended)
	assert.Contains(t, extended.Extends, "User")
}

func TestTypeScriptParser_ParseTypeAliases(t *testing.T) {
	const testCode = `
type Status = 'active' | 'inactive' | 'pending';

export type UserID = string;

type UserList = User[];
`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	assert.Len(t, pf.TypeAliases, 3)

	// Check Status type
	status := findTypeAlias(pf.TypeAliases, "Status")
	require.NotNil(t, status)
	assert.Contains(t, status.Type, "'active'")
	assert.False(t, status.IsExported)

	// Check UserID is exported
	userID := findTypeAlias(pf.TypeAliases, "UserID")
	require.NotNil(t, userID)
	assert.True(t, userID.IsExported)
}

func TestTypeScriptParser_ExtractZodSchemas(t *testing.T) {
	const testCode = `
import { z } from 'zod';

const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string().min(1).max(100),
  email: z.string().email(),
});

export const CreateUserSchema = z.object({
  name: z.string(),
  email: z.string().email(),
});

const SimpleString = z.string();
`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	assert.Len(t, pf.ZodSchemas, 3)

	// Check UserSchema
	userSchema := findZodSchema(pf.ZodSchemas, "UserSchema")
	require.NotNil(t, userSchema)
	assert.NotNil(t, userSchema.Node)
	assert.False(t, userSchema.IsExported)

	// Check CreateUserSchema is exported
	createSchema := findZodSchema(pf.ZodSchemas, "CreateUserSchema")
	require.NotNil(t, createSchema)
	assert.True(t, createSchema.IsExported)

	// Check SimpleString
	simpleString := findZodSchema(pf.ZodSchemas, "SimpleString")
	require.NotNil(t, simpleString)
	assert.NotNil(t, simpleString.Node)
}

func TestTypeScriptParser_IsSupported(t *testing.T) {
	parser := NewTypeScriptParser()
	defer parser.Close()

	assert.True(t, parser.IsSupported())
}

func TestTypeScriptParser_SupportedExtensions(t *testing.T) {
	parser := NewTypeScriptParser()
	defer parser.Close()

	exts := parser.SupportedExtensions()
	assert.Contains(t, exts, ".ts")
	assert.Contains(t, exts, ".tsx")
	assert.Contains(t, exts, ".js")
	assert.Contains(t, exts, ".jsx")
	assert.Contains(t, exts, ".mts")
	assert.Contains(t, exts, ".mjs")
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
		{"Array<string>", "array", ""},
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
		{"Array<string>", TSKindArray},
		{"string | number", TSKindUnion},
		{"User | null", TSKindUnion},
		{"A & B", TSKindIntersection},
		{"Promise<User>", TSKindGeneric},
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
	assert.Equal(t, "A user in the system", iface.Description)
	assert.True(t, iface.IsExported)
	assert.Equal(t, 10, iface.Line)
	assert.Equal(t, []string{"BaseEntity"}, iface.Extends)
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
	assert.Equal(t, "A unique user identifier", alias.Description)
	assert.True(t, alias.IsExported)
	assert.Equal(t, 5, alias.Line)
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

func TestTypeScriptParser_FindCallExpressions(t *testing.T) {
	const testCode = `
const app = new Hono();
app.get('/users', () => {});
app.post('/users', createUser);
console.log('test');
`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	calls := parser.FindCallExpressions(pf.RootNode, pf.Content)
	// Should find: Hono(), app.get(), app.post(), console.log()
	assert.GreaterOrEqual(t, len(calls), 3)
}

func TestTypeScriptParser_GetCalleeText(t *testing.T) {
	const testCode = `app.get('/users', handler);`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	calls := parser.FindCallExpressions(pf.RootNode, pf.Content)
	require.Len(t, calls, 1)

	calleeText := parser.GetCalleeText(calls[0], pf.Content)
	assert.Equal(t, "app.get", calleeText)
}

func TestTypeScriptParser_GetCallArguments(t *testing.T) {
	const testCode = `app.get('/users', handler, middleware);`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	calls := parser.FindCallExpressions(pf.RootNode, pf.Content)
	require.Len(t, calls, 1)

	args := parser.GetCallArguments(calls[0], pf.Content)
	assert.Len(t, args, 3)
}

func TestTypeScriptParser_ExtractStringLiteral(t *testing.T) {
	const testCode = `const path = '/users/:id';`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	// Find the string node
	var stringNode = pf.RootNode
	parser.walkNodes(pf.RootNode, func(n *sitter.Node) bool {
		if n.Type() == "string" {
			stringNode = n
			return false
		}
		return true
	})

	str, ok := parser.ExtractStringLiteral(stringNode, pf.Content)
	assert.True(t, ok)
	assert.Equal(t, "/users/:id", str)
}

func TestTypeScriptParser_GetMemberExpressionParts(t *testing.T) {
	const testCode = `app.get('/users', handler);`

	parser := NewTypeScriptParser()
	defer parser.Close()

	pf, err := parser.ParseSource("test.ts", testCode)
	require.NoError(t, err)
	defer pf.Close()

	calls := parser.FindCallExpressions(pf.RootNode, pf.Content)
	require.Len(t, calls, 1)

	// Get the member expression (callee)
	callee := calls[0].Child(0)
	require.NotNil(t, callee)

	obj, prop := parser.GetMemberExpressionParts(callee, pf.Content)
	assert.Equal(t, "app", obj)
	assert.Equal(t, "get", prop)
}

// Helper functions

func findInterface(interfaces []TSInterface, name string) *TSInterface {
	for i := range interfaces {
		if interfaces[i].Name == name {
			return &interfaces[i]
		}
	}
	return nil
}

func findProperty(properties []TSProperty, name string) *TSProperty {
	for i := range properties {
		if properties[i].Name == name {
			return &properties[i]
		}
	}
	return nil
}

func findTypeAlias(aliases []TSTypeAlias, name string) *TSTypeAlias {
	for i := range aliases {
		if aliases[i].Name == name {
			return &aliases[i]
		}
	}
	return nil
}

func findZodSchema(schemas []ZodSchema, name string) *ZodSchema {
	for i := range schemas {
		if schemas[i].Name == name {
			return &schemas[i]
		}
	}
	return nil
}

// sitter is imported at top of file
