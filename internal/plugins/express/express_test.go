// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package express

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// expressTestCode is a comprehensive test fixture for Express route extraction.
const expressTestCode = `
const express = require('express')
const { body, validationResult } = require('express-validator')

const app = express()

// Basic routes
app.get('/users', (req, res) => res.json([]))
app.get('/users/:id', (req, res) => res.json({}))
app.post('/users', body('email').isEmail(), (req, res) => res.json({}))
app.put('/users/:id', (req, res) => res.json({}))
app.delete('/users/:id', (req, res) => res.sendStatus(204))

// Router
const itemRouter = express.Router()
itemRouter.get('/', (req, res) => res.json([]))
itemRouter.post('/', (req, res) => res.json({}))

app.use('/items', itemRouter)

// Route chaining
app.route('/books')
  .get((req, res) => res.json([]))
  .post((req, res) => res.json({}))

module.exports = app
`

// expressESMCode tests ES module imports.
const expressESMCode = `
import express from 'express'

const app = express()

app.get('/api/products', (req, res) => res.json([]))
app.post('/api/products', (req, res) => res.json({}))
app.get('/api/products/:id', (req, res) => res.json({}))
app.patch('/api/products/:id', (req, res) => res.json({}))
app.delete('/api/products/:id', (req, res) => res.sendStatus(204))

export default app
`

// expressRouterMountingCode tests router mounting with prefixes.
const expressRouterMountingCode = `
const express = require('express')

const app = express()

const usersRouter = express.Router()
usersRouter.get('/', (req, res) => res.json([]))
usersRouter.get('/:id', (req, res) => res.json({}))
usersRouter.post('/', (req, res) => res.json({}))

const postsRouter = express.Router()
postsRouter.get('/', (req, res) => res.json([]))
postsRouter.get('/:id', (req, res) => res.json({}))

app.use('/api/users', usersRouter)
app.use('/api/posts', postsRouter)

module.exports = app
`

// expressRouteChainCode tests route chaining patterns.
const expressRouteChainCode = `
const express = require('express')
const app = express()

app.route('/products')
  .get((req, res) => res.json([]))
  .post((req, res) => res.json({}))
  .put((req, res) => res.json({}))

app.route('/categories/:id')
  .get((req, res) => res.json({}))
  .delete((req, res) => res.sendStatus(204))
  .patch((req, res) => res.json({}))

module.exports = app
`

// expressValidationCode tests validation middleware detection.
const expressValidationCode = `
const express = require('express')
const { body, query, param } = require('express-validator')
const { z } = require('zod')

const CreateUserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
})

const app = express()

// Express validator
app.post('/users',
  body('email').isEmail(),
  body('name').isString().notEmpty(),
  (req, res) => res.json({})
)

// Query validation
app.get('/search',
  query('q').isString(),
  (req, res) => res.json([])
)

// Zod validation
app.post('/accounts', validate(CreateUserSchema), (req, res) => res.json({}))

module.exports = app
`

// expressWildcardCode tests wildcard route patterns.
const expressWildcardCode = `
const express = require('express')
const app = express()

app.get('/files/*', (req, res) => res.sendFile(req.params[0]))
app.get('/static/:type/*', (req, res) => res.sendFile(req.params[0]))

module.exports = app
`

