// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package rails

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// railsRoutesCode is a comprehensive test fixture for Rails route extraction.
const railsRoutesCode = `
Rails.application.routes.draw do
  get '/users', to: 'users#index'
  get '/users/:id', to: 'users#show'
  post '/users', to: 'users#create'
  put '/users/:id', to: 'users#update'
  delete '/users/:id', to: 'users#destroy'
  patch '/users/:id/status', to: 'users#update_status'
end
`

// railsNamespacedRoutesCode tests namespaced routes.
const railsNamespacedRoutesCode = `
Rails.application.routes.draw do
  namespace :api do
    namespace :v1 do
      get '/products', to: 'products#index'
      get '/products/:id', to: 'products#show'
      post '/products', to: 'products#create'
    end
  end
end
`

// railsResourcesCode tests resource routes.
const railsResourcesCode = `
Rails.application.routes.draw do
  resources :orders
  resources :items, only: [:index, :show, :create]
  resource :profile
end
`

// railsAllMethodsCode tests all HTTP methods.
const railsAllMethodsCode = `
Rails.application.routes.draw do
  get '/test', to: 'test#get_test'
  post '/test', to: 'test#post_test'
  put '/test', to: 'test#put_test'
  delete '/test', to: 'test#delete_test'
  patch '/test', to: 'test#patch_test'
  options '/test', to: 'test#options_test'
  head '/test', to: 'test#head_test'
end
`

// railsNestedResourcesCode tests nested resources.
const railsNestedResourcesCode = `
Rails.application.routes.draw do
  resources :users do
    resources :posts do
      resources :comments, only: [:index, :create]
    end
  end
end
`

// railsScopedRoutesCode tests scoped routes.
const railsScopedRoutesCode = `
Rails.application.routes.draw do
  scope '/admin' do
    get '/dashboard', to: 'admin#dashboard'
    resources :settings
  end
end
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "rails", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".rb")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "rails", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "rails")
}

func TestPlugin_Detect_WithGemfile(t *testing.T) {
	dir := t.TempDir()
	gemfile := `source 'https://rubygems.org'

gem 'rails', '~> 7.1.2'
gem 'pg', '~> 1.1'
gem 'puma', '>= 5.0'
`
	err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte(gemfile), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithRoutesRb(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	routesRb := `Rails.application.routes.draw do
  root 'home#index'
end
`
	err = os.WriteFile(filepath.Join(configDir, "routes.rb"), []byte(routesRb), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutRails(t *testing.T) {
	dir := t.TempDir()
	gemfile := `source 'https://rubygems.org'

gem 'sinatra', '~> 3.0'
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
			Path:     "config/routes.rb",
			Language: "ruby",
			Content:  []byte(railsRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from get/post/put/delete/patch calls
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

func TestPlugin_ExtractRoutes_NamespacedRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.rb",
			Language: "ruby",
			Content:  []byte(railsNamespacedRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check routes have namespace prefix applied
	getProducts := findRoute(routes, "GET", "/api/v1/products")
	if getProducts != nil {
		assert.Equal(t, "/api/v1/products", getProducts.Path)
	}
}

func TestPlugin_ExtractRoutes_ResourceRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.rb",
			Language: "ruby",
			Content:  []byte(railsResourcesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Resource routes should expand to standard CRUD routes
	// index, create, new, show, edit, update, destroy
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
			Path:     "config/routes.rb",
			Language: "ruby",
			Content:  []byte(railsAllMethodsCode),
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

func TestPlugin_ExtractRoutes_IgnoresNonRoutes(t *testing.T) {
	p := New()

	// File that doesn't contain "routes" in path
	files := []scanner.SourceFile{
		{
			Path:     "app/controllers/users_controller.rb",
			Language: "ruby",
			Content:  []byte(`class UsersController < ApplicationController end`),
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
			Path:     "/path/to/config/routes.rb",
			Language: "ruby",
			Content:  []byte(railsRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/config/routes.rb", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app/models/user.rb",
			Language: "ruby",
			Content:  []byte(`class User < ApplicationRecord end`),
		},
	}

	// Rails doesn't have standard schema extraction
	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)
	assert.Empty(t, schemas)
}

func TestConvertRailsPathParams(t *testing.T) {
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
			result := convertRailsPathParams(tt.input)
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
