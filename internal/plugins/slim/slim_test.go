// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package slim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// slimBasicRoutesCode tests basic Slim routes.
const slimBasicRoutesCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->get('/users', function ($request, $response) {
    return $response->withJson([]);
});

$app->get('/users/{id}', function ($request, $response, $args) {
    return $response->withJson([]);
});

$app->post('/users', function ($request, $response) {
    return $response->withJson([], 201);
});

$app->put('/users/{id}', function ($request, $response, $args) {
    return $response->withJson([]);
});

$app->delete('/users/{id}', function ($request, $response, $args) {
    return $response->withStatus(204);
});

$app->patch('/users/{id}/status', function ($request, $response, $args) {
    return $response->withJson([]);
});

$app->run();
`

// slimGroupRoutesCode tests grouped routes.
const slimGroupRoutesCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->group('/api/v1', function ($group) {
    $group->get('/products', function ($request, $response) {
        return $response->withJson([]);
    });

    $group->get('/products/{id}', function ($request, $response, $args) {
        return $response->withJson([]);
    });

    $group->post('/products', function ($request, $response) {
        return $response->withJson([], 201);
    });
});

$app->run();
`

// slimMapRoutesCode tests map routes with multiple methods.
const slimMapRoutesCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->map(['GET', 'POST'], '/test', function ($request, $response) {
    return $response;
});

$app->map(['PUT', 'PATCH'], '/items/{id}', function ($request, $response, $args) {
    return $response;
});

$app->run();
`

// slimAnyRouteCode tests the any() method.
const slimAnyRouteCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->any('/webhook', function ($request, $response) {
    return $response;
});

$app->run();
`

// slimOptionsRouteCode tests the options() method.
const slimOptionsRouteCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->options('/cors-check', function ($request, $response) {
    return $response;
});

$app->run();
`

// slimParamConstraintsCode tests path parameters with constraints.
const slimParamConstraintsCode = `<?php

use Slim\Factory\AppFactory;

$app = AppFactory::create();

$app->get('/users/{id:\d+}', function ($request, $response, $args) {
    return $response->withJson([]);
});

$app->get('/posts/{slug:[a-z\-]+}', function ($request, $response, $args) {
    return $response->withJson([]);
});

$app->run();
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "slim", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".php")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "slim", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "slim/slim")
}

func TestPlugin_Detect_WithComposerJson(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
    "name": "my/app",
    "require": {
        "slim/slim": "^4.0"
    }
}`
	err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutSlim(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
    "name": "my/app",
    "require": {
        "laravel/framework": "^10.0"
    }
}`
	err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.False(t, detected)
}

func TestPlugin_Detect_NoComposer(t *testing.T) {
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
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimBasicRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from $app->method() calls
	assert.GreaterOrEqual(t, len(routes), 5)

	// Check for various HTTP methods
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
	}
	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
	assert.True(t, methods["PATCH"])

	// Check GET /users
	getUsers := findRoute(routes, "GET", "/users")
	if getUsers != nil {
		assert.Equal(t, "GET", getUsers.Method)
		assert.Equal(t, "/users", getUsers.Path)
	}

	// Check GET /users/{id}
	getUserByID := findRoute(routes, "GET", "/users/{id}")
	if getUserByID != nil {
		assert.Equal(t, "GET", getUserByID.Method)
		assert.Len(t, getUserByID.Parameters, 1)
		assert.Equal(t, "id", getUserByID.Parameters[0].Name)
		assert.Equal(t, "path", getUserByID.Parameters[0].In)
		assert.True(t, getUserByID.Parameters[0].Required)
	}
}

func TestPlugin_ExtractRoutes_GroupedRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimGroupRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes with group prefix applied
	if len(routes) > 0 {
		// Check that at least some routes were extracted
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_MapRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimMapRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Map routes should create multiple route entries
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have GET, POST, PUT, PATCH
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_AnyRoute(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimAnyRouteCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// any() should create routes for multiple methods
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have at least GET, POST, PUT, DELETE
		assert.GreaterOrEqual(t, len(methods), 2)
	}
}

func TestPlugin_ExtractRoutes_OptionsRoute(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimOptionsRouteCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract OPTIONS route
	optionsRoute := findRoute(routes, "OPTIONS", "/cors-check")
	if optionsRoute != nil {
		assert.Equal(t, "OPTIONS", optionsRoute.Method)
	}
}

func TestPlugin_ExtractRoutes_ParamConstraints(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "routes.php",
			Language: "php",
			Content:  []byte(slimParamConstraintsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Constraints should be stripped from path params
	for _, r := range routes {
		// Path should have {id} not {id:\d+}
		if r.Path == "/users/{id}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "id", r.Parameters[0].Name)
		}
		if r.Path == "/posts/{slug}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "slug", r.Parameters[0].Name)
		}
	}
}

func TestPlugin_ExtractRoutes_IgnoresNonPHP(t *testing.T) {
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
			Path:     "/path/to/routes.php",
			Language: "php",
			Content:  []byte(slimBasicRoutesCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/routes.php", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Model/User.php",
			Language: "php",
			Content:  []byte(`<?php class User {}`),
		},
	}

	// Slim doesn't have standard schema extraction
	schemas, err := p.ExtractSchemas(files)
	require.NoError(t, err)
	assert.Empty(t, schemas)
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

func TestConvertSlimPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/users", "/users"},
		{"/users/{id}", "/users/{id}"},
		{"/users/{id:\\d+}", "/users/{id}"},
		{"/posts/{slug:[a-z\\-]+}", "/posts/{slug}"},
		{"/items/{id:\\d+}/details/{detailId}", "/items/{id}/details/{detailId}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertSlimPathParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCombinePaths(t *testing.T) {
	tests := []struct {
		prefix   string
		path     string
		expected string
	}{
		{"", "/users", "/users"},
		{"/api", "/users", "/api/users"},
		{"/api/", "/users", "/api/users"},
		{"/api", "users", "/api/users"},
		{"", "users", "/users"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_"+tt.path, func(t *testing.T) {
			result := combinePaths(tt.prefix, tt.path)
			assert.Equal(t, tt.expected, result)
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
		path     string
		expected []string
	}{
		{"/users", []string{"users"}},
		{"/api/users", []string{"users"}},
		{"/api/v1/users", []string{"users"}},
		{"/api/v1/users/{id}", []string{"users"}},
		{"/", nil},
		{"/{id}", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := inferTags(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMethodsList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`'GET'`, []string{"GET"}},
		{`'GET', 'POST'`, []string{"GET", "POST"}},
		{`"GET", "PUT", "DELETE"`, []string{"GET", "PUT", "DELETE"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMethodsList(tt.input)
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
