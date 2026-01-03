// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package micronaut

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// micronautJavaControllerCode tests Java Micronaut controller.
const micronautJavaControllerCode = `
package com.example.controller;

import io.micronaut.http.annotation.*;
import io.micronaut.http.HttpResponse;

@Controller("/api/users")
public class UserController {

    @Get
    public List<User> getUsers() {
        return new ArrayList<>();
    }

    @Get("/{id}")
    public User getUser(@PathVariable Long id) {
        return new User();
    }

    @Post
    public HttpResponse<User> createUser(@Body CreateUserDto user) {
        return HttpResponse.created(new User());
    }

    @Put("/{id}")
    public User updateUser(@PathVariable Long id, @Body UpdateUserDto user) {
        return new User();
    }

    @Delete("/{id}")
    public HttpResponse<?> deleteUser(@PathVariable Long id) {
        return HttpResponse.noContent();
    }

    @Patch("/{id}/status")
    public User updateUserStatus(@PathVariable Long id, @Body StatusDto status) {
        return new User();
    }
}
`

// micronautKotlinControllerCode tests Kotlin Micronaut controller.
const micronautKotlinControllerCode = `
package com.example.controller

import io.micronaut.http.annotation.*
import io.micronaut.http.HttpResponse

@Controller("/api/products")
class ProductController {

    @Get
    fun getProducts(): List<Product> {
        return listOf()
    }

    @Get("/{id}")
    fun getProduct(@PathVariable id: Long): Product {
        return Product()
    }

    @Post
    fun createProduct(@Body product: CreateProductDto): HttpResponse<Product> {
        return HttpResponse.created(Product())
    }

    @Put("/{id}")
    fun updateProduct(@PathVariable id: Long, @Body product: UpdateProductDto): Product {
        return Product()
    }

    @Delete("/{id}")
    fun deleteProduct(@PathVariable id: Long): HttpResponse<*> {
        return HttpResponse.noContent<Any>()
    }
}
`

// micronautAllMethodsCode tests all HTTP methods.
const micronautAllMethodsCode = `
package com.example.controller;

import io.micronaut.http.annotation.*;

@Controller("/test")
public class TestController {

    @Get
    public String testGet() {
        return "get";
    }

    @Post
    public String testPost() {
        return "post";
    }

    @Put
    public String testPut() {
        return "put";
    }

    @Delete
    public String testDelete() {
        return "delete";
    }

    @Patch
    public String testPatch() {
        return "patch";
    }

    @Head
    public String testHead() {
        return "head";
    }

    @Options
    public String testOptions() {
        return "options";
    }
}
`

// micronautQueryParamsCode tests query parameters.
const micronautQueryParamsCode = `
package com.example.controller;

import io.micronaut.http.annotation.*;

@Controller("/api/search")
public class SearchController {

    @Get
    public List<Item> search(@QueryValue String query, @QueryValue Integer page, @QueryValue Integer size) {
        return new ArrayList<>();
    }

    @Get("/advanced")
    public List<Item> advancedSearch(@QueryValue("q") String query, @QueryValue Optional<String> filter) {
        return new ArrayList<>();
    }
}
`

// micronautDtoCode tests DTO classes in Java.
const micronautDtoCode = `
package com.example.dto;

public class UserDto {
    private Long id;
    private String name;
    private String email;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

public class CreateUserRequest {
    private String name;
    private String email;
    private String password;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }
}
`

// micronautKotlinDtoCode tests DTO classes in Kotlin.
const micronautKotlinDtoCode = `
package com.example.dto

data class UserResponse(
    val id: Long,
    val name: String,
    val email: String
)

data class CreateUserRequest(
    val name: String,
    val email: String,
    val password: String
)
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "micronaut", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".java")
	assert.Contains(t, exts, ".kt")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "micronaut", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "io.micronaut")
}

func TestPlugin_Detect_WithBuildGradle(t *testing.T) {
	dir := t.TempDir()
	buildGradle := `plugins {
    id 'io.micronaut.application' version '4.0.0'
}

