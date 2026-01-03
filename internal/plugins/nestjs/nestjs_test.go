// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package nestjs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// nestjsBasicController is a test fixture for basic NestJS controller.
const nestjsBasicController = `
import { Controller, Get, Post, Put, Delete, Patch, Param, Body } from '@nestjs/common';

@Controller('users')
export class UsersController {
  @Get()
  findAll() {
    return [];
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return {};
  }

  @Post()
  create(@Body() createUserDto: CreateUserDto) {
    return {};
  }

  @Put(':id')
  update(@Param('id') id: string, @Body() updateUserDto: UpdateUserDto) {
    return {};
  }

  @Delete(':id')
  remove(@Param('id') id: string) {
    return;
  }

  @Patch(':id')
  patch(@Param('id') id: string, @Body() patchUserDto: PatchUserDto) {
    return {};
  }
}
`

// nestjsControllerWithPath tests controller with explicit path in decorator.
const nestjsControllerWithPath = `
import { Controller, Get, Post } from '@nestjs/common';

@Controller('api/products')
export class ProductsController {
  @Get()
  findAll() {
    return [];
  }

  @Get('active')
  findActive() {
    return [];
  }

  @Post('bulk')
  createBulk(@Body() products: Product[]) {
    return [];
  }
}
`

// nestjsControllerWithVersion tests versioned controller.
const nestjsControllerWithVersion = `
import { Controller, Get } from '@nestjs/common';

@Controller({ path: 'items', version: '1' })
export class ItemsController {
  @Get()
  findAll() {
    return [];
  }

  @Get(':id')
  findOne(@Param('id') id: string) {
    return {};
  }
}
`

// nestjsControllerWithQuery tests @Query decorator.
const nestjsControllerWithQuery = `
import { Controller, Get, Query } from '@nestjs/common';

@Controller('search')
export class SearchController {
  @Get()
  search(@Query('q') query: string, @Query('limit') limit?: number) {
    return [];
  }
}
`

// nestjsControllerWithHttpCode tests @HttpCode decorator.
const nestjsControllerWithHttpCode = `
import { Controller, Post, HttpCode } from '@nestjs/common';

@Controller('orders')
export class OrdersController {
  @Post()
  @HttpCode(201)
  create(@Body() order: CreateOrderDto) {
    return {};
  }
}
`

// nestjsAllMethods tests all HTTP method decorators.
const nestjsAllMethods = `
import { Controller, Get, Post, Put, Delete, Patch, Head, Options, All } from '@nestjs/common';

@Controller('test')
export class TestController {
  @Get('get')
  testGet() { return 'get'; }

  @Post('post')
  testPost() { return 'post'; }

  @Put('put')
  testPut() { return 'put'; }

  @Delete('delete')
  testDelete() { return 'delete'; }

  @Patch('patch')
  testPatch() { return 'patch'; }

  @Head('head')
  testHead() { return 'head'; }

  @Options('options')
  testOptions() { return 'options'; }

  @All('all')
  testAll() { return 'all'; }
}
`

// nestjsNoController tests file without controller decorator.
const nestjsNoController = `
import { Injectable } from '@nestjs/common';

@Injectable()
export class UsersService {
  findAll() {
    return [];
  }
}
`

