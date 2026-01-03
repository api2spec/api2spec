// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package fastify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// fastifyTestCode is a comprehensive test fixture for Fastify route extraction.
const fastifyTestCode = `
import Fastify from 'fastify'
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

const fastify = Fastify()

fastify.get('/users', async (request, reply) => {
  return []
})

fastify.get('/users/:id', async (request, reply) => {
  return {}
})

fastify.post('/users', {
  schema: {
    body: {
      type: 'object',
      properties: {
        name: { type: 'string' },
        email: { type: 'string', format: 'email' }
      },
      required: ['name', 'email']
    }
  }
}, async (request, reply) => {
  return {}
})

fastify.put('/users/:id', async (request, reply) => {
  return {}
})

fastify.patch('/users/:id', async (request, reply) => {
  return {}
})

fastify.delete('/users/:id', async (request, reply) => {
  reply.code(204).send()
})

export default fastify
`

// fastifyRouteMethodCode tests the fastify.route() pattern.
const fastifyRouteMethodCode = `
import Fastify from 'fastify'

const fastify = Fastify()

fastify.route({
  method: 'GET',
  url: '/products',
  schema: {
    response: {
      200: {
        type: 'array',
        items: {
          type: 'object',
          properties: {
            id: { type: 'string' },
            name: { type: 'string' }
          }
        }
      }
    }
  },
  handler: async (request, reply) => {
    return []
  }
})

fastify.route({
  method: 'POST',
  url: '/products',
  schema: {
    body: {
      type: 'object',
      properties: {
        name: { type: 'string' },
        price: { type: 'number' }
      },
      required: ['name', 'price']
    },
    response: {
      201: {
        type: 'object',
        properties: {
          id: { type: 'string' },
          name: { type: 'string' },
          price: { type: 'number' }
        }
      }
    }
  },
  handler: async (request, reply) => {
    reply.code(201)
    return {}
  }
})

fastify.route({
  method: 'GET',
  url: '/products/:id',
  handler: async (request, reply) => {
    return {}
  }
})

export default fastify
`

// fastifySchemaValidationCode tests Fastify's built-in JSON Schema validation.
const fastifySchemaValidationCode = `
import Fastify from 'fastify'

const fastify = Fastify()

fastify.post('/orders', {
  schema: {
    body: {
      type: 'object',
      properties: {
        productId: { type: 'string', format: 'uuid' },
        quantity: { type: 'integer' },
        notes: { type: 'string' }
      },
      required: ['productId', 'quantity']
    },
    response: {
      201: {
        type: 'object',
        properties: {
          orderId: { type: 'string', format: 'uuid' },
          status: { type: 'string', enum: ['pending', 'confirmed'] }
        }
      },
      400: {
        type: 'object',
        properties: {
          error: { type: 'string' }
        }
      }
    }
  }
}, async (request, reply) => {
  reply.code(201)
  return {}
})

export default fastify
`

// fastifyAllMethodsCode tests all HTTP methods.
const fastifyAllMethodsCode = `
import Fastify from 'fastify'

const fastify = Fastify()

fastify.get('/test', async () => 'get')
fastify.post('/test', async () => 'post')
fastify.put('/test', async () => 'put')
fastify.delete('/test', async () => 'delete')
fastify.patch('/test', async () => 'patch')
fastify.head('/test', async () => 'head')
fastify.options('/test', async () => 'options')

export default fastify
`

// fastifyPluginPrefixCode tests plugin registration with prefix.
const fastifyPluginPrefixCode = `
import Fastify from 'fastify'

const fastify = Fastify()

async function userRoutes(fastify) {
  fastify.get('/', async () => [])
  fastify.get('/:id', async () => ({}))
  fastify.post('/', async () => ({}))
}

fastify.register(userRoutes, { prefix: '/api/users' })

export default fastify
`

// fastifyRequireCode tests CommonJS require pattern.
const fastifyRequireCode = `
const Fastify = require('fastify')

const fastify = Fastify()

fastify.get('/items', async (request, reply) => {
  return []
})

fastify.get('/items/:id', async (request, reply) => {
  return {}
})

fastify.post('/items', async (request, reply) => {
  return {}
})

module.exports = fastify
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "fastify", p.Name())
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

	assert.Equal(t, "fastify", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "fastify")
}

func TestPlugin_Detect_WithFastify(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "fastify": "^4.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithFastifyDevDep(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "devDependencies": {
    "fastify": "^4.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutFastify(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "express": "^4.0.0"
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
			Content:  []byte(fastifyTestCode),
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

	// Check POST /users with schema
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)
	// Should have request body from inline schema
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

func TestPlugin_ExtractRoutes_RouteMethod(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(fastifyRouteMethodCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 3 routes
	assert.Len(t, routes, 3)

	// Check GET /products
	getProducts := findRoute(routes, "GET", "/products")
	require.NotNil(t, getProducts)
	assert.Equal(t, "GET", getProducts.Method)

	// Check POST /products with schema
	postProducts := findRoute(routes, "POST", "/products")
	require.NotNil(t, postProducts)
	assert.NotNil(t, postProducts.RequestBody)

	// Check GET /products/{id}
	getProductByID := findRoute(routes, "GET", "/products/{id}")
	require.NotNil(t, getProductByID)
	assert.Len(t, getProductByID.Parameters, 1)
}

func TestPlugin_ExtractRoutes_SchemaValidation(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(fastifySchemaValidationCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.Len(t, routes, 1)

	// Check POST /orders with full schema
	postOrders := findRoute(routes, "POST", "/orders")
	require.NotNil(t, postOrders)

	// Should have request body schema
	assert.NotNil(t, postOrders.RequestBody)
	assert.True(t, postOrders.RequestBody.Required)

	bodyContent := postOrders.RequestBody.Content["application/json"]
	require.NotNil(t, bodyContent.Schema)
	assert.Equal(t, "object", bodyContent.Schema.Type)

	// Should have response schemas
	assert.NotNil(t, postOrders.Responses)
	if len(postOrders.Responses) > 0 {
		resp201, ok := postOrders.Responses["201"]
		if ok {
			assert.NotEmpty(t, resp201.Content)
		}
	}
}

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(fastifyAllMethodsCode),
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

func TestPlugin_ExtractRoutes_RequireSyntax(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(fastifyRequireCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.Len(t, routes, 3)

	// Check routes
	getItems := findRoute(routes, "GET", "/items")
	require.NotNil(t, getItems)

	getItemByID := findRoute(routes, "GET", "/items/{id}")
	require.NotNil(t, getItemByID)

	postItems := findRoute(routes, "POST", "/items")
	require.NotNil(t, postItems)
}

func TestPlugin_ExtractRoutes_IgnoresNonJS(t *testing.T) {
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

func TestPlugin_ExtractRoutes_IgnoresNonFastify(t *testing.T) {
	p := New()

	// File that doesn't import Fastify
	code := `
import express from 'express';

const app = express();
app.get('/users', (req, res) => res.json([]));
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
			Content:  []byte(fastifyTestCode),
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
			Content:  []byte(fastifyTestCode),
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