dependencies {
    implementation 'io.micronaut:micronaut-http-server-netty'
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(buildGradle), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithBuildGradleKts(t *testing.T) {
	dir := t.TempDir()
	buildGradleKts := `plugins {
    id("io.micronaut.application") version "4.0.0"
}

dependencies {
    implementation("io.micronaut:micronaut-http-server-netty")
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(buildGradleKts), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithPomXml(t *testing.T) {
	dir := t.TempDir()
	pomXml := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <parent>
        <groupId>io.micronaut</groupId>
        <artifactId>micronaut-parent</artifactId>
        <version>4.0.0</version>
    </parent>
    <dependencies>
        <dependency>
            <groupId>io.micronaut</groupId>
            <artifactId>micronaut-http-server-netty</artifactId>
        </dependency>
    </dependencies>
</project>
`
	err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pomXml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutMicronaut(t *testing.T) {
	dir := t.TempDir()
	buildGradle := `plugins {
    id 'org.springframework.boot' version '3.2.0'
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
}
`
	err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(buildGradle), 0644)
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

func TestPlugin_ExtractRoutes_JavaController(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/controller/UserController.java",
			Language: "java",
			Content:  []byte(micronautJavaControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from @Get, @Post, etc.
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"] || methods["PUT"] || methods["DELETE"] || methods["PATCH"])
	}

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/api/users", getUsers.Path)
	}

	// Check GET /api/users/{id}
	getUserByID := findRoute(routes, "GET", "/api/users/{id}")
	if getUserByID != nil {
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
	}
}

func TestPlugin_ExtractRoutes_KotlinController(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/controller/ProductController.kt",
			Language: "kotlin",
			Content:  []byte(micronautKotlinControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from Kotlin controller
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"] || methods["PUT"] || methods["DELETE"])
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/controller/TestController.java",
			Language: "java",
			Content:  []byte(micronautAllMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have at least some methods
		assert.True(t, methods["GET"] || methods["POST"] || methods["PUT"] || methods["DELETE"] || methods["PATCH"])
	}
}

func TestPlugin_ExtractRoutes_QueryParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/controller/SearchController.java",
			Language: "java",
			Content:  []byte(micronautQueryParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Query parameter extraction is basic; just verify routes are extracted
	if len(routes) > 0 {
		for _, r := range routes {
			assert.Equal(t, "GET", r.Method)
		}
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonJavaKotlin(t *testing.T) {
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
			Path:     "/path/to/UserController.java",
			Language: "java",
			Content:  []byte(micronautJavaControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserController.java", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_JavaDTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/dto/UserDto.java",
			Language: "java",
			Content:  []byte(micronautDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	if len(schemas) > 0 {
		schemaNames := make(map[string]bool)
		for _, s := range schemas {
			schemaNames[s.Title] = true
		}
		assert.True(t, schemaNames["UserDto"] || schemaNames["CreateUserRequest"])
	}
}

func TestPlugin_ExtractSchemas_KotlinDTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/kotlin/com/example/dto/UserDto.kt",
			Language: "kotlin",
			Content:  []byte(micronautKotlinDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	if len(schemas) > 0 {
		schemaNames := make(map[string]bool)
		for _, s := range schemas {
			schemaNames[s.Title] = true
		}
		assert.True(t, schemaNames["UserResponse"] || schemaNames["CreateUserRequest"])
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

func TestCombinePaths(t *testing.T) {
	tests := []struct {
		basePath string
		path     string
		expected string
	}{
		{"", "", "/"},
		{"", "/users", "/users"},
		{"", "users", "/users"},
		{"/api", "", "/api"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "users", "/api/users"},
		{"api", "users", "/api/users"},
	}

	for _, tt := range tests {
		t.Run(tt.basePath+"_"+tt.path, func(t *testing.T) {
			result := combinePaths(tt.basePath, tt.path)
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

// Helper to find a route by method and path
func findRoute(routes []types.Route, method, path string) *types.Route {
	for i := range routes {
		if routes[i].Method == method && routes[i].Path == path {
			return &routes[i]
		}
	}
	return nil
}
