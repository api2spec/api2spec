// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package express provides a plugin for extracting routes from Express.js framework applications.
package express

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

// httpMethods maps HTTP method names to their uppercase forms.
var httpMethods = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"delete":  "DELETE",
	"patch":   "PATCH",
	"head":    "HEAD",
	"options": "OPTIONS",
	"all":     "ALL",
}

// Plugin implements the FrameworkPlugin interface for Express framework.
type Plugin struct {
	tsParser  *parser.TypeScriptParser
	zodParser *schema.ZodParser
}

// New creates a new Express plugin instance.
func New() *Plugin {
	tsParser := parser.NewTypeScriptParser()
	return &Plugin{
		tsParser:  tsParser,
		zodParser: schema.NewZodParser(tsParser),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "express"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "express",
		Version:     "1.0.0",
		Description: "Extracts routes from Express.js framework applications",
		SupportedFrameworks: []string{
			"express",
		},
	}
}

// Detect checks if Express is used in the project by looking at package.json.
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

	// Check for express in dependencies or devDependencies
	if _, ok := pkg.Dependencies["express"]; ok {
		return true, nil
	}
	if _, ok := pkg.DevDependencies["express"]; ok {
		return true, nil
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Express route definitions.
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

// routerInfo tracks information about an Express app or router variable.
type routerInfo struct {
	name       string
	basePath   string
	isRouter   bool
	parentName string // For tracking nested routers
}

// extractRoutesFromFile extracts routes from a single TypeScript/JavaScript file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.tsParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports or requires Express
	if !p.hasExpressImport(pf.RootNode, file.Content) {
		return nil, nil
	}

	var routes []types.Route

	// Build a map of Zod schema names to their nodes
	zodSchemas := make(map[string]*sitter.Node)
	for _, zs := range pf.ZodSchemas {
		zodSchemas[zs.Name] = zs.Node
	}

	// Track router/app variables and their base paths
	routers := p.findRouterVariables(pf.RootNode, file.Content)

	// Track router mounting (app.use('/prefix', router))
	routerMounts := p.findRouterMounts(pf.RootNode, file.Content, routers)

	// Find all call expressions
	calls := p.tsParser.FindCallExpressions(pf.RootNode, file.Content)

	for _, call := range calls {
		extractedRoutes := p.extractRoutesFromCall(call, file.Content, routers, routerMounts, zodSchemas)
		for i := range extractedRoutes {
			extractedRoutes[i].SourceFile = file.Path
			routes = append(routes, extractedRoutes[i])
		}
	}

	return routes, nil
}