// nestjsNoNestImport tests file without NestJS imports.
const nestjsNoNestImport = `
import express from 'express';

const app = express();
app.get('/users', (req, res) => res.json([]));
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "nestjs", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()

	assert.Contains(t, exts, ".ts")
	assert.Contains(t, exts, ".tsx")
	assert.Contains(t, exts, ".js")
	assert.Contains(t, exts, ".jsx")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "nestjs", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "@nestjs/core")
	assert.Contains(t, info.SupportedFrameworks, "@nestjs/common")
}

func TestPlugin_Detect_WithNestJS(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithNestJSDevDep(t *testing.T) {
	dir := t.TempDir()
	packageJSON := `{
  "name": "test-app",
  "devDependencies": {
    "@nestjs/common": "^10.0.0"
  }
}`
	err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutNestJS(t *testing.T) {
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
	assert.False(t, detected)
}

func TestPlugin_Detect_NoPackageJSON(t *testing.T) {
	dir := t.TempDir()

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_ExtractRoutes_BasicController(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "users.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsBasicController),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if assert.NotNil(t, getUsers, "Should find GET /users") {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
		assert.Contains(t, getUsers.Tags, "users")
		assert.Equal(t, "UsersController.findAll", getUsers.Handler)
	}

	// Check GET /users/{id}
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	if assert.NotNil(t, getUserByID, "Should find GET /users/{id}") {
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	if assert.NotNil(t, postUsers, "Should find POST /users") {
		assert.Equal(t, "POST", postUsers.Method)
		assert.NotNil(t, postUsers.RequestBody)
	}

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	assert.NotNil(t, putUser, "Should find PUT /users/{id}")

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	assert.NotNil(t, deleteUser, "Should find DELETE /users/{id}")

	// Check PATCH /users/{id}
	patchUser := findRoute(routes, "PATCH", "/users/{id}")
	assert.NotNil(t, patchUser, "Should find PATCH /users/{id}")
}

func TestPlugin_ExtractRoutes_ControllerWithPath(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "products.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsControllerWithPath),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check GET /api/products
	getProducts := findRoute(routes, "GET", "/api/products")
	if getProducts != nil {
		assert.Equal(t, "/api/products", getProducts.Path)
	}

	// Check GET /api/products/active
	getActive := findRoute(routes, "GET", "/api/products/active")
	if getActive != nil {
		assert.Equal(t, "/api/products/active", getActive.Path)
	}

	// Check POST /api/products/bulk
	postBulk := findRoute(routes, "POST", "/api/products/bulk")
	if postBulk != nil {
		assert.Equal(t, "/api/products/bulk", postBulk.Path)
	}
}

func TestPlugin_ExtractRoutes_VersionedController(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "items.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsControllerWithVersion),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check for versioned paths
	getItems := findRoute(routes, "GET", "/v1/items")
	if getItems != nil {
		assert.Equal(t, "/v1/items", getItems.Path)
	}

	getItemByID := findRoute(routes, "GET", "/v1/items/{id}")
	if getItemByID != nil {
		assert.Equal(t, "/v1/items/{id}", getItemByID.Path)
	}
}

func TestPlugin_ExtractRoutes_QueryParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "search.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsControllerWithQuery),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	getSearch := findRoute(routes, "GET", "/search")
	if assert.NotNil(t, getSearch, "Should find GET /search") {
		// Check for query parameters
		queryParams := filterParamsByIn(getSearch.Parameters, "query")
		assert.GreaterOrEqual(t, len(queryParams), 1)

		// Check for 'q' parameter
		qParam := findParamByName(queryParams, "q")
		if qParam != nil {
			assert.Equal(t, "query", qParam.In)
			assert.True(t, qParam.Required)
		}
	}
}

func TestPlugin_ExtractRoutes_HttpCode(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "orders.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsControllerWithHttpCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	postOrder := findRoute(routes, "POST", "/orders")
	if assert.NotNil(t, postOrder, "Should find POST /orders") {
		// Check for custom HTTP code response
		if postOrder.Responses != nil {
			_, has201 := postOrder.Responses["201"]
			assert.True(t, has201, "Should have 201 response")
		}
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "test.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsAllMethods),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}

	assert.True(t, methods["GET"], "Should have GET")
	assert.True(t, methods["POST"], "Should have POST")
	assert.True(t, methods["PUT"], "Should have PUT")
	assert.True(t, methods["DELETE"], "Should have DELETE")
	assert.True(t, methods["PATCH"], "Should have PATCH")
	assert.True(t, methods["HEAD"], "Should have HEAD")
	assert.True(t, methods["OPTIONS"], "Should have OPTIONS")
	assert.True(t, methods["ALL"], "Should have ALL")
}

func TestPlugin_ExtractRoutes_NoController(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "users.service.ts",
			Language: "typescript",
			Content:  []byte(nestjsNoController),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_NoNestImport(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.ts",
			Language: "typescript",
			Content:  []byte(nestjsNoNestImport),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
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

func TestPlugin_ExtractRoutes_SourceInfo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "/path/to/users.controller.ts",
			Language: "typescript",
			Content:  []byte(nestjsBasicController),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	for _, r := range routes {
		assert.Equal(t, "/path/to/users.controller.ts", r.SourceFile)
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
		{"/users/:userId/posts/:postId", "/users/{userId}/posts/{postId}"},
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

func TestBuildPath(t *testing.T) {
	tests := []struct {
		basePath   string
		version    string
		methodPath string
		expected   string
	}{
		{"", "", "", "/"},
		{"users", "", "", "/users"},
		{"users", "", "active", "/users/active"},
		{"users", "1", "", "/v1/users"},
		{"users", "1", "active", "/v1/users/active"},
		{"/users/", "", "/active/", "/users/active"},
		{"api/users", "", ":id", "/api/users/:id"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := buildPath(tt.basePath, tt.version, tt.methodPath)
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
		{"GET", "/users", "findAll", "getfindAll"},
		{"POST", "/users", "create", "postcreate"},
		{"GET", "/users/{id}", "findOne", "getfindOne"},
		{"GET", "/users", "", "getUsers"},
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
		controllerName string
		path           string
		expected       []string
	}{
		{"UsersController", "/users", []string{"users"}},
		{"ProductsController", "/api/products", []string{"products"}},
		{"", "/orders", []string{"orders"}},
		{"", "/api/v1/items", []string{"items"}},
		{"TestController", "/", []string{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.controllerName+"_"+tt.path, func(t *testing.T) {
			result := inferTags(tt.controllerName, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapTypeScriptToOpenAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"number", "number"},
		{"boolean", "boolean"},
		{"Date", "string"},
		{"any", "string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapTypeScriptToOpenAPI(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

func findRoute(routes []types.Route, method, path string) *types.Route {
	for i := range routes {
		if routes[i].Method == method && routes[i].Path == path {
			return &routes[i]
		}
	}
	return nil
}

func filterParamsByIn(params []types.Parameter, in string) []types.Parameter {
	var result []types.Parameter
	for _, p := range params {
		if p.In == in {
			result = append(result, p)
		}
	}
	return result
}

func findParamByName(params []types.Parameter, name string) *types.Parameter {
	for i := range params {
		if params[i].Name == name {
			return &params[i]
		}
	}
	return nil
}
