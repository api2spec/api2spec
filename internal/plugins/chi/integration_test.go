// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package chi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// TestChiPlugin_EndToEnd tests the full flow from Go source to OpenAPI routes and schemas.
// This simulates a real-world chi API project with routes and models.
func TestChiPlugin_EndToEnd(t *testing.T) {
	// Simulated project structure:
	// models/user.go - Contains User and related types
	// routes/routes.go - Contains chi route definitions

	modelsSource := `package models

import "time"

// User represents a user in the system.
type User struct {
	// ID is the unique identifier for the user.
	ID        string    ` + "`json:\"id\"`" + `

	// Name is the user's display name.
	Name      string    ` + "`json:\"name\" validate:\"required,min=1,max=255\"`" + `

	// Email is the user's email address.
	Email     string    ` + "`json:\"email\" validate:\"required,email\"`" + `

	// Age is the user's age in years.
	Age       *int      ` + "`json:\"age,omitempty\" validate:\"min=0,max=150\"`" + `

	// Role is the user's role in the system.
	Role      string    ` + "`json:\"role\" validate:\"oneof=admin user guest\"`" + `

	// Tags are custom tags for the user.
	Tags      []string  ` + "`json:\"tags\"`" + `

	// Metadata contains additional user metadata.
	Metadata  map[string]string ` + "`json:\"metadata,omitempty\"`" + `

	// CreatedAt is when the user was created.
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `

	// UpdatedAt is when the user was last updated.
	UpdatedAt *time.Time ` + "`json:\"updated_at,omitempty\"`" + `

	// Password is never serialized.
	Password  string    ` + "`json:\"-\"`" + `
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Name     string   ` + "`json:\"name\" validate:\"required,min=1,max=255\"`" + `
	Email    string   ` + "`json:\"email\" validate:\"required,email\"`" + `
	Password string   ` + "`json:\"password\" validate:\"required,min=8\"`" + `
	Role     string   ` + "`json:\"role\" validate:\"oneof=admin user guest\"`" + `
	Tags     []string ` + "`json:\"tags,omitempty\"`" + `
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Name  *string  ` + "`json:\"name,omitempty\" validate:\"omitempty,min=1,max=255\"`" + `
	Email *string  ` + "`json:\"email,omitempty\" validate:\"omitempty,email\"`" + `
	Role  *string  ` + "`json:\"role,omitempty\" validate:\"omitempty,oneof=admin user guest\"`" + `
	Tags  []string ` + "`json:\"tags,omitempty\"`" + `
}

// Post represents a blog post.
type Post struct {
	ID        string    ` + "`json:\"id\"`" + `
	Title     string    ` + "`json:\"title\" validate:\"required,min=1,max=500\"`" + `
	Content   string    ` + "`json:\"content\" validate:\"required\"`" + `
	AuthorID  string    ` + "`json:\"author_id\"`" + `
	Published bool      ` + "`json:\"published\"`" + `
	Tags      []string  ` + "`json:\"tags\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`

	routesSource := `package routes

import (
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", HealthCheck)

		r.Route("/v1", func(r chi.Router) {
			// User routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/", ListUsers)
				r.Post("/", CreateUser)

				r.Route("/{userId}", func(r chi.Router) {
					r.Get("/", GetUser)
					r.Put("/", UpdateUser)
					r.Delete("/", DeleteUser)

					// User's posts
					r.Route("/posts", func(r chi.Router) {
						r.Get("/", ListUserPosts)
						r.Post("/", CreateUserPost)
					})
				})
			})

			// Post routes
			r.Route("/posts", func(r chi.Router) {
				r.Get("/", ListPosts)
				r.Get("/{postId}", GetPost)
				r.Put("/{postId}", UpdatePost)
				r.Delete("/{postId}", DeletePost)
			})
		})
	})
}
`

	// Create plugin and files
	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "models/user.go",
			Language: "go",
			Content:  []byte(modelsSource),
		},
		{
			Path:     "routes/routes.go",
			Language: "go",
			Content:  []byte(routesSource),
		},
	}

	// Extract routes
	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Extract schemas
	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)

	// Verify routes
	t.Run("routes extraction", func(t *testing.T) {
		// We should have extracted routes for health, users, and posts
		require.GreaterOrEqual(t, len(routes), 10, "expected at least 10 routes")

		// Create a map of routes for easier lookup
		routeMap := make(map[string]types.Route)
		for _, r := range routes {
			key := r.Method + " " + r.Path
			routeMap[key] = r
		}

		// Verify specific routes exist
		expectedRoutes := []string{
			"GET /api/health",
			"GET /api/v1/users",
			"POST /api/v1/users",
			"GET /api/v1/users/{userId}",
			"PUT /api/v1/users/{userId}",
			"DELETE /api/v1/users/{userId}",
			"GET /api/v1/users/{userId}/posts",
			"POST /api/v1/users/{userId}/posts",
			"GET /api/v1/posts",
			"GET /api/v1/posts/{postId}",
		}

		for _, expected := range expectedRoutes {
			route, exists := routeMap[expected]
			assert.True(t, exists, "route %s should exist", expected)
			if exists {
				assert.NotEmpty(t, route.Handler, "route %s should have a handler", expected)
			}
		}

		// Verify path parameters are extracted correctly
		getUserRoute := routeMap["GET /api/v1/users/{userId}"]
		require.Len(t, getUserRoute.Parameters, 1, "GET /users/{userId} should have 1 path param")
		assert.Equal(t, "userId", getUserRoute.Parameters[0].Name)
		assert.Equal(t, "path", getUserRoute.Parameters[0].In)
		assert.True(t, getUserRoute.Parameters[0].Required)

		// Verify tags are inferred correctly
		assert.Contains(t, getUserRoute.Tags, "users")
	})

	// Verify schemas
	t.Run("schemas extraction", func(t *testing.T) {
		require.GreaterOrEqual(t, len(schemas), 4, "expected at least 4 schemas")

		// Create a map of schemas for easier lookup
		schemaMap := make(map[string]types.Schema)
		for _, s := range schemas {
			schemaMap[s.Title] = s
		}

		// Verify User schema
		userSchema, exists := schemaMap["User"]
		require.True(t, exists, "User schema should exist")
		assert.Equal(t, "object", userSchema.Type)
		assert.Contains(t, userSchema.Required, "name")
		assert.Contains(t, userSchema.Required, "email")

		// Verify User properties
		require.NotNil(t, userSchema.Properties)
		assert.NotNil(t, userSchema.Properties["id"])
		assert.NotNil(t, userSchema.Properties["name"])
		assert.NotNil(t, userSchema.Properties["email"])
		assert.NotNil(t, userSchema.Properties["role"])
		assert.NotNil(t, userSchema.Properties["tags"])
		assert.NotNil(t, userSchema.Properties["created_at"])

		// Password should not be in properties (json:"-")
		assert.NotContains(t, userSchema.Properties, "password")

		// Verify email format
		emailProp := userSchema.Properties["email"]
		assert.Equal(t, "email", emailProp.Format)

		// Verify role enum
		roleProp := userSchema.Properties["role"]
		assert.Len(t, roleProp.Enum, 3)

		// Verify tags array
		tagsProp := userSchema.Properties["tags"]
		assert.Equal(t, "array", tagsProp.Type)
		assert.NotNil(t, tagsProp.Items)
		assert.Equal(t, "string", tagsProp.Items.Type)

		// Verify created_at is datetime
		createdAtProp := userSchema.Properties["created_at"]
		assert.Equal(t, "string", createdAtProp.Type)
		assert.Equal(t, "date-time", createdAtProp.Format)

		// Verify age is nullable (pointer)
		ageProp := userSchema.Properties["age"]
		assert.Equal(t, "integer", ageProp.Type)
		assert.True(t, ageProp.Nullable)

		// Verify CreateUserRequest schema
		createUserSchema, exists := schemaMap["CreateUserRequest"]
		require.True(t, exists, "CreateUserRequest schema should exist")
		assert.Contains(t, createUserSchema.Required, "name")
		assert.Contains(t, createUserSchema.Required, "email")
		assert.Contains(t, createUserSchema.Required, "password")

		// Verify UpdateUserRequest schema (all fields optional)
		updateUserSchema, exists := schemaMap["UpdateUserRequest"]
		require.True(t, exists, "UpdateUserRequest schema should exist")
		// All fields are pointers, so none should be required
		assert.NotContains(t, updateUserSchema.Required, "name")
		assert.NotContains(t, updateUserSchema.Required, "email")

		// Verify Post schema
		postSchema, exists := schemaMap["Post"]
		require.True(t, exists, "Post schema should exist")
		assert.Contains(t, postSchema.Required, "title")
		assert.Contains(t, postSchema.Required, "content")
	})
}

