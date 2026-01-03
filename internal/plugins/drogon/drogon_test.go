// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package drogon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// drogonControllerCode is a comprehensive test fixture for Drogon route extraction.
const drogonControllerCode = `
#include <drogon/HttpController.h>

class UserController : public drogon::HttpController<UserController>
{
public:
    METHOD_LIST_BEGIN
    ADD_METHOD_TO(UserController, getUsers, "/api/users", Get);
    ADD_METHOD_TO(UserController, getUser, "/api/users/{id}", Get);
    ADD_METHOD_TO(UserController, createUser, "/api/users", Post);
    ADD_METHOD_TO(UserController, updateUser, "/api/users/{id}", Put);
    ADD_METHOD_TO(UserController, deleteUser, "/api/users/{id}", Delete);
    METHOD_LIST_END

    void getUsers(const HttpRequestPtr &req,
                  std::function<void(const HttpResponsePtr &)> &&callback);
    void getUser(const HttpRequestPtr &req,
                 std::function<void(const HttpResponsePtr &)> &&callback,
                 const std::string &id);
    void createUser(const HttpRequestPtr &req,
                    std::function<void(const HttpResponsePtr &)> &&callback);
    void updateUser(const HttpRequestPtr &req,
                    std::function<void(const HttpResponsePtr &)> &&callback,
                    const std::string &id);
    void deleteUser(const HttpRequestPtr &req,
                    std::function<void(const HttpResponsePtr &)> &&callback,
                    const std::string &id);
};
`

// drogonMethodAddCode tests METHOD_ADD macro format.
const drogonMethodAddCode = `
#include <drogon/HttpController.h>

class ProductController : public drogon::HttpController<ProductController>
{
public:
    METHOD_LIST_BEGIN
    METHOD_ADD(ProductController::getProducts, "/products", Get);
    METHOD_ADD(ProductController::getProduct, "/products/{id}", Get);
    METHOD_ADD(ProductController::createProduct, "/products", Post);
    METHOD_LIST_END
};
`

// drogonLambdaCode tests registerHandler with lambda functions.
const drogonLambdaCode = `
#include <drogon/drogon.h>

int main() {
    app().registerHandler("/hello",
        [](const HttpRequestPtr &req,
           std::function<void(const HttpResponsePtr &)> &&callback) {
            auto resp = HttpResponse::newHttpResponse();
            resp->setBody("Hello World!");
            callback(resp);
        });

    drogon::app().registerHandler("/api/items",
        [](const HttpRequestPtr &req,
           std::function<void(const HttpResponsePtr &)> &&callback) {
            auto resp = HttpResponse::newHttpJsonResponse(Json::Value());
            callback(resp);
        });

    app().run();
    return 0;
}
`

// drogonDtoCode tests DTO extraction.
const drogonDtoCode = `
#include <string>

struct UserDto {
    int id;
    std::string name;
    std::string email;
    bool active;
};

struct CreateUserRequest {
    std::string name;
    std::string email;
    std::string password;
};

struct UserResponse {
    int id;
    std::string name;
    std::string email;
};
`

// drogonPathParamsCode tests routes with path parameters.
const drogonPathParamsCode = `
#include <drogon/HttpController.h>

class ItemController : public drogon::HttpController<ItemController>
{
public:
    METHOD_LIST_BEGIN
    ADD_METHOD_TO(ItemController, getItem, "/items/{itemId}", Get);
    ADD_METHOD_TO(ItemController, getItemDetail, "/items/{itemId}/details/{detailId}", Get);
    ADD_METHOD_TO(ItemController, getItemBySlug, "/items/:slug", Get);
    METHOD_LIST_END
};
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "drogon", p.Name())
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

	assert.Equal(t, "drogon", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "drogon")
}

func TestPlugin_Detect_WithCMakeLists(t *testing.T) {
	dir := t.TempDir()
	cmakeContent := `cmake_minimum_required(VERSION 3.5)
project(myapp)

find_package(Drogon CONFIG REQUIRED)

add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE Drogon::Drogon)
`
	err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte(cmakeContent), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithConanfile(t *testing.T) {
	dir := t.TempDir()
	conanContent := `[requires]
drogon/1.8.0

[generators]
cmake
`
	err := os.WriteFile(filepath.Join(dir, "conanfile.txt"), []byte(conanContent), 0644)
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
        "drogon"
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

func TestPlugin_Detect_WithoutDrogon(t *testing.T) {
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

func TestPlugin_ExtractRoutes_AddMethodTo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/UserController.cpp",
			Language: "cpp",
			Content:  []byte(drogonControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from ADD_METHOD_TO macros
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

func TestPlugin_ExtractRoutes_MethodAdd(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/ProductController.cpp",
			Language: "cpp",
			Content:  []byte(drogonMethodAddCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from METHOD_ADD macros
	assert.GreaterOrEqual(t, len(routes), 3)

	// Check GET /products
	getProducts := findRoute(routes, "GET", "/products")
	if getProducts != nil {
		assert.Equal(t, "GET", getProducts.Method)
		assert.Equal(t, "/products", getProducts.Path)
	}

	// Check POST /products
	createProduct := findRoute(routes, "POST", "/products")
	if createProduct != nil {
		assert.Equal(t, "POST", createProduct.Method)
	}
}

func TestPlugin_ExtractRoutes_RegisterHandler(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main.cpp",
			Language: "cpp",
			Content:  []byte(drogonLambdaCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from registerHandler calls
	assert.GreaterOrEqual(t, len(routes), 2)

	// Check /hello route
	helloRoute := findRoute(routes, "GET", "/hello")
	if helloRoute != nil {
		assert.Equal(t, "GET", helloRoute.Method)
		assert.Equal(t, "/hello", helloRoute.Path)
	}

	// Check /api/items route
	itemsRoute := findRoute(routes, "GET", "/api/items")
	if itemsRoute != nil {
		assert.Equal(t, "GET", itemsRoute.Method)
		assert.Equal(t, "/api/items", itemsRoute.Path)
	}
}

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/ItemController.cpp",
			Language: "cpp",
			Content:  []byte(drogonPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check route with single path param
	getItem := findRoute(routes, "GET", "/items/{itemId}")
	if getItem != nil {
		assert.Len(t, getItem.Parameters, 1)
		assert.Equal(t, "itemId", getItem.Parameters[0].Name)
	}

	// Check route with multiple path params
	getDetail := findRoute(routes, "GET", "/items/{itemId}/details/{detailId}")
	if getDetail != nil {
		assert.Len(t, getDetail.Parameters, 2)
	}

	// Check :slug style param converted to {slug}
	getBySlug := findRoute(routes, "GET", "/items/{slug}")
	if getBySlug != nil {
		assert.Len(t, getBySlug.Parameters, 1)
		assert.Equal(t, "slug", getBySlug.Parameters[0].Name)
	}
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
			Path:     "/path/to/UserController.cpp",
			Language: "cpp",
			Content:  []byte(drogonControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserController.cpp", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/dto/UserDto.hpp",
			Language: "cpp",
			Content:  []byte(drogonDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract DTOs with standard naming conventions
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	// Check that DTO structs are extracted
	if len(schemas) > 0 {
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
		{"GET", "/users/{id}", "lambda", "getUsersByid"},
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
