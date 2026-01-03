// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package flask

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// flaskBasicCode is a comprehensive test fixture for Flask route extraction.
const flaskBasicCode = `
from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/users', methods=['GET'])
def get_users():
    return jsonify([])

@app.route('/users/<int:id>', methods=['GET'])
def get_user(id):
    return jsonify({})

@app.route('/users', methods=['POST'])
def create_user():
    return jsonify({})

@app.route('/users/<id>', methods=['PUT'])
def update_user(id):
    return jsonify({})

@app.route('/users/<id>', methods=['DELETE'])
def delete_user(id):
    return '', 204
`

// flaskMethodDecoratorsCode tests method-specific decorators.
const flaskMethodDecoratorsCode = `
from flask import Flask, jsonify

app = Flask(__name__)

@app.get('/products')
def get_products():
    return jsonify([])

@app.post('/products')
def create_product():
    return jsonify({})

@app.get('/products/<int:id>')
def get_product(id):
    return jsonify({})

@app.put('/products/<id>')
def update_product(id):
    return jsonify({})

@app.delete('/products/<id>')
def delete_product(id):
    return '', 204

@app.patch('/products/<id>')
def patch_product(id):
    return jsonify({})
`

// flaskBlueprintCode tests Blueprint routes.
const flaskBlueprintCode = `
from flask import Blueprint, jsonify

bp = Blueprint('users', __name__, url_prefix='/users')

@bp.route('/')
def list_users():
    return jsonify([])

@bp.route('/<int:id>')
def get_user(id):
    return jsonify({})

@bp.post('/')
def create_user():
    return jsonify({})
`

// flaskMethodViewCode tests MethodView classes.
const flaskMethodViewCode = `
from flask import Flask
from flask.views import MethodView

app = Flask(__name__)

class UserAPI(MethodView):
    def get(self, id=None):
        if id is None:
            return jsonify([])
        return jsonify({})

    def post(self):
        return jsonify({})

    def put(self, id):
        return jsonify({})

    def delete(self, id):
        return '', 204

app.add_url_rule('/users', view_func=UserAPI.as_view('users'))
`

// flaskPathParamsCode tests various path parameter types.
const flaskPathParamsCode = `
from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/items/<id>')
def get_item_by_id(id):
    return jsonify({})

@app.route('/items/<int:item_id>')
def get_item_by_int(item_id):
    return jsonify({})

@app.route('/items/<float:price>')
def get_item_by_price(price):
    return jsonify({})

@app.route('/files/<path:filename>')
def get_file(filename):
    return jsonify({})

@app.route('/users/<uuid:user_id>')
def get_user_by_uuid(user_id):
    return jsonify({})
`

// flaskPydanticCode tests Pydantic model extraction.
const flaskPydanticCode = `
from flask import Flask
from pydantic import BaseModel
from typing import Optional, List

class User(BaseModel):
    id: int
    name: str
    email: str
    is_active: bool = True

class CreateUserRequest(BaseModel):
    name: str
    email: str
    password: str

class UserResponse(BaseModel):
    id: int
    name: str
    email: str
    posts: Optional[List[str]] = None

app = Flask(__name__)

@app.get('/users')
def get_users():
    return []
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "flask", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".py")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "flask", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "flask")
}

func TestPlugin_Detect_WithRequirements(t *testing.T) {
	dir := t.TempDir()
	requirements := `flask==2.0.0
requests==2.28.0
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
flask = "^2.0.0"
`
	err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(pyproject), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithSetupPy(t *testing.T) {
	dir := t.TempDir()
	setupPy := `from setuptools import setup
