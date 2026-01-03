// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package echo provides a plugin for extracting routes from Echo framework applications.
package echo

import (
	"bufio"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/internal/schema"
	"github.com/api2spec/api2spec/pkg/types"
)

// echoImportPaths are the known import paths for Echo framework.
var echoImportPaths = []string{
	"github.com/labstack/echo",
	"github.com/labstack/echo/v4",
	"github.com/labstack/echo/v5",
}

// httpMethods are the HTTP methods supported by Echo.
var httpMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
	"CONNECT": true,
	"Any":     true,
}

// Plugin implements the FrameworkPlugin interface for Echo framework.
type Plugin struct {
	goParser        *parser.GoParser
	schemaExtractor *schema.GoSchemaExtractor
}

// New creates a new Echo plugin instance.
func New() *Plugin {
	return &Plugin{
		goParser:        parser.NewGoParser(),
		schemaExtractor: schema.NewGoSchemaExtractor(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "echo"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".go"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "echo",
		Version:     "1.0.0",
		Description: "Extracts routes from Echo framework applications",
		SupportedFrameworks: []string{
			"github.com/labstack/echo",
			"github.com/labstack/echo/v4",
			"github.com/labstack/echo/v5",
		},
	}
}

// Detect checks if Echo is used in the project by looking at go.mod.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")

	file, err := os.Open(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanr := bufio.NewScanner(file)
	for scanr.Scan() {
		line := scanr.Text()
		for _, importPath := range echoImportPaths {
			if strings.Contains(line, importPath) {
				return true, nil
			}
		}
	}

	if err := scanr.Err(); err != nil {
		return false, fmt.Errorf("failed to read go.mod: %w", err)
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Echo route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "go" {
			continue
		}

		fileRoutes, err := p.extractRoutesFromFile(file)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single Go file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.goParser.ParseSource(file.Path, string(file.Content))
	if err != nil {
		return nil, err
	}

	// Check if this file imports Echo
	if !p.hasEchoImport(pf) {
		return nil, nil
	}

	// Find route definitions
	var routes []types.Route

	// Track route groups and prefixes
	ctx := &extractionContext{
		file:        pf,
		parser:      p.goParser,
		prefixStack: []string{},
	}

	ast.Inspect(pf.AST, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		// Look for function declarations that might set up routes
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			funcRoutes := p.extractRoutesFromFunc(funcDecl, ctx)
			routes = append(routes, funcRoutes...)
		}

		return true
	})

	// Set source file for all routes
	for i := range routes {
		routes[i].SourceFile = file.Path
	}

	return routes, nil
}

// extractionContext tracks context during route extraction.
type extractionContext struct {
	file        *parser.ParsedFile
	parser      *parser.GoParser
	prefixStack []string
}

// currentPrefix returns the current route prefix.
func (ctx *extractionContext) currentPrefix() string {
	return strings.Join(ctx.prefixStack, "")
}

// extractRoutesFromFunc extracts routes from a function body.
func (p *Plugin) extractRoutesFromFunc(funcDecl *ast.FuncDecl, ctx *extractionContext) []types.Route {
	var routes []types.Route

	if funcDecl.Body == nil {
		return routes
	}

	// Walk the function body looking for method calls
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := selExpr.Sel.Name

		// Check for Group() calls - e.g., e.Group("/api")
		if methodName == "Group" {
			nestedRoutes := p.handleRouteGroup(callExpr, ctx)
			routes = append(routes, nestedRoutes...)
			return false // Don't recurse into this, we handled it
		}

		// Check for Add() calls - e.g., e.Add("GET", "/path", handler)
		if methodName == "Add" {
			route := p.extractRouteFromAdd(callExpr, ctx)
			if route != nil {
				routes = append(routes, *route)
			}
			return true
		}

		// Check for HTTP method calls (GET, POST, etc.)
		if httpMethods[methodName] {
			route := p.extractRouteFromCall(callExpr, methodName, ctx)
			if route != nil {
				routes = append(routes, *route)
			}
		}

		return true
	})

	return routes
}

