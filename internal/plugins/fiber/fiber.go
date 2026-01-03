// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package fiber provides a plugin for extracting routes from Fiber framework applications.
package fiber

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

// fiberImportPaths are the known import paths for Fiber framework.
var fiberImportPaths = []string{
	"github.com/gofiber/fiber",
	"github.com/gofiber/fiber/v2",
	"github.com/gofiber/fiber/v3",
}

// httpMethods are the HTTP methods supported by Fiber.
var httpMethods = map[string]bool{
	"Get":     true,
	"Post":    true,
	"Put":     true,
	"Delete":  true,
	"Patch":   true,
	"Head":    true,
	"Options": true,
	"Trace":   true,
	"Connect": true,
	"All":     true,
}

// Plugin implements the FrameworkPlugin interface for Fiber framework.
type Plugin struct {
	goParser        *parser.GoParser
	schemaExtractor *schema.GoSchemaExtractor
}

// New creates a new Fiber plugin instance.
func New() *Plugin {
	return &Plugin{
		goParser:        parser.NewGoParser(),
		schemaExtractor: schema.NewGoSchemaExtractor(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "fiber"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".go"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "fiber",
		Version:     "1.0.0",
		Description: "Extracts routes from Fiber framework applications",
		SupportedFrameworks: []string{
			"github.com/gofiber/fiber/v2",
			"github.com/gofiber/fiber/v3",
		},
	}
}

// Detect checks if Fiber is used in the project by looking at go.mod.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")

	file, err := os.Open(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer func() { _ = file.Close() }()

	scanr := bufio.NewScanner(file)
	for scanr.Scan() {
		line := scanr.Text()
		for _, importPath := range fiberImportPaths {
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

// ExtractRoutes parses source files and extracts Fiber route definitions.
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

	// Check if this file imports Fiber
	if !p.hasFiberImport(pf) {
		return nil, nil
	}

	// Find route definitions
	var routes []types.Route

	// Track route groups and prefixes
	ctx := &extractionContext{
		file:        pf,
		parser:      p.goParser,
		prefixStack: []string{},
		groupVars:   make(map[string]string), // Maps variable name to prefix
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
	groupVars   map[string]string // Maps variable name to its accumulated prefix
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

	// First pass: find all group assignments to build prefix map
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		assignStmt, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		// Look for: api := app.Group("/api")
		if len(assignStmt.Lhs) != 1 || len(assignStmt.Rhs) != 1 {
			return true
		}

		callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if selExpr.Sel.Name != "Group" {
			return true
		}

		// Get the variable name being assigned
		ident, ok := assignStmt.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}

		// Get the prefix from the Group call
		if len(callExpr.Args) < 1 {
			return true
		}

		prefix, ok := parser.ExtractStringLiteral(callExpr.Args[0])
		if !ok {
			return true
		}

		// Get the object of the method call (e.g., "app" in app.Group)
		var parentPrefix string
		if parentIdent, ok := selExpr.X.(*ast.Ident); ok {
			parentPrefix = ctx.groupVars[parentIdent.Name]
		}

		// Combine parent prefix with this prefix
		fullPrefix := parentPrefix + prefix
		ctx.groupVars[ident.Name] = fullPrefix

		return true
	})

	// Second pass: extract routes
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

		// Check for Route() calls with inline function
		if methodName == "Route" {
			nestedRoutes := p.handleRouteCallback(callExpr, ctx)
			routes = append(routes, nestedRoutes...)
			return false // Don't recurse into Route callback, we handled it
		}

		// Check for HTTP method calls
		if httpMethods[methodName] {
			route := p.extractRouteFromCall(callExpr, methodName, selExpr.X, ctx)
			if route != nil {
				routes = append(routes, *route)
			}
		}

		return true
	})

	return routes
}

// handleRouteCallback handles app.Route("/path", func(router fiber.Router) { ... }) calls.
func (p *Plugin) handleRouteCallback(callExpr *ast.CallExpr, ctx *extractionContext) []types.Route {
	var routes []types.Route

	if len(callExpr.Args) < 2 {
		return routes
	}

	// First argument is the prefix
	prefix, ok := parser.ExtractStringLiteral(callExpr.Args[0])
	if !ok {
		return routes
	}

	// Get the object's prefix
	var objectPrefix string
	if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selExpr.X.(*ast.Ident); ok {
			objectPrefix = ctx.groupVars[ident.Name]
		}
	}

	fullPrefix := objectPrefix + prefix

	// Second argument should be the function literal
	funcLit, ok := callExpr.Args[1].(*ast.FuncLit)
	if !ok {
		return routes
	}

	// Get the router parameter name
	var routerParamName string
	if funcLit.Type.Params != nil && len(funcLit.Type.Params.List) > 0 {
		if len(funcLit.Type.Params.List[0].Names) > 0 {
			routerParamName = funcLit.Type.Params.List[0].Names[0].Name
		}
	}

	// Push prefix onto stack
	ctx.prefixStack = append(ctx.prefixStack, fullPrefix)

	// Save the router variable's prefix
	if routerParamName != "" {
		ctx.groupVars[routerParamName] = fullPrefix
	}

	// Extract routes from the callback
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		nestedCall, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := nestedCall.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		nestedMethod := selExpr.Sel.Name

		if httpMethods[nestedMethod] {
			route := p.extractRouteFromCall(nestedCall, nestedMethod, selExpr.X, ctx)
			if route != nil {
				routes = append(routes, *route)
			}
		}

		return true
	})

	// Pop prefix from stack
	ctx.prefixStack = ctx.prefixStack[:len(ctx.prefixStack)-1]

	return routes
}

// extractRouteFromCall extracts a Route from an HTTP method call.
func (p *Plugin) extractRouteFromCall(callExpr *ast.CallExpr, method string, receiver ast.Expr, ctx *extractionContext) *types.Route {
	if len(callExpr.Args) < 1 {
		return nil
	}

	// First argument is the path
	path, ok := parser.ExtractStringLiteral(callExpr.Args[0])
	if !ok {
		return nil
	}

	// Get prefix from the receiver variable
	var prefix string
	if ident, ok := receiver.(*ast.Ident); ok {
		prefix = ctx.groupVars[ident.Name]
	}

	// Combine with current context prefix
	if prefix == "" {
		prefix = ctx.currentPrefix()
	}

	fullPath := prefix + path

	// Normalize path
	fullPath = normalizePath(fullPath)

	// Convert Fiber :param syntax to OpenAPI {param} syntax
	fullPath = convertFiberPathParams(fullPath)

	// Extract handler name if present
	var handlerName string
	if len(callExpr.Args) >= 2 {
		handlerName = p.extractHandlerName(callExpr.Args[len(callExpr.Args)-1])
	}

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Get HTTP method
	httpMethod := strings.ToUpper(method)
	if method == "All" {
		httpMethod = "ALL"
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

// hasFiberImport checks if the file imports Fiber.
func (p *Plugin) hasFiberImport(pf *parser.ParsedFile) bool {
	for _, importPath := range fiberImportPaths {
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

// colonParamRegex matches Fiber path parameters like :param.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// wildcardRegex matches Fiber wildcard like *.
var wildcardRegex = regexp.MustCompile(`\*`)

// convertFiberPathParams converts Fiber :param and * syntax to OpenAPI {param} syntax.
func convertFiberPathParams(path string) string {
	// Convert :param to {param}
	path = colonParamRegex.ReplaceAllString(path, "{$1}")

	// Convert * to {path} (wildcard)
	path = wildcardRegex.ReplaceAllString(path, "{path}")

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

// Register registers the Fiber plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
