// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package nestjs provides a plugin for extracting routes from NestJS framework applications.
package nestjs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/internal/schema"
	"github.com/api2spec/api2spec/pkg/types"
)

// httpMethodDecorators maps NestJS decorator names to HTTP methods.
var httpMethodDecorators = map[string]string{
	"Get":     "GET",
	"Post":    "POST",
	"Put":     "PUT",
	"Delete":  "DELETE",
	"Patch":   "PATCH",
	"Head":    "HEAD",
	"Options": "OPTIONS",
	"All":     "ALL",
}

// Plugin implements the FrameworkPlugin interface for NestJS framework.
type Plugin struct {
	tsParser  *parser.TypeScriptParser
	zodParser *schema.ZodParser
}

// New creates a new NestJS plugin instance.
func New() *Plugin {
	tsParser := parser.NewTypeScriptParser()
	return &Plugin{
		tsParser:  tsParser,
		zodParser: schema.NewZodParser(tsParser),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "nestjs"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "nestjs",
		Version:     "1.0.0",
		Description: "Extracts routes from NestJS framework applications",
		SupportedFrameworks: []string{
			"@nestjs/core",
			"@nestjs/common",
		},
	}
}

// Detect checks if NestJS is used in the project by looking at package.json.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	packageJSONPath := filepath.Join(projectRoot, "package.json")

	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Check for @nestjs/core or @nestjs/common in dependencies
	nestjsDeps := []string{"@nestjs/core", "@nestjs/common"}
	for _, dep := range nestjsDeps {
		if _, ok := pkg.Dependencies[dep]; ok {
			return true, nil
		}
		if _, ok := pkg.DevDependencies[dep]; ok {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts NestJS route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "typescript" && file.Language != "javascript" {
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

// controllerInfo holds information about a NestJS controller.
type controllerInfo struct {
	name       string
	basePath   string
	version    string
	classNode  *sitter.Node
	sourceLine int
}

// extractRoutesFromFile extracts routes from a single TypeScript file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.tsParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports NestJS
	if !p.hasNestJSImport(pf.RootNode, file.Content) {
		return nil, nil
	}

	var routes []types.Route

	// Find controller classes and their routes
	controllers := p.findControllers(pf.RootNode, file.Content)

	for _, ctrl := range controllers {
		controllerRoutes := p.extractRoutesFromController(ctrl, file.Content)
		for i := range controllerRoutes {
			controllerRoutes[i].SourceFile = file.Path
		}
		routes = append(routes, controllerRoutes...)
	}

	return routes, nil
}

// hasNestJSImport checks if the file imports NestJS.
func (p *Plugin) hasNestJSImport(rootNode *sitter.Node, content []byte) bool {
	hasImport := false

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check for import statement
		if node.Type() == "import_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "string" {
					source := child.Content(content)
					source = strings.Trim(source, `"'`)
					if strings.HasPrefix(source, "@nestjs/") {
						hasImport = true
						return false
					}
				}
			}
		}
		return true
	})

	return hasImport
}

// findControllers finds all controller classes in the file.
func (p *Plugin) findControllers(rootNode *sitter.Node, content []byte) []*controllerInfo {
	var controllers []*controllerInfo

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Handle export statements wrapping class declarations with decorators
		// Structure: export_statement -> [decorator, export, class_declaration]
		if node.Type() == "export_statement" {
			var decorators []*sitter.Node
			var classDecl *sitter.Node

			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "decorator" {
					decorators = append(decorators, child)
				}
				if child.Type() == "class_declaration" {
					classDecl = child
				}
			}

			if classDecl != nil {
				ctrl := p.parseControllerClassWithDecorators(classDecl, decorators, content)
				if ctrl != nil {
					controllers = append(controllers, ctrl)
				}
			}
			return false // Don't recurse into export_statement
		}

		// Look for standalone class declarations (rare but possible)
		if node.Type() == "class_declaration" {
			ctrl := p.parseControllerClass(node, content)
			if ctrl != nil {
				controllers = append(controllers, ctrl)
			}
			return false // Don't recurse into the class
		}

		return true
	})

	return controllers
}