// handleRouteGroup handles e.Group("/prefix") calls and extracts nested routes.
func (p *Plugin) handleRouteGroup(callExpr *ast.CallExpr, ctx *extractionContext) []types.Route {
	var routes []types.Route

	// Extract the prefix from Group("/prefix")
	if len(callExpr.Args) < 1 {
		return routes
	}

	prefix, ok := parser.ExtractStringLiteral(callExpr.Args[0])
	if !ok {
		return routes
	}

	// Check for chained calls on the same expression
	// e.g., e.Group("/api").GET("/users", handler)
	parent := findParentCall(ctx.file.AST, callExpr)
	if parent != nil {
		selExpr, ok := parent.Fun.(*ast.SelectorExpr)
		if ok && httpMethods[selExpr.Sel.Name] {
			// Push prefix onto stack
			ctx.prefixStack = append(ctx.prefixStack, prefix)

			// Extract the route from the chained call
			route := p.extractRouteFromCall(parent, selExpr.Sel.Name, ctx)
			if route != nil {
				routes = append(routes, *route)
			}

			// Pop prefix from stack
			ctx.prefixStack = ctx.prefixStack[:len(ctx.prefixStack)-1]
		}
	}

	return routes
}

// findParentCall finds the parent CallExpr that contains this expression.
func findParentCall(file *ast.File, target *ast.CallExpr) *ast.CallExpr {
	var result *ast.CallExpr

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if this call's receiver is our target
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if selExpr.X == target {
			result = callExpr
			return false
		}

		return true
	})

	return result
}

// extractRouteFromCall extracts a Route from an HTTP method call.
// e.g., e.GET("/users/:id", handler)
func (p *Plugin) extractRouteFromCall(callExpr *ast.CallExpr, method string, ctx *extractionContext) *types.Route {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// First argument is the path
	path, ok := parser.ExtractStringLiteral(callExpr.Args[0])
	if !ok {
		return nil
	}

	// Combine with prefix
	fullPath := ctx.currentPrefix() + path

	// Normalize path
	fullPath = normalizePath(fullPath)

	// Convert Echo :param syntax to OpenAPI {param} syntax
	fullPath = convertEchoPathParams(fullPath)

	// Extract handler name if present
	var handlerName string
	if len(callExpr.Args) >= 2 {
		handlerName = p.extractHandlerName(callExpr.Args[len(callExpr.Args)-1])
	}

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Get HTTP method (handle "Any" case)
	httpMethod := strings.ToUpper(method)
	if method == "Any" {
		httpMethod = "ANY"
	}

	// Generate operation ID
	operationID := generateOperationID(httpMethod, fullPath, handlerName)

	// Infer tags from path
	tags := inferTags(fullPath)

	route := &types.Route{
		Method:      httpMethod,
		Path:        fullPath,
		Handler:     handlerName,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceLine:  ctx.file.FileSet.Position(callExpr.Pos()).Line,
	}

	return route
}

// extractRouteFromAdd extracts a Route from e.Add("GET", "/path", handler).
func (p *Plugin) extractRouteFromAdd(callExpr *ast.CallExpr, ctx *extractionContext) *types.Route {
	if len(callExpr.Args) < 2 {
		return nil
	}

	// First argument is the HTTP method
	method, ok := parser.ExtractStringLiteral(callExpr.Args[0])
	if !ok {
		return nil
	}

	// Second argument is the path
	path, ok := parser.ExtractStringLiteral(callExpr.Args[1])
	if !ok {
		return nil
	}

	// Combine with prefix
	fullPath := ctx.currentPrefix() + path

	// Normalize path
	fullPath = normalizePath(fullPath)

	// Convert Echo :param syntax to OpenAPI {param} syntax
	fullPath = convertEchoPathParams(fullPath)

	// Extract handler name if present
	var handlerName string
	if len(callExpr.Args) >= 3 {
		handlerName = p.extractHandlerName(callExpr.Args[len(callExpr.Args)-1])
	}

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Generate operation ID
	operationID := generateOperationID(method, fullPath, handlerName)

	// Infer tags from path
	tags := inferTags(fullPath)

	route := &types.Route{
		Method:      strings.ToUpper(method),
		Path:        fullPath,
		Handler:     handlerName,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceLine:  ctx.file.FileSet.Position(callExpr.Pos()).Line,
	}

	return route
}

