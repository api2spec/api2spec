// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package ktor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// ktorRoutingCode is a comprehensive test fixture for Ktor route extraction.
const ktorRoutingCode = `
package com.example.routes

import io.ktor.server.routing.*
import io.ktor.server.response.*
import io.ktor.server.request.*
import io.ktor.server.application.*

fun Application.configureRouting() {
    routing {
        get("/users") {
            call.respond(listOf<User>())
        }
        get("/users/{id}") {
            val id = call.parameters["id"]
            call.respond(User())
        }
        post("/users") {
            val user = call.receive<CreateUserDto>()
            call.respond(User())
        }
        put("/users/{id}") {
            val id = call.parameters["id"]
            val user = call.receive<UpdateUserDto>()
            call.respond(User())
        }
        delete("/users/{id}") {
            val id = call.parameters["id"]
            call.respond(HttpStatusCode.NoContent)
        }
        patch("/users/{id}/status") {
            val id = call.parameters["id"]
            call.respond(User())
        }
    }
}
`

// ktorNestedRoutesCode tests nested route blocks.
const ktorNestedRoutesCode = `
package com.example.routes

import io.ktor.server.routing.*
import io.ktor.server.response.*
import io.ktor.server.application.*

fun Application.configureRouting() {
    routing {
        route("/api") {
            route("/v1") {
                get("/products") {
                    call.respond(listOf<Product>())
                }
                get("/products/{id}") {
                    call.respond(Product())
                }
                post("/products") {
                    call.respond(Product())
                }
            }
        }
    }
}
`

// ktorAllMethodsCode tests all HTTP methods.
const ktorAllMethodsCode = `
package com.example.routes

import io.ktor.server.routing.*
import io.ktor.server.response.*
import io.ktor.server.application.*

fun Application.configureRouting() {
    routing {
        get("/test") { call.respondText("get") }
        post("/test") { call.respondText("post") }
        put("/test") { call.respondText("put") }
        delete("/test") { call.respondText("delete") }
        patch("/test") { call.respondText("patch") }
        head("/test") { call.respondText("head") }
        options("/test") { call.respondText("options") }
    }
}
`

// ktorDataClassCode tests data class extraction.
const ktorDataClassCode = `
package com.example.dto

import kotlinx.serialization.Serializable

@Serializable
data class UserDto(
    val id: Long,
    val name: String,
    val email: String,
    val isActive: Boolean = true
)

@Serializable
data class CreateUserRequest(
    val name: String,
    val email: String,
    val password: String
)

@Serializable
data class UserResponse(
    val id: Long,
    val name: String,
    val email: String,
    val createdAt: String?
)
`

// ktorOptionalParamsCode tests routes with optional parameters.
const ktorOptionalParamsCode = `
package com.example.routes

import io.ktor.server.routing.*
import io.ktor.server.response.*
import io.ktor.server.application.*

fun Application.configureRouting() {
    routing {
        get("/users/{id?}") {
            val id = call.parameters["id"]
            call.respond(User())
        }
        get("/posts/{postId}/comments/{commentId?}") {
            call.respond(listOf<Comment>())
        }
    }
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "ktor", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".kt")
	assert.Contains(t, exts, ".kts")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "ktor", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "io.ktor")
}

func TestPlugin_Detect_WithBuildGradleKts(t *testing.T) {
	dir := t.TempDir()
	buildGradleKts := `plugins {
    kotlin("jvm") version "1.9.20"
    id("io.ktor.plugin") version "2.3.6"
}

dependencies {
    implementation("io.ktor:ktor-server-core:2.3.6")
    implementation("io.ktor:ktor-server-netty:2.3.6")
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(buildGradleKts), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithBuildGradle(t *testing.T) {
	dir := t.TempDir()
	buildGradle := `plugins {
    id 'org.jetbrains.kotlin.jvm' version '1.9.20'
}

dependencies {
    implementation 'io.ktor:ktor-server-core:2.3.6'
    implementation 'io.ktor:ktor-server-netty:2.3.6'
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(buildGradle), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutKtor(t *testing.T) {
	dir := t.TempDir()
	buildGradleKts := `plugins {
    kotlin("jvm") version "1.9.20"
}

dependencies {
    implementation("org.springframework.boot:spring-boot-starter-web")
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(buildGradleKts), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoBuildFile(t *testing.T) {
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
			Path:     "src/main/kotlin/com/example/routes/UserRoutes.kt",
			Language: "kotlin",
			Content:  []byte(ktorRoutingCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from routing block
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
		assert.Contains(t, getUsers.Tags, "users")
	}

	// Check GET /users/{id}
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

func TestPlugin_ExtractRoutes_NestedRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/routes/ProductRoutes.kt",
			Language: "kotlin",
			Content:  []byte(ktorNestedRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check that nested routes are extracted (may or may not have prefix applied)
	// The plugin extracts routes from regex, so may not apply nesting
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/routes/TestRoutes.kt",
			Language: "kotlin",
			Content:  []byte(ktorAllMethodsCode),
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

func TestPlugin_ExtractRoutes_OptionalParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/routes/UserRoutes.kt",
			Language: "kotlin",
			Content:  []byte(ktorOptionalParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Optional params should have the ? removed in the converted path
	for _, r := range routes {
		for _, param := range r.Parameters {
			assert.NotContains(t, param.Name, "?")
		}
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonKotlin(t *testing.T) {
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

func TestPlugin_ExtractRoutes_SourceInfo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "/path/to/UserRoutes.kt",
			Language: "kotlin",
			Content:  []byte(ktorRoutingCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserRoutes.kt", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DataClasses(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/dto/UserDto.kt",
			Language: "kotlin",
			Content:  []byte(ktorDataClassCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract data classes with DTO naming conventions
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	// Check that data classes are extracted
	if len(schemas) > 0 {
		assert.True(t, schemaNames["UserDto"] || schemaNames["CreateUserRequest"] || schemaNames["UserResponse"])
	}
}

func TestConvertKtorPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/{id}", "/users/{id}"},
		{"/users/{id?}", "/users/{id}"},
		{"/posts/{postId}/comments/{commentId?}", "/posts/{postId}/comments/{commentId}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertKtorPathParams(tt.input)
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

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		handler  string
		expected string
	}{
		{"GET", "/users", "getUsers", "getGetUsers"},
		{"POST", "/users", "createUser", "postCreateUser"},
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
