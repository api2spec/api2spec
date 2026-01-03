// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package vapor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// vaporBasicCode is a comprehensive test fixture for Vapor route extraction.
const vaporBasicCode = `
import Vapor

func routes(_ app: Application) throws {
    app.get("users") { req in
        return User.query(on: req.db).all()
    }

    app.get("users", ":id") { req -> User in
        guard let user = try await User.find(req.parameters.get("id"), on: req.db) else {
            throw Abort(.notFound)
        }
        return user
    }

    app.post("users") { req -> User in
        let user = try req.content.decode(CreateUser.self)
        try await user.save(on: req.db)
        return user
    }

    app.put("users", ":id") { req -> User in
        let update = try req.content.decode(UpdateUser.self)
        guard let user = try await User.find(req.parameters.get("id"), on: req.db) else {
            throw Abort(.notFound)
        }
        user.name = update.name
        try await user.save(on: req.db)
        return user
    }

    app.delete("users", ":id") { req -> HTTPStatus in
        guard let user = try await User.find(req.parameters.get("id"), on: req.db) else {
            throw Abort(.notFound)
        }
        try await user.delete(on: req.db)
        return .ok
    }
}
`

// vaporGroupedRoutesCode tests route group patterns.
const vaporGroupedRoutesCode = `
import Vapor

func routes(_ app: Application) throws {
    let api = app.grouped("api")

    api.get("users") { req in
        return User.query(on: req.db).all()
    }

    api.post("users") { req -> User in
        let user = try req.content.decode(CreateUser.self)
        return user
    }

    let users = api.grouped("users")

    users.get(":id") { req -> User in
        guard let user = try await User.find(req.parameters.get("id"), on: req.db) else {
            throw Abort(.notFound)
        }
        return user
    }

    users.delete(":id") { req -> HTTPStatus in
        return .ok
    }
}
`

// vaporInlineGroupedCode tests inline grouped routes.
const vaporInlineGroupedCode = `
import Vapor

func routes(_ app: Application) throws {
    app.grouped("api").get("health") { req in
        return ["status": "ok"]
    }

    app.grouped("api").get("users") { req in
        return User.query(on: req.db).all()
    }

    app.grouped("v1").post("items") { req -> Item in
        let item = try req.content.decode(Item.self)
        return item
    }
}
`

// vaporControllerCode tests controller-style routes.
const vaporControllerCode = `
import Vapor

struct UserController: RouteCollection {
    func boot(routes: RoutesBuilder) throws {
        let users = routes.grouped("users")

        users.get(use: index)
        users.post(use: create)
        users.get(":id", use: show)
        users.put(":id", use: update)
        users.delete(":id", use: delete)
    }

    func index(req: Request) async throws -> [User] {
        return try await User.query(on: req.db).all()
    }

    func create(req: Request) async throws -> User {
        let user = try req.content.decode(CreateUser.self)
        try await user.save(on: req.db)
        return user
    }

    func show(req: Request) async throws -> User {
        guard let user = try await User.find(req.parameters.get("id"), on: req.db) else {
            throw Abort(.notFound)
        }
        return user
    }

    func update(req: Request) async throws -> User {
        return User()
    }

    func delete(req: Request) async throws -> HTTPStatus {
        return .ok
    }
}
`

// vaporAllMethodsCode tests all HTTP methods.
const vaporAllMethodsCode = `
import Vapor

func routes(_ app: Application) throws {
    app.get("test") { req in return "GET" }
    app.post("test") { req in return "POST" }
    app.put("test") { req in return "PUT" }
    app.delete("test") { req in return "DELETE" }
    app.patch("test") { req in return "PATCH" }
    app.head("test") { req in return "HEAD" }
    app.options("test") { req in return "OPTIONS" }
}
`

// vaporNestedPathsCode tests nested path segments.
const vaporNestedPathsCode = `
import Vapor

func routes(_ app: Application) throws {
    app.get("api", "v1", "users") { req in
        return User.query(on: req.db).all()
    }

    app.get("users", ":userId", "posts") { req in
        return Post.query(on: req.db).all()
    }

    app.get("users", ":userId", "posts", ":postId", "comments") { req in
        return Comment.query(on: req.db).all()
    }
}
`

