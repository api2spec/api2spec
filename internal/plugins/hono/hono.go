// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package hono provides a plugin for extracting routes from Hono framework applications.
package hono

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
	"all":     "ALL", // Hono supports 'all' for all methods
}

// Plugin implements the FrameworkPlugin interface for Hono framework.
type Plugin struct {
	tsParser  *parser.TypeScriptParser
	zodParser *schema.ZodParser
}

// New creates a new Hono plugin instance.
func New() *Plugin {
	tsParser := parser.NewTypeScriptParser()
	return &Plugin{
		tsParser:  tsParser,
		zodParser: schema.NewZodParser(tsParser),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "hono"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "hono",
		Version:     "1.0.0",
		Description: "Extracts routes from Hono framework applications",
		SupportedFrameworks: []string{
			"hono",
		},
	}
}

// Detect checks if Hono is used in the project by looking at package.json.
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

	// Check for hono in dependencies or devDependencies
	if _, ok := pkg.Dependencies["hono"]; ok {
		return true, nil
	}
	if _, ok := pkg.DevDependencies["hono"]; ok {
		return true, nil
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Hono route definitions.
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

// extractRoutesFromFile extracts routes from a single TypeScript/JavaScript file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.tsParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports Hono
	if !p.hasHonoImport(pf.RootNode, file.Content) {
		return nil, nil
	}

	var routes []types.Route

	// Build a map of Zod schema names to their nodes
	zodSchemas := make(map[string]*sitter.Node)
	for _, zs := range pf.ZodSchemas {
		zodSchemas[zs.Name] = zs.Node
	}

	// Track router variables and their base paths
	routerVars := p.findRouterVariables(pf.RootNode, file.Content)

	// Find all call expressions
	calls := p.tsParser.FindCallExpressions(pf.RootNode, file.Content)

	for _, call := range calls {
		route := p.extractRouteFromCall(call, file.Content, routerVars, zodSchemas)
		if route != nil {
			route.SourceFile = file.Path
			routes = append(routes, *route)
		}
	}

	return routes, nil
}

