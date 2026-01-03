// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package sinatra

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// sinatraBasicCode is a comprehensive test fixture for Sinatra route extraction.
const sinatraBasicCode = `
require 'sinatra'

get '/users' do
  json users: []
end

get '/users/:id' do
  json user: {}
end

post '/users' do
  json user: {}
end

put '/users/:id' do
  json user: {}
end

delete '/users/:id' do
  status 204
end

patch '/users/:id/status' do
  json user: {}
end
`

// sinatraClassCode tests class-based Sinatra app.
const sinatraClassCode = `
require 'sinatra/base'

class MyApp < Sinatra::Base
  get '/products' do
    json products: []
  end

  get '/products/:id' do
    json product: {}
  end

  post '/products' do
    json product: {}
  end

  put '/products/:id' do
    json product: {}
  end

  delete '/products/:id' do
    status 204
  end
end
`

// sinatraAllMethodsCode tests all HTTP methods.
const sinatraAllMethodsCode = `
require 'sinatra'

get '/test' do
  'get'
end

post '/test' do
  'post'
end

put '/test' do
  'put'
end

delete '/test' do
  'delete'
end

patch '/test' do
  'patch'
end

options '/test' do
  'options'
end

head '/test' do
  'head'
end
`

// sinatraSplatCode tests splat (wildcard) routes.
const sinatraSplatCode = `
require 'sinatra'

get '/files/*' do
  params['splat']
end

get '/static/*.*' do
  params['splat']
end

get '/download/*/:filename' do
  params['filename']
end
`

// sinatraRegexParamsCode tests routes with regex constraints.
const sinatraRegexParamsCode = `
require 'sinatra'

get '/users/:id' do |id|
  "User #{id}"
end

get '/posts/:year/:month/:day' do |year, month, day|
  "Post from #{year}-#{month}-#{day}"
end
`

// sinatraModularCode tests modular Sinatra with explicit require.
const sinatraModularCode = `
require "sinatra/base"
require "sinatra/json"

class API < Sinatra::Application
  get '/api/items' do
    json items: []
  end

  get '/api/items/:id' do
    json item: {}
  end

  post '/api/items' do
    json item: {}
  end
end
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "sinatra", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".rb")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "sinatra", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "sinatra")
}

func TestPlugin_Detect_WithGemfile(t *testing.T) {
	dir := t.TempDir()
	gemfile := `source 'https://rubygems.org'

gem 'sinatra', '~> 3.1'
gem 'puma', '>= 5.0'
gem 'rack', '~> 2.2'
`
	err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutSinatra(t *testing.T) {
	dir := t.TempDir()
	gemfile := `source 'https://rubygems.org'

gem 'rails', '~> 7.1.2'
gem 'puma', '>= 5.0'
`
	err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoGemfile(t *testing.T) {
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
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from get/post/put/delete/patch blocks
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
		assert.Contains(t, getUsers.Tags, "users")
	}

	// Check GET /users/{id} (converted from :id)
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
	}

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	if putUser != nil {
		assert.Equal(t, "PUT", putUser.Method)
	}

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	if deleteUser != nil {
		assert.Equal(t, "DELETE", deleteUser.Method)
	}
}

func TestPlugin_ExtractRoutes_ClassBasedApp(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraClassCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from class that inherits from Sinatra::Base
	assert.GreaterOrEqual(t, len(routes), 4)

	// Check GET /products
	getProducts := findRoute(routes, "GET", "/products")
	if getProducts != nil {
		assert.Equal(t, "GET", getProducts.Method)
	}
}

func TestPlugin_ExtractRoutes_ModularApp(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "api.rb",
			Language: "ruby",
			Content:  []byte(sinatraModularCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from modular Sinatra app
	assert.GreaterOrEqual(t, len(routes), 2)
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraAllMethodsCode),
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

func TestPlugin_ExtractRoutes_SplatRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraSplatCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Splat routes should be converted to {splat} params
	for _, r := range routes {
		if r.Path == "/files/{splat}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "splat", r.Parameters[0].Name)
		}
	}
}

func TestPlugin_ExtractRoutes_MultipleParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraRegexParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check route with multiple params
	dateRoute := findRoute(routes, "GET", "/posts/{year}/{month}/{day}")
	if dateRoute != nil {
		assert.Len(t, dateRoute.Parameters, 3)
		paramNames := make(map[string]bool)
		for _, param := range dateRoute.Parameters {
			paramNames[param.Name] = true
		}
		assert.True(t, paramNames["year"])
		assert.True(t, paramNames["month"])
		assert.True(t, paramNames["day"])
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonRuby(t *testing.T) {
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

func TestPlugin_ExtractRoutes_IgnoresNonSinatra(t *testing.T) {
	p := New()

	// File without Sinatra require or class inheritance
	code := `
class MyController
  def index
    @users = User.all
  end
end
`
	files := []scanner.SourceFile{
		{
			Path:     "controller.rb",
			Language: "ruby",
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
			Path:     "/path/to/app.rb",
			Language: "ruby",
			Content:  []byte(sinatraBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/app.rb", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.rb",
			Language: "ruby",
			Content:  []byte(sinatraBasicCode),
		},
	}

	// Sinatra doesn't have standard schema extraction
	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestConvertSinatraPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:post_id", "/users/{id}/posts/{post_id}"},
		{"/api/v1/:resource/:id", "/api/v1/{resource}/{id}"},
		{"/:a/:b/:c", "/{a}/{b}/{c}"},
		{"/files/*", "/files/{splat}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertSinatraPathParams(tt.input)
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
		{"/files/{splat}", 1, []string{"splat"}},
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
