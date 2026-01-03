// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package play

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// playRoutesFile is a comprehensive test fixture for Play routes extraction.
const playRoutesFile = `
# Routes
# This file defines all application routes (Higher priority routes first)

# Home page
GET     /                           controllers.HomeController.index

# Users API
GET     /api/users                  controllers.UserController.list
GET     /api/users/:id              controllers.UserController.get(id: Long)
POST    /api/users                  controllers.UserController.create
PUT     /api/users/:id              controllers.UserController.update(id: Long)
DELETE  /api/users/:id              controllers.UserController.delete(id: Long)

# Products API
GET     /api/products               controllers.ProductController.list
GET     /api/products/:slug         controllers.ProductController.getBySlug(slug: String)
POST    /api/products               controllers.ProductController.create
PATCH   /api/products/:id/status    controllers.ProductController.updateStatus(id: Long)

# Map static resources from the /public folder to the /assets URL path
GET     /assets/*file               controllers.Assets.versioned(path="/public", file: Asset)
`

// playRoutesWithParams tests routes with various parameter formats.
const playRoutesWithParams = `
# Routes with different parameter formats

# Colon style parameters
GET     /users/:userId              controllers.UserController.get(userId: Long)
GET     /users/:userId/posts/:postId controllers.PostController.get(userId: Long, postId: Long)

# Dollar style parameters with regex
GET     /files/$path<.+>            controllers.FileController.get(path: String)
GET     /items/$id<[0-9]+>          controllers.ItemController.get(id: Long)

# Optional parameters (query strings)
GET     /search                     controllers.SearchController.search(q: String, page: Int ?= 1)
`

// playSubRoutesFile tests sub-route includes.
const playSubRoutesFile = `
# Main routes file with includes

GET     /                           controllers.HomeController.index

# Include API routes
->      /api                        api.Routes

# Include admin routes
->      /admin                      admin.Routes
`

// playCaseClassCode tests case class schema extraction.
const playCaseClassCode = `
package models

import play.api.libs.json._

case class UserDto(
  id: Long,
  name: String,
  email: String,
  active: Boolean
)

case class CreateUserRequest(
  name: String,
  email: String,
  password: String
)

case class UserResponse(
  id: Long,
  name: String,
  email: String,
  createdAt: java.time.LocalDateTime
)

case class PaginatedResponse[T](
  items: List[T],
  total: Long,
  page: Int,
  pageSize: Int
)

case class UserForm(
  name: String,
  email: Option[String],
  age: Option[Int]
)
`

// playAllMethodsRoutes tests all HTTP methods.
const playAllMethodsRoutes = `
# All HTTP methods

GET     /test       controllers.TestController.testGet
POST    /test       controllers.TestController.testPost
PUT     /test       controllers.TestController.testPut
DELETE  /test       controllers.TestController.testDelete
PATCH   /test       controllers.TestController.testPatch
HEAD    /test       controllers.TestController.testHead
OPTIONS /test       controllers.TestController.testOptions
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "play", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".scala")
	assert.Contains(t, exts, ".sc")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "play", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "Play Framework")
}

func TestPlugin_Detect_WithBuildSbt(t *testing.T) {
	dir := t.TempDir()
	buildSbt := `name := "my-play-app"

version := "1.0"

scalaVersion := "2.13.12"

libraryDependencies ++= Seq(
  "com.typesafe.play" %% "play" % "2.9.0"
)
`
	err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(buildSbt), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithRoutesFile(t *testing.T) {
	dir := t.TempDir()

	// Create conf directory and routes file
	confDir := filepath.Join(dir, "conf")
	err := os.MkdirAll(confDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(confDir, "routes"), []byte(playRoutesFile), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithPluginsSbt(t *testing.T) {
	dir := t.TempDir()

	// Create project directory
	projectDir := filepath.Join(dir, "project")
	err := os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	pluginsSbt := `addSbtPlugin("com.typesafe.play" % "sbt-plugin" % "2.9.0")`
	err = os.WriteFile(filepath.Join(projectDir, "plugins.sbt"), []byte(pluginsSbt), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutPlay(t *testing.T) {
	dir := t.TempDir()
	buildSbt := `name := "my-app"

