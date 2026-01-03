// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package crow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// crowBasicCode is a comprehensive test fixture for Crow route extraction.
const crowBasicCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    CROW_ROUTE(app, "/")
    ([]() {
        return "Hello world!";
    });

    CROW_ROUTE(app, "/api/users")
    ([](const crow::request& req) {
        return crow::response(200);
    });

    CROW_ROUTE(app, "/api/users/<int>")
    ([](int id) {
        return "User " + std::to_string(id);
    });

    app.port(8080).run();
    return 0;
}
`

// crowMethodsCode tests routes with explicit HTTP methods.
const crowMethodsCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    CROW_ROUTE(app, "/api/users")
    .methods("GET"_method)
    ([](const crow::request& req) {
        return crow::response(200);
    });

    CROW_ROUTE(app, "/api/users")
    .methods("POST"_method)
    ([](const crow::request& req) {
        return crow::response(201);
    });

    CROW_ROUTE(app, "/api/users/<int>")
    .methods("PUT"_method, "PATCH"_method)
    ([](const crow::request& req, int id) {
        return crow::response(200);
    });

    CROW_ROUTE(app, "/api/users/<int>")
    .methods("DELETE"_method)
    ([](int id) {
        return crow::response(204);
    });

    app.run();
    return 0;
}
`

// crowBlueprintCode tests blueprint routes.
const crowBlueprintCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;
    crow::Blueprint bp("api");

    CROW_BP_ROUTE(bp, "/items")
    ([](const crow::request& req) {
        return crow::response(200);
    });

    CROW_BP_ROUTE(bp, "/items/<int>")
    .methods("GET"_method)
    ([](int id) {
        return crow::response(200);
    });

    CROW_BP_ROUTE(bp, "/items/<int>")
    .methods("DELETE"_method)
    ([](int id) {
        return crow::response(204);
    });

    app.register_blueprint(bp);
    app.run();
    return 0;
}
`

// crowPathParamsCode tests various path parameter formats.
const crowPathParamsCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    // Integer parameter
    CROW_ROUTE(app, "/users/<int>")
    ([](int id) {
        return "User: " + std::to_string(id);
    });

    // Unsigned integer parameter
    CROW_ROUTE(app, "/posts/<uint>")
    ([](unsigned int id) {
        return "Post: " + std::to_string(id);
    });

    // Double parameter
    CROW_ROUTE(app, "/prices/<double>")
    ([](double price) {
        return "Price: " + std::to_string(price);
    });

    // String parameter
    CROW_ROUTE(app, "/products/<string>")
    ([](const std::string& slug) {
        return "Product: " + slug;
    });

    // Multiple parameters
    CROW_ROUTE(app, "/users/<int>/posts/<int>")
    ([](int userId, int postId) {
        return "User " + std::to_string(userId) + " Post " + std::to_string(postId);
    });

    app.run();
    return 0;
}
`

// crowNamedParamsCode tests named path parameters.
const crowNamedParamsCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    CROW_ROUTE(app, "/users/<int: userId>")
    ([](int userId) {
        return "User: " + std::to_string(userId);
    });

    CROW_ROUTE(app, "/products/<string: slug>")
    ([](const std::string& slug) {
        return "Product: " + slug;
    });

    CROW_ROUTE(app, "/orders/<int: orderId>/items/<int: itemId>")
    ([](int orderId, int itemId) {
        return "Order: " + std::to_string(orderId) + " Item: " + std::to_string(itemId);
    });

    app.run();
    return 0;
}
`

// crowAllMethodsCode tests all HTTP methods.
const crowAllMethodsCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    CROW_ROUTE(app, "/test").methods("GET"_method)([](){ return "get"; });
    CROW_ROUTE(app, "/test").methods("POST"_method)([](){ return "post"; });
    CROW_ROUTE(app, "/test").methods("PUT"_method)([](){ return "put"; });
    CROW_ROUTE(app, "/test").methods("DELETE"_method)([](){ return "delete"; });
    CROW_ROUTE(app, "/test").methods("PATCH"_method)([](){ return "patch"; });
    CROW_ROUTE(app, "/test").methods("HEAD"_method)([](){ return "head"; });
    CROW_ROUTE(app, "/test").methods("OPTIONS"_method)([](){ return "options"; });

    app.run();
    return 0;
}
`

// crowHTTPMethodEnumCode tests crow::HTTPMethod enum style.
const crowHTTPMethodEnumCode = `
#include "crow.h"

int main() {
    crow::SimpleApp app;

    CROW_ROUTE(app, "/api/data")
    .methods(crow::HTTPMethod::GET, crow::HTTPMethod::POST)
    ([](const crow::request& req) {
        return crow::response(200);
    });

    app.run();
    return 0;
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "crow", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".cpp")
	assert.Contains(t, exts, ".hpp")
	assert.Contains(t, exts, ".h")
	assert.Contains(t, exts, ".cc")
	assert.Contains(t, exts, ".cxx")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "crow", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "crow")
}

func TestPlugin_Detect_WithCMakeLists(t *testing.T) {
	dir := t.TempDir()
	cmakeContent := `cmake_minimum_required(VERSION 3.5)
project(myapp)

find_package(Crow CONFIG REQUIRED)

