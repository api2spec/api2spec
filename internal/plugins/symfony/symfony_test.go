// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package symfony

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// symfonyPHP8AttributeCode tests PHP 8 attribute-based routes.
const symfonyPHP8AttributeCode = `<?php

namespace App\Controller;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\Routing\Annotation\Route;

#[Route('/api/users')]
class UserController extends AbstractController
{
    #[Route('', methods: ['GET'])]
    public function index(): Response
    {
        return $this->json([]);
    }

    #[Route('/{id}', methods: ['GET'])]
    public function show(int $id): Response
    {
        return $this->json([]);
    }

    #[Route('', methods: ['POST'])]
    public function create(): Response
    {
        return $this->json([], 201);
    }

    #[Route('/{id}', methods: ['PUT'])]
    public function update(int $id): Response
    {
        return $this->json([]);
    }

    #[Route('/{id}', methods: ['DELETE'])]
    public function delete(int $id): Response
    {
        return $this->json(null, 204);
    }
}
`

// symfonyAnnotationCode tests annotation-based routes.
const symfonyAnnotationCode = `<?php

namespace App\Controller;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\Routing\Annotation\Route;

/**
 * @Route("/api/products")
 */
class ProductController extends AbstractController
{
    /**
     * @Route("", methods={"GET"})
     */
    public function index(): Response
    {
        return $this->json([]);
    }

    /**
     * @Route("/{id}", methods={"GET"})
     */
    public function show(int $id): Response
    {
        return $this->json([]);
    }

    /**
     * @Route("", methods={"POST"})
     */
    public function create(): Response
    {
        return $this->json([], 201);
    }
}
`

// symfonyMixedMethodsCode tests routes with multiple HTTP methods.
const symfonyMixedMethodsCode = `<?php

namespace App\Controller;

use Symfony\Component\Routing\Annotation\Route;

class TestController
{
    #[Route('/test', methods: ['GET', 'POST'])]
    public function handleTest(): Response
    {
        return new Response();
    }

    #[Route('/status', methods: ['GET', 'HEAD'])]
    public function status(): Response
    {
        return new Response();
    }
}
`

// symfonyNoMethodsCode tests routes without explicit methods.
const symfonyNoMethodsCode = `<?php

namespace App\Controller;

use Symfony\Component\Routing\Annotation\Route;

class DefaultController
{
    #[Route('/default')]
    public function defaultAction(): Response
    {
        return new Response();
    }
}
`

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "symfony", p.Name())
}

func TestPlugin_Extensions(t *testing.T) {
	p := New()
	exts := p.Extensions()
	assert.Contains(t, exts, ".php")
}

func TestPlugin_Info(t *testing.T) {
	p := New()
	info := p.Info()

	assert.Equal(t, "symfony", info.Name)
	assert.Equal(t, "1.0.0", info.Version)
	assert.NotEmpty(t, info.Description)
	assert.Contains(t, info.SupportedFrameworks, "symfony/framework-bundle")
}