// vaporSchemaCode tests schema extraction from Content structs.
const vaporSchemaCode = `
import Vapor

struct User: Content {
    var id: UUID?
    var name: String
    var email: String
    var age: Int
    var isActive: Bool
}

struct CreateUser: Content {
    var name: String
    var email: String
    var password: String
}

struct UpdateUser: Content {
    var name: String?
    var email: String?
}

struct Post: Content {
    var id: Int
    var title: String
    var body: String
    var authorId: UUID
    var createdAt: Date?
}

// This struct does not conform to Content and should be ignored
struct InternalData {
    var value: String
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "vapor", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".swift")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "vapor", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "vapor")
}

func TestPlugin_Detect_WithPackageSwift(t *testing.T) {
	dir := t.TempDir()
	packageSwift := `// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "MyVaporApp",
    dependencies: [
        .package(url: "https://github.com/vapor/vapor.git", from: "4.0.0"),
    ],
    targets: [
        .target(
            name: "App",
            dependencies: [
                .product(name: "Vapor", package: "vapor"),
            ]
        ),
    ]
)
`
	err := os.WriteFile(filepath.Join(dir, "Package.swift"), []byte(packageSwift), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutVapor(t *testing.T) {
	dir := t.TempDir()
	packageSwift := `// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "MyApp",
    dependencies: [
        .package(url: "https://github.com/apple/swift-argument-parser", from: "1.0.0"),
    ],
    targets: [
        .target(name: "MyApp"),
    ]
)
`
	err := os.WriteFile(filepath.Join(dir, "Package.swift"), []byte(packageSwift), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoPackageSwift(t *testing.T) {
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
			Path:     "Sources/App/routes.swift",
			Language: "swift",
			Content:  []byte(vaporBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 5 routes
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsersRoute := findRoute(routes, "GET", "/users")
	if getUsersRoute != nil {
		assert.Equal(t, "GET", getUsersRoute.Method)
		assert.Equal(t, "/users", getUsersRoute.Path)
	}

	// Check GET /users/{id}
	getUserRoute := findRoute(routes, "GET", "/users/{id}")
	if getUserRoute != nil {
		assert.Equal(t, "GET", getUserRoute.Method)
		assert.Len(t, getUserRoute.Parameters, 1)
		assert.Equal(t, "id", getUserRoute.Parameters[0].Name)
		assert.Equal(t, "path", getUserRoute.Parameters[0].In)
		assert.True(t, getUserRoute.Parameters[0].Required)
	}

	// Check POST /users
	postUserRoute := findRoute(routes, "POST", "/users")
	if postUserRoute != nil {
		assert.Equal(t, "POST", postUserRoute.Method)
	}

	// Check DELETE /users/{id}
	deleteUserRoute := findRoute(routes, "DELETE", "/users/{id}")
	if deleteUserRoute != nil {
		assert.Equal(t, "DELETE", deleteUserRoute.Method)
	}
}

func TestPlugin_ExtractRoutes_GroupedRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/routes.swift",
			Language: "swift",
			Content:  []byte(vaporGroupedRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes with group prefixes
	assert.GreaterOrEqual(t, len(routes), 2)

	// Check GET /api/users
	apiUsersRoute := findRoute(routes, "GET", "/api/users")
	if apiUsersRoute != nil {
		assert.Equal(t, "/api/users", apiUsersRoute.Path)
	}

	// Check POST /api/users
	postApiUsersRoute := findRoute(routes, "POST", "/api/users")
	if postApiUsersRoute != nil {
		assert.Equal(t, "/api/users", postApiUsersRoute.Path)
	}
}

func TestPlugin_ExtractRoutes_InlineGrouped(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/routes.swift",
			Language: "swift",
			Content:  []byte(vaporInlineGroupedCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract inline grouped routes
	assert.GreaterOrEqual(t, len(routes), 3)

	// Check GET /api/health
	healthRoute := findRoute(routes, "GET", "/api/health")
	if healthRoute != nil {
		assert.Equal(t, "/api/health", healthRoute.Path)
	}

	// Check GET /api/users
	usersRoute := findRoute(routes, "GET", "/api/users")
	if usersRoute != nil {
		assert.Equal(t, "/api/users", usersRoute.Path)
	}

	// Check POST /v1/items
	itemsRoute := findRoute(routes, "POST", "/v1/items")
	if itemsRoute != nil {
		assert.Equal(t, "/v1/items", itemsRoute.Path)
	}
}

func TestPlugin_ExtractRoutes_ControllerRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/Controllers/UserController.swift",
			Language: "swift",
			Content:  []byte(vaporControllerCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract controller routes with handlers (at least some of them)
	assert.GreaterOrEqual(t, len(routes), 3)

	// Check routes have handler names
	foundWithHandler := false
	for _, route := range routes {
		if route.Handler != "" {
			foundWithHandler = true
			break
		}
	}
	assert.True(t, foundWithHandler, "At least one route should have a handler name")
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/routes.swift",
			Language: "swift",
			Content:  []byte(vaporAllMethodsCode),
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

func TestPlugin_ExtractRoutes_NestedPaths(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/routes.swift",
			Language: "swift",
			Content:  []byte(vaporNestedPathsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check nested static paths
	apiV1UsersRoute := findRoute(routes, "GET", "/api/v1/users")
	if apiV1UsersRoute != nil {
		assert.Equal(t, "/api/v1/users", apiV1UsersRoute.Path)
	}

	// Check nested paths with params
	userPostsRoute := findRoute(routes, "GET", "/users/{userId}/posts")
	if userPostsRoute != nil {
		assert.Len(t, userPostsRoute.Parameters, 1)
		assert.Equal(t, "userId", userPostsRoute.Parameters[0].Name)
	}

	// Check deeply nested paths
	commentsRoute := findRoute(routes, "GET", "/users/{userId}/posts/{postId}/comments")
	if commentsRoute != nil {
		assert.Len(t, commentsRoute.Parameters, 2)
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonSwift(t *testing.T) {
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
			Path:     "/path/to/routes.swift",
			Language: "swift",
			Content:  []byte(vaporBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/routes.swift", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_ContentStructs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "Sources/App/Models/User.swift",
			Language: "swift",
			Content:  []byte(vaporSchemaCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract only structs conforming to Content
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	assert.True(t, schemaNames["User"])
	assert.True(t, schemaNames["CreateUser"])
	assert.True(t, schemaNames["UpdateUser"])
	assert.True(t, schemaNames["Post"])
	// InternalData should NOT be included
	assert.False(t, schemaNames["InternalData"])

	// Check User properties
	for _, s := range schemas {
		if s.Title == "User" {
			assert.NotNil(t, s.Properties["id"])
			assert.NotNil(t, s.Properties["name"])
			assert.NotNil(t, s.Properties["email"])
			assert.NotNil(t, s.Properties["age"])
			assert.NotNil(t, s.Properties["isActive"])

			assert.Equal(t, "string", s.Properties["id"].Type)
			assert.Equal(t, "uuid", s.Properties["id"].Format)
			assert.Equal(t, "string", s.Properties["name"].Type)
			assert.Equal(t, "integer", s.Properties["age"].Type)
			assert.Equal(t, "boolean", s.Properties["isActive"].Type)

			// Optional fields should be nullable
			assert.True(t, s.Properties["id"].Nullable)

			// Non-optional fields should be required
			assert.Contains(t, s.Required, "name")
			assert.Contains(t, s.Required, "email")
		}

		// Check optional fields in UpdateUser
		if s.Title == "UpdateUser" {
			if nameProp := s.Properties["name"]; nameProp != nil {
				assert.True(t, nameProp.Nullable)
			}
			// Optional fields should not be required
			assert.NotContains(t, s.Required, "name")
			assert.NotContains(t, s.Required, "email")
		}
	}
}

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		handler  string
		expected string
	}{
		{"GET", "/users", "index", "getIndex"},
		{"POST", "/users", "create", "postCreate"},
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

func TestBuildPath(t *testing.T) {
	tests := []struct {
		prefix   string
		pathStr  string
		expected string
	}{
		{"", `"users"`, "/users"},
		{"api", `"users"`, "/api/users"},
		{"", `"users", ":id"`, "/users/{id}"},
		{"api", `"users", ":id"`, "/api/users/{id}"},
		{"", "", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := buildPath(tt.prefix, tt.pathStr)
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
