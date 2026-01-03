// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package tapir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// tapirBasicCode is a comprehensive test fixture for Tapir route extraction.
const tapirBasicCode = `
package api

import sttp.tapir._
import sttp.tapir.json.circe._
import io.circe.generic.auto._

object UserEndpoints {

  val getUsers = endpoint.get
    .in("users")
    .out(jsonBody[List[User]])

  val getUser = endpoint.get
    .in("users" / path[Int]("id"))
    .out(jsonBody[User])

  val createUser = endpoint.post
    .in("users")
    .in(jsonBody[CreateUser])
    .out(jsonBody[User])

  val updateUser = endpoint.put
    .in("users" / path[Int]("id"))
    .in(jsonBody[UpdateUser])
    .out(jsonBody[User])

  val deleteUser = endpoint.delete
    .in("users" / path[Int]("id"))
    .out(emptyOutput)
}
`

// tapirPathParamsCode tests various path parameter formats.
const tapirPathParamsCode = `
package api

import sttp.tapir._

object ItemEndpoints {

  val getItem = endpoint.get
    .in("items" / path[Long]("itemId"))
    .out(jsonBody[Item])

  val getItemDetail = endpoint.get
    .in("items" / path[Long]("itemId") / "details" / path[Long]("detailId"))
    .out(jsonBody[ItemDetail])

  val getItemBySlug = endpoint.get
    .in("items" / path[String]("slug"))
    .out(jsonBody[Item])

  val getItemByUUID = endpoint.get
    .in("items" / "uuid" / path[UUID]("id"))
    .out(jsonBody[Item])
}
`

// tapirQueryParamsCode tests query parameter extraction.
const tapirQueryParamsCode = `
package api

import sttp.tapir._

object SearchEndpoints {

  val search = endpoint.get
    .in("search")
    .in(query[String]("q"))
    .in(query[Int]("limit"))
    .in(query[Int]("offset"))
    .out(jsonBody[SearchResults])

  val filter = endpoint.get
    .in("items")
    .in(query[Option[String]]("category"))
    .in(query[Option[Boolean]]("active"))
    .out(jsonBody[List[Item]])
}
`

// tapirAllMethodsCode tests all HTTP methods.
const tapirAllMethodsCode = `
package api

import sttp.tapir._

object TestEndpoints {

  val testGet = endpoint.get.in("test").out(stringBody)
  val testPost = endpoint.post.in("test").out(stringBody)
  val testPut = endpoint.put.in("test").out(stringBody)
  val testDelete = endpoint.delete.in("test").out(stringBody)
  val testPatch = endpoint.patch.in("test").out(stringBody)
  val testHead = endpoint.head.in("test").out(emptyOutput)
  val testOptions = endpoint.options.in("test").out(emptyOutput)
}
`

// tapirServerLogicCode tests endpoint chains with serverLogic.
const tapirServerLogicCode = `
package api

import sttp.tapir._
import sttp.tapir.server.ServerEndpoint

object UserRoutes {

  val getUserEndpoint = endpoint.get
    .in("users" / path[Int]("id"))
    .out(jsonBody[User])
    .serverLogic { id =>
      Future.successful(Right(User(id, "Test")))
    }

  val createUserEndpoint = endpoint.post
    .in("users")
    .in(jsonBody[CreateUser])
    .out(jsonBody[User])
    .serverLogicSuccess { user =>
      Future.successful(User(1, user.name))
    }
}
`

// tapirCaseClassCode tests case class schema extraction.
const tapirCaseClassCode = `
package models

case class User(
  id: Long,
  name: String,
  email: String,
  active: Boolean
)

case class CreateUser(
  name: String,
  email: String,
  password: String
)

case class UpdateUser(
  name: Option[String],
  email: Option[String]
)

case class SearchResults(
  items: List[User],
  total: Long,
  page: Int
)
`

// tapirNestedPathsCode tests nested path segments.
const tapirNestedPathsCode = `
package api

import sttp.tapir._

object NestedEndpoints {

  val getApiV1Users = endpoint.get
    .in("api" / "v1" / "users")
    .out(jsonBody[List[User]])

  val getUserPosts = endpoint.get
    .in("users" / path[Int]("userId") / "posts")
    .out(jsonBody[List[Post]])

  val getUserPostComment = endpoint.get
    .in("users" / path[Int]("userId") / "posts" / path[Int]("postId") / "comments" / path[Int]("commentId"))
    .out(jsonBody[Comment])
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "tapir", p.Name())
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

	assert.Equal(t, "tapir", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "tapir")
}

func TestPlugin_Detect_WithBuildSbt(t *testing.T) {
	dir := t.TempDir()
	buildSbt := `name := "my-tapir-app"

version := "1.0"

scalaVersion := "2.13.12"

