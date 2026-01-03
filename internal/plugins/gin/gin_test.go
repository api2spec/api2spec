// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package gin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
)

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "gin", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	assert.Equal(t, []string{".go"}, p.Extensions())
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "gin", info.Name)
	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "github.com/gin-gonic/gin")
}

func TestPlugin_Detect(t *testing.T) {
	tests := []struct {
		name         string
		goModContent string
		wantDetect   bool
	}{
		{
			name: "gin detected",
			goModContent: `module example.com/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
)
`,
			wantDetect: true,
		},
		{
			name: "gin with specific version",
			goModContent: `module example.com/myapp

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/some/other v1.0.0
)
`,
			wantDetect: true,
		},
		{
			name: "no gin - has chi",
			goModContent: `module example.com/myapp

go 1.21

require (
	github.com/go-chi/chi/v5 v5.0.10
)
`,
			wantDetect: false,
		},
		{
			name: "no gin - has echo",
			goModContent: `module example.com/myapp

go 1.21

require (
	github.com/labstack/echo/v4 v4.11.0
)
`,
			wantDetect: false,
		},
		{
			name: "empty go.mod",
			goModContent: `module example.com/myapp

go 1.21
`,
			wantDetect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			goModPath := filepath.Join(tmpDir, "go.mod")
			err := os.WriteFile(goModPath, []byte(tt.goModContent), 0644)
			require.NoError(t, err)

			p := New()
			detected, err := p.Detect(tmpDir)

			require.NoError(t, err)
			assert.Equal(t, tt.wantDetect, detected)
		})
	}
}

func TestPlugin_Detect_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	p := New()
	detected, err := p.Detect(tmpDir)

	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_ExtractRoutes_Basic(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/users", ListUsers)
	r.POST("/users", CreateUser)
	r.GET("/users/:id", GetUser)
	r.PUT("/users/:id", UpdateUser)
	r.DELETE("/users/:id", DeleteUser)
	r.PATCH("/users/:id/status", UpdateUserStatus)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 6)

	// Verify GET /users
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/users", routes[0].Path)
	assert.Equal(t, "ListUsers", routes[0].Handler)
	assert.Contains(t, routes[0].Tags, "users")

	// Verify POST /users
	assert.Equal(t, "POST", routes[1].Method)
	assert.Equal(t, "/users", routes[1].Path)
	assert.Equal(t, "CreateUser", routes[1].Handler)

	// Verify GET /users/{id} - note: :id converted to {id}
	assert.Equal(t, "GET", routes[2].Method)
	assert.Equal(t, "/users/{id}", routes[2].Path)
	require.Len(t, routes[2].Parameters, 1)
	assert.Equal(t, "id", routes[2].Parameters[0].Name)
	assert.Equal(t, "path", routes[2].Parameters[0].In)
	assert.True(t, routes[2].Parameters[0].Required)

	// Verify PUT /users/{id}
	assert.Equal(t, "PUT", routes[3].Method)
	assert.Equal(t, "/users/{id}", routes[3].Path)

	// Verify DELETE /users/{id}
	assert.Equal(t, "DELETE", routes[4].Method)
	assert.Equal(t, "/users/{id}", routes[4].Path)

	// Verify PATCH /users/{id}/status
	assert.Equal(t, "PATCH", routes[5].Method)
	assert.Equal(t, "/users/{id}/status", routes[5].Path)
	assert.Len(t, routes[5].Parameters, 1)
}

func TestPlugin_ExtractRoutes_Handle(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.Handle("GET", "/items", ListItems)
	r.Handle("POST", "/items", CreateItem)
	r.Handle("DELETE", "/items/:id", DeleteItem)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 3)

	// Verify Handle calls
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/items", routes[0].Path)
	assert.Equal(t, "ListItems", routes[0].Handler)

	assert.Equal(t, "POST", routes[1].Method)
	assert.Equal(t, "/items", routes[1].Path)

	assert.Equal(t, "DELETE", routes[2].Method)
	assert.Equal(t, "/items/{id}", routes[2].Path)
}

func TestPlugin_ExtractRoutes_MethodHandler(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

type API struct{}

func SetupRoutes(r *gin.Engine, api *API) {
	r.GET("/users", api.ListUsers)
	r.GET("/health", handlers.HealthCheck)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 2)

	assert.Equal(t, "api.ListUsers", routes[0].Handler)
	assert.Equal(t, "handlers.HealthCheck", routes[1].Handler)
}

func TestPlugin_ExtractRoutes_AnonymousHandler(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/inline", func(c *gin.Context) {
		c.String(200, "ok")
	})
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	assert.Equal(t, "<anonymous>", routes[0].Handler)
}

