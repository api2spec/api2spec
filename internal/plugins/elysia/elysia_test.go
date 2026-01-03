// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package elysia

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// elysiaTestCode is a comprehensive test fixture for Elysia route extraction.
const elysiaTestCode = `
import { Elysia, t } from 'elysia'
import { z } from 'zod'

const CreateUserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
})

const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string(),
  email: z.string().email(),
  createdAt: z.string().datetime(),
})

const app = new Elysia()
  .get('/users', () => [])
  .get('/users/:id', () => ({}))
  .post('/users', ({ body }) => ({}), {
    body: t.Object({
      name: t.String(),
      email: t.String()
    })
  })
  .put('/users/:id', () => ({}))
  .patch('/users/:id', () => ({}))
  .delete('/users/:id', () => {})

export default app
`

// elysiaGroupCode tests the group() pattern.
const elysiaGroupCode = `
import { Elysia } from 'elysia'

const app = new Elysia()
  .group('/api', app => app
    .get('/users', () => [])
    .get('/users/:id', () => ({}))
    .post('/users', () => ({}))
  )

export default app
`

// elysiaTypeBoxCode tests TypeBox validation.
const elysiaTypeBoxCode = `
import { Elysia, t } from 'elysia'

const app = new Elysia()
  .post('/products', ({ body }) => ({}), {
    body: t.Object({
      name: t.String(),
      price: t.Number(),
      description: t.Optional(t.String()),
      tags: t.Array(t.String())
    })
  })
  .post('/orders', ({ body }) => ({}), {
    body: t.Object({
      productId: t.String(),
      quantity: t.Integer()
    })
  })

export default app
`

// elysiaZodCode tests Zod validation via elysia-zod.
const elysiaZodCode = `
import { Elysia } from 'elysia'
import { z } from 'zod'

const UserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
})

const app = new Elysia()
  .post('/users', ({ body }) => ({}), {
    body: UserSchema
  })

export default app
`

// elysiaAllMethodsCode tests all HTTP methods.
const elysiaAllMethodsCode = `
import { Elysia } from 'elysia'

const app = new Elysia()
  .get('/test', () => 'get')
  .post('/test', () => 'post')
  .put('/test', () => 'put')
  .delete('/test', () => 'delete')
  .patch('/test', () => 'patch')
  .head('/test', () => 'head')
  .options('/test', () => 'options')

export default app
`

// elysiaChainedCode tests chained method calls.
const elysiaChainedCode = `
import { Elysia } from 'elysia'

const app = new Elysia()
  .use(somePlugin)
  .get('/items', () => [])
  .get('/items/:id', () => ({}))
  .post('/items', () => ({}))
  .listen(3000)

export default app
`

// elysiaNestedGroupCode tests nested groups.
const elysiaNestedGroupCode = `
import { Elysia } from 'elysia'

const app = new Elysia()
  .group('/api/v1', app => app
    .get('/health', () => 'ok')
    .group('/users', app => app
      .get('/', () => [])
      .get('/:id', () => ({}))
    )
  )

export default app
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "elysia", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()

	assert.Contains(t, exts, ".ts")
	assert.Contains(t, exts, ".tsx")
	assert.Contains(t, exts, ".js")
	assert.Contains(t, exts, ".jsx")
	assert.Contains(t, exts, ".mts")
	assert.Contains(t, exts, ".mjs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "elysia", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "elysia")
}

func TestPlugin_Detect_WithElysia(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "elysia": "^1.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithElysiaDevDep(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "devDependencies": {
    "elysia": "^1.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutElysia(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "hono": "^4.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoPackageJSON(t *testing.T) {
	dir := t.TempDir()

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_ExtractRoutes_BasicRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaTestCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 6 routes
	assert.Len(t, routes, 6)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	require.NotNil(t, getUsers)
	assert.Equal(t, "GET", getUsers.Method)
	assert.Equal(t, "/users", getUsers.Path)
	assert.Contains(t, getUsers.Tags, "users")

	// Check GET /users/:id -> /users/{id}
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	require.NotNil(t, getUserByID)
	assert.Equal(t, "GET", getUserByID.Method)
	assert.Len(t, getUserByID.Parameters, 1)
	assert.Equal(t, "id", getUserByID.Parameters[0].Name)
	assert.Equal(t, "path", getUserByID.Parameters[0].In)
	assert.True(t, getUserByID.Parameters[0].Required)

	// Check POST /users with TypeBox validation
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)
	// Should have request body from TypeBox schema
	assert.NotNil(t, postUsers.RequestBody)
	assert.True(t, postUsers.RequestBody.Required)

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	require.NotNil(t, putUser)

	// Check PATCH /users/{id}
	patchUser := findRoute(routes, "PATCH", "/users/{id}")
	require.NotNil(t, patchUser)
	assert.Equal(t, "PATCH", patchUser.Method)

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	require.NotNil(t, deleteUser)
	assert.Equal(t, "DELETE", deleteUser.Method)
}

func TestPlugin_ExtractRoutes_Group(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaGroupCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 3 routes with /api prefix
	assert.Len(t, routes, 3)

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	require.NotNil(t, getUsers)
	assert.Equal(t, "/api/users", getUsers.Path)

	// Check GET /api/users/{id}
	getUserByID := findRoute(routes, "GET", "/api/users/{id}")
	require.NotNil(t, getUserByID)
	assert.Len(t, getUserByID.Parameters, 1)

	// Check POST /api/users
	postUsers := findRoute(routes, "POST", "/api/users")
	require.NotNil(t, postUsers)
}

func TestPlugin_ExtractRoutes_TypeBoxValidation(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaTypeBoxCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.Len(t, routes, 2)

	// Check POST /products with TypeBox schema
	postProducts := findRoute(routes, "POST", "/products")
	require.NotNil(t, postProducts)
	assert.NotNil(t, postProducts.RequestBody)
	assert.True(t, postProducts.RequestBody.Required)

	bodyContent := postProducts.RequestBody.Content["application/json"]
	require.NotNil(t, bodyContent.Schema)
	assert.Equal(t, "object", bodyContent.Schema.Type)

	// Check POST /orders
	postOrders := findRoute(routes, "POST", "/orders")
	require.NotNil(t, postOrders)
	assert.NotNil(t, postOrders.RequestBody)
}

func TestPlugin_ExtractRoutes_ZodValidation(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaZodCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.Len(t, routes, 1)

	// Check POST /users with Zod schema reference
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.NotNil(t, postUsers.RequestBody)

	bodyContent := postUsers.RequestBody.Content["application/json"]
	require.NotNil(t, bodyContent.Schema)
	// Should have a reference to UserSchema
	assert.Contains(t, bodyContent.Schema.Ref, "UserSchema")
}

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaAllMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.Len(t, routes, 7)

	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["PATCH"])
	assert.True(t, methods["HEAD"])
	assert.True(t, methods["OPTIONS"])
}

func TestPlugin_ExtractRoutes_ChainedMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaChainedCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 3 routes (ignoring .use() and .listen())
	assert.Len(t, routes, 3)

	getItems := findRoute(routes, "GET", "/items")
	require.NotNil(t, getItems)

	getItemByID := findRoute(routes, "GET", "/items/{id}")
	require.NotNil(t, getItemByID)

	postItems := findRoute(routes, "POST", "/items")
	require.NotNil(t, postItems)
}

func TestPlugin_ExtractRoutes_IgnoresNonTS(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.go",
			Language: "go",
			Content:  []byte(`package main`),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_IgnoresNonElysia(t *testing.T) {
	p := New()

	// File that doesn't import Elysia
	code := `