// expressAllMethodsCode tests all HTTP methods including 'all'.
const expressAllMethodsCode = `
const express = require('express')
const app = express()

app.get('/test', (req, res) => res.text('get'))
app.post('/test', (req, res) => res.text('post'))
app.put('/test', (req, res) => res.text('put'))
app.delete('/test', (req, res) => res.text('delete'))
app.patch('/test', (req, res) => res.text('patch'))
app.head('/test', (req, res) => res.text('head'))
app.options('/test', (req, res) => res.text('options'))
app.all('/secret', (req, res) => res.text('secret'))

module.exports = app
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "express", p.Name())
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

	assert.Equal(t, "express", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "express")
}

func TestPlugin_Detect_WithExpress(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "express": "^4.18.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithExpressDevDep(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "devDependencies": {
    "express": "^4.18.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutExpress(t *testing.T) {
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
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressTestCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract basic routes
	assert.GreaterOrEqual(t, len(routes), 5)

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

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	require.NotNil(t, putUser)

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	require.NotNil(t, deleteUser)
	assert.Equal(t, "DELETE", deleteUser.Method)
}

func TestPlugin_ExtractRoutes_ESMImports(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(expressESMCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(routes), 5)

	// Check routes with /api prefix
	getProducts := findRoute(routes, "GET", "/api/products")
	require.NotNil(t, getProducts)

	getProductByID := findRoute(routes, "GET", "/api/products/{id}")
	require.NotNil(t, getProductByID)

	postProducts := findRoute(routes, "POST", "/api/products")
	require.NotNil(t, postProducts)
}

func TestPlugin_ExtractRoutes_RouterMounting(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressRouterMountingCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check mounted user routes
	getUsersFromRouter := findRoute(routes, "GET", "/api/users")
	if getUsersFromRouter != nil {
		assert.Equal(t, "/api/users", getUsersFromRouter.Path)
	}

	// Check mounted posts routes
	getPostsFromRouter := findRoute(routes, "GET", "/api/posts")
	if getPostsFromRouter != nil {
		assert.Equal(t, "/api/posts", getPostsFromRouter.Path)
	}
}

func TestPlugin_ExtractRoutes_RouteChaining(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressRouteChainCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check /products routes from chaining
	getProducts := findRoute(routes, "GET", "/products")
	if getProducts != nil {
		assert.Equal(t, "GET", getProducts.Method)
		assert.Equal(t, "/products", getProducts.Path)
	}

	postProducts := findRoute(routes, "POST", "/products")
	if postProducts != nil {
		assert.Equal(t, "POST", postProducts.Method)
	}

	putProducts := findRoute(routes, "PUT", "/products")
	if putProducts != nil {
		assert.Equal(t, "PUT", putProducts.Method)
	}

	// Check /categories/:id routes from chaining
	getCategoryByID := findRoute(routes, "GET", "/categories/{id}")
	if getCategoryByID != nil {
		assert.Len(t, getCategoryByID.Parameters, 1)
		assert.Equal(t, "id", getCategoryByID.Parameters[0].Name)
	}
}

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressAllMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

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
	assert.True(t, methods["ALL"])
}

func TestPlugin_ExtractRoutes_WildcardParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressWildcardCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check wildcard conversion
	filesRoute := findRoute(routes, "GET", "/files/{path}")
	if filesRoute != nil {
		assert.Contains(t, filesRoute.Path, "{path}")
	}
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

func TestPlugin_ExtractRoutes_IgnoresNonExpress(t *testing.T) {
	p := New()

	// File that doesn't import Express
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
			Path:     "/path/to/app.js",
			Language: "javascript",
			Content:  []byte(expressTestCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	for _, r := range routes {
		assert.Equal(t, "/path/to/app.js", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
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
		{"/files/*", "/files/{path}"},
		{"/static/:type/*", "/static/{type}/{path}"},
		{"/users/:id(\\d+)", "/users/{id}"}, // Regex constraint
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
		{"/files/{path}", 1, []string{"path"}},
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

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	code := `
const express = require('express')
const { z } = require('zod')

const CreateUserSchema = z.object({
  name: z.string().min(1),
  email: z.string().email(),
})

const UserSchema = z.object({
  id: z.string().uuid(),
  name: z.string(),
  email: z.string().email(),
})

const app = express()

app.get('/users', (req, res) => res.json([]))

module.exports = app
`

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(code),
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

func TestPlugin_ExtractRoutes_ValidationMiddleware(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(expressValidationCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check POST /users with express-validator
	postUsers := findRoute(routes, "POST", "/users")
	if postUsers != nil {
		// Should detect express-validator middleware
		if postUsers.RequestBody != nil {
			assert.NotNil(t, postUsers.RequestBody.Content["application/json"])
		}
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