// parseControllerClass parses a class declaration to check if it's a NestJS controller.
// This handles standalone class declarations (not wrapped in export_statement).
func (p *Plugin) parseControllerClass(classNode *sitter.Node, content []byte) *controllerInfo {
	// Look for @Controller decorator in parent's children (siblings before class)
	var decorators []*sitter.Node

	parent := classNode.Parent()
	if parent != nil {
		for i := 0; i < int(parent.ChildCount()); i++ {
			sibling := parent.Child(i)
			if sibling == classNode {
				break
			}
			if sibling.Type() == "decorator" {
				decorators = append(decorators, sibling)
			}
		}
	}

	return p.parseControllerClassWithDecorators(classNode, decorators, content)
}

// parseControllerClassWithDecorators parses a class with explicit decorators.
func (p *Plugin) parseControllerClassWithDecorators(classNode *sitter.Node, decorators []*sitter.Node, content []byte) *controllerInfo {
	// Find the @Controller decorator
	var controllerDecorator *sitter.Node
	for _, dec := range decorators {
		if p.isControllerDecorator(dec, content) {
			controllerDecorator = dec
			break
		}
	}

	if controllerDecorator == nil {
		return nil
	}

	// Extract controller info
	ctrl := &controllerInfo{
		classNode:  classNode,
		sourceLine: int(classNode.StartPoint().Row) + 1,
	}

	// Get class name
	for i := 0; i < int(classNode.ChildCount()); i++ {
		child := classNode.Child(i)
		if child.Type() == "type_identifier" || child.Type() == "identifier" {
			ctrl.name = child.Content(content)
			break
		}
	}

	// Extract base path from @Controller decorator
	ctrl.basePath, ctrl.version = p.extractControllerPath(controllerDecorator, content)

	return ctrl
}

// isControllerDecorator checks if a decorator is @Controller.
func (p *Plugin) isControllerDecorator(node *sitter.Node, content []byte) bool {
	decoratorText := node.Content(content)
	return strings.Contains(decoratorText, "@Controller")
}

// extractControllerPath extracts the base path from a @Controller decorator.
func (p *Plugin) extractControllerPath(decorator *sitter.Node, content []byte) (path, version string) {
	// Find the call_expression inside the decorator
	var callExpr *sitter.Node
	p.walkNodes(decorator, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			callExpr = n
			return false
		}
		return true
	})

	if callExpr == nil {
		return "", ""
	}

	args := p.tsParser.GetCallArguments(callExpr, content)
	if len(args) == 0 {
		return "", ""
	}

	firstArg := args[0]

	// Check for string literal: @Controller('users')
	if firstArg.Type() == "string" {
		path, _ = p.tsParser.ExtractStringLiteral(firstArg, content)
		return path, ""
	}

	// Check for object literal: @Controller({ path: 'users', version: '1' })
	if firstArg.Type() == "object" {
		p.walkNodes(firstArg, func(n *sitter.Node) bool {
			if n.Type() == "pair" || n.Type() == "property_assignment" {
				key, value := p.extractPairKeyValue(n, content)
				switch key {
				case "path":
					path = strings.Trim(value, `"'`)
				case "version":
					version = strings.Trim(value, `"'`)
				}
			}
			return true
		})
	}

	return path, version
}

// extractPairKeyValue extracts key and value from a pair node.
func (p *Plugin) extractPairKeyValue(node *sitter.Node, content []byte) (key, value string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_identifier", "identifier", "string":
			if key == "" {
				key = strings.Trim(child.Content(content), `"'`)
			} else if value == "" {
				value = child.Content(content)
			}
		case "string_fragment":
			if value == "" {
				value = child.Content(content)
			}
		}
	}
	return key, value
}

// extractRoutesFromController extracts routes from a controller class.
func (p *Plugin) extractRoutesFromController(ctrl *controllerInfo, content []byte) []types.Route {
	var routes []types.Route

	// Find class body
	var classBody *sitter.Node
	for i := 0; i < int(ctrl.classNode.ChildCount()); i++ {
		child := ctrl.classNode.Child(i)
		if child.Type() == "class_body" {
			classBody = child
			break
		}
	}

	if classBody == nil {
		return routes
	}

	// Iterate through class body children.
	// Decorators appear as siblings before method_definition.
	// Structure: class_body -> { decorator, decorator, method_definition, decorator, method_definition, ... }
	var pendingDecorators []*sitter.Node

	for i := 0; i < int(classBody.ChildCount()); i++ {
		child := classBody.Child(i)

		if child.Type() == "decorator" {
			pendingDecorators = append(pendingDecorators, child)
			continue
		}

		if child.Type() == "method_definition" || child.Type() == "public_field_definition" {
			methodRoutes := p.extractRoutesFromMethodWithDecorators(child, pendingDecorators, ctrl, content)
			routes = append(routes, methodRoutes...)
			pendingDecorators = nil // Reset decorators after use
		}
	}

	return routes
}