add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE Crow::Crow)
`
	err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte(cmakeContent), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithCrowHeader(t *testing.T) {
	dir := t.TempDir()

	// Create crow.h header file
	err := os.WriteFile(filepath.Join(dir, "crow.h"), []byte("// Crow header"), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithCrowAllHeader(t *testing.T) {
	dir := t.TempDir()

	// Create crow_all.h header file
	err := os.WriteFile(filepath.Join(dir, "crow_all.h"), []byte("// Crow all-in-one header"), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithVcpkg(t *testing.T) {
	dir := t.TempDir()
	vcpkgContent := `{
    "name": "myapp",
    "version": "1.0.0",
    "dependencies": [
        "crow"
    ]
}
`
	err := os.WriteFile(filepath.Join(dir, "vcpkg.json"), []byte(vcpkgContent), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutCrow(t *testing.T) {
	dir := t.TempDir()
	cmakeContent := `cmake_minimum_required(VERSION 3.5)
project(myapp)

find_package(Boost REQUIRED)
`
	err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte(cmakeContent), 0644)
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
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from CROW_ROUTE macros
	assert.GreaterOrEqual(t, len(routes), 3)

	// Check / route
	rootRoute := findRoute(routes, "GET", "/")
	if rootRoute != nil {
		assert.Equal(t, "GET", rootRoute.Method)
		assert.Equal(t, "/", rootRoute.Path)
	}

	// Check /api/users route
	usersRoute := findRoute(routes, "GET", "/api/users")
	if usersRoute != nil {
		assert.Equal(t, "GET", usersRoute.Method)
		assert.Equal(t, "/api/users", usersRoute.Path)
	}
}

func TestPlugin_ExtractRoutes_WithMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
	}

	// Check POST /api/users
	postUsers := findRoute(routes, "POST", "/api/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
	}

	// Check DELETE route
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}
	assert.True(t, methods["DELETE"])
}

func TestPlugin_ExtractRoutes_Blueprint(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowBlueprintCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from CROW_BP_ROUTE macros
	assert.GreaterOrEqual(t, len(routes), 2)

	// Check /items route
	itemsRoute := findRoute(routes, "GET", "/items")
	if itemsRoute != nil {
		assert.Equal(t, "GET", itemsRoute.Method)
		assert.Equal(t, "/items", itemsRoute.Path)
	}

	// Check DELETE /items/{param1}
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}
	assert.True(t, methods["DELETE"])
}

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check integer param route
	for _, r := range routes {
		if r.Path == "/users/{param1}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "param1", r.Parameters[0].Name)
			assert.Equal(t, "integer", r.Parameters[0].Schema.Type)
		}
	}
}

func TestPlugin_ExtractRoutes_NamedParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowNamedParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check named param route
	userRoute := findRoute(routes, "GET", "/users/{userId}")
	if userRoute != nil {
		assert.Len(t, userRoute.Parameters, 1)
		assert.Equal(t, "userId", userRoute.Parameters[0].Name)
		assert.Equal(t, "integer", userRoute.Parameters[0].Schema.Type)
	}

	// Check string named param
	productRoute := findRoute(routes, "GET", "/products/{slug}")
	if productRoute != nil {
		assert.Len(t, productRoute.Parameters, 1)
		assert.Equal(t, "slug", productRoute.Parameters[0].Name)
		assert.Equal(t, "string", productRoute.Parameters[0].Schema.Type)
	}

	// Check multiple named params
	orderRoute := findRoute(routes, "GET", "/orders/{orderId}/items/{itemId}")
	if orderRoute != nil {
		assert.Len(t, orderRoute.Parameters, 2)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowAllMethodsCode),
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

func TestPlugin_ExtractRoutes_HTTPMethodEnum(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowHTTPMethodEnumCode),
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
}

func TestPlugin_ExtractRoutes_IgnoresNonCpp(t *testing.T) {
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
			Path:     "/path/to/main.cpp",
			Language: "cpp",
			Content:  []byte(crowBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/main.cpp", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(crowBasicCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Crow doesn't have a schema system, so should be empty
	assert.Empty(t, schemas)
}

func TestConvertCrowPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/<int>", "/users/{param1}"},
		{"/users/<int>/posts/<int>", "/users/{param1}/posts/{param2}"},
		{"/users/<string>", "/users/{param1}"},
		{"/users/<int: userId>", "/users/{userId}"},
		{"/orders/<int: orderId>/items/<int: itemId>", "/orders/{orderId}/items/{itemId}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertCrowPathParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCrowTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		crowType   string
		wantType   string
		wantFormat string
	}{
		{"int", "integer", ""},
		{"uint", "integer", ""},
		{"double", "number", ""},
		{"string", "string", ""},
		{"unknown", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.crowType, func(t *testing.T) {
			gotType, gotFormat := crowTypeToOpenAPI(tt.crowType)
			assert.Equal(t, tt.wantType, gotType)
			assert.Equal(t, tt.wantFormat, gotFormat)
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
		path string
		want []string
	}{
		{"/users", []string{"users"}},
		{"/api/users", []string{"users"}},
		{"/api/v1/users", []string{"users"}},
		{"/", nil},
		{"/{id}", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := inferTags(tt.path)
			assert.Equal(t, tt.want, result)
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
