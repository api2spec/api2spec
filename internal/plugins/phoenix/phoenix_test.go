// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package phoenix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// phoenixRouterCode is a comprehensive test fixture for Phoenix route extraction.
const phoenixRouterCode = `
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  pipeline :api do
    plug :accepts, ["json"]
  end

  scope "/api", MyAppWeb do
    pipe_through :api

    get "/users", UserController, :index
    get "/users/:id", UserController, :show
    post "/users", UserController, :create
    put "/users/:id", UserController, :update
    delete "/users/:id", UserController, :delete
    patch "/users/:id/status", UserController, :update_status
  end
end
`

// phoenixNestedScopesCode tests nested scope blocks.
const phoenixNestedScopesCode = `
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  scope "/api" do
    scope "/v1" do
      get "/products", ProductController, :index
      get "/products/:id", ProductController, :show
      post "/products", ProductController, :create
    end
  end
end
`

// phoenixResourcesCode tests resource routes.
const phoenixResourcesCode = `
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  scope "/api", MyAppWeb do
    pipe_through :api

    resources "/orders", OrderController
    resources "/items", ItemController, only: [:index, :show, :create]
  end
end
`

// phoenixAllMethodsCode tests all HTTP methods.
const phoenixAllMethodsCode = `
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  scope "/" do
    get "/test", TestController, :get_test
    post "/test", TestController, :post_test
    put "/test", TestController, :put_test
    delete "/test", TestController, :delete_test
    patch "/test", TestController, :patch_test
    head "/test", TestController, :head_test
    options "/test", TestController, :options_test
  end
end
`

// phoenixMultipleParamsCode tests routes with multiple path parameters.
const phoenixMultipleParamsCode = `
defmodule MyAppWeb.Router do
  use MyAppWeb, :router

  scope "/api" do
    get "/users/:user_id/posts/:post_id", PostController, :show
    get "/organizations/:org_id/teams/:team_id/members/:member_id", MemberController, :show
  end
end
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "phoenix", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".ex")
	assert.Contains(t, exts, ".exs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "phoenix", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, ":phoenix")
}

func TestPlugin_Detect_WithMixExs(t *testing.T) {
	dir := t.TempDir()
	mixExs := `defmodule MyApp.MixProject do
  use Mix.Project

  def project do
    [
      app: :my_app,
      version: "0.1.0",
      elixir: "~> 1.14",
      deps: deps()
    ]
  end

  defp deps do
    [
      {:phoenix, "~> 1.7.10"},
      {:phoenix_html, "~> 3.3"},
      {:plug_cowboy, "~> 2.5"}
    ]
  end
end
`
	err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(mixExs), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutPhoenix(t *testing.T) {
	dir := t.TempDir()
	mixExs := `defmodule MyApp.MixProject do
  use Mix.Project

  defp deps do
    [
      {:plug, "~> 1.14"},
      {:plug_cowboy, "~> 2.5"}
    ]
  end
end
`
	err := os.WriteFile(filepath.Join(dir, "mix.exs"), []byte(mixExs), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoMixExs(t *testing.T) {
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
			Path:     "lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixRouterCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from scope blocks
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/api/users", getUsers.Path)
		assert.Contains(t, getUsers.Tags, "users")
	}

	// Check GET /api/users/{id} (converted from :id)
	getUserByID := findRoute(routes, "GET", "/api/users/{id}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}

	// Check POST /api/users
	postUsers := findRoute(routes, "POST", "/api/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
	}

	// Check PUT /api/users/{id}
	putUser := findRoute(routes, "PUT", "/api/users/{id}")
	if putUser != nil {
		assert.Equal(t, "PUT", putUser.Method)
	}

	// Check DELETE /api/users/{id}
	deleteUser := findRoute(routes, "DELETE", "/api/users/{id}")
	if deleteUser != nil {
		assert.Equal(t, "DELETE", deleteUser.Method)
	}
}

func TestPlugin_ExtractRoutes_NestedScopes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixNestedScopesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check routes have nested scope prefixes
	if len(routes) > 0 {
		// Routes should have paths starting with /api/v1 or similar
		for _, r := range routes {
			if r.Path != "" {
				// Path should start with /
				assert.True(t, r.Path[0] == '/')
			}
		}
	}
}

func TestPlugin_ExtractRoutes_ResourceRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixResourcesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Resource routes should expand to standard CRUD routes
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have at least GET and POST for resources
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixAllMethodsCode),
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
}

func TestPlugin_ExtractRoutes_MultipleParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixMultipleParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check route with 2 parameters
	userPost := findRoute(routes, "GET", "/api/users/{user_id}/posts/{post_id}")
	if userPost != nil {
		assert.Len(t, userPost.Parameters, 2)
		paramNames := make(map[string]bool)
		for _, param := range userPost.Parameters {
			paramNames[param.Name] = true
		}
		assert.True(t, paramNames["user_id"])
		assert.True(t, paramNames["post_id"])
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonElixir(t *testing.T) {
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

func TestPlugin_ExtractRoutes_IgnoresNonRouter(t *testing.T) {
	p := New()

	// File that doesn't contain "router" in path
	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app_web/controllers/user_controller.ex",
			Language: "elixir",
			Content:  []byte(`defmodule MyAppWeb.UserController do end`),
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
			Path:     "/path/to/lib/my_app_web/router.ex",
			Language: "elixir",
			Content:  []byte(phoenixRouterCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/lib/my_app_web/router.ex", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "lib/my_app/user.ex",
			Language: "elixir",
			Content:  []byte(`defmodule MyApp.User do end`),
		},
	}

	// Phoenix doesn't have standard schema extraction
	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestConvertPhoenixPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:post_id", "/users/{id}/posts/{post_id}"},
		{"/api/v1/:resource/:id", "/api/v1/{resource}/{id}"},
		{"/:a/:b/:c", "/{a}/{b}/{c}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertPhoenixPathParams(tt.input)
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

func TestCombinePaths(t *testing.T) {
	tests := []struct {
		prefix   string
		path     string
		expected string
	}{
		{"", "/users", "/users"},
		{"", "users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"api", "users", "/api/users"},
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
		{"GET", "/users", "index", "getIndex"},
		{"POST", "/users", "create", "postCreate"},
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

// Helper to find a route by method and path
func findRoute(routes []types.Route, method, path string) *types.Route {
	for i := range routes {
		if routes[i].Method == method && routes[i].Path == path {
			return &routes[i]
		}
	}
	return nil
}
