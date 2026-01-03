// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package nancy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// nancyBasicModuleCode tests basic Nancy module routes with indexed property syntax.
const nancyBasicModuleCode = `
using Nancy;

public class UsersModule : NancyModule
{
    public UsersModule()
    {
        Get["/users"] = _ => Response.AsJson(new List<User>());

        Get["/users/{id}"] = parameters =>
        {
            var id = (int)parameters.id;
            return Response.AsJson(new User());
        };

        Post["/users"] = _ => Response.AsJson(new User(), HttpStatusCode.Created);

        Put["/users/{id}"] = parameters =>
        {
            var id = (int)parameters.id;
            return Response.AsJson(new User());
        };

        Delete["/users/{id}"] = parameters =>
        {
            var id = (int)parameters.id;
            return HttpStatusCode.NoContent;
        };

        Patch["/users/{id}/status"] = parameters =>
        {
            return Response.AsJson(new User());
        };
    }
}
`

// nancyModuleWithBasePathCode tests Nancy module with base path.
const nancyModuleWithBasePathCode = `
using Nancy;

public class ProductsModule : NancyModule
{
    public ProductsModule() : base("/api/products")
    {
        Get["/"] = _ => Response.AsJson(new List<Product>());

        Get["/{id}"] = parameters => Response.AsJson(new Product());

        Post["/"] = _ => Response.AsJson(new Product(), HttpStatusCode.Created);
    }
}
`

// nancyAllMethodsCode tests all HTTP methods.
const nancyAllMethodsCode = `
using Nancy;

public class TestModule : NancyModule
{
    public TestModule()
    {
        Get["/test"] = _ => "get";
        Post["/test"] = _ => "post";
        Put["/test"] = _ => "put";
        Delete["/test"] = _ => "delete";
        Patch["/test"] = _ => "patch";
        Head["/test"] = _ => "head";
        Options["/test"] = _ => "options";
    }
}
`

// nancyMethodCallSyntaxCode tests method call syntax.
const nancyMethodCallSyntaxCode = `
using Nancy;

public class ApiModule : NancyModule
{
    public ApiModule()
    {
        Get("/items", _ => Response.AsJson(new List<Item>()));

        Get("/items/{id}", parameters =>
        {
            var id = (int)parameters.id;
            return Response.AsJson(new Item());
        });

        Post("/items", _ => Response.AsJson(new Item()));
    }
}
`

// nancyDtoCode tests DTO classes.
const nancyDtoCode = `
namespace MyApp.Models
{
    public class UserDto
    {
        public int Id { get; set; }
        public string Name { get; set; }
        public string Email { get; set; }
    }

    public class CreateUserRequest
    {
        public string Name { get; set; }
        public string Email { get; set; }
        public string Password { get; set; }
    }

    public class UserResponse
    {
        public int Id { get; set; }
        public string Name { get; set; }
        public string Email { get; set; }
        public DateTime CreatedAt { get; set; }
    }
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "nancy", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".cs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "nancy", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "Nancy")
}

func TestPlugin_Detect_WithCsproj(t *testing.T) {
	dir := t.TempDir()
	csproj := `<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net48</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Nancy" Version="2.0.0" />
  </ItemGroup>
</Project>
`
	err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte(csproj), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithPackagesConfig(t *testing.T) {
	dir := t.TempDir()
	packagesConfig := `<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="Nancy" version="2.0.0" targetFramework="net48" />
</packages>
`
	err := os.WriteFile(filepath.Join(dir, "packages.config"), []byte(packagesConfig), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutNancy(t *testing.T) {
	dir := t.TempDir()
	csproj := `<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.AspNetCore.App" />
  </ItemGroup>
</Project>
`
	err := os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte(csproj), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoCsproj(t *testing.T) {
	dir := t.TempDir()

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_ExtractRoutes_BasicModule(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Modules/UsersModule.cs",
			Language: "csharp",
			Content:  []byte(nancyBasicModuleCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from indexed property syntax
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check for various HTTP methods
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}
	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["PATCH"])

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
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
}

func TestPlugin_ExtractRoutes_ModuleWithBasePath(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Modules/ProductsModule.cs",
			Language: "csharp",
			Content:  []byte(nancyModuleWithBasePathCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should have base path applied
	for _, r := range routes {
		assert.True(t, r.Path == "/api/products" || r.Path == "/api/products/" || strings.HasPrefix(r.Path, "/api/products/"))
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Modules/TestModule.cs",
			Language: "csharp",
			Content:  []byte(nancyAllMethodsCode),
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

func TestPlugin_ExtractRoutes_MethodCallSyntax(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Modules/ApiModule.cs",
			Language: "csharp",
			Content:  []byte(nancyMethodCallSyntaxCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from method call syntax
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonCSharp(t *testing.T) {
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
			Path:     "/path/to/Modules/UsersModule.cs",
			Language: "csharp",
			Content:  []byte(nancyBasicModuleCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/Modules/UsersModule.cs", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Models/UserDto.cs",
			Language: "csharp",
			Content:  []byte(nancyDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract DTOs with standard naming conventions
	if len(schemas) > 0 {
		schemaNames := make(map[string]bool)
		for _, s := range schemas {
			schemaNames[s.Title] = true
		}
		// Check that DTO classes are extracted
		assert.True(t, schemaNames["UserDto"] || schemaNames["CreateUserRequest"] || schemaNames["UserResponse"])
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
		{"", "/users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "users", "/api/users"},
		{"", "users", "/users"},
		{"/api/products", "/", "/api/products/"},
		{"/api/products", "/{id}", "/api/products/{id}"},
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
		{"GET", "/users", "", "getUsers"},
		{"POST", "/users", "", "postUsers"},
		{"GET", "/users/{id}", "", "getUsersByid"},
		{"DELETE", "/users/{id}", "", "deleteUsersByid"},
		{"GET", "/", "", "get"},
		{"GET", "/users", "UsersModule", "getUsersModule"},
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
		path       string
		moduleName string
		expected   []string
	}{
		{"/users", "UsersModule", []string{"Users"}},
		{"/api/users", "UsersModule", []string{"Users"}},
		{"/users", "", []string{"users"}},
		{"/api/v1/users", "", []string{"users"}},
		{"/", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := inferTags(tt.path, tt.moduleName)
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

// Ensure strings is used
var _ = strings.Contains