// extractRoutesFromMethodWithDecorators extracts routes from a method with its decorators.
func (p *Plugin) extractRoutesFromMethodWithDecorators(methodNode *sitter.Node, decorators []*sitter.Node, ctrl *controllerInfo, content []byte) []types.Route {
	var routes []types.Route

	// Find HTTP method decorators and HttpCode
	var httpDecorators []*sitter.Node
	var httpCode int

	for _, dec := range decorators {
		decoratorText := dec.Content(content)
		// Check for HTTP method decorators
		for decoratorName := range httpMethodDecorators {
			if strings.Contains(decoratorText, "@"+decoratorName+"(") ||
				strings.Contains(decoratorText, "@"+decoratorName+")") ||
				decoratorText == "@"+decoratorName {
				httpDecorators = append(httpDecorators, dec)
			}
		}
		// Check for @HttpCode decorator
		if strings.Contains(decoratorText, "@HttpCode(") {
			httpCode = p.extractHttpCode(dec, content)
		}
	}

	// Get method name from method_definition
	var methodName string
	for i := 0; i < int(methodNode.ChildCount()); i++ {
		child := methodNode.Child(i)
		if child.Type() == "property_identifier" || child.Type() == "identifier" {
			methodName = child.Content(content)
			break
		}
	}

	// Extract routes from HTTP decorators
	for _, decorator := range httpDecorators {
		route := p.extractRouteFromDecorator(decorator, methodName, ctrl, content)
		if route != nil {
			if httpCode > 0 {
				route.Responses = map[string]types.Response{
					fmt.Sprintf("%d", httpCode): {Description: "Success response"},
				}
			}
			route.SourceLine = int(methodNode.StartPoint().Row) + 1

			// Extract request body info from @Body decorator in method parameters
			requestBody := p.extractRequestBodyFromMethod(methodNode, content)
			if requestBody != nil {
				route.RequestBody = requestBody
			}

			// Extract query parameters from @Query decorator
			queryParams := p.extractQueryParamsFromMethod(methodNode, content)
			route.Parameters = append(route.Parameters, queryParams...)

			routes = append(routes, *route)
		}
	}

	return routes
}


// extractRouteFromDecorator extracts a route from an HTTP method decorator.
func (p *Plugin) extractRouteFromDecorator(decorator *sitter.Node, methodName string, ctrl *controllerInfo, content []byte) *types.Route {
	decoratorText := decorator.Content(content)

	// Determine HTTP method
	var httpMethod string
	for decoratorName, method := range httpMethodDecorators {
		if strings.Contains(decoratorText, "@"+decoratorName) {
			httpMethod = method
			break
		}
	}

	if httpMethod == "" {
		return nil
	}

	// Extract path from decorator
	decoratorPath := p.extractPathFromDecorator(decorator, content)

	// Build full path
	fullPath := buildPath(ctrl.basePath, ctrl.version, decoratorPath)

	// Convert path parameters to OpenAPI format
	fullPath = convertPathParams(fullPath)

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Generate operation ID
	operationID := generateOperationID(httpMethod, fullPath, methodName)

	// Infer tags from controller name or path
	tags := inferTags(ctrl.name, fullPath)

	return &types.Route{
		Method:      httpMethod,
		Path:        fullPath,
		Handler:     ctrl.name + "." + methodName,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
	}
}

// extractPathFromDecorator extracts the path from an HTTP decorator.
func (p *Plugin) extractPathFromDecorator(decorator *sitter.Node, content []byte) string {
	// Find call_expression inside decorator
	var callExpr *sitter.Node
	p.walkNodes(decorator, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			callExpr = n
			return false
		}
		return true
	})

	if callExpr == nil {
		return ""
	}

	args := p.tsParser.GetCallArguments(callExpr, content)
	if len(args) == 0 {
		return ""
	}

	// First argument is the path
	if args[0].Type() == "string" {
		path, _ := p.tsParser.ExtractStringLiteral(args[0], content)
		return path
	}

	return ""
}

