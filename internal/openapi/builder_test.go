// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/pkg/types"
)

func TestNewBuilder(t *testing.T) {
	cfg := config.Default()
	builder := NewBuilder(cfg)

	assert.NotNil(t, builder)
	assert.Equal(t, cfg, builder.config)
}

func TestBuilder_Build_EmptyRoutesAndSchemas(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Info.Title = "Test API"
	cfg.OpenAPI.Info.Version = "1.0.0"

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "Test API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)
	assert.Empty(t, doc.Paths)
}

func TestBuilder_Build_WithRoutes(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Info.Title = "Test API"

	routes := []types.Route{
		{
			Method:      "GET",
			Path:        "/users",
			Summary:     "List users",
			Description: "Returns a list of all users",
			Tags:        []string{"users"},
		},
		{
			Method:  "POST",
			Path:    "/users",
			Summary: "Create user",
			Tags:    []string{"users"},
		},
		{
			Method:  "GET",
			Path:    "/users/{id}",
			Summary: "Get user by ID",
			Tags:    []string{"users"},
			Parameters: []types.Parameter{
				{
					Name:     "id",
					In:       "path",
					Required: true,
					Schema:   &types.Schema{Type: "string"},
				},
			},
		},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	assert.Len(t, doc.Paths, 2)

	// Check /users path
	usersPath, ok := doc.Paths["/users"]
	require.True(t, ok)
	assert.NotNil(t, usersPath.Get)
	assert.NotNil(t, usersPath.Post)
	assert.Equal(t, "List users", usersPath.Get.Summary)
	assert.Equal(t, "Create user", usersPath.Post.Summary)

	// Check /users/{id} path
	userPath, ok := doc.Paths["/users/{id}"]
	require.True(t, ok)
	assert.NotNil(t, userPath.Get)
	assert.Len(t, userPath.Get.Parameters, 1)
	assert.Equal(t, "id", userPath.Get.Parameters[0].Name)
}

func TestBuilder_Build_WithSchemas(t *testing.T) {
	cfg := config.Default()

	schemas := []types.Schema{
		{
			Title:       "User",
			Type:        "object",
			Description: "A user object",
			Properties: map[string]*types.Schema{
				"id":   {Type: "string"},
				"name": {Type: "string"},
			},
			Required: []string{"id", "name"},
		},
		{
			Title: "Error",
			Type:  "object",
			Properties: map[string]*types.Schema{
				"code":    {Type: "integer"},
				"message": {Type: "string"},
			},
		},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, schemas)

	require.NoError(t, err)
	assert.NotNil(t, doc.Components)
	assert.Len(t, doc.Components.Schemas, 2)
	assert.Contains(t, doc.Components.Schemas, "User")
	assert.Contains(t, doc.Components.Schemas, "Error")
}

func TestBuilder_Build_WithServers(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Servers = []config.ServerConfig{
		{URL: "https://api.example.com", Description: "Production"},
		{URL: "https://staging.example.com", Description: "Staging"},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.Len(t, doc.Servers, 2)
	assert.Equal(t, "https://api.example.com", doc.Servers[0].URL)
	assert.Equal(t, "Production", doc.Servers[0].Description)
}

func TestBuilder_Build_WithTags(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Tags = []config.TagConfig{
		{Name: "users", Description: "User operations"},
		{Name: "posts", Description: "Post operations"},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.Len(t, doc.Tags, 2)
	assert.Equal(t, "users", doc.Tags[0].Name)
	assert.Equal(t, "User operations", doc.Tags[0].Description)
}

func TestBuilder_Build_WithSecurity(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Security = config.SecurityConfig{
		Schemes: map[string]config.SecuritySchemeConfig{
			"bearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT authentication",
			},
		},
		Default: []string{"bearerAuth"},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, doc.Components)
	assert.Len(t, doc.Components.SecuritySchemes, 1)
	assert.Contains(t, doc.Components.SecuritySchemes, "bearerAuth")
	assert.Len(t, doc.Security, 1)
}

func TestBuilder_Build_WithContact(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Info.Contact = config.ContactConfig{
		Name:  "API Support",
		Email: "support@example.com",
		URL:   "https://example.com/support",
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, doc.Info.Contact)
	assert.Equal(t, "API Support", doc.Info.Contact.Name)
	assert.Equal(t, "support@example.com", doc.Info.Contact.Email)
}

func TestBuilder_Build_WithLicense(t *testing.T) {
	cfg := config.Default()
	cfg.OpenAPI.Info.License = config.LicenseConfig{
		Name: "MIT",
		URL:  "https://opensource.org/licenses/MIT",
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, doc.Info.License)
	assert.Equal(t, "MIT", doc.Info.License.Name)
	assert.Equal(t, "https://opensource.org/licenses/MIT", doc.Info.License.URL)
}

func TestBuilder_Build_AllHTTPMethods(t *testing.T) {
	cfg := config.Default()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "TRACE"}
	routes := make([]types.Route, len(methods))
	for i, method := range methods {
		routes[i] = types.Route{
			Method: method,
			Path:   "/test",
		}
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	pathItem := doc.Paths["/test"]
	assert.NotNil(t, pathItem.Get)
	assert.NotNil(t, pathItem.Post)
	assert.NotNil(t, pathItem.Put)
	assert.NotNil(t, pathItem.Delete)
	assert.NotNil(t, pathItem.Patch)
	assert.NotNil(t, pathItem.Options)
	assert.NotNil(t, pathItem.Head)
	assert.NotNil(t, pathItem.Trace)
}

func TestBuilder_Build_InvalidMethod(t *testing.T) {
	cfg := config.Default()

	routes := []types.Route{
		{Method: "INVALID", Path: "/test"},
	}

	builder := NewBuilder(cfg)
	_, err := builder.Build(routes, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported HTTP method")
}

func TestBuilder_Build_DefaultResponses(t *testing.T) {
	cfg := config.Default()
	cfg.Generation.DefaultResponses = []string{"200", "400", "500"}

	routes := []types.Route{
		{Method: "GET", Path: "/test"},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	assert.Len(t, doc.Paths["/test"].Get.Responses, 3)
	assert.Contains(t, doc.Paths["/test"].Get.Responses, "200")
	assert.Contains(t, doc.Paths["/test"].Get.Responses, "400")
	assert.Contains(t, doc.Paths["/test"].Get.Responses, "500")
}

func TestBuilder_Build_RouteWithRequestBody(t *testing.T) {
	cfg := config.Default()

	routes := []types.Route{
		{
			Method: "POST",
			Path:   "/users",
			RequestBody: &types.RequestBody{
				Description: "User to create",
				Required:    true,
				Content: map[string]types.MediaType{
					"application/json": {
						Schema: &types.Schema{
							Ref: "#/components/schemas/CreateUser",
						},
					},
				},
			},
		},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	assert.NotNil(t, doc.Paths["/users"].Post.RequestBody)
	assert.Equal(t, "User to create", doc.Paths["/users"].Post.RequestBody.Description)
	assert.True(t, doc.Paths["/users"].Post.RequestBody.Required)
}

func TestBuilder_Build_RouteWithResponses(t *testing.T) {
	cfg := config.Default()

	routes := []types.Route{
		{
			Method: "GET",
			Path:   "/users",
			Responses: map[string]types.Response{
				"200": {
					Description: "List of users",
					Content: map[string]types.MediaType{
						"application/json": {
							Schema: &types.Schema{
								Type:  "array",
								Items: &types.Schema{Ref: "#/components/schemas/User"},
							},
						},
					},
				},
				"401": {Description: "Unauthorized"},
			},
		},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	assert.Len(t, doc.Paths["/users"].Get.Responses, 2)
	assert.Equal(t, "List of users", doc.Paths["/users"].Get.Responses["200"].Description)
}

func TestBuilder_Build_DeprecatedRoute(t *testing.T) {
	cfg := config.Default()

	routes := []types.Route{
		{
			Method:     "GET",
			Path:       "/old-endpoint",
			Summary:    "Deprecated endpoint",
			Deprecated: true,
		},
	}

	builder := NewBuilder(cfg)
	doc, err := builder.Build(routes, nil)

	require.NoError(t, err)
	assert.True(t, doc.Paths["/old-endpoint"].Get.Deprecated)
}

func TestSchemaRef(t *testing.T) {
	ref := SchemaRef("User")
	assert.Equal(t, "#/components/schemas/User", ref.Ref)
}

func TestSortedPaths(t *testing.T) {
	paths := map[string]types.PathItem{
		"/users":     {},
		"/posts":     {},
		"/comments":  {},
		"/admin":     {},
	}

	sorted := SortedPaths(paths)
	assert.Equal(t, []string{"/admin", "/comments", "/posts", "/users"}, sorted)
}

func TestSortedSchemas(t *testing.T) {
	schemas := map[string]*types.Schema{
		"User":    {},
		"Post":    {},
		"Comment": {},
		"Admin":   {},
	}

	sorted := SortedSchemas(schemas)
	assert.Equal(t, []string{"Admin", "Comment", "Post", "User"}, sorted)
}
