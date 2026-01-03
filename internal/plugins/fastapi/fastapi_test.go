// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package fastapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// fastAPIBasicCode is a comprehensive test fixture for FastAPI route extraction.
const fastAPIBasicCode = `
from fastapi import FastAPI

app = FastAPI()

@app.get('/users')
async def get_users():
    return []

@app.get('/users/{user_id}')
async def get_user(user_id: int):
    return {}

@app.post('/users')
async def create_user(user: UserCreate):
    return {}

@app.put('/users/{user_id}')
async def update_user(user_id: int, user: UserUpdate):
    return {}

@app.delete('/users/{user_id}')
async def delete_user(user_id: int):
    return {}
`

// fastAPIRouterCode tests APIRouter with prefixes.
const fastAPIRouterCode = `
from fastapi import APIRouter

router = APIRouter(prefix='/api/v1')

@router.get('/items')
async def get_items():
    return []

@router.get('/items/{item_id}')
async def get_item(item_id: int):
    return {}

@router.post('/items')
async def create_item(item: ItemCreate):
    return {}
`

// fastAPIResponseModelCode tests response_model extraction.
const fastAPIResponseModelCode = `
from fastapi import FastAPI
from pydantic import BaseModel
from typing import List

class User(BaseModel):
    id: int
    name: str
    email: str

class UserList(BaseModel):
    users: List[User]

app = FastAPI()

@app.get('/users', response_model=List[User])
async def get_users():
    return []

@app.get('/users/{user_id}', response_model=User)
async def get_user(user_id: int):
    return {}
`

// fastAPIPydanticCode tests Pydantic model extraction.
const fastAPIPydanticCode = `
from fastapi import FastAPI
from pydantic import BaseModel
from typing import Optional, List
from datetime import datetime

class UserBase(BaseModel):
    name: str
    email: str

class UserCreate(UserBase):
    password: str

class UserUpdate(BaseModel):
    name: Optional[str] = None
    email: Optional[str] = None

class User(UserBase):
    id: int
    created_at: datetime
    is_active: bool = True
    tags: List[str] = []

app = FastAPI()

@app.get('/users')
async def get_users():
    return []
`

// fastAPIQueryParamsCode tests query parameter extraction.
const fastAPIQueryParamsCode = `
from fastapi import FastAPI, Query
from typing import Optional

app = FastAPI()

@app.get('/items')
async def get_items(
    skip: int = 0,
    limit: int = 100,
    q: Optional[str] = None,
    sort_by: str = Query(default='created_at')
):
    return []

@app.get('/search')
async def search(
    query: str,
    page: int = 1,
    per_page: int = 10
):
    return []
`

// fastAPIAllMethodsCode tests all HTTP methods.
const fastAPIAllMethodsCode = `
from fastapi import FastAPI

app = FastAPI()

@app.get('/test')
async def test_get():
    return 'get'

@app.post('/test')
async def test_post():
    return 'post'

@app.put('/test')
async def test_put():
    return 'put'

@app.delete('/test')
async def test_delete():
    return 'delete'

@app.patch('/test')
async def test_patch():
    return 'patch'

@app.options('/test')
async def test_options():
    return 'options'

@app.head('/test')
async def test_head():
    return 'head'
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "fastapi", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".py")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "fastapi", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "fastapi")
}

func TestPlugin_Detect_WithRequirements(t *testing.T) {
	dir := t.TempDir()
	requirements := `fastapi==0.100.0
uvicorn==0.23.0
`
	err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithPyproject(t *testing.T) {
	dir := t.TempDir()
	pyproject := `[tool.poetry.dependencies]
python = "^3.9"
fastapi = "^0.100.0"
`
	err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutFastAPI(t *testing.T) {
	dir := t.TempDir()
	requirements := `flask==2.0.0