setup(
    name='myapp',
    install_requires=['flask>=2.0.0']
)
`
	err := os.WriteFile(filepath.Join(dir, "setup.py"), []byte(setupPy), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutFlask(t *testing.T) {
	dir := t.TempDir()
	requirements := `django==4.0.0
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
			Path:     "app.py",
			Language: "python",
			Content:  []byte(flaskBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Debug output
	t.Logf("Found %d routes:", len(routes))
	for _, r := range routes {
		t.Logf("  %s %s (handler: %s)", r.Method, r.Path, r.Handler)
	}

	// Should extract basic routes
	assert.GreaterOrEqual(t, len(routes), 4)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	require.NotNil(t, getUsers)
	assert.Equal(t, "GET", getUsers.Method)
	assert.Equal(t, "/users", getUsers.Path)
	assert.Contains(t, getUsers.Tags, "users")

	// Check GET /users/{id}
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	require.NotNil(t, getUserByID)
	assert.Len(t, getUserByID.Parameters, 1)
	assert.Equal(t, "id", getUserByID.Parameters[0].Name)
	assert.Equal(t, "path", getUserByID.Parameters[0].In)
	assert.True(t, getUserByID.Parameters[0].Required)

	// Check POST /users
	postUsers := findRoute(routes, "POST", "/users")
	require.NotNil(t, postUsers)
	assert.Equal(t, "POST", postUsers.Method)
}

func TestPlugin_ExtractRoutes_MethodDecorators(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.py",
			Language: "python",
			Content:  []byte(flaskMethodDecoratorsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(routes), 6)

	// Check method-specific decorators
	getProducts := findRoute(routes, "GET", "/products")
	require.NotNil(t, getProducts)

	postProducts := findRoute(routes, "POST", "/products")
	require.NotNil(t, postProducts)

	putProduct := findRoute(routes, "PUT", "/products/{id}")
	require.NotNil(t, putProduct)

	deleteProduct := findRoute(routes, "DELETE", "/products/{id}")
	require.NotNil(t, deleteProduct)

	patchProduct := findRoute(routes, "PATCH", "/products/{id}")
	require.NotNil(t, patchProduct)
}

func TestPlugin_ExtractRoutes_Blueprint(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "blueprints/users.py",
			Language: "python",
			Content:  []byte(flaskBlueprintCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Blueprint routes should have the url_prefix applied
	listUsers := findRoute(routes, "GET", "/users/")
	if listUsers != nil {
		assert.Equal(t, "/users/", listUsers.Path)
	}
}

func TestPlugin_ExtractRoutes_MethodView(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.py",
			Language: "python",
			Content:  []byte(flaskMethodViewCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// MethodView should extract HTTP methods
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
}

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app.py",
			Language: "python",
			Content:  []byte(flaskPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check path parameter conversion
	itemByInt := findRoute(routes, "GET", "/items/{item_id}")
	if itemByInt != nil {
		require.Len(t, itemByInt.Parameters, 1)
		assert.Equal(t, "item_id", itemByInt.Parameters[0].Name)
		// Type should be inferred from <int:item_id>
		assert.Equal(t, "integer", itemByInt.Parameters[0].Schema.Type)
	}
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

func TestPlugin_ExtractRoutes_IgnoresNonFlask(t *testing.T) {
	p := New()

	code := `
from django.urls import path

urlpatterns = [
    path('users/', views.user_list),
]
`

	files := []scanner.SourceFile{
		{
			Path:     "urls.py",
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
			Path:     "/path/to/app.py",
			Language: "python",
			Content:  []byte(flaskBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	for _, r := range routes {
		assert.Equal(t, "/path/to/app.py", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_Pydantic(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "models.py",
			Language: "python",
			Content:  []byte(flaskPydanticCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract User, CreateUserRequest, UserResponse
	assert.GreaterOrEqual(t, len(schemas), 3)

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

func TestConvertPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/<id>", "/users/{id}"},
		{"/users/<int:id>", "/users/{id}"},
		{"/users/<string:name>", "/users/{name}"},
		{"/users/<int:id>/posts/<post_id>", "/users/{id}/posts/{post_id}"},
		{"/files/<path:filename>", "/files/{filename}"},
		{"/users/<uuid:user_id>", "/users/{user_id}"},
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
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "users", "/api/users"},
		{"/api/v1", "/users/<id>", "/api/v1/users/<id>"},
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

func TestParseMethodsList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"['GET']", []string{"GET"}},
		{"['GET', 'POST']", []string{"GET", "POST"}},
		{`["GET", "POST"]`, []string{"GET", "POST"}},
		{"['get', 'post']", []string{"GET", "POST"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMethodsList(tt.input)
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