import { Hono } from 'hono';

const app = new Hono();
app.get('/users', (c) => c.json([]));
`

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(code),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_SourceInfo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "/path/to/app.ts",
			Language: "typescript",
			Content:  []byte(elysiaTestCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	for _, r := range routes {
		assert.Equal(t, "/path/to/app.ts", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(elysiaTestCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract CreateUserSchema and UserSchema
	assert.GreaterOrEqual(t, len(schemas), 2)

	// Find CreateUserSchema
	var createUserSchema *struct{ Title, Type string }
	for _, s := range schemas {
		if s.Title == "CreateUserSchema" {
			createUserSchema = &struct{ Title, Type string }{s.Title, s.Type}
			break
		}
	}
	require.NotNil(t, createUserSchema)
	assert.Equal(t, "object", createUserSchema.Type)
}

func TestConvertPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:postId", "/users/{id}/posts/{postId}"},
		{"/api/v1/:resource/:id", "/api/v1/{resource}/{id}"},
		{"/:a/:b/:c", "/{a}/{b}/{c}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertPathParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path       string
		wantCount  int
		wantParams []string
	}{
		{"/users", 0, nil},
		{"/users/{id}", 1, []string{"id"}},
		{"/users/{id}/posts/{postId}", 2, []string{"id", "postId"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := extractPathParams(tt.path)
			assert.Len(t, params, tt.wantCount)

			for i, expectedName := range tt.wantParams {
				assert.Equal(t, expectedName, params[i].Name)
				assert.Equal(t, "path", params[i].In)
				assert.True(t, params[i].Required)
			}
		})
	}
}

func TestCombinePaths(t *testing.T) {
	tests := []struct {
		prefix   string
		path     string
		expected string
	}{
		{"", "/users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "users", "/api/users"},
		{"/api/v1", "/users/:id", "/api/v1/users/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_"+tt.path, func(t *testing.T) {
			result := combinePaths(tt.prefix, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		handler  string
		expected string
	}{
		{"GET", "/users", "", "getUsers"},
		{"POST", "/users", "", "postUsers"},
		{"GET", "/users/{id}", "", "getUsersByid"},
		{"DELETE", "/users/{id}", "", "deleteUsersByid"},
		{"GET", "/", "", "get"},
		{"GET", "/api/v1/users", "", "getApiV1Users"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := generateOperationID(tt.method, tt.path, tt.handler)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferTags(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/users", []string{"users"}},
		{"/api/users", []string{"users"}},
		{"/api/v1/users", []string{"users"}},
		{"/api/v1/users/{id}", []string{"users"}},
		{"/", nil},
		{"/{id}", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := inferTags(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper to find a route by method and path
func findRoute(routes []types.Route, method, path string) *types.Route {
	for i := range routes {
		if routes[i].Method == method && routes[i].Path == path {
			return &routes[i]
		}
	}
	return nil
}
