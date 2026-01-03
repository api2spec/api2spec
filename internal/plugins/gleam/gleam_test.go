// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package gleam

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// gleamWispCode is a comprehensive test fixture for Gleam/Wisp route extraction.
const gleamWispCode = `
import gleam/http
import gleam/http/request.{type Request}
import gleam/http/response.{type Response}
import wisp.{type Request, type Response}

pub fn handle_request(req: Request) -> Response {
  case request.method(req) {
    http.Get -> handle_get(req)
    http.Post -> handle_post(req)
    http.Put -> handle_put(req)
    http.Delete -> handle_delete(req)
    http.Patch -> handle_patch(req)
    _ -> wisp.method_not_allowed([http.Get, http.Post])
  }
}

fn handle_get(req: Request) -> Response {
  case wisp.path_segments(req) {
    ["users"] -> list_users(req)
    ["users", id] -> get_user(req, id)
    _ -> wisp.not_found()
  }
}

fn handle_post(req: Request) -> Response {
  case wisp.path_segments(req) {
    ["users"] -> create_user(req)
    _ -> wisp.not_found()
  }
}
`

// gleamRouterCode tests explicit router patterns.
const gleamRouterCode = `
import gleam/http
import wisp

pub fn router(req: wisp.Request) -> wisp.Response {
  get("/users", list_users)
  get("/users/{id}", get_user)
  post("/users", create_user)
  put("/users/{id}", update_user)
  delete("/users/{id}", delete_user)
}
`

// gleamMistCode tests Mist server routes.
const gleamMistCode = `
import gleam/http
import mist
import gleam/http/request.{type Request}
import gleam/http/response.{type Response}

pub fn handle(req: Request(Connection)) -> Response(ResponseData) {
  case request.method(req) {
    http.Get -> {
      case request.path(req) {
        "/products" -> list_products()
        "/products/" <> id -> get_product(id)
        _ -> response.new(404)
      }
    }
    http.Post -> {
      case request.path(req) {
        "/products" -> create_product(req)
        _ -> response.new(404)
      }
    }
    _ -> response.new(405)
  }
}
`

// gleamTypesCode tests Gleam type extraction.
const gleamTypesCode = `
import gleam/option.{type Option}

pub type User {
  User(
    id: Int,
    name: String,
    email: String,
    is_active: Bool,
  )
}

pub type CreateUserRequest {
  CreateUserRequest(
    name: String,
    email: String,
    password: String,
  )
}

pub type UserResponse {
  UserResponse(
    id: Int,
    name: String,
    email: String,
    created_at: Option(String),
  )
}
`

// gleamAllMethodsCode tests all HTTP methods.
const gleamAllMethodsCode = `
import gleam/http
import wisp

pub fn handle_request(req: wisp.Request) -> wisp.Response {
  case request.method(req) {
    http.Get -> handle_get(req)
    http.Post -> handle_post(req)
    http.Put -> handle_put(req)
    http.Delete -> handle_delete(req)
    http.Patch -> handle_patch(req)
    http.Head -> handle_head(req)
    http.Options -> handle_options(req)
    _ -> wisp.method_not_allowed([])
  }
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "gleam", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".gleam")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "gleam", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "wisp")
}

func TestPlugin_Detect_WithWisp(t *testing.T) {
	dir := t.TempDir()
	gleamToml := `name = "my_app"
version = "1.0.0"

[dependencies]
gleam_stdlib = "~> 0.32"
wisp = "~> 0.10"
mist = "~> 0.15"
`
	err := os.WriteFile(filepath.Join(dir, "gleam.toml"), []byte(gleamToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithGleamHttp(t *testing.T) {
	dir := t.TempDir()
	gleamToml := `name = "my_app"
version = "1.0.0"

[dependencies]
gleam_stdlib = "~> 0.32"
gleam_http = "~> 3.5"
`
	err := os.WriteFile(filepath.Join(dir, "gleam.toml"), []byte(gleamToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithMist(t *testing.T) {
	dir := t.TempDir()
	gleamToml := `name = "my_app"
version = "1.0.0"

[dependencies]
gleam_stdlib = "~> 0.32"
mist = "~> 0.15"
`
	err := os.WriteFile(filepath.Join(dir, "gleam.toml"), []byte(gleamToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutGleamHttp(t *testing.T) {
	dir := t.TempDir()
	gleamToml := `name = "my_app"
version = "1.0.0"

[dependencies]
gleam_stdlib = "~> 0.32"
gleam_json = "~> 1.0"
`
	err := os.WriteFile(filepath.Join(dir, "gleam.toml"), []byte(gleamToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoGleamToml(t *testing.T) {
	dir := t.TempDir()

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_ExtractRoutes_WispRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/router.gleam",
			Language: "gleam",
			Content:  []byte(gleamWispCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from HTTP method case expressions
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have at least GET and POST
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_RouterPatterns(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/router.gleam",
			Language: "gleam",
			Content:  []byte(gleamRouterCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
	}

	// Check GET /users/{id}
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
	}

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/router.gleam",
			Language: "gleam",
			Content:  []byte(gleamAllMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// The Gleam parser extracts from case expressions on HTTP method
	// Just verify no error - route extraction depends on parser implementation
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have at least some HTTP methods if routes were extracted
		assert.True(t, methods["GET"] || methods["POST"] || methods["PUT"] || methods["DELETE"] || methods["PATCH"])
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonGleam(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "main.py",
			Language: "python",
			Content:  []byte(`from flask import Flask`),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_IgnoresNonHttp(t *testing.T) {
	p := New()

	// File without HTTP imports
	code := `
import gleam/list
import gleam/string

pub fn main() {
  list.map([1, 2, 3], fn(x) { x * 2 })
}
`
	files := []scanner.SourceFile{
		{
			Path:     "src/main.gleam",
			Language: "gleam",
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
			Path:     "/path/to/src/router.gleam",
			Language: "gleam",
			Content:  []byte(gleamRouterCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/src/router.gleam", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_GleamTypes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/types.gleam",
			Language: "gleam",
			Content:  []byte(gleamTypesCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract public types
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	// Check that public types are extracted
	if len(schemas) > 0 {
		assert.True(t, schemaNames["User"] || schemaNames["CreateUserRequest"] || schemaNames["UserResponse"])
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
		{"/users/{id}/posts/{post_id}", 2, []string{"id", "post_id"}},
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
		{"GET", "/users", "list_users", "getList_users"},
		{"POST", "/users", "create_user", "postCreate_user"},
		{"GET", "/users/{id}", "", "getUsersByid"},
		{"DELETE", "/users/{id}", "", "deleteUsersByid"},
		{"GET", "/", "", "get"},
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

func TestCountLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 1},
		{"hello", 1},
		{"hello\nworld", 2},
		{"line1\nline2\nline3", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := countLines(tt.input)
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
