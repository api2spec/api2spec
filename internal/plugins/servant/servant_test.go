// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package servant

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// servantBasicCode is a comprehensive test fixture for Servant API extraction.
const servantBasicCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Api where

import Servant

type GetUsers = "users" :> Get '[JSON] [User]

type GetUser = "users" :> Capture "id" Int :> Get '[JSON] User

type CreateUser = "users" :> ReqBody '[JSON] CreateUser :> Post '[JSON] User

type UpdateUser = "users" :> Capture "id" Int :> ReqBody '[JSON] UpdateUser :> Put '[JSON] User

type DeleteUser = "users" :> Capture "id" Int :> Delete '[JSON] NoContent
`

// servantPathParamsCode tests various path parameter formats.
const servantPathParamsCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Items where

import Servant

type GetItem = "items" :> Capture "itemId" Int :> Get '[JSON] Item

type GetItemDetail = "items" :> Capture "itemId" Int :> "details" :> Capture "detailId" Int :> Get '[JSON] ItemDetail

type GetItemBySlug = "items" :> Capture "slug" Text :> Get '[JSON] Item

type GetItemByUUID = "items" :> "uuid" :> Capture "id" UUID :> Get '[JSON] Item
`

// servantQueryParamsCode tests query parameter extraction.
const servantQueryParamsCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Search where

import Servant

type Search = "search" :> QueryParam "q" Text :> QueryParam "limit" Int :> QueryParam "offset" Int :> Get '[JSON] SearchResults

type Filter = "items" :> QueryParam "category" Text :> QueryParam "active" Bool :> Get '[JSON] [Item]

type RequiredParam = "items" :> QueryParam' '[Required] "id" Int :> Get '[JSON] Item
`

// servantAllMethodsCode tests all HTTP methods.
const servantAllMethodsCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Methods where

import Servant

type TestGet = "test" :> Get '[JSON] Text
type TestPost = "test" :> Post '[JSON] Text
type TestPut = "test" :> Put '[JSON] Text
type TestDelete = "test" :> Delete '[JSON] NoContent
type TestPatch = "test" :> Patch '[JSON] Text
type TestHead = "test" :> Head '[JSON] NoContent
type TestOptions = "test" :> Options '[JSON] NoContent
`

// servantCombinedAPICode tests combined APIs using :<|>.
const servantCombinedAPICode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Combined where

import Servant

type UserAPI = "users" :> Get '[JSON] [User]
           :<|> "users" :> Capture "id" Int :> Get '[JSON] User
           :<|> "users" :> ReqBody '[JSON] CreateUser :> Post '[JSON] User

type ProductAPI = "products" :> Get '[JSON] [Product]
              :<|> "products" :> Capture "id" Int :> Get '[JSON] Product

type API = UserAPI :<|> ProductAPI
`

// servantNestedPathsCode tests nested path segments.
const servantNestedPathsCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module Nested where

import Servant

type ApiV1Users = "api" :> "v1" :> "users" :> Get '[JSON] [User]

type UserPosts = "users" :> Capture "userId" Int :> "posts" :> Get '[JSON] [Post]

type UserPostComments = "users" :> Capture "userId" Int :> "posts" :> Capture "postId" Int :> "comments" :> Get '[JSON] [Comment]
`

// servantDataTypesCode tests Haskell data type extraction.
const servantDataTypesCode = `
{-# LANGUAGE DeriveGeneric #-}

module Models where

import GHC.Generics
import Data.Aeson

data User = User
  { userId :: Int
  , userName :: Text
  , userEmail :: Text
  , userActive :: Bool
  } deriving (Generic, Show)

instance ToJSON User
instance FromJSON User

data CreateUser = CreateUser
  { createUserName :: Text
  , createUserEmail :: Text
  , createUserPassword :: Text
  } deriving (Generic, Show)

data UpdateUser = UpdateUser
  { updateUserName :: Maybe Text
  , updateUserEmail :: Maybe Text
  } deriving (Generic, Show)

data SearchResults = SearchResults
  { searchItems :: [User]
  , searchTotal :: Int
  , searchPage :: Int
  } deriving (Generic, Show)
`

// servantReqBodyCode tests request body extraction.
const servantReqBodyCode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module ReqBody where

import Servant

type CreateItem = "items" :> ReqBody '[JSON] NewItem :> Post '[JSON] Item

type UpdateItem = "items" :> Capture "id" Int :> ReqBody '[JSON] UpdateItem :> Put '[JSON] Item

