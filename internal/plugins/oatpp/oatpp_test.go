// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package oatpp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// oatppControllerCode is a comprehensive test fixture for Oat++ route extraction.
const oatppControllerCode = `
#include "oatpp/web/server/api/ApiController.hpp"

class UserController : public oatpp::web::server::api::ApiController {
public:
    ENDPOINT_INFO(getUsers) {
        info->summary = "Get all users";
    }
    ENDPOINT("GET", "/api/users", getUsers) {
        return createDtoResponse(Status::CODE_200, m_database->getUsers());
    }

    ENDPOINT_INFO(getUser) {
        info->summary = "Get user by ID";
    }
    ENDPOINT("GET", "/api/users/{userId}", getUser,
             PATH(Int32, userId)) {
        return createDtoResponse(Status::CODE_200, m_database->getUser(userId));
    }

    ENDPOINT("POST", "/api/users", createUser,
             BODY_DTO(Object<UserDto>, body)) {
        return createDtoResponse(Status::CODE_201, m_database->createUser(body));
    }

    ENDPOINT("PUT", "/api/users/{userId}", updateUser,
             PATH(Int32, userId),
             BODY_DTO(Object<UserDto>, body)) {
        return createDtoResponse(Status::CODE_200, m_database->updateUser(userId, body));
    }

    ENDPOINT("DELETE", "/api/users/{userId}", deleteUser,
             PATH(Int32, userId)) {
        return createDtoResponse(Status::CODE_204, nullptr);
    }
};
`

// oatppQueryParamsCode tests QUERY parameter extraction.
const oatppQueryParamsCode = `
#include "oatpp/web/server/api/ApiController.hpp"

class SearchController : public oatpp::web::server::api::ApiController {
public:
    ENDPOINT("GET", "/api/search", search,
             QUERY(String, q),
             QUERY(Int32, limit),
             QUERY(Int32, offset)) {
        return createDtoResponse(Status::CODE_200, m_database->search(q, limit, offset));
    }

    ENDPOINT("GET", "/api/items", getItems,
             QUERY(String, category),
             QUERY(Boolean, active)) {
        return createDtoResponse(Status::CODE_200, m_database->getItems(category, active));
    }
};
`

// oatppDtoCode tests DTO extraction.
const oatppDtoCode = `
#include "oatpp/core/macro/codegen.hpp"

#include OATPP_CODEGEN_BEGIN(DTO)

class UserDto : public oatpp::DTO {
    DTO_INIT(UserDto, DTO)

    DTO_FIELD(Int64, id);
    DTO_FIELD(String, name);
    DTO_FIELD(String, email);
    DTO_FIELD(Boolean, active);
};

class CreateUserDto : public oatpp::DTO {
    DTO_INIT(CreateUserDto, DTO)

    DTO_FIELD(String, name);
    DTO_FIELD(String, email);
    DTO_FIELD(String, password);
};

class UserResponseDto : public oatpp::DTO {
    DTO_INIT(UserResponseDto, DTO)

    DTO_FIELD(Int64, id);
    DTO_FIELD(String, name);
    DTO_FIELD(String, email);
    DTO_FIELD(Object<UserDto>, user);
    DTO_FIELD(List<Object<UserDto>>, users);
};

#include OATPP_CODEGEN_END(DTO)
`

// oatppAllMethodsCode tests all HTTP methods.
const oatppAllMethodsCode = `
#include "oatpp/web/server/api/ApiController.hpp"

class TestController : public oatpp::web::server::api::ApiController {
public:
    ENDPOINT("GET", "/test", testGet) { return createResponse(Status::CODE_200); }
    ENDPOINT("POST", "/test", testPost) { return createResponse(Status::CODE_201); }
    ENDPOINT("PUT", "/test", testPut) { return createResponse(Status::CODE_200); }
    ENDPOINT("DELETE", "/test", testDelete) { return createResponse(Status::CODE_204); }
    ENDPOINT("PATCH", "/test", testPatch) { return createResponse(Status::CODE_200); }
    ENDPOINT("HEAD", "/test", testHead) { return createResponse(Status::CODE_200); }
    ENDPOINT("OPTIONS", "/test", testOptions) { return createResponse(Status::CODE_200); }
};
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "oatpp", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".cpp")
	assert.Contains(t, exts, ".hpp")
	assert.Contains(t, exts, ".h")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "oatpp", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "oatpp")
}

func TestPlugin_Detect_WithCMakeLists(t *testing.T) {
	dir := t.TempDir()
	cmakeContent := `cmake_minimum_required(VERSION 3.5)
