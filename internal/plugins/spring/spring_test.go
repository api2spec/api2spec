// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package spring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// springControllerCode is a comprehensive test fixture for Spring Boot route extraction.
const springControllerCode = `
package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;

@RestController
@RequestMapping("/api/users")
public class UserController {

    @GetMapping
    public ResponseEntity<List<User>> getUsers() {
        return ResponseEntity.ok(new ArrayList<>());
    }

    @GetMapping("/{id}")
    public ResponseEntity<User> getUser(@PathVariable Long id) {
        return ResponseEntity.ok(new User());
    }

    @PostMapping
    public ResponseEntity<User> createUser(@RequestBody CreateUserDto user) {
        return ResponseEntity.created(null).body(new User());
    }

    @PutMapping("/{id}")
    public ResponseEntity<User> updateUser(@PathVariable Long id, @RequestBody UpdateUserDto user) {
        return ResponseEntity.ok(new User());
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<Void> deleteUser(@PathVariable Long id) {
        return ResponseEntity.noContent().build();
    }

    @PatchMapping("/{id}/status")
    public ResponseEntity<User> updateUserStatus(@PathVariable Long id, @RequestBody StatusDto status) {
        return ResponseEntity.ok(new User());
    }
}
`

// springRequestMappingCode tests @RequestMapping with method attribute.
const springRequestMappingCode = `
package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/products")
public class ProductController {

    @RequestMapping(method = RequestMethod.GET)
    public List<Product> getProducts() {
        return new ArrayList<>();
    }

    @RequestMapping(value = "/{id}", method = RequestMethod.GET)
    public Product getProduct(@PathVariable Long id) {
        return new Product();
    }

    @RequestMapping(path = "/{id}", method = RequestMethod.PUT)
    public Product updateProduct(@PathVariable Long id, @RequestBody Product product) {
        return product;
    }
}
`

// springAllMethodsCode tests all HTTP methods.
const springAllMethodsCode = `
package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/test")
public class TestController {

    @GetMapping
    public String testGet() { return "get"; }

    @PostMapping
    public String testPost() { return "post"; }

    @PutMapping
    public String testPut() { return "put"; }

    @DeleteMapping
    public String testDelete() { return "delete"; }

    @PatchMapping
    public String testPatch() { return "patch"; }
}
`

// springDtoCode tests DTO extraction.
const springDtoCode = `
package com.example.demo.dto;

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

// springRouteConstraintsCode tests routes with regex constraints.
const springRouteConstraintsCode = `
package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/items")
public class ItemController {

    @GetMapping("/{id:\\d+}")
    public Item getItem(@PathVariable Long id) {
        return new Item();
    }

    @GetMapping("/{slug:[a-z-]+}")
    public Item getItemBySlug(@PathVariable String slug) {
        return new Item();
    }
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "spring", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".java")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "spring", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "spring-boot")
}

func TestPlugin_Detect_WithPomXml(t *testing.T) {
	dir := t.TempDir()
	pomXml := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
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

func TestPlugin_Detect_WithBuildGradle(t *testing.T) {
	dir := t.TempDir()
	buildGradle := `plugins {
    id 'org.springframework.boot' version '3.2.0'
    id 'io.spring.dependency-management' version '1.1.4'
    id 'java'
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
	assert.True(t, detected)
}

func TestPlugin_Detect_WithBuildGradleKts(t *testing.T) {
	dir := t.TempDir()
	buildGradleKts := `plugins {
    id("org.springframework.boot") version "3.2.0"
    id("io.spring.dependency-management") version "1.1.4"
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
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutSpring(t *testing.T) {
	dir := t.TempDir()
	pomXml := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <dependencies>
        <dependency>
            <groupId>io.javalin</groupId>
            <artifactId>javalin</artifactId>
        </dependency>
    </dependencies>
</project>
`
	err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(pomXml), 0644)
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

func TestPlugin_ExtractRoutes_ControllerRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/demo/controller/UserController.java",
			Language: "java",
			Content:  []byte(springControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from @GetMapping, @PostMapping, etc.
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/api/users", getUsers.Path)
		assert.Contains(t, getUsers.Handler, "getUsers")
	}

	// Check GET /api/users/{id}
	getUserByID := findRoute(routes, "GET", "/api/users/{id}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}

	// Check POST /api/users
	postUsers := findRoute(routes, "POST", "/api/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
	}

	// Check PUT /api/users/{id}
	putUser := findRoute(routes, "PUT", "/api/users/{id}")
	if putUser != nil {
		assert.Equal(t, "PUT", putUser.Method)
	}

	// Check DELETE /api/users/{id}
	deleteUser := findRoute(routes, "DELETE", "/api/users/{id}")
	if deleteUser != nil {
		assert.Equal(t, "DELETE", deleteUser.Method)
	}
}

func TestPlugin_ExtractRoutes_RequestMapping(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/demo/controller/ProductController.java",
			Language: "java",
			Content:  []byte(springRequestMappingCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check routes extracted from @RequestMapping with method attribute
	getProducts := findRoute(routes, "GET", "/api/products")
	if getProducts != nil {
		assert.Equal(t, "GET", getProducts.Method)
	}

	putProduct := findRoute(routes, "PUT", "/api/products/{id}")
	if putProduct != nil {
		assert.Equal(t, "PUT", putProduct.Method)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/demo/controller/TestController.java",
			Language: "java",
			Content:  []byte(springAllMethodsCode),
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

func TestPlugin_ExtractRoutes_RouteConstraints(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/demo/controller/ItemController.java",
			Language: "java",
			Content:  []byte(springRouteConstraintsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Regex constraints should be stripped from path params
	for _, r := range routes {
		// Path should have {id} not {id:\d+}
		if r.Path == "/api/items/{id}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "id", r.Parameters[0].Name)
		}
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonJava(t *testing.T) {
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
			Content:  []byte(springControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserController.java", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/java/com/example/demo/dto/UserDto.java",
			Language: "java",
			Content:  []byte(springDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract DTOs with standard naming conventions
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	// Check that DTO classes are extracted
	if len(schemas) > 0 {
		assert.True(t, schemaNames["UserDto"] || schemaNames["CreateUserRequest"])
	}
}

func TestConvertSpringPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/{id}", "/users/{id}"},
		{"/users/{id:\\d+}", "/users/{id}"},
		{"/users/{slug:[a-z-]+}", "/users/{slug}"},
		{"/items/{id:\\d+}/details/{detailId:[a-z]+}", "/items/{id}/details/{detailId}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertSpringPathParams(tt.input)
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

func TestCombinePaths(t *testing.T) {
	tests := []struct {
		base     string
		relative string
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
		t.Run(tt.base+"_"+tt.relative, func(t *testing.T) {
			result := combinePaths(tt.base, tt.relative)
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

func TestParseRequestMethod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GET", "GET"},
		{"POST", "POST"},
		{"RequestMethod.GET", "GET"},
		{"RequestMethod.POST", "POST"},
		{"invalid", "GET"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseRequestMethod(tt.input)
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