// hasExpressImport checks if the file imports or requires 'express'.
func (p *Plugin) hasExpressImport(rootNode *sitter.Node, content []byte) bool {
	hasImport := false

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check for ES6 import: import express from 'express'
		if node.Type() == "import_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "string" {
					source := child.Content(content)
					source = strings.Trim(source, `"'`)
					if source == "express" {
						hasImport = true
						return false
					}
				}
			}
		}

		// Check for require: const express = require('express')
		if node.Type() == "call_expression" {
			calleeText := p.tsParser.GetCalleeText(node, content)
			if calleeText == "require" {
				args := p.tsParser.GetCallArguments(node, content)
				if len(args) > 0 {
					argText := args[0].Content(content)
					argText = strings.Trim(argText, `"'`)
					if argText == "express" {
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

// findRouterVariables finds variables that are Express app or Router instances.
func (p *Plugin) findRouterVariables(rootNode *sitter.Node, content []byte) map[string]*routerInfo {
	routers := make(map[string]*routerInfo)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "lexical_declaration" || node.Type() == "variable_declaration" {
			p.walkNodes(node, func(n *sitter.Node) bool {
				if n.Type() == "variable_declarator" {
					var name string
					var isExpress, isRouter bool

					for i := 0; i < int(n.ChildCount()); i++ {
						child := n.Child(i)
						if child.Type() == "identifier" && name == "" {
							name = child.Content(content)
						}
						if child.Type() == "call_expression" {
							callText := child.Content(content)
							// Check for express() or express.Router()
							if strings.HasPrefix(callText, "express()") || callText == "express()" {
								isExpress = true
							}
							// Check for express.Router()
							calleeNode := child.Child(0)
							if calleeNode != nil {
								calleeText := calleeNode.Content(content)
								if calleeText == "express.Router" {
									isRouter = true
								}
							}
						}
					}

					if name != "" && (isExpress || isRouter) {
						routers[name] = &routerInfo{
							name:     name,
							isRouter: isRouter,
						}
					}
				}
				return true
			})
		}
		return true
	})

	// Default to "app" if no routers found
	if len(routers) == 0 {
		routers["app"] = &routerInfo{name: "app"}
	}

	return routers
}

// findRouterMounts finds app.use('/prefix', router) calls to track path prefixes.
func (p *Plugin) findRouterMounts(rootNode *sitter.Node, content []byte, routers map[string]*routerInfo) map[string]string {
	mounts := make(map[string]string)

	calls := p.tsParser.FindCallExpressions(rootNode, content)

	for _, call := range calls {
		callee := call.Child(0)
		if callee == nil || callee.Type() != "member_expression" {
			continue
		}

		object, method := p.tsParser.GetMemberExpressionParts(callee, content)
		if method != "use" {
			continue
		}

		// Check if the object is an Express app or router
		if _, ok := routers[object]; !ok {
			// Also check if it's a known app variable
			if object != "app" {
				continue
			}
		}

		args := p.tsParser.GetCallArguments(call, content)
		if len(args) < 2 {
			continue
		}

		// First arg should be the path prefix
		pathArg := args[0]
		path := ""
		if pathArg.Type() == "string" || pathArg.Type() == "template_string" {
			path, _ = p.tsParser.ExtractStringLiteral(pathArg, content)
		}

		if path == "" {
			continue
		}

		// Second arg should be the router variable
		routerArg := args[1]
		if routerArg.Type() == "identifier" {
			routerName := routerArg.Content(content)
			mounts[routerName] = path
		}
	}

	return mounts
}

// extractRoutesFromCall extracts routes from a call expression.
// Returns multiple routes for route chaining patterns.
func (p *Plugin) extractRoutesFromCall(
	node *sitter.Node,
	content []byte,
	routers map[string]*routerInfo,
	routerMounts map[string]string,
	zodSchemas map[string]*sitter.Node,
) []types.Route {
	// Get the callee (function being called)
	callee := node.Child(0)
	if callee == nil {
		return nil
	}

	// Check for route chaining: app.route('/path').get().post()
	if chainedRoutes := p.extractRouteChain(node, content, routers, routerMounts, zodSchemas); len(chainedRoutes) > 0 {
		return chainedRoutes
	}

	// Check if this is a method call (member_expression)
	if callee.Type() != "member_expression" {
		return nil
	}

	// Get object and method
	object, method := p.tsParser.GetMemberExpressionParts(callee, content)
	if object == "" || method == "" {
		return nil
	}

	// Check if method is an HTTP method
	httpMethod, isHTTPMethod := httpMethods[strings.ToLower(method)]
	if !isHTTPMethod {
		return nil
	}

	// Check if object is a known router or app
	prefix := ""
	if mount, ok := routerMounts[object]; ok {
		prefix = mount
	}

	// Get arguments
	args := p.tsParser.GetCallArguments(node, content)
	if len(args) == 0 {
		return nil
	}

	// First argument should be the path
	pathArg := args[0]
	path := ""
	if pathArg.Type() == "string" || pathArg.Type() == "template_string" {
		path, _ = p.tsParser.ExtractStringLiteral(pathArg, content)
	}

	if path == "" {
		return nil
	}

	// Combine prefix and path
	fullPath := combinePaths(prefix, path)

	// Convert Express path parameters (:param) to OpenAPI format ({param})
	fullPath = convertPathParams(fullPath)

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Look for validation middleware to determine request body schema
	var requestBody *types.RequestBody
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg.Type() == "call_expression" {
			schemaRef := p.extractValidatorSchema(arg, content, zodSchemas)
			if schemaRef != nil {
				requestBody = &types.RequestBody{
					Required: true,
					Content: map[string]types.MediaType{
						"application/json": {
							Schema: schemaRef,
						},
					},
				}
				break
			}
		}
	}

	// Generate operation ID
	operationID := generateOperationID(httpMethod, fullPath, "")

	// Infer tags from path
	tags := inferTags(fullPath)

	route := types.Route{
		Method:      httpMethod,
		Path:        fullPath,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		RequestBody: requestBody,
		SourceLine:  int(node.StartPoint().Row) + 1,
	}

	return []types.Route{route}
}

// extractRouteChain handles app.route('/path').get().post() patterns.
// TODO: Use routers for prefix tracking in nested routes.
func (p *Plugin) extractRouteChain(
	node *sitter.Node,
	content []byte,
	_ map[string]*routerInfo,
	routerMounts map[string]string,
	_ map[string]*sitter.Node,
) []types.Route {
	// Build the chain of method calls
	chain := p.buildMethodChain(node, content)
	if len(chain) == 0 {
		return nil
	}

	// Find the base route() call
	var basePath string
	var baseRouterName string
	var routeCallFound bool

	for i := len(chain) - 1; i >= 0; i-- {
		item := chain[i]
		if item.method == "route" && len(item.args) > 0 {
			basePath, _ = p.tsParser.ExtractStringLiteral(item.args[0], content)
			baseRouterName = item.object
			routeCallFound = true
			break
		}
	}

	if !routeCallFound || basePath == "" {
		return nil
	}

	// Apply prefix if router is mounted
	prefix := ""
	if mount, ok := routerMounts[baseRouterName]; ok {
		prefix = mount
	}

	fullPath := combinePaths(prefix, basePath)
	fullPath = convertPathParams(fullPath)
	params := extractPathParams(fullPath)
	tags := inferTags(fullPath)

	var routes []types.Route

	// Extract HTTP method calls from the chain
	for _, item := range chain {
		if httpMethod, isHTTP := httpMethods[strings.ToLower(item.method)]; isHTTP {
			operationID := generateOperationID(httpMethod, fullPath, "")
			route := types.Route{
				Method:      httpMethod,
				Path:        fullPath,
				OperationID: operationID,
				Tags:        tags,
				Parameters:  params,
				SourceLine:  int(node.StartPoint().Row) + 1,
			}
			routes = append(routes, route)
		}
	}

	return routes
}

// methodCall represents a method call in a chain.
type methodCall struct {
	object string
	method string
	args   []*sitter.Node
}

// buildMethodChain builds a list of chained method calls from innermost to outermost.
func (p *Plugin) buildMethodChain(node *sitter.Node, content []byte) []methodCall {
	var chain []methodCall

	current := node
	for current != nil && current.Type() == "call_expression" {
		callee := current.Child(0)
		if callee == nil {
			break
		}

		if callee.Type() == "member_expression" {
			object, method := p.tsParser.GetMemberExpressionParts(callee, content)
			args := p.tsParser.GetCallArguments(current, content)

			chain = append(chain, methodCall{
				object: object,
				method: method,
				args:   args,
			})

			// Move to the object of the member expression (for chaining)
			objNode := callee.Child(0)
			if objNode != nil && objNode.Type() == "call_expression" {
				current = objNode
			} else {
				break
			}
		} else {
			break
		}
	}

	return chain
}

// extractValidatorSchema extracts the schema reference from validation middleware.
func (p *Plugin) extractValidatorSchema(
	node *sitter.Node,
	content []byte,
	zodSchemas map[string]*sitter.Node,
) *types.Schema {
	calleeText := p.tsParser.GetCalleeText(node, content)

	// Handle express-validator: body('email').isEmail()
	if strings.HasPrefix(calleeText, "body(") || strings.HasPrefix(calleeText, "body") ||
		strings.HasPrefix(calleeText, "query(") || strings.HasPrefix(calleeText, "param(") {
		return p.extractExpressValidatorSchema(node, content)
	}

	// Handle celebrate/joi: celebrate({ body: schema })
	if calleeText == "celebrate" {
		return p.extractCelebrateSchema(node, content)
	}

	// Handle Zod validation middleware: validate(schema) or zValidator('json', schema)
	if calleeText == "validate" || calleeText == "zValidator" {
		return p.extractZodValidatorSchema(node, content, zodSchemas)
	}

	return nil
}

// extractExpressValidatorSchema attempts to extract schema info from express-validator.
func (p *Plugin) extractExpressValidatorSchema(node *sitter.Node, content []byte) *types.Schema {
	// For express-validator, we can only infer basic types from the validation chain
	// This is a simplified implementation
	nodeText := node.Content(content)

	props := make(map[string]*types.Schema)
	var required []string

	// Look for body('fieldName') patterns and extract field names
	bodyFieldRegex := regexp.MustCompile(`body\(['"]([^'"]+)['"]\)`)
	matches := bodyFieldRegex.FindAllStringSubmatch(nodeText, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			fieldName := match[1]
			fieldSchema := &types.Schema{Type: "string"} // Default to string

			// Try to infer type from validation chain
			if strings.Contains(nodeText, ".isInt()") || strings.Contains(nodeText, ".isNumeric()") {
				fieldSchema.Type = "integer"
			} else if strings.Contains(nodeText, ".isFloat()") || strings.Contains(nodeText, ".isDecimal()") {
				fieldSchema.Type = "number"
			} else if strings.Contains(nodeText, ".isBoolean()") {
				fieldSchema.Type = "boolean"
			} else if strings.Contains(nodeText, ".isEmail()") {
				fieldSchema.Format = "email"
			} else if strings.Contains(nodeText, ".isURL()") {
				fieldSchema.Format = "uri"
			} else if strings.Contains(nodeText, ".isUUID()") {
				fieldSchema.Format = "uuid"
			}

			props[fieldName] = fieldSchema

			// If not marked as optional, assume required
			if !strings.Contains(nodeText, ".optional()") {
				required = append(required, fieldName)
			}
		}
	}

	if len(props) == 0 {
		return nil
	}

	return &types.Schema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

// extractCelebrateSchema extracts schema from celebrate({ body: schema }) patterns.
func (p *Plugin) extractCelebrateSchema(node *sitter.Node, content []byte) *types.Schema {
	args := p.tsParser.GetCallArguments(node, content)
	if len(args) == 0 {
		return nil
	}

	// First arg is an object with body, query, params, etc.
	objArg := args[0]
	if objArg.Type() != "object" {
		return nil
	}

	// Look for body property
	p.walkNodes(objArg, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			// Get property key
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "property_identifier" {
					keyName := child.Content(content)
					if keyName == "body" {
						// Found body property - could extract Joi schema here
						// For now, return a placeholder
						return false
					}
				}
			}
		}
		return true
	})

	// Celebrate/Joi schema extraction is complex - return placeholder for now
	return &types.Schema{
		Type: "object",
	}
}

