// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package hono

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// honoTestCode is a comprehensive test fixture for Hono route extraction.
const honoTestCode = `
import { Hono } from 'hono'
import { zValidator } from '@hono/zod-validator'
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

const app = new Hono()

app.get('/users', (c) => c.json([]))
app.get('/users/:id', (c) => c.json({}))
app.post('/users', zValidator('json', CreateUserSchema), (c) => c.json({}, 201))
app.put('/users/:id', zValidator('json', CreateUserSchema), (c) => c.json({}))
app.patch('/users/:id', (c) => c.json({}))
app.delete('/users/:id', (c) => c.text('', 204))

export default app
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "hono", p.Name())
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

	assert.Equal(t, "hono", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "hono")
}

func TestPlugin_Detect_WithHono(t *testing.T) {
	// Create a temp directory with a package.json containing hono
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
	assert.True(t, detected)
}

func TestPlugin_Detect_WithHonoDevDep(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "devDependencies": {
    "hono": "^4.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutHono(t *testing.T) {
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

func TestPlugin_ExtractRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(honoTestCode),
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

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)
	// Should have request body from zValidator
	assert.NotNil(t, postUsers.RequestBody)
	assert.True(t, postUsers.RequestBody.Required)
	assert.NotNil(t, postUsers.RequestBody.Content["application/json"])

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	require.NotNil(t, putUser)
	assert.NotNil(t, putUser.RequestBody)

	// Check PATCH /users/{id}
	patchUser := findRoute(routes, "PATCH", "/users/{id}")
	require.NotNil(t, patchUser)
	assert.Equal(t, "PATCH", patchUser.Method)

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	require.NotNil(t, deleteUser)
	assert.Equal(t, "DELETE", deleteUser.Method)
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

func TestPlugin_ExtractRoutes_IgnoresNonHono(t *testing.T) {
	p := New()

	// File that doesn't import Hono
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

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(honoTestCode),
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

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	code := `
import { Hono } from 'hono'

const app = new Hono()

app.get('/test', (c) => c.text('get'))
app.post('/test', (c) => c.text('post'))
app.put('/test', (c) => c.text('put'))
app.delete('/test', (c) => c.text('delete'))
app.patch('/test', (c) => c.text('patch'))
app.head('/test', (c) => c.text('head'))
app.options('/test', (c) => c.text('options'))
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(code),
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

func TestPlugin_ExtractRoutes_SourceInfo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "/path/to/app.ts",
			Language: "typescript",
			Content:  []byte(honoTestCode),
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

// Helper to find a route by method and path
func findRoute(routes []types.Route, method, path string) *types.Route {
	for i := range routes {
		if routes[i].Method == method && routes[i].Path == path {
			return &routes[i]
		}
	}
	return nil
}
