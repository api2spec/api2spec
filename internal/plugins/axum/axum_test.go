// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package axum

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// axumBasicCode is a comprehensive test fixture for Axum route extraction.
const axumBasicCode = `
use axum::{
    routing::{get, post, put, delete},
    Router,
    Json,
};

async fn get_users() -> Json<Vec<User>> {
    Json(vec![])
}

async fn get_user(Path(id): Path<u32>) -> Json<User> {
    Json(User::default())
}

async fn create_user(Json(user): Json<CreateUser>) -> Json<User> {
    Json(User::default())
}

async fn update_user(Path(id): Path<u32>, Json(user): Json<UpdateUser>) -> Json<User> {
    Json(User::default())
}

async fn delete_user(Path(id): Path<u32>) -> impl IntoResponse {
    StatusCode::NO_CONTENT
}

pub fn router() -> Router {
    Router::new()
        .route("/users", get(get_users).post(create_user))
        .route("/users/:id", get(get_user).put(update_user).delete(delete_user))
}
`

// axumNestedCode tests nested routers with prefixes.
const axumNestedCode = `
use axum::{routing::get, Router};

async fn get_items() -> impl IntoResponse {
    Json(vec![])
}

async fn get_item(Path(id): Path<u32>) -> impl IntoResponse {
    Json(Item::default())
}

fn api_routes() -> Router {
    Router::new()
        .route("/items", get(get_items))
        .route("/items/:id", get(get_item))
}

pub fn app() -> Router {
    Router::new()
        .nest("/api/v1", api_routes())
}
`

// axumAllMethodsCode tests all HTTP methods.
const axumAllMethodsCode = `
use axum::{
    routing::{get, post, put, delete, patch, head, options, trace},
    Router,
};

async fn handler() -> impl IntoResponse {
    "ok"
}

pub fn router() -> Router {
    Router::new()
        .route("/test", get(handler))
        .route("/test", post(handler))
        .route("/test", put(handler))
        .route("/test", delete(handler))
        .route("/test", patch(handler))
        .route("/test", head(handler))
        .route("/test", options(handler))
}
`

// axumStructsCode tests struct extraction with serde.
const axumStructsCode = `
use axum::{routing::get, Router, Json};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct User {
    pub id: u32,
    pub name: String,
    pub email: String,
    #[serde(rename = "isActive")]
    pub is_active: bool,
}

#[derive(Debug, Deserialize)]
pub struct CreateUser {
    pub name: String,
    pub email: String,
    pub password: String,
}

#[derive(Debug, Serialize)]
pub struct UserResponse {
    pub id: u32,
    pub name: String,
    pub email: String,
    pub posts: Option<Vec<String>>,
}

async fn get_users() -> Json<Vec<User>> {
    Json(vec![])
}

pub fn router() -> Router {
    Router::new()
        .route("/users", get(get_users))
}
`

// axumChainedMethodsCode tests chained HTTP methods on a single route.
const axumChainedMethodsCode = `
use axum::{routing::{get, post, put, delete}, Router};

async fn list_products() -> impl IntoResponse { Json(vec![]) }
async fn create_product() -> impl IntoResponse { Json(Product::default()) }
async fn get_product() -> impl IntoResponse { Json(Product::default()) }
async fn update_product() -> impl IntoResponse { Json(Product::default()) }
async fn delete_product() -> impl IntoResponse { StatusCode::NO_CONTENT }

pub fn router() -> Router {
    Router::new()
        .route("/products", get(list_products).post(create_product))
        .route("/products/:id", get(get_product).put(update_product).delete(delete_product))
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "axum", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".rs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "axum", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "axum")
}

func TestPlugin_Detect_WithCargoToml(t *testing.T) {
	dir := t.TempDir()
	cargoToml := `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
axum = "0.7"
tokio = { version = "1", features = ["full"] }
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutAxum(t *testing.T) {
	dir := t.TempDir()
	cargoToml := `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
actix-web = "4"
tokio = { version = "1", features = ["full"] }
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoCargoToml(t *testing.T) {
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
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(axumBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from Router::new().route() calls
	assert.GreaterOrEqual(t, len(routes), 2)

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
}

func TestPlugin_ExtractRoutes_ChainedMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(axumChainedMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check for multiple methods on same path
	methods := make(map[string]bool)
	for _, r := range routes {
		if r.Path == "/products" || r.Path == "/products/{id}" {
			methods[r.Method] = true
		}
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(axumAllMethodsCode),
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

func TestPlugin_ExtractRoutes_IgnoresNonRust(t *testing.T) {
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

func TestPlugin_ExtractRoutes_IgnoresNonAxum(t *testing.T) {
	p := New()

	code := `
use actix_web::{web, App, HttpServer};

async fn hello() -> impl Responder {
    "Hello!"
}

fn main() {
    App::new()
        .route("/hello", web::get().to(hello))
}
`

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
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
			Path:     "/path/to/src/main.rs",
			Language: "rust",
			Content:  []byte(axumBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/src/main.rs", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_SerdeStructs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/models.rs",
			Language: "rust",
			Content:  []byte(axumStructsCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract User and UserResponse (both have Serialize)
	assert.GreaterOrEqual(t, len(schemas), 2)

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
		{"/users/:id", "/users/{id}"},
		{"/users/:id/posts/:post_id", "/users/{id}/posts/{post_id}"},
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
		{"Vec<String>", "String"},
		{"Option<i32>", "i32"},
		{"HashMap<String, i32>", "String, i32"},
		{"String", ""},
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