version := "1.0"

scalaVersion := "2.13.12"

libraryDependencies ++= Seq(
  "org.typelevel" %% "cats-core" % "2.10.0"
)
`
	err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(buildSbt), 0644)
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
			Path:     "conf/routes",
			Language: "",
			Content:  []byte(playRoutesFile),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from routes file
	assert.GreaterOrEqual(t, len(routes), 10)

	// Check GET /
	homeRoute := findRoute(routes, "GET", "/")
	if homeRoute != nil {
		assert.Equal(t, "GET", homeRoute.Method)
		assert.Equal(t, "/", homeRoute.Path)
		assert.Contains(t, homeRoute.Handler, "index")
	}

	// Check GET /api/users
	usersRoute := findRoute(routes, "GET", "/api/users")
	if usersRoute != nil {
		assert.Equal(t, "GET", usersRoute.Method)
		assert.Equal(t, "/api/users", usersRoute.Path)
		assert.Contains(t, usersRoute.Handler, "list")
	}

	// Check GET /api/users/{id}
	userByIDRoute := findRoute(routes, "GET", "/api/users/{id}")
	if userByIDRoute != nil {
		assert.Equal(t, "GET", userByIDRoute.Method)
		assert.Len(t, userByIDRoute.Parameters, 1)
		assert.Equal(t, "id", userByIDRoute.Parameters[0].Name)
		assert.Equal(t, "path", userByIDRoute.Parameters[0].In)
		assert.True(t, userByIDRoute.Parameters[0].Required)
	}

	// Check POST /api/users
	createUserRoute := findRoute(routes, "POST", "/api/users")
	if createUserRoute != nil {
		assert.Equal(t, "POST", createUserRoute.Method)
	}

	// Check DELETE /api/users/{id}
	deleteUserRoute := findRoute(routes, "DELETE", "/api/users/{id}")
	if deleteUserRoute != nil {
		assert.Equal(t, "DELETE", deleteUserRoute.Method)
	}
}

func TestPlugin_ExtractRoutes_WithParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "conf/routes",
			Language: "",
			Content:  []byte(playRoutesWithParams),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check single path param
	userRoute := findRoute(routes, "GET", "/users/{userId}")
	if userRoute != nil {
		assert.Len(t, userRoute.Parameters, 1)
		assert.Equal(t, "userId", userRoute.Parameters[0].Name)
	}

	// Check multiple path params
	postRoute := findRoute(routes, "GET", "/users/{userId}/posts/{postId}")
	if postRoute != nil {
		assert.Len(t, postRoute.Parameters, 2)
	}

	// Check dollar style params with regex
	fileRoute := findRoute(routes, "GET", "/files/{path}")
	if fileRoute != nil {
		assert.Len(t, fileRoute.Parameters, 1)
		assert.Equal(t, "path", fileRoute.Parameters[0].Name)
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "conf/routes",
			Language: "",
			Content:  []byte(playAllMethodsRoutes),
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

func TestPlugin_ExtractRoutes_SourceInfo(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "/path/to/conf/routes",
			Language: "",
			Content:  []byte(playRoutesFile),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/conf/routes", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_CaseClasses(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "app/models/User.scala",
			Language: "scala",
			Content:  []byte(playCaseClassCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract case classes with DTO-like names
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	assert.True(t, schemaNames["UserDto"])
	assert.True(t, schemaNames["CreateUserRequest"])
	assert.True(t, schemaNames["UserResponse"])
	assert.True(t, schemaNames["UserForm"])

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

		// Check optional fields
		if s.Title == "UserForm" {
			if emailProp := s.Properties["email"]; emailProp != nil {
				assert.True(t, emailProp.Nullable)
			}
			if ageProp := s.Properties["age"]; ageProp != nil {
				assert.True(t, ageProp.Nullable)
			}
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
		{"GET", "/users", "list", "getList"},
		{"POST", "/users", "create", "postCreate"},
		{"GET", "/users/{id}", "get", "getGet"},
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
		{"/:id", nil},
		{"/$id", nil},
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