// extractHandlerName extracts the handler function name from an expression.
func (p *Plugin) extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		// e.g., handlers.GetUser
		if ident, ok := e.X.(*ast.Ident); ok {
			return ident.Name + "." + e.Sel.Name
		}
		return e.Sel.Name
	case *ast.FuncLit:
		return "<anonymous>"
	case *ast.CallExpr:
		// e.g., middleware(handler)
		return "<wrapped>"
	default:
		return ""
	}
}

// hasEchoImport checks if the file imports Echo.
func (p *Plugin) hasEchoImport(pf *parser.ParsedFile) bool {
	for _, importPath := range echoImportPaths {
		if p.goParser.HasImport(pf, importPath) {
			return true
		}
	}
	return false
}

// ExtractSchemas extracts schema definitions from Go structs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	for _, file := range files {
		if file.Language != "go" {
			continue
		}

		pf, err := p.goParser.ParseSource(file.Path, string(file.Content))
		if err != nil {
			continue
		}

		structs := p.goParser.ExtractStructs(pf)
		for _, def := range structs {
			p.schemaExtractor.ExtractFromStruct(def)
		}
	}

	return p.schemaExtractor.Registry().ToSlice(), nil
}

// --- Helper Functions ---

// normalizePath normalizes a route path.
func normalizePath(path string) string {
	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove double slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Remove trailing slash (except for root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

// convertEchoPathParams converts Echo :param syntax to OpenAPI {param} syntax.
func convertEchoPathParams(path string) string {
	// Convert :param to {param}
	path = regexp.MustCompile(`:(\w+)`).ReplaceAllString(path, "{$1}")

	// Convert * (catch-all in Echo) to {path} - Echo uses trailing * for catch-all
	// Note: Echo's catch-all is just *, we'll convert it to a generic {path} param
	path = strings.ReplaceAll(path, "/*", "/{path}")

	return path
}

// pathParamRegex matches OpenAPI path parameters like {id} or {userId}.
var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// extractPathParams extracts path parameters from a route path.
func extractPathParams(path string) []types.Parameter {
	var params []types.Parameter

	matches := pathParamRegex.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		paramName := match[1]

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "path",
			Required: true,
			Schema: &types.Schema{
				Type: "string",
			},
		})
	}

	return params
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" && handler != "<anonymous>" && handler != "<wrapped>" {
		// Remove package prefix and clean up
		parts := strings.Split(handler, ".")
		name := parts[len(parts)-1]
		return strings.ToLower(method) + name
	}

	// Generate from path
	// Remove parameter syntax and convert to camelCase
	path = pathParamRegex.ReplaceAllString(path, "By${1}")
	path = strings.ReplaceAll(path, "/", " ")
	path = strings.TrimSpace(path)

	words := strings.Fields(path)
	if len(words) == 0 {
		return strings.ToLower(method)
	}

	// Build camelCase operation ID
	var sb strings.Builder
	sb.WriteString(strings.ToLower(method))

	titleCaser := cases.Title(language.English)
	for _, word := range words {
		word = titleCaser.String(strings.ToLower(word))
		sb.WriteString(word)
	}

	return sb.String()
}

// inferTags infers tags from the route path.
func inferTags(path string) []string {
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		return nil
	}

	// Skip common prefixes like "api", "v1", etc.
	skipPrefixes := map[string]bool{
		"api": true,
		"v1":  true,
		"v2":  true,
		"v3":  true,
	}

	// Find the first meaningful segment
	var tagPart string
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Skip version/api prefixes
		if skipPrefixes[part] {
			continue
		}
		// Skip if it's a parameter
		if strings.HasPrefix(part, "{") {
			continue
		}
		tagPart = part
		break
	}

	if tagPart == "" {
		return nil
	}

	return []string{tagPart}
}

// Register registers the Echo plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