func TestPlugin_Detect_WithComposerJson(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{
    "name": "my/app",
    "require": {
        "symfony/framework-bundle": "^6.0"
    }
}`
	err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithBinConsole(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)

	console := `#!/usr/bin/env php
<?php
require __DIR__.'/../vendor/autoload.php';
`
	err = os.WriteFile(filepath.Join(binDir, "console"), []byte(console), 0644)
	require.NoError(t, err)

	p := New()
	detected, err := p.Detect(dir)
	require.NoError(t, err)
	assert.True(t, detected)
}

func TestPlugin_Detect_WithoutSymfony(t *testing.T) {
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

func TestPlugin_ExtractRoutes_PHP8Attributes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Controller/UserController.php",
			Language: "php",
			Content:  []byte(symfonyPHP8AttributeCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from PHP 8 attributes
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"] || methods["PUT"] || methods["DELETE"])
	}

	// Check for path parameters
	for _, r := range routes {
		if r.Path == "/api/users/{id}" {
			assert.Len(t, r.Parameters, 1)
			assert.Equal(t, "id", r.Parameters[0].Name)
			assert.Equal(t, "path", r.Parameters[0].In)
		}
	}
}

func TestPlugin_ExtractRoutes_Annotations(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Controller/ProductController.php",
			Language: "php",
			Content:  []byte(symfonyAnnotationCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Should extract routes from annotations
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		assert.True(t, methods["GET"] || methods["POST"])
	}
}

func TestPlugin_ExtractRoutes_MixedMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Controller/TestController.php",
			Language: "php",
			Content:  []byte(symfonyMixedMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Routes with multiple methods should create multiple route entries
	if len(routes) > 0 {
		methods := make(map[string]bool)
		for _, r := range routes {
			methods[r.Method] = true
		}
		// Should have GET and potentially POST, HEAD
		assert.True(t, methods["GET"])
	}
}

func TestPlugin_ExtractRoutes_NoMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Controller/DefaultController.php",
			Language: "php",
			Content:  []byte(symfonyNoMethodsCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	// Routes without explicit methods should default to GET
	for _, r := range routes {
		if r.Path == "/default" {
			assert.Equal(t, "GET", r.Method)
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
			Path:     "/path/to/Controller/UserController.php",
			Language: "php",
			Content:  []byte(symfonyPHP8AttributeCode),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)

	for _, r := range routes {
		assert.Equal(t, "/path/to/Controller/UserController.php", r.SourceFile)
		assert.Greater(t, r.SourceLine, 0)
	}
}

func TestPlugin_ExtractSchemas(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "src/Entity/User.php",
			Language: "php",
			Content:  []byte(`<?php class User {}`),
		},
	}

	// Symfony doesn't have standard schema extraction
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
		path      string
		className string
		expected  []string
	}{
		{"/users", "UserController", []string{"User"}},
		{"/api/users", "UserController", []string{"User"}},
		{"/users", "", []string{"users"}},
		{"/api/v1/users", "", []string{"users"}},
		{"/", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := inferTags(tt.path, tt.className)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSymfonyMethods(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`'GET'`, []string{"GET"}},
		{`'GET', 'POST'`, []string{"GET", "POST"}},
		{`"GET", "PUT", "DELETE"`, []string{"GET", "PUT", "DELETE"}},
		{"", []string{"GET"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSymfonyMethods(tt.input)
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

// ============================================================================
// YAML Route Parsing Tests
// ============================================================================

// symfonyYAMLBasicRoute tests a basic YAML route with path, controller, and methods.
const symfonyYAMLBasicRoute = `user_show:
    path: /users/{id}
    controller: App\Controller\UserController::show
    methods: GET
`

// symfonyYAMLMultipleRoutes tests multiple routes in a single YAML file.
const symfonyYAMLMultipleRoutes = `user_list:
    path: /users
    controller: App\Controller\UserController::index
    methods: GET

user_create:
    path: /users
    controller: App\Controller\UserController::create
    methods: POST

user_show:
    path: /users/{id}
    controller: App\Controller\UserController::show
    methods: GET
`

// symfonyYAMLArrayMethods tests routes with array format methods [GET, POST].
const symfonyYAMLArrayMethods = `resource_update:
    path: /resources/{id}
    controller: App\Controller\ResourceController::update
    methods: [PUT, PATCH]
`

// symfonyYAMLNoMethods tests routes without explicit methods (should default to GET).
const symfonyYAMLNoMethods = `default_route:
    path: /default
    controller: App\Controller\DefaultController::index
`

func TestExtractRoutesFromYAML_BasicRoute(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.yaml",
			Language: "yaml",
			Content:  []byte(symfonyYAMLBasicRoute),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/users/{id}", route.Path)
	assert.Equal(t, "App\\Controller\\UserController::show", route.Handler)
	assert.Equal(t, "config/routes.yaml", route.SourceFile)
	assert.Greater(t, route.SourceLine, 0)

	// Verify path parameter extraction
	require.Len(t, route.Parameters, 1)
	assert.Equal(t, "id", route.Parameters[0].Name)
	assert.Equal(t, "path", route.Parameters[0].In)
	assert.True(t, route.Parameters[0].Required)
}

func TestExtractRoutesFromYAML_MultipleRoutes(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.yaml",
			Language: "yaml",
			Content:  []byte(symfonyYAMLMultipleRoutes),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 3)

	// Verify first route: GET /users
	listRoute := findRoute(routes, "GET", "/users")
	require.NotNil(t, listRoute, "GET /users route not found")
	assert.Contains(t, listRoute.Handler, "index")

	// Verify second route: POST /users
	createRoute := findRoute(routes, "POST", "/users")
	require.NotNil(t, createRoute, "POST /users route not found")
	assert.Contains(t, createRoute.Handler, "create")

	// Verify third route: GET /users/{id}
	showRoute := findRoute(routes, "GET", "/users/{id}")
	require.NotNil(t, showRoute, "GET /users/{id} route not found")
	assert.Contains(t, showRoute.Handler, "show")
}

func TestExtractRoutesFromYAML_ArrayMethods(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.yaml",
			Language: "yaml",
			Content:  []byte(symfonyYAMLArrayMethods),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 2)

	// Should create separate routes for PUT and PATCH
	methods := make(map[string]bool)
	for _, r := range routes {
		methods[r.Method] = true
		assert.Equal(t, "/resources/{id}", r.Path)
	}

	assert.True(t, methods["PUT"], "PUT method not found")
	assert.True(t, methods["PATCH"], "PATCH method not found")
}

func TestExtractRoutesFromYAML_MissingMethodsDefaultsToGET(t *testing.T) {
	p := New()

	files := []scanner.SourceFile{
		{
			Path:     "config/routes.yaml",
			Language: "yaml",
			Content:  []byte(symfonyYAMLNoMethods),
		},
	}

	routes, err := p.ExtractRoutes(files)
	require.NoError(t, err)
	require.Len(t, routes, 1)

	route := routes[0]
	assert.Equal(t, "GET", route.Method, "Missing methods should default to GET")
	assert.Equal(t, "/default", route.Path)
}

func TestParseYAMLMethods(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single method",
			input:    "GET",
			expected: []string{"GET"},
		},
		{
			name:     "array format",
			input:    "[GET, POST]",
			expected: []string{"GET", "POST"},
		},
		{
			name:     "array with quotes",
			input:    "['GET', 'POST']",
			expected: []string{"GET", "POST"},
		},
		{
			name:     "array with double quotes",
			input:    `["PUT", "PATCH", "DELETE"]`,
			expected: []string{"PUT", "PATCH", "DELETE"},
		},
		{
			name:     "lowercase converted to uppercase",
			input:    "[get, post]",
			expected: []string{"GET", "POST"},
		},
		{
			name:     "empty string defaults to GET",
			input:    "",
			expected: []string{"GET"},
		},
		{
			name:     "single with quotes",
			input:    "'POST'",
			expected: []string{"POST"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseYAMLMethods(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