project(myapp)

find_package(oatpp REQUIRED)

add_executable(myapp main.cpp)
target_link_libraries(myapp PRIVATE oatpp::oatpp)
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
oatpp/1.3.0

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
        "oatpp"
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

func TestPlugin_Detect_WithoutOatpp(t *testing.T) {
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

func TestPlugin_ExtractRoutes_Endpoints(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/UserController.hpp",
			Language: "cpp",
			Content:  []byte(oatppControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from ENDPOINT macros
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /api/users
	getUsers := findRoute(routes, "GET", "/api/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/api/users", getUsers.Path)
		assert.Equal(t, "getUsers", getUsers.Handler)
	}

	// Check GET /api/users/{userId}
	getUserByID := findRoute(routes, "GET", "/api/users/{userId}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "userId", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}

	// Check POST /api/users
	postUsers := findRoute(routes, "POST", "/api/users")
	if postUsers != nil {
		assert.Equal(t, "POST", postUsers.Method)
		assert.Equal(t, "createUser", postUsers.Handler)
	}

	// Check PUT /api/users/{userId}
	putUser := findRoute(routes, "PUT", "/api/users/{userId}")
	if putUser != nil {
		assert.Equal(t, "PUT", putUser.Method)
	}

	// Check DELETE /api/users/{userId}
	deleteUser := findRoute(routes, "DELETE", "/api/users/{userId}")
	if deleteUser != nil {
		assert.Equal(t, "DELETE", deleteUser.Method)
	}
}

func TestPlugin_ExtractRoutes_QueryParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/SearchController.hpp",
			Language: "cpp",
			Content:  []byte(oatppQueryParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check search endpoint with query params
	searchRoute := findRoute(routes, "GET", "/api/search")
	if searchRoute != nil {
		assert.Equal(t, "GET", searchRoute.Method)

		// Should have query parameters
		queryParams := filterParamsByIn(searchRoute.Parameters, "query")
		assert.GreaterOrEqual(t, len(queryParams), 1)
	}

	// Check items endpoint
	itemsRoute := findRoute(routes, "GET", "/api/items")
	if itemsRoute != nil {
		assert.Equal(t, "GET", itemsRoute.Method)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/controllers/TestController.hpp",
			Language: "cpp",
			Content:  []byte(oatppAllMethodsCode),
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
			Path:     "/path/to/UserController.hpp",
			Language: "cpp",
			Content:  []byte(oatppControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserController.hpp", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DTOs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/dto/UserDto.hpp",
			Language: "cpp",
			Content:  []byte(oatppDtoCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract DTO classes
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	assert.True(t, schemaNames["UserDto"])
	assert.True(t, schemaNames["CreateUserDto"])
	assert.True(t, schemaNames["UserResponseDto"])

	// Check UserDto properties
	for _, s := range schemas {
		if s.Title == "UserDto" {
			assert.NotNil(t, s.Properties["id"])
			assert.NotNil(t, s.Properties["name"])
			assert.NotNil(t, s.Properties["email"])
			assert.NotNil(t, s.Properties["active"])

			assert.Equal(t, "integer", s.Properties["id"].Type)
			assert.Equal(t, "string", s.Properties["name"].Type)
			assert.Equal(t, "boolean", s.Properties["active"].Type)
		}
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
		{"/users/{userId}/posts/{postId}", 2, []string{"userId", "postId"}},
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

func TestOatppTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		oatppType   string
		wantType    string
		wantFormat  string
	}{
		{"String", "string", ""},
		{"oatpp::String", "string", ""},
		{"Int32", "integer", ""},
		{"Int64", "integer", ""},
		{"Float32", "number", ""},
		{"Float64", "number", ""},
		{"Boolean", "boolean", ""},
		{"Object<UserDto>", "object", ""},
		{"List<Object<UserDto>>", "array", ""},
	}

	for _, tt := range tests {
		t.Run(tt.oatppType, func(t *testing.T) {
			gotType, gotFormat := oatppTypeToOpenAPI(tt.oatppType)
			assert.Equal(t, tt.wantType, gotType)
			assert.Equal(t, tt.wantFormat, gotFormat)
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

// Helper to filter parameters by "in" type
func filterParamsByIn(params []types.Parameter, in string) []types.Parameter {
	var result []types.Parameter
	for _, p := range params {
		if p.In == in {
			result = append(result, p)
		}
	}
	return result
}