// extractZodValidatorSchema extracts Zod schema from validation middleware.
// TODO: Use zodSchemas to resolve and validate schema references.
func (p *Plugin) extractZodValidatorSchema(
	node *sitter.Node,
	content []byte,
	_ map[string]*sitter.Node,
) *types.Schema {
	calleeText := p.tsParser.GetCalleeText(node, content)
	args := p.tsParser.GetCallArguments(node, content)

	if calleeText == "zValidator" && len(args) >= 2 {
		// zValidator('json', Schema) pattern
		target := ""
		if args[0].Type() == "string" {
			target, _ = p.tsParser.ExtractStringLiteral(args[0], content)
		}

		if target != "json" && target != "body" {
			return nil
		}

		schemaArg := args[1]
		if schemaArg.Type() == "identifier" {
			schemaName := schemaArg.Content(content)
			return schema.SchemaRef(schemaName)
		}

		// Inline Zod schema
		if schemaArg.Type() == "call_expression" {
			parsedSchema, _ := p.zodParser.ParseZodSchema(schemaArg, content)
			return parsedSchema
		}
	} else if calleeText == "validate" && len(args) >= 1 {
		// validate(Schema) pattern
		schemaArg := args[0]
		if schemaArg.Type() == "identifier" {
			schemaName := schemaArg.Content(content)
			return schema.SchemaRef(schemaName)
		}

		// Inline Zod schema
		if schemaArg.Type() == "call_expression" {
			parsedSchema, _ := p.zodParser.ParseZodSchema(schemaArg, content)
			return parsedSchema
		}
	}

	return nil
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

// ExtractSchemas extracts schema definitions from Zod schemas in TypeScript files.
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

// pathParamRegex matches path parameters in the format :param or :param(...) or *.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)(?:\([^)]*\))?`)
var wildcardRegex = regexp.MustCompile(`\*`)
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts Express-style path params (:id, *) to OpenAPI format ({id}, {path}).
func convertPathParams(path string) string {
	// Convert :param and :param(regex) to {param}
	result := colonParamRegex.ReplaceAllString(path, "{$1}")

	// Convert * to {path} (wildcard)
	result = wildcardRegex.ReplaceAllString(result, "{path}")

	return result
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

// combinePaths combines a prefix and path, handling slashes correctly.
func combinePaths(prefix, path string) string {
	if prefix == "" {
		return path
	}

	// Remove trailing slash from prefix
	prefix = strings.TrimSuffix(prefix, "/")

	// Ensure path starts with slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// generateOperationID generates an operation ID from method and path.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" && handler != "<anonymous>" {
		parts := strings.Split(handler, ".")
		name := parts[len(parts)-1]
		return strings.ToLower(method) + name
	}

	// Generate from path
	// Remove parameter syntax and convert to camelCase
	path = braceParamRegex.ReplaceAllString(path, "By${1}")
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

// Register registers the Express plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