// TestChiPlugin_ComplexRoutePatterns tests various complex chi routing patterns.
func TestChiPlugin_ComplexRoutePatterns(t *testing.T) {
	source := `package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(r chi.Router) {
	// Middleware usage (should not affect route extraction)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Simple routes
	r.Get("/", Home)
	r.Get("/health", Health)
	r.Get("/version", Version)

	// RESTful resources
	r.Route("/articles", func(r chi.Router) {
		r.Get("/", ListArticles)      // GET /articles
		r.Post("/", CreateArticle)    // POST /articles

		r.Route("/{articleSlug}", func(r chi.Router) {
			r.Get("/", GetArticle)         // GET /articles/{articleSlug}
			r.Put("/", UpdateArticle)      // PUT /articles/{articleSlug}
			r.Delete("/", DeleteArticle)   // DELETE /articles/{articleSlug}

			// Nested resources
			r.Route("/comments", func(r chi.Router) {
				r.Get("/", ListComments)       // GET /articles/{articleSlug}/comments
				r.Post("/", CreateComment)     // POST /articles/{articleSlug}/comments
				r.Get("/{commentId}", GetComment) // GET /articles/{articleSlug}/comments/{commentId}
			})
		})
	})

	// Groups with shared configuration
	r.Group(func(r chi.Router) {
		r.Get("/admin/dashboard", AdminDashboard)
		r.Get("/admin/users", AdminUsers)
		r.Get("/admin/settings", AdminSettings)
	})

	// Route with regex constraint (should still extract param name)
	r.Get("/files/{path:.*}", ServeFile)
	r.Get("/users/{id:[0-9]+}", GetUserByID)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "api/routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Create map for easier lookup
	routeMap := make(map[string]types.Route)
	for _, r := range routes {
		key := r.Method + " " + r.Path
		routeMap[key] = r
	}

	// Test simple routes
	assert.Contains(t, routeMap, "GET /")
	assert.Contains(t, routeMap, "GET /health")
	assert.Contains(t, routeMap, "GET /version")

	// Test RESTful routes
	assert.Contains(t, routeMap, "GET /articles")
	assert.Contains(t, routeMap, "POST /articles")
	assert.Contains(t, routeMap, "GET /articles/{articleSlug}")
	assert.Contains(t, routeMap, "PUT /articles/{articleSlug}")
	assert.Contains(t, routeMap, "DELETE /articles/{articleSlug}")

	// Test nested routes
	assert.Contains(t, routeMap, "GET /articles/{articleSlug}/comments")
	assert.Contains(t, routeMap, "POST /articles/{articleSlug}/comments")
	assert.Contains(t, routeMap, "GET /articles/{articleSlug}/comments/{commentId}")

	// Test group routes
	assert.Contains(t, routeMap, "GET /admin/dashboard")
	assert.Contains(t, routeMap, "GET /admin/users")
	assert.Contains(t, routeMap, "GET /admin/settings")

	// Test regex path params
	filesRoute := routeMap["GET /files/{path:.*}"]
	require.Len(t, filesRoute.Parameters, 1)
	assert.Equal(t, "path", filesRoute.Parameters[0].Name)

	usersRoute := routeMap["GET /users/{id:[0-9]+}"]
	require.Len(t, usersRoute.Parameters, 1)
	assert.Equal(t, "id", usersRoute.Parameters[0].Name)

	// Verify multiple path params
	commentRoute := routeMap["GET /articles/{articleSlug}/comments/{commentId}"]
	require.Len(t, commentRoute.Parameters, 2)
	paramNames := []string{commentRoute.Parameters[0].Name, commentRoute.Parameters[1].Name}
	assert.Contains(t, paramNames, "articleSlug")
	assert.Contains(t, paramNames, "commentId")
}
