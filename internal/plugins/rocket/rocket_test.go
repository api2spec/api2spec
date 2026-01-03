// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package rocket

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// rocketBasicCode is a comprehensive test fixture for Rocket route extraction.
const rocketBasicCode = `
use rocket::{get, post, put, delete, patch, routes};
use rocket::serde::json::Json;

#[get("/users")]
fn get_users() -> Json<Vec<User>> {
    Json(vec![])
}

#[get("/users/<id>")]
fn get_user(id: u32) -> Json<User> {
    Json(User::default())
}

#[post("/users", data = "<user>")]
fn create_user(user: Json<CreateUser>) -> Json<User> {
    Json(User::default())
}

#[put("/users/<id>", data = "<user>")]
fn update_user(id: u32, user: Json<UpdateUser>) -> Json<User> {
    Json(User::default())
}

#[delete("/users/<id>")]
fn delete_user(id: u32) -> &'static str {
    "deleted"
}

#[patch("/users/<id>/status")]
fn update_user_status(id: u32) -> Json<User> {
    Json(User::default())
}

fn main() {
    rocket::build()
        .mount("/api", routes![get_users, get_user, create_user, update_user, delete_user])
        .launch();
}
`

// rocketAllMethodsCode tests all HTTP methods.
const rocketAllMethodsCode = `
use rocket::{get, post, put, delete, patch, head, options};

#[get("/test")]
fn test_get() -> &'static str { "get" }

#[post("/test")]
fn test_post() -> &'static str { "post" }

#[put("/test")]
fn test_put() -> &'static str { "put" }

#[delete("/test")]
fn test_delete() -> &'static str { "delete" }

#[patch("/test")]
fn test_patch() -> &'static str { "patch" }

#[head("/test")]
fn test_head() -> &'static str { "head" }

#[options("/test")]
fn test_options() -> &'static str { "options" }
`

// rocketStructsCode tests struct extraction with serde.
const rocketStructsCode = `
use rocket::serde::{Deserialize, Serialize};

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
`

// rocketMultipleParamsCode tests routes with multiple path parameters.
const rocketMultipleParamsCode = `
use rocket::get;

#[get("/users/<user_id>/posts/<post_id>")]
fn get_user_post(user_id: u32, post_id: u32) -> String {
    format!("user {} post {}", user_id, post_id)
}

#[get("/organizations/<org_id>/teams/<team_id>/members/<member_id>")]
fn get_team_member(org_id: u32, team_id: u32, member_id: u32) -> String {
    "member".to_string()
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "rocket", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".rs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "rocket", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "rocket")
}

func TestPlugin_Detect_WithCargoToml(t *testing.T) {
	dir := t.TempDir()
	cargoToml := `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
rocket = "0.5"
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithDevDependency(t *testing.T) {
	dir := t.TempDir()
	cargoToml := `[package]
name = "myapp"
version = "0.1.0"

[dev-dependencies]
rocket = "0.5"
`
	err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(cargoToml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutRocket(t *testing.T) {
	dir := t.TempDir()
	cargoToml := `[package]
name = "myapp"
version = "0.1.0"

[dependencies]
axum = "0.7"
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
			Content:  []byte(rocketBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from #[method("/path")] attributes
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
		assert.Equal(t, "get_users", getUsers.Handler)
		assert.Contains(t, getUsers.Tags, "users")
	}

	// Check GET /users/{id} (converted from <id>)
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
		assert.Equal(t, "create_user", postUsers.Handler)
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

	// Check PATCH /users/{id}/status
	patchUser := findRoute(routes, "PATCH", "/users/{id}/status")
	if patchUser != nil {
		assert.Equal(t, "PATCH", patchUser.Method)
		assert.Len(t, patchUser.Parameters, 1)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.rs",
			Language: "rust",
			Content:  []byte(rocketAllMethodsCode),
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
			Path:     "src/routes.rs",
			Language: "rust",
			Content:  []byte(rocketMultipleParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check route with 2 parameters
	userPost := findRoute(routes, "GET", "/users/{user_id}/posts/{post_id}")
	if userPost != nil {
		assert.Len(t, userPost.Parameters, 2)
		paramNames := make(map[string]bool)
		for _, p := range userPost.Parameters {
			paramNames[p.Name] = true
		}
		assert.True(t, paramNames["user_id"])
		assert.True(t, paramNames["post_id"])
	}

	// Check route with 3 parameters
	teamMember := findRoute(routes, "GET", "/organizations/{org_id}/teams/{team_id}/members/{member_id}")
	if teamMember != nil {
		assert.Len(t, teamMember.Parameters, 3)
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

func TestPlugin_ExtractRoutes_IgnoresNonRocket(t *testing.T) {
	p := New()

	code := `
use axum::{routing::get, Router};

async fn hello() -> &'static str {
    "Hello!"
}

fn main() {
    let app = Router::new().route("/hello", get(hello));
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
			Content:  []byte(rocketBasicCode),
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
			Content:  []byte(rocketStructsCode),
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
	if userSchema != nil {
		assert.Equal(t, "object", userSchema.Type)
		assert.Contains(t, userSchema.Properties, "id")
		assert.Contains(t, userSchema.Properties, "name")
		assert.Contains(t, userSchema.Properties, "email")
	}
}

func TestConvertRocketPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/<id>", "/users/{id}"},
		{"/users/<id>/posts/<post_id>", "/users/{id}/posts/{post_id}"},
		{"/api/v1/<resource>/<id>", "/api/v1/{resource}/{id}"},
		{"/<a>/<b>/<c>", "/{a}/{b}/{c}"},
		{"/files/<path..>", "/files/{path}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertRocketPathParams(tt.input)
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