func TestPlugin_ExtractRoutes_NoGinImport(t *testing.T) {
	source := `package main

import "net/http"

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "handler.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_NonGoFiles(t *testing.T) {
	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "main.ts",
			Language: "typescript",
			Content:  []byte("export const foo = 1"),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	assert.Empty(t, routes)
}

func TestPlugin_ExtractRoutes_CatchAllParam(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/files/*filepath", ServeFiles)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	// Verify *filepath is converted to {filepath}
	assert.Equal(t, "/files/{filepath}", routes[0].Path)
	require.Len(t, routes[0].Parameters, 1)
	assert.Equal(t, "filepath", routes[0].Parameters[0].Name)
}

func TestPlugin_ExtractRoutes_MultipleParams(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/users/:userId/posts/:postId", GetUserPost)
	r.GET("/orgs/:orgId/teams/:teamId/members/:memberId", GetTeamMember)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 2)

	// Verify first route
	assert.Equal(t, "/users/{userId}/posts/{postId}", routes[0].Path)
	require.Len(t, routes[0].Parameters, 2)
	assert.Equal(t, "userId", routes[0].Parameters[0].Name)
	assert.Equal(t, "postId", routes[0].Parameters[1].Name)

	// Verify second route
	assert.Equal(t, "/orgs/{orgId}/teams/{teamId}/members/{memberId}", routes[1].Path)
	require.Len(t, routes[1].Parameters, 3)
}

func TestPlugin_ExtractRoutes_AnyMethod(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.Any("/webhook", HandleWebhook)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	assert.Equal(t, "ANY", routes[0].Method)
	assert.Equal(t, "/webhook", routes[0].Path)
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	source := `package models

import "time"

// User represents a user in the system.
type User struct {
	ID        string    ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\" validate:\"required\"`" + `
	Email     string    ` + "`json:\"email\" validate:\"required,email\"`" + `
	Age       *int      ` + "`json:\"age,omitempty\"`" + `
	Tags      []string  ` + "`json:\"tags\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}

type Address struct {
	Street string ` + "`json:\"street\"`" + `
	City   string ` + "`json:\"city\"`" + `
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "models.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)
	require.Len(t, schemas, 2)

	// Find User schema
	for _, s := range schemas {
		if s.Title == "User" {
			assert.Equal(t, "object", s.Type)
			assert.Contains(t, s.Required, "name")
			assert.Contains(t, s.Required, "email")
			assert.NotContains(t, s.Required, "age")

			// Check properties
			assert.NotNil(t, s.Properties["id"])
			assert.Equal(t, "string", s.Properties["id"].Type)

			assert.NotNil(t, s.Properties["email"])
			assert.Equal(t, "email", s.Properties["email"].Format)

			assert.NotNil(t, s.Properties["tags"])
			assert.Equal(t, "array", s.Properties["tags"].Type)

			assert.NotNil(t, s.Properties["created_at"])
			assert.Equal(t, "date-time", s.Properties["created_at"].Format)
			break
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"users", "/users"},
		{"/users/", "/users"},
		{"/users//posts", "/users/posts"},
		{"/", "/"},
		{"", "/"},
		{"//users", "/users"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertGinPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users/:id", "/users/{id}"},
		{"/users/:userId/posts/:postId", "/users/{userId}/posts/{postId}"},
		{"/files/*filepath", "/files/{filepath}"},
		{"/users", "/users"},
		{"/:param1/:param2/*catchall", "/{param1}/{param2}/{catchall}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertGinPathParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path       string
		paramNames []string
	}{
		{"/users", nil},
		{"/users/{id}", []string{"id"}},
		{"/users/{userId}/posts/{postId}", []string{"userId", "postId"}},
		{"/files/{filepath}", []string{"filepath"}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			params := extractPathParams(tt.path)

			if tt.paramNames == nil {
				assert.Empty(t, params)
				return
			}

			require.Len(t, params, len(tt.paramNames))
			for i, name := range tt.paramNames {
				assert.Equal(t, name, params[i].Name)
				assert.Equal(t, "path", params[i].In)
				assert.True(t, params[i].Required)
			}
		})
	}
}

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		method  string
		path    string
		handler string
		want    string
	}{
		{"GET", "/users", "ListUsers", "getListUsers"},
		{"POST", "/users", "CreateUser", "postCreateUser"},
		{"GET", "/users/{id}", "GetUser", "getGetUser"},
		{"DELETE", "/users/{id}", "", "deleteUsersByid"},
		{"GET", "/", "", "get"},
		{"GET", "/api/v1/users", "ListUsers", "getListUsers"},
		{"GET", "/users", "<anonymous>", "getUsers"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			result := generateOperationID(tt.method, tt.path, tt.handler)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestInferTags(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/users", []string{"users"}},
		{"/users/{id}", []string{"users"}},
		{"/api/users", []string{"users"}},
		{"/api/v1/users", []string{"users"}},
		{"/v1/orders", []string{"orders"}},
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

func TestPlugin_ExtractRoutes_AllHTTPMethods(t *testing.T) {
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/get", Handler)
	r.POST("/post", Handler)
	r.PUT("/put", Handler)
	r.DELETE("/delete", Handler)
	r.PATCH("/patch", Handler)
	r.HEAD("/head", Handler)
	r.OPTIONS("/options", Handler)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 7)

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
	source := `package main

import "github.com/gin-gonic/gin"

func SetupRoutes(r *gin.Engine) {
	r.GET("/users", ListUsers)
}
`

	p := New()
	files := []scanner.SourceFile{
		{
			Path:     "/path/to/routes.go",
			Language: "go",
			Content:  []byte(source),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	assert.Equal(t, "/path/to/routes.go", routes[0].SourceFile)
	assert.Greater(t, routes[0].SourceLine, 0)
}
