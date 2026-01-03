// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package actix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// actixBasicCode is a comprehensive test fixture for Actix-web route extraction.
const actixBasicCode = `
use actix_web::{get, post, put, delete, web, App, HttpServer, Responder};

#[get("/users")]
async fn get_users() -> impl Responder {
    web::Json(vec![])
}

#[get("/users/{id}")]
async fn get_user(path: web::Path<u32>) -> impl Responder {
    web::Json(User::default())
}

#[post("/users")]
async fn create_user(body: web::Json<CreateUser>) -> impl Responder {
    web::Json(User::default())
}

#[put("/users/{id}")]
async fn update_user(path: web::Path<u32>, body: web::Json<UpdateUser>) -> impl Responder {
    web::Json(User::default())
}

#[delete("/users/{id}")]
async fn delete_user(path: web::Path<u32>) -> impl Responder {
    HttpResponse::NoContent()
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    HttpServer::new(|| {
        App::new()
            .service(get_users)
            .service(get_user)
            .service(create_user)
            .service(update_user)
            .service(delete_user)
    })
    .bind("127.0.0.1:8080")?
    .run()
    .await
}
`

// actixAllMethodsCode tests all HTTP methods.
const actixAllMethodsCode = `
use actix_web::{get, post, put, delete, patch, head, options, web, Responder};

#[get("/test")]
async fn test_get() -> impl Responder {
    "get"
}

#[post("/test")]
async fn test_post() -> impl Responder {
    "post"
}

#[put("/test")]
async fn test_put() -> impl Responder {
    "put"
}

#[delete("/test")]
async fn test_delete() -> impl Responder {
    "delete"
}

#[patch("/test")]
async fn test_patch() -> impl Responder {
    "patch"
}

#[head("/test")]
async fn test_head() -> impl Responder {
    "head"
}

#[options("/test")]
async fn test_options() -> impl Responder {
    "options"
}
`

// actixStructsCode tests struct extraction with serde.
const actixStructsCode = `
use actix_web::{get, web, Responder};
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

#[get("/users")]
async fn get_users() -> impl Responder {
    web::Json(vec![])
}
`

// actixPathParamsCode tests path parameter extraction.
const actixPathParamsCode = `
use actix_web::{get, web, Responder};

#[get("/items/{id}")]
async fn get_item(path: web::Path<u32>) -> impl Responder {
    "item"
}

#[get("/users/{user_id}/posts/{post_id}")]
async fn get_user_post(path: web::Path<(u32, u32)>) -> impl Responder {
    "post"
}

#[get("/categories/{category}/items/{item_id}")]
async fn get_category_item(path: web::Path<(String, u32)>) -> impl Responder {
    "item"
}
`

// actixRequestBodyCode tests request body extraction.
const actixRequestBodyCode = `
use actix_web::{post, put, web, Responder};
use serde::Deserialize;

#[derive(Deserialize)]
struct CreateItem {
    name: String,
    price: f64,
}

#[derive(Deserialize)]
struct UpdateItem {
    name: Option<String>,
    price: Option<f64>,
}

#[post("/items")]
async fn create_item(body: web::Json<CreateItem>) -> impl Responder {
    "created"
}

#[put("/items/{id}")]
async fn update_item(path: web::Path<u32>, body: web::Json<UpdateItem>) -> impl Responder {
    "updated"
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "actix", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".rs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "actix", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "actix-web")
}

func TestPlugin_Detect_WithCargoToml(t *testing.T) {
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
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutActix(t *testing.T) {
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
			Content:  []byte(actixBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from attribute macros
	assert.GreaterOrEqual(t, len(routes), 5)

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

	// Check PUT /users/{id}
	putUser := findRoute(routes, "PUT", "/users/{id}")
	require.NotNil(t, putUser)

	// Check DELETE /users/{id}
	deleteUser := findRoute(routes, "DELETE", "/users/{id}")
	require.NotNil(t, deleteUser)
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(actixAllMethodsCode),
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
}

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(actixPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check single path param
	getItem := findRoute(routes, "GET", "/items/{id}")
	if getItem != nil {
		assert.Len(t, getItem.Parameters, 1)
		assert.Equal(t, "id", getItem.Parameters[0].Name)
	}

	// Check multiple path params
	getUserPost := findRoute(routes, "GET", "/users/{user_id}/posts/{post_id}")
	if getUserPost != nil {
		assert.Len(t, getUserPost.Parameters, 2)
	}
}

func TestPlugin_ExtractRoutes_RequestBody(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(actixRequestBodyCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check POST with request body
	createItem := findRoute(routes, "POST", "/items")
	if createItem != nil && createItem.RequestBody != nil {
		assert.True(t, createItem.RequestBody.Required)
		assert.Contains(t, createItem.RequestBody.Content, "application/json")
	}
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

func TestPlugin_ExtractRoutes_IgnoresNonActix(t *testing.T) {
	p := New()

	code := `
use axum::{routing::get, Router};

async fn hello() -> &'static str {
    "Hello!"
}

fn main() {
    Router::new().route("/hello", get(hello))
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
			Content:  []byte(actixBasicCode),
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
			Content:  []byte(actixStructsCode),
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

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"users", "/users"},
		{"/users/", "/users"},
		{"/users//items", "/users/items"},
		{"/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
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
		{"web::Json<CreateUser>", "CreateUser"},
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