type CreateWithPlainText = "text" :> ReqBody '[PlainText] Text :> Post '[PlainText] Text
`

// servantNestedParenAPICode tests APIs with shared prefixes using parentheses.
const servantNestedParenAPICode = `
{-# LANGUAGE DataKinds #-}
{-# LANGUAGE TypeOperators #-}

module NestedParen where

import Servant

type UsersAPI = "users" :>
  (    Get '[JSON] [User]
  :<|> Capture "id" Int :> Get '[JSON] User
  :<|> ReqBody '[JSON] CreateUser :> PostCreated '[JSON] User
  :<|> Capture "id" Int :> DeleteNoContent
  )
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "servant", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".hs")
	assert.Contains(t, exts, ".lhs")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "servant", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "servant")
}

func TestPlugin_Detect_WithPackageYaml(t *testing.T) {
	dir := t.TempDir()
	packageYaml := `name: my-servant-app
version: 0.1.0.0

dependencies:
  - base >= 4.7 && < 5
  - servant
  - servant-server
  - aeson
  - warp

library:
  source-dirs: src
`
	err := os.WriteFile(filepath.Join(dir, "package.yaml"), []byte(packageYaml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithCabalFile(t *testing.T) {
	dir := t.TempDir()
	cabalFile := `name:                my-servant-app
version:             0.1.0.0
build-type:          Simple

library
  exposed-modules:     Api
  build-depends:       base >= 4.7 && < 5
                     , servant
                     , servant-server
                     , aeson
                     , warp
  default-language:    Haskell2010
`
	err := os.WriteFile(filepath.Join(dir, "my-servant-app.cabal"), []byte(cabalFile), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutServant(t *testing.T) {
	dir := t.TempDir()
	packageYaml := `name: my-app
version: 0.1.0.0

dependencies:
  - base >= 4.7 && < 5
  - aeson
  - text
`
	err := os.WriteFile(filepath.Join(dir, "package.yaml"), []byte(packageYaml), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoPackageFile(t *testing.T) {
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
			Path:     "src/Api.hs",
			Language: "haskell",
			Content:  []byte(servantBasicCode),
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
		assert.Equal(t, "GetUsers", getUsersRoute.Handler)
	}

	// Check GET /users/{id}
	getUserRoute := findRoute(routes, "GET", "/users/{id}")
	if getUserRoute != nil {
		assert.Equal(t, "GET", getUserRoute.Method)
		assert.Equal(t, "/users/{id}", getUserRoute.Path)
		// Should have path parameter
		pathParams := filterParamsByIn(getUserRoute.Parameters, "path")
		assert.Len(t, pathParams, 1)
		assert.Equal(t, "id", pathParams[0].Name)
		assert.True(t, pathParams[0].Required)
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

func TestPlugin_ExtractRoutes_PathParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Items.hs",
			Language: "haskell",
			Content:  []byte(servantPathParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check single path param
	getItemRoute := findRoute(routes, "GET", "/items/{itemId}")
	if getItemRoute != nil {
		pathParams := filterParamsByIn(getItemRoute.Parameters, "path")
		assert.Len(t, pathParams, 1)
		assert.Equal(t, "itemId", pathParams[0].Name)
		assert.Equal(t, "integer", pathParams[0].Schema.Type)
	}

	// Check string path param
	getBySlugRoute := findRoute(routes, "GET", "/items/{slug}")
	if getBySlugRoute != nil {
		pathParams := filterParamsByIn(getBySlugRoute.Parameters, "path")
		assert.Len(t, pathParams, 1)
		assert.Equal(t, "slug", pathParams[0].Name)
		assert.Equal(t, "string", pathParams[0].Schema.Type)
	}

	// Check UUID path param
	getByUUIDRoute := findRoute(routes, "GET", "/items/uuid/{id}")
	if getByUUIDRoute != nil {
		pathParams := filterParamsByIn(getByUUIDRoute.Parameters, "path")
		assert.Len(t, pathParams, 1)
		assert.Equal(t, "id", pathParams[0].Name)
		assert.Equal(t, "uuid", pathParams[0].Schema.Format)
	}
}

func TestPlugin_ExtractRoutes_QueryParams(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Search.hs",
			Language: "haskell",
			Content:  []byte(servantQueryParamsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Check search endpoint with query params
	searchRoute := findRoute(routes, "GET", "/search")
	if searchRoute != nil {
		queryParams := filterParamsByIn(searchRoute.Parameters, "query")
		assert.GreaterOrEqual(t, len(queryParams), 1)

		// Default QueryParam should not be required
		for _, param := range queryParams {
			if param.Name == "q" {
				assert.False(t, param.Required)
			}
		}
	}

	// Check required query param
	requiredRoute := findRoute(routes, "GET", "/items")
	if requiredRoute != nil {
		queryParams := filterParamsByIn(requiredRoute.Parameters, "query")
		for _, param := range queryParams {
			if param.Name == "id" {
				assert.True(t, param.Required)
			}
		}
	}
}

func TestPlugin_ExtractRoutes_AllMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Methods.hs",
			Language: "haskell",
			Content:  []byte(servantAllMethodsCode),
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
			Path:     "src/Nested.hs",
			Language: "haskell",
			Content:  []byte(servantNestedPathsCode),
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
		pathParams := filterParamsByIn(userPostsRoute.Parameters, "path")
		assert.Len(t, pathParams, 1)
		assert.Equal(t, "userId", pathParams[0].Name)
	}

	// Check deeply nested paths
	commentsRoute := findRoute(routes, "GET", "/users/{userId}/posts/{postId}/comments")
	if commentsRoute != nil {
		pathParams := filterParamsByIn(commentsRoute.Parameters, "path")
		assert.Len(t, pathParams, 2)
	}
}

func TestPlugin_ExtractRoutes_CombinedAPIs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Combined.hs",
			Language: "haskell",
			Content:  []byte(servantCombinedAPICode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 5 routes from the combined APIs (UserAPI: 3, ProductAPI: 2)
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check UserAPI routes
	getUsersRoute := findRoute(routes, "GET", "/users")
	assert.NotNil(t, getUsersRoute, "Expected GET /users route")

	getUserRoute := findRoute(routes, "GET", "/users/{id}")
	assert.NotNil(t, getUserRoute, "Expected GET /users/{id} route")

	postUserRoute := findRoute(routes, "POST", "/users")
	assert.NotNil(t, postUserRoute, "Expected POST /users route")

	// Check ProductAPI routes
	getProductsRoute := findRoute(routes, "GET", "/products")
	assert.NotNil(t, getProductsRoute, "Expected GET /products route")

	getProductRoute := findRoute(routes, "GET", "/products/{id}")
	assert.NotNil(t, getProductRoute, "Expected GET /products/{id} route")
}

func TestPlugin_ExtractRoutes_NestedParenAPIs(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/NestedParen.hs",
			Language: "haskell",
			Content:  []byte(servantNestedParenAPICode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract 4 routes from the nested parentheses API
	assert.GreaterOrEqual(t, len(routes), 4)

	// All routes should have the shared /users prefix
	getUsersRoute := findRoute(routes, "GET", "/users")
	assert.NotNil(t, getUsersRoute, "Expected GET /users route")

	getUserRoute := findRoute(routes, "GET", "/users/{id}")
	assert.NotNil(t, getUserRoute, "Expected GET /users/{id} route")

	postUserRoute := findRoute(routes, "POST", "/users")
	assert.NotNil(t, postUserRoute, "Expected POST /users route")

	deleteUserRoute := findRoute(routes, "DELETE", "/users/{id}")
	assert.NotNil(t, deleteUserRoute, "Expected DELETE /users/{id} route")
}

func TestPlugin_ExtractRoutes_IgnoresNonHaskell(t *testing.T) {
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
			Path:     "/path/to/Api.hs",
			Language: "haskell",
			Content:  []byte(servantBasicCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/Api.hs", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas_DataTypes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Models.hs",
			Language: "haskell",
			Content:  []byte(servantDataTypesCode),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Should extract all data types
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
			assert.NotNil(t, s.Properties["userId"])
			assert.NotNil(t, s.Properties["userName"])
			assert.NotNil(t, s.Properties["userEmail"])
			assert.NotNil(t, s.Properties["userActive"])

			assert.Equal(t, "integer", s.Properties["userId"].Type)
			assert.Equal(t, "string", s.Properties["userName"].Type)
			assert.Equal(t, "boolean", s.Properties["userActive"].Type)

			// Non-optional fields should be required
			assert.Contains(t, s.Required, "userId")
			assert.Contains(t, s.Required, "userName")
		}

		// Check optional fields in UpdateUser
		if s.Title == "UpdateUser" {
			if nameProp := s.Properties["updateUserName"]; nameProp != nil {
				assert.True(t, nameProp.Nullable)
			}
			// Optional fields should not be required
			assert.NotContains(t, s.Required, "updateUserName")
			assert.NotContains(t, s.Required, "updateUserEmail")
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
		{"GET", "/users", "GetUsers", "getGetUsers"},
		{"POST", "/users", "CreateUser", "postCreateUser"},
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