// extractHttpCode extracts the HTTP status code from @HttpCode decorator.
func (p *Plugin) extractHttpCode(decorator *sitter.Node, content []byte) int {
	var callExpr *sitter.Node
	p.walkNodes(decorator, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			callExpr = n
			return false
		}
		return true
	})

	if callExpr == nil {
		return 0
	}

	args := p.tsParser.GetCallArguments(callExpr, content)
	if len(args) == 0 {
		return 0
	}

	if args[0].Type() == "number" {
		codeStr := args[0].Content(content)
		var code int
		_, _ = fmt.Sscanf(codeStr, "%d", &code) // Ignore error - if parsing fails, code stays 0
		return code
	}

	return 0
}

// extractRequestBodyFromMethod looks for @Body decorator in method parameters.
func (p *Plugin) extractRequestBodyFromMethod(methodNode *sitter.Node, content []byte) *types.RequestBody {
	// Find formal_parameters
	var formalParams *sitter.Node
	p.walkNodes(methodNode, func(n *sitter.Node) bool {
		if n.Type() == "formal_parameters" {
			formalParams = n
			return false
		}
		return true
	})

	if formalParams == nil {
		return nil
	}

	// Look for @Body decorator in parameters
	var bodyType string
	p.walkNodes(formalParams, func(n *sitter.Node) bool {
		if n.Type() == "required_parameter" || n.Type() == "optional_parameter" {
			hasBodyDecorator := false
			var typeAnnotation string

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "decorator" {
					decoratorText := child.Content(content)
					if strings.Contains(decoratorText, "@Body") {
						hasBodyDecorator = true
					}
				}
				if child.Type() == "type_annotation" {
					if child.ChildCount() > 1 {
						typeNode := child.Child(1)
						typeAnnotation = typeNode.Content(content)
					}
				}
			}

			if hasBodyDecorator && typeAnnotation != "" {
				bodyType = typeAnnotation
				return false
			}
		}
		return true
	})

	if bodyType == "" {
		return nil
	}

	return &types.RequestBody{
		Required: true,
		Content: map[string]types.MediaType{
			"application/json": {
				Schema: schema.SchemaRef(bodyType),
			},
		},
	}
}

// extractQueryParamsFromMethod extracts @Query parameters from method.
func (p *Plugin) extractQueryParamsFromMethod(methodNode *sitter.Node, content []byte) []types.Parameter {
	var params []types.Parameter

	// Find formal_parameters
	var formalParams *sitter.Node
	p.walkNodes(methodNode, func(n *sitter.Node) bool {
		if n.Type() == "formal_parameters" {
			formalParams = n
			return false
		}
		return true
	})

	if formalParams == nil {
		return params
	}

	// Look for @Query decorator in parameters
	p.walkNodes(formalParams, func(n *sitter.Node) bool {
		if n.Type() == "required_parameter" || n.Type() == "optional_parameter" {
			var queryName string
			var paramName string
			var paramType string
			hasQueryDecorator := false
			isOptional := n.Type() == "optional_parameter"

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				switch child.Type() {
				case "decorator":
					decoratorText := child.Content(content)
					if strings.Contains(decoratorText, "@Query") {
						hasQueryDecorator = true
						// Extract query param name from decorator if provided
						queryName = p.extractDecoratorArgString(child, content)
					}
				case "identifier":
					paramName = child.Content(content)
				case "type_annotation":
					if child.ChildCount() > 1 {
						typeNode := child.Child(1)
						paramType = typeNode.Content(content)
					}
				}
			}

			if hasQueryDecorator {
				name := queryName
				if name == "" {
					name = paramName
				}
				if name != "" {
					param := types.Parameter{
						Name:     name,
						In:       "query",
						Required: !isOptional,
						Schema: &types.Schema{
							Type: mapTypeScriptToOpenAPI(paramType),
						},
					}
					params = append(params, param)
				}
			}
		}
		return true
	})

	return params
}