// hasHonoImport checks if the file imports from 'hono'.
func (p *Plugin) hasHonoImport(rootNode *sitter.Node, content []byte) bool {
	hasImport := false

	p.tsParser.FindCallExpressions(rootNode, content)

	// Look for import statements
	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "import_statement" {
			// Check the source (module name)
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "string" {
					source := child.Content(content)
					source = strings.Trim(source, `"'`)
					if source == "hono" || strings.HasPrefix(source, "hono/") {
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

// routerInfo tracks information about a router variable.
type routerInfo struct {
	name     string
	basePath string
}

// findRouterVariables finds variables that are Hono instances.
func (p *Plugin) findRouterVariables(rootNode *sitter.Node, content []byte) map[string]routerInfo {
	routers := make(map[string]routerInfo)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Look for: const app = new Hono()
		if node.Type() == "lexical_declaration" || node.Type() == "variable_declaration" {
			p.walkNodes(node, func(n *sitter.Node) bool {
				if n.Type() == "variable_declarator" {
					var name string
					isHono := false

					for i := 0; i < int(n.ChildCount()); i++ {
						child := n.Child(i)
						if child.Type() == "identifier" && name == "" {
							name = child.Content(content)
						}
						if child.Type() == "new_expression" {
							// Check if it's new Hono()
							calleeText := ""
							for j := 0; j < int(child.ChildCount()); j++ {
								c := child.Child(j)
								if c.Type() == "identifier" {
									calleeText = c.Content(content)
									break
								}
							}
							if calleeText == "Hono" {
								isHono = true
							}
						}
					}

					if name != "" && isHono {
						routers[name] = routerInfo{name: name}
					}
				}
				return true
			})
		}
		return true
	})

	// Default router names if not found
	if len(routers) == 0 {
		routers["app"] = routerInfo{name: "app"}
	}

	return routers
}

// extractRouteFromCall extracts a route from a call expression.
// TODO: Use routers for prefix tracking in nested routes.
func (p *Plugin) extractRouteFromCall(
	node *sitter.Node,
	content []byte,
	_ map[string]routerInfo,
	zodSchemas map[string]*sitter.Node,
) *types.Route {
	// Get the callee (function being called)
	callee := node.Child(0)
	if callee == nil {
		return nil
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
		// Check for route() method for sub-routing
		if method == "route" {
			// This is a sub-router mount, we could track this for prefixes
			return nil
		}
		return nil
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

	// Convert Hono path parameters (:param) to OpenAPI format ({param})
	path = convertPathParams(path)

	// Extract path parameters
	params := extractPathParams(path)

	// Look for zValidator middleware to determine request body schema
	var requestBody *types.RequestBody
	for i := 1; i < len(args)-1; i++ { // Skip path (first) and handler (last)
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
			}
		}
	}

	// Generate operation ID
	operationID := generateOperationID(httpMethod, path, "")

	// Infer tags from path
	tags := inferTags(path)

	route := &types.Route{
		Method:      httpMethod,
		Path:        path,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		RequestBody: requestBody,
		SourceLine:  int(node.StartPoint().Row) + 1,
	}

	return route
}

// extractValidatorSchema extracts the schema reference from a zValidator call.
// TODO: Use zodSchemas to resolve and validate schema references.
func (p *Plugin) extractValidatorSchema(
	node *sitter.Node,
	content []byte,
	_ map[string]*sitter.Node,
) *types.Schema {
	calleeText := p.tsParser.GetCalleeText(node, content)

	// Check for zValidator('json', Schema)
	if calleeText != "zValidator" {
		return nil
	}

	args := p.tsParser.GetCallArguments(node, content)
	if len(args) < 2 {
		return nil
	}

	// First arg is the target ('json', 'query', 'param', etc.)
	target := ""
	if args[0].Type() == "string" {
		target, _ = p.tsParser.ExtractStringLiteral(args[0], content)
	}

	// Only handle 'json' for request body
	if target != "json" {
		return nil
	}

	// Second arg is the schema (identifier or inline)
	schemaArg := args[1]
	if schemaArg.Type() == "identifier" {
		schemaName := schemaArg.Content(content)
		// Create a reference to the schema
		return schema.SchemaRef(schemaName)
	}

	// Inline Zod schema
	if schemaArg.Type() == "call_expression" {
		parsedSchema, _ := p.zodParser.ParseZodSchema(schemaArg, content)
		return parsedSchema
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

// ExtractSchemas extracts schema definitions from TypeScript interfaces and Zod schemas.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	tsExtractor := schema.NewTypeScriptSchemaExtractor()

	for _, file := range files {
		if file.Language != "typescript" && file.Language != "javascript" {
			continue
		}

		pf, err := p.tsParser.Parse(file.Path, file.Content)
		if err != nil {
			continue
		}

		// Extract TypeScript interfaces
		for _, iface := range pf.Interfaces {
			tsExtractor.ExtractAndRegister(iface)
		}

		// Extract Zod schemas (if any)
		for _, zs := range pf.ZodSchemas {
			p.zodParser.ExtractAndRegister(zs.Name, zs.Node, file.Content)
		}

		pf.Close()
	}

	// Merge Zod schemas into the registry
	tsExtractor.Registry().Merge(p.zodParser.Registry())

	return tsExtractor.Registry().ToSlice(), nil
}

// --- Helper Functions ---

// pathParamRegex matches path parameters in the format :param or {param}.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts Hono-style path params (:id) to OpenAPI format ({id}).
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

		// Check for regex pattern in param (e.g., {id:[0-9]+})
		if idx := strings.Index(paramName, ":"); idx > 0 {
			paramName = paramName[:idx]
		}

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

// Register registers the Hono plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