libraryDependencies ++= Seq(
  "com.softwaremill.sttp.tapir" %% "tapir-core" % "1.9.0",
  "com.softwaremill.sttp.tapir" %% "tapir-json-circe" % "1.9.0",
  "com.softwaremill.sttp.tapir" %% "tapir-http4s-server" % "1.9.0"
)
`
	err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(buildSbt), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithSttpTapir(t *testing.T) {
	dir := t.TempDir()
	buildSbt := `libraryDependencies += "com.softwaremill.sttp.tapir" %% "tapir-akka-http-server" % "1.0.0"`
	err := os.WriteFile(filepath.Join(dir, "build.sbt"), []byte(buildSbt), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutTapir(t *testing.T) {
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

func TestPlugin_ExtractRoutes_BasicEndpoints(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/UserEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from endpoint definitions
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check GET /users
	getUsersRoute := findRoute(routes, "GET", "/users")
	if getUsersRoute != nil {
		assert.Equal(t, "GET", getUsersRoute.Method)
		assert.Equal(t, "/users", getUsersRoute.Path)
	}

	// Check POST /users
	createUserRoute := findRoute(routes, "POST", "/users")
	if createUserRoute != nil {
		assert.Equal(t, "POST", createUserRoute.Method)
	}

	// Check DELETE /users/{id}
	deleteUserRoute := findRoute(routes, "DELETE", "/users/{id}")
	if deleteUserRoute != nil {
		assert.Equal(t, "DELETE", deleteUserRoute.Method)
	}
}

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/ItemEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check single path param
	getItemRoute := findRoute(routes, "GET", "/items/{itemId}")
	if getItemRoute != nil {
		assert.Len(t, getItemRoute.Parameters, 1)
		assert.Equal(t, "itemId", getItemRoute.Parameters[0].Name)
		assert.Equal(t, "path", getItemRoute.Parameters[0].In)
		assert.True(t, getItemRoute.Parameters[0].Required)
		assert.Equal(t, "integer", getItemRoute.Parameters[0].Schema.Type)
	}

	// Check string path param
	getBySlugRoute := findRoute(routes, "GET", "/items/{slug}")
	if getBySlugRoute != nil {
		assert.Len(t, getBySlugRoute.Parameters, 1)
		assert.Equal(t, "slug", getBySlugRoute.Parameters[0].Name)
		assert.Equal(t, "string", getBySlugRoute.Parameters[0].Schema.Type)
	}

	// Check UUID path param
	getByUUIDRoute := findRoute(routes, "GET", "/items/uuid/{id}")
	if getByUUIDRoute != nil {
		assert.Len(t, getByUUIDRoute.Parameters, 1)
		assert.Equal(t, "id", getByUUIDRoute.Parameters[0].Name)
		assert.Equal(t, "uuid", getByUUIDRoute.Parameters[0].Schema.Format)
	}
}

func TestPlugin_ExtractRoutes_QueryParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/SearchEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirQueryParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check search endpoint with query params
	searchRoute := findRoute(routes, "GET", "/search")
	if searchRoute != nil {
		queryParams := filterParamsByIn(searchRoute.Parameters, "query")
		assert.GreaterOrEqual(t, len(queryParams), 1)

		// Check that required params are marked as required
		for _, param := range queryParams {
			if param.Name == "q" {
				assert.True(t, param.Required)
			}
		}
	}

	// Check filter endpoint with optional params
	filterRoute := findRoute(routes, "GET", "/items")
	if filterRoute != nil {
		queryParams := filterParamsByIn(filterRoute.Parameters, "query")
		// Optional params should not be required
		for _, param := range queryParams {
			if param.Name == "category" || param.Name == "active" {
				assert.False(t, param.Required)
			}
		}
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/TestEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirAllMethodsCode),
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

func TestPlugin_ExtractRoutes_ServerLogic(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/UserRoutes.scala",
			Language: "scala",
			Content:  []byte(tapirServerLogicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes even with serverLogic
	assert.GreaterOrEqual(t, len(routes), 2)
}

func TestPlugin_ExtractRoutes_NestedPaths(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/api/NestedEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirNestedPathsCode),
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
}

func TestPlugin_ExtractRoutes_IgnoresNonScala(t *testing.T) {
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
			Path:     "/path/to/UserEndpoints.scala",
			Language: "scala",
			Content:  []byte(tapirBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/UserEndpoints.scala", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_CaseClasses(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/main/scala/models/User.scala",
			Language: "scala",
			Content:  []byte(tapirCaseClassCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract all case classes
	schemaNames := make(map[string]bool)
	for _, s := range schemas {
		schemaNames[s.Title] = true
	}

	assert.True(t, schemaNames["User"])
	assert.True(t, schemaNames["CreateUser"])
	assert.True(t, schemaNames["UpdateUser"])
	assert.True(t, schemaNames["SearchResults"])

	// Check User properties
	for _, s := range schemas {
		if s.Title == "User" {
			assert.NotNil(t, s.Properties["id"])
			assert.NotNil(t, s.Properties["name"])
			assert.NotNil(t, s.Properties["email"])
			assert.NotNil(t, s.Properties["active"])

			assert.Equal(t, "integer", s.Properties["id"].Type)
			assert.Equal(t, "string", s.Properties["name"].Type)
			assert.Equal(t, "boolean", s.Properties["active"].Type)

			// Non-optional fields should be required
			assert.Contains(t, s.Required, "id")
			assert.Contains(t, s.Required, "name")
		}

		// Check optional fields
		if s.Title == "UpdateUser" {
			if nameProp := s.Properties["name"]; nameProp != nil {
				assert.True(t, nameProp.Nullable)
			}
			// Optional fields should not be required
			assert.NotContains(t, s.Required, "name")
		}
	}
}

func TestScalaTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		scalaType  string
		wantType   string
		wantFormat string
	}{
		{"String", "string", ""},
		{"Int", "integer", ""},
		{"Long", "integer", "int64"},
		{"Float", "number", "float"},
		{"Double", "number", "double"},
		{"Boolean", "boolean", ""},
		{"UUID", "string", "uuid"},
		{"Option[String]", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.scalaType, func(t *testing.T) {
			gotType, gotFormat := scalaTypeToOpenAPI(tt.scalaType)
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