// extractDecoratorArgString extracts a string argument from a decorator.
func (p *Plugin) extractDecoratorArgString(decorator *sitter.Node, content []byte) string {
	var callExpr *sitter.Node
	p.walkNodes(decorator, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			callExpr = n
			return false
		}
		return true
	})

	if callExpr == nil {
		return ""
	}

	args := p.tsParser.GetCallArguments(callExpr, content)
	if len(args) == 0 {
		return ""
	}

	if args[0].Type() == "string" {
		val, _ := p.tsParser.ExtractStringLiteral(args[0], content)
		return val
	}

	return ""
}

// walkNodes walks all nodes in the tree.
func (p *Plugin) walkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
	if node == nil {
		return
	}

	if !fn(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNodes(node.Child(i), fn)
	}
}

// ExtractSchemas extracts schema definitions from TypeScript files.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	for _, file := range files {
		if file.Language != "typescript" && file.Language != "javascript" {
			continue
		}

		pf, err := p.tsParser.Parse(file.Path, file.Content)
		if err != nil {
			continue
		}

		// Extract and register Zod schemas
		for _, zs := range pf.ZodSchemas {
			p.zodParser.ExtractAndRegister(zs.Name, zs.Node, file.Content)
		}

		pf.Close()
	}

	return p.zodParser.Registry().ToSlice(), nil
}

// --- Helper Functions ---

// colonParamRegex matches path parameters in the format :param.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// braceParamRegex matches path parameters in the format {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts NestJS-style path params (:id) to OpenAPI format ({id}).
func convertPathParams(path string) string {
	return colonParamRegex.ReplaceAllString(path, "{$1}")
}

// extractPathParams extracts path parameters from a route path.
func extractPathParams(path string) []types.Parameter {
	var params []types.Parameter

	matches := braceParamRegex.FindAllStringSubmatch(path, -1)
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

// buildPath builds the full path from controller path, version, and method path.
func buildPath(basePath, version, methodPath string) string {
	var parts []string

	// Add version prefix if specified
	if version != "" {
		parts = append(parts, "v"+version)
	}

	// Add base path
	if basePath != "" {
		basePath = strings.TrimPrefix(basePath, "/")
		basePath = strings.TrimSuffix(basePath, "/")
		if basePath != "" {
			parts = append(parts, basePath)
		}
	}

	// Add method path
	if methodPath != "" {
		methodPath = strings.TrimPrefix(methodPath, "/")
		methodPath = strings.TrimSuffix(methodPath, "/")
		if methodPath != "" {
			parts = append(parts, methodPath)
		}
	}

	if len(parts) == 0 {
		return "/"
	}

	return "/" + strings.Join(parts, "/")
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" {
		parts := strings.Split(handler, ".")
		name := parts[len(parts)-1]
		return strings.ToLower(method) + name
	}

	// Generate from path
	path = braceParamRegex.ReplaceAllString(path, "By${1}")
	path = strings.ReplaceAll(path, "/", " ")
	path = strings.TrimSpace(path)

	words := strings.Fields(path)
	if len(words) == 0 {
		return strings.ToLower(method)
	}

	var sb strings.Builder
	sb.WriteString(strings.ToLower(method))

	titleCaser := cases.Title(language.English)
	for _, word := range words {
		word = titleCaser.String(strings.ToLower(word))
		sb.WriteString(word)
	}

	return sb.String()
}

// inferTags infers tags from the controller name or path.
func inferTags(controllerName, path string) []string {
	// Try to extract tag from controller name (remove "Controller" suffix)
	if controllerName != "" {
		name := strings.TrimSuffix(controllerName, "Controller")
		name = strings.TrimSuffix(name, "controller")
		if name != "" {
			return []string{strings.ToLower(name)}
		}
	}

	// Fall back to path-based inference
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	skipPrefixes := map[string]bool{
		"api": true,
		"v1":  true,
		"v2":  true,
		"v3":  true,
	}

	for _, part := range parts {
		if part == "" {
			continue
		}
		if skipPrefixes[part] {
			continue
		}
		if strings.HasPrefix(part, "{") {
			continue
		}
		return []string{part}
	}

	return nil
}

// mapTypeScriptToOpenAPI maps TypeScript types to OpenAPI types.
func mapTypeScriptToOpenAPI(tsType string) string {
	switch strings.TrimSpace(tsType) {
	case "string":
		return "string"
	case "number":
		return "number"
	case "boolean":
		return "boolean"
	case "Date":
		return "string"
	default:
		return "string"
	}
}

// Register registers the NestJS plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