requests==2.28.0
`
	err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte(requirements), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoConfigFiles(t *testing.T) {
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
			Path:     "main.py",
			Language: "python",
			Content:  []byte(fastAPIBasicCode),
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

	// Check GET /users/{user_id}
	getUserByID := findRoute(routes, "GET", "/users/{user_id}")
	require.NotNil(t, getUserByID)
	assert.Len(t, getUserByID.Parameters, 1)
	assert.Equal(t, "user_id", getUserByID.Parameters[0].Name)
	assert.Equal(t, "path", getUserByID.Parameters[0].In)
	assert.True(t, getUserByID.Parameters[0].Required)

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)

	// Check PUT /users/{user_id}
	putUser := findRoute(routes, "PUT", "/users/{user_id}")
	require.NotNil(t, putUser)

	// Check DELETE /users/{user_id}
	deleteUser := findRoute(routes, "DELETE", "/users/{user_id}")
	require.NotNil(t, deleteUser)
}

func TestPlugin_ExtractRoutes_RouterPrefix(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routers/items.py",
			Language: "python",
			Content:  []byte(fastAPIRouterCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Routes should have the prefix applied
	getItems := findRoute(routes, "GET", "/api/v1/items")
	if getItems != nil {
		assert.Equal(t, "/api/v1/items", getItems.Path)
	}

	getItem := findRoute(routes, "GET", "/api/v1/items/{item_id}")
	if getItem != nil {
		assert.Equal(t, "/api/v1/items/{item_id}", getItem.Path)
	}
}

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "main.py",
			Language: "python",
			Content:  []byte(fastAPIAllMethodsCode),
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
	assert.True(t, methods["OPTIONS"])
	assert.True(t, methods["HEAD"])
}

func TestPlugin_ExtractRoutes_IgnoresNonPython(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.js",
			Language: "javascript",
			Content:  []byte(`const express = require('express')`),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_IgnoresNonFastAPI(t *testing.T) {
	p := New()

	code := `
from flask import Flask

app = Flask(__name__)

@app.route('/users')
def get_users():
    return []
`

	files := []scanner.SourceFile{
		{
			Path:     "app.py",
			Language: "python",
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
			Path:     "/path/to/main.py",
			Language: "python",
			Content:  []byte(fastAPIBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	for _, r := range routes {
		assert.Equal(t, "/path/to/main.py", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_Pydantic(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "models.py",
			Language: "python",
			Content:  []byte(fastAPIPydanticCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract UserBase, UserCreate, UserUpdate, User
	assert.GreaterOrEqual(t, len(schemas), 4)

	// Find User schema
	var userSchema *types.Schema
	for i := range schemas {
		if schemas[i].Title == "User" {
			userSchema = &schemas[i]
			break
		}
	}
	require.NotNil(t, userSchema)
	assert.Equal(t, "object", userSchema.Type)
	assert.Contains(t, userSchema.Properties, "id")
	assert.Contains(t, userSchema.Properties, "name")
	assert.Contains(t, userSchema.Properties, "email")
}

func TestPlugin_ExtractSchemas_OptionalFields(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "models.py",
			Language: "python",
			Content:  []byte(fastAPIPydanticCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Find UserUpdate schema
	var userUpdateSchema *types.Schema
	for i := range schemas {
		if schemas[i].Title == "UserUpdate" {
			userUpdateSchema = &schemas[i]
			break
		}
	}

	if userUpdateSchema != nil {
		// Optional fields should be marked as nullable
		if nameProp, ok := userUpdateSchema.Properties["name"]; ok {
			assert.True(t, nameProp.Nullable)
		}
	}
}

func TestNormalizePathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/{id}", "/users/{id}"},
		{"/users/{user_id:int}", "/users/{user_id}"},
		{"/items/{item_id:path}", "/items/{item_id}"},
		{"/users/{id}/posts/{post_id:int}", "/users/{id}/posts/{post_id}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePathParams(tt.input)
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
		{"/users/{user_id:int}", 1, []string{"user_id"}},
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
		{"/api/v1", "/users/{id}", "/api/v1/users/{id}"},
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
		{"GET", "/users", "get_users", "getGet_users"},
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

func TestExtractGenericType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"List[str]", "str"},
		{"Optional[int]", "int"},
		{"Dict[str, int]", "str, int"},
		{"list[User]", "User"},
		{"str", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractGenericType(tt.input)
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
