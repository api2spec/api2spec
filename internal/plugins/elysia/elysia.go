// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package elysia provides a plugin for extracting routes from Elysia framework applications.
package elysia

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

// Plugin implements the FrameworkPlugin interface for Elysia framework.
type Plugin struct {
	tsParser  *parser.TypeScriptParser
	zodParser *schema.ZodParser
}

// New creates a new Elysia plugin instance.
func New() *Plugin {
	tsParser := parser.NewTypeScriptParser()
	return &Plugin{
		tsParser:  tsParser,
		zodParser: schema.NewZodParser(tsParser),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "elysia"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "elysia",
		Version:     "1.0.0",
		Description: "Extracts routes from Elysia framework applications",
		SupportedFrameworks: []string{
			"elysia",
		},
	}
}

// Detect checks if Elysia is used in the project by looking at package.json.
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

	// Check for elysia in dependencies or devDependencies
	if _, ok := pkg.Dependencies["elysia"]; ok {
		return true, nil
	}
	if _, ok := pkg.DevDependencies["elysia"]; ok {
		return true, nil
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Elysia route definitions.
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

// elysiaInfo tracks information about an Elysia instance variable.
type elysiaInfo struct {
	name   string
	prefix string
}

// extractRoutesFromFile extracts routes from a single TypeScript/JavaScript file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.tsParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports Elysia
	if !p.hasElysiaImport(pf.RootNode, file.Content) {
		return nil, nil
	}

	var routes []types.Route

	// Build a map of Zod schema names to their nodes
	zodSchemas := make(map[string]*sitter.Node)
	for _, zs := range pf.ZodSchemas {
		zodSchemas[zs.Name] = zs.Node
	}

	// Track Elysia instances
	elysiaInstances := p.findElysiaInstances(pf.RootNode, file.Content)

	// Track group prefixes from chained calls
	groupPrefixes := make(map[string]string)

	// Find all call expressions
	calls := p.tsParser.FindCallExpressions(pf.RootNode, file.Content)

	for _, call := range calls {
		extractedRoutes := p.extractRoutesFromCall(call, file.Content, elysiaInstances, groupPrefixes, zodSchemas)
		for i := range extractedRoutes {
			extractedRoutes[i].SourceFile = file.Path
			routes = append(routes, extractedRoutes[i])
		}
	}

	return routes, nil
}

// hasElysiaImport checks if the file imports from 'elysia'.
func (p *Plugin) hasElysiaImport(rootNode *sitter.Node, content []byte) bool {
	hasImport := false

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check for ES6 import statements
		if node.Type() == "import_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "string" {
					source := child.Content(content)
					source = strings.Trim(source, `"'`)
					if source == "elysia" || strings.HasPrefix(source, "elysia/") ||
						strings.HasPrefix(source, "@elysiajs/") {
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

// findElysiaInstances finds variables that are Elysia instances.
func (p *Plugin) findElysiaInstances(rootNode *sitter.Node, content []byte) map[string]*elysiaInfo {
	instances := make(map[string]*elysiaInfo)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "lexical_declaration" || node.Type() == "variable_declaration" {
			p.walkNodes(node, func(n *sitter.Node) bool {
				if n.Type() == "variable_declarator" {
					var name string
					var isElysia bool

					for i := 0; i < int(n.ChildCount()); i++ {
						child := n.Child(i)
						if child.Type() == "identifier" && name == "" {
							name = child.Content(content)
						}
						if child.Type() == "new_expression" {
							// Check if it's new Elysia()
							for j := 0; j < int(child.ChildCount()); j++ {
								c := child.Child(j)
								if c.Type() == "identifier" {
									if c.Content(content) == "Elysia" {
										isElysia = true
									}
								}
							}
						}
						if child.Type() == "call_expression" {
							// Handle method chaining like new Elysia().use(...)
							callContent := child.Content(content)
							if strings.Contains(callContent, "new Elysia()") ||
								strings.HasPrefix(callContent, "new Elysia(") {
								isElysia = true
							}
						}
					}

					if name != "" && isElysia {
						instances[name] = &elysiaInfo{name: name}
					}
				}
				return true
			})
		}
		return true
	})

	// Default to common names if not found
	if len(instances) == 0 {
		instances["app"] = &elysiaInfo{name: "app"}
		instances["elysia"] = &elysiaInfo{name: "elysia"}
	}

	return instances
}

// isInsideGroupCallback checks if the node is inside a .group() callback function.
func (p *Plugin) isInsideGroupCallback(node *sitter.Node, content []byte) bool {
	// Walk up the tree to check if we're inside an arrow_function
	// that is the second argument of a group() call
	current := node.Parent()
	for current != nil {
		// Check if we're inside an arrow_function or function_expression
		if current.Type() == "arrow_function" || current.Type() == "function_expression" {
			// Check if this function's parent is an arguments node
			argsParent := current.Parent()
			if argsParent != nil && argsParent.Type() == "arguments" {
				// Check if the arguments belong to a group() call
				callExpr := argsParent.Parent()
				if callExpr != nil && callExpr.Type() == "call_expression" {
					callee := callExpr.Child(0)
					if callee != nil && callee.Type() == "member_expression" {
						_, method := p.tsParser.GetMemberExpressionParts(callee, content)
						if method == "group" {
							return true
						}
					}
				}
			}
		}
		current = current.Parent()
	}
	return false
}

// extractRoutesFromCall extracts routes from a call expression.
func (p *Plugin) extractRoutesFromCall(
	node *sitter.Node,
	content []byte,
	instances map[string]*elysiaInfo,
	groupPrefixes map[string]string,
	zodSchemas map[string]*sitter.Node,
) []types.Route {
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

	// Check for group() method - extract routes from within the group
	if method == "group" {
		return p.extractRoutesFromGroup(node, content, instances, zodSchemas)
	}

	// Check if method is an HTTP method
	httpMethod, isHTTPMethod := httpMethods[strings.ToLower(method)]
	if !isHTTPMethod {
		return nil
	}

	// Skip if this call is inside a group() callback (already handled by extractRoutesFromGroup)
	if p.isInsideGroupCallback(node, content) {
		return nil
	}

	// Check if this is chained from an Elysia instance or another chain
	if !p.isElysiaChain(callee, content, instances) {
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

	// Apply any group prefix
	if prefix, ok := groupPrefixes[object]; ok {
		path = combinePaths(prefix, path)
	}

	// Convert Elysia path parameters (:param) to OpenAPI format ({param})
	path = convertPathParams(path)

	// Extract path parameters
	params := extractPathParams(path)

	// Look for validation options in the last argument
	var requestBody *types.RequestBody
	if len(args) >= 2 {
		// Check if last argument is an options object with validation
		lastArg := args[len(args)-1]
		if lastArg.Type() == "object" {
			requestBody = p.extractValidationSchema(lastArg, content, zodSchemas)
		}
	}

	// Generate operation ID
	operationID := generateOperationID(httpMethod, path, "")

	// Infer tags from path
	tags := inferTags(path)

	route := types.Route{
		Method:      httpMethod,
		Path:        path,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		RequestBody: requestBody,
		SourceLine:  int(node.StartPoint().Row) + 1,
	}

	return []types.Route{route}
}

// isElysiaChain checks if the member expression is chained from an Elysia instance.
func (p *Plugin) isElysiaChain(memberExpr *sitter.Node, content []byte, instances map[string]*elysiaInfo) bool {
	// Walk up the chain to find if it originates from an Elysia instance
	current := memberExpr
	for current != nil {
		if current.Type() == "member_expression" {
			objNode := current.Child(0)
			if objNode != nil {
				if objNode.Type() == "identifier" {
					name := objNode.Content(content)
					if _, ok := instances[name]; ok {
						return true
					}
				}
				if objNode.Type() == "call_expression" {
					// Check if this call creates an Elysia instance
					callContent := objNode.Content(content)
					if strings.Contains(callContent, "new Elysia()") ||
						strings.HasPrefix(callContent, "new Elysia(") {
						return true
					}
					// Continue traversing the chain
					callCallee := objNode.Child(0)
					if callCallee != nil && callCallee.Type() == "member_expression" {
						current = callCallee
						continue
					}
				}
				if objNode.Type() == "new_expression" {
					// Check if it's new Elysia()
					for i := 0; i < int(objNode.ChildCount()); i++ {
						c := objNode.Child(i)
						if c.Type() == "identifier" && c.Content(content) == "Elysia" {
							return true
						}
					}
				}
			}
		}
		// Try to continue up the chain
		if current.Type() == "member_expression" && current.Child(0) != nil {
			objNode := current.Child(0)
			if objNode.Type() == "call_expression" {
				callCallee := objNode.Child(0)
				if callCallee != nil && callCallee.Type() == "member_expression" {
					current = callCallee
					continue
				}
			}
		}
		break
	}

	return false
}

// extractRoutesFromGroup extracts routes from a .group('/prefix', app => app.get(...)) pattern.
func (p *Plugin) extractRoutesFromGroup(
	node *sitter.Node,
	content []byte,
	instances map[string]*elysiaInfo,
	zodSchemas map[string]*sitter.Node,
) []types.Route {
	args := p.tsParser.GetCallArguments(node, content)
	if len(args) < 2 {
		return nil
	}

	// First arg is the prefix
	prefixArg := args[0]
	prefix := ""
	if prefixArg.Type() == "string" || prefixArg.Type() == "template_string" {
		prefix, _ = p.tsParser.ExtractStringLiteral(prefixArg, content)
	}

	if prefix == "" {
		return nil
	}

	// Second arg is the callback function
	callbackArg := args[1]
	if callbackArg.Type() != "arrow_function" && callbackArg.Type() != "function_expression" {
		return nil
	}

	// Find route calls within the callback
	var routes []types.Route
	p.walkNodes(callbackArg, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			callee := n.Child(0)
			if callee != nil && callee.Type() == "member_expression" {
				_, method := p.tsParser.GetMemberExpressionParts(callee, content)
				if httpMethod, isHTTP := httpMethods[strings.ToLower(method)]; isHTTP {
					// Extract route info
					innerArgs := p.tsParser.GetCallArguments(n, content)
					if len(innerArgs) > 0 {
						pathArg := innerArgs[0]
						path := ""
						if pathArg.Type() == "string" || pathArg.Type() == "template_string" {
							path, _ = p.tsParser.ExtractStringLiteral(pathArg, content)
						}

						if path != "" {
							fullPath := combinePaths(prefix, path)
							fullPath = convertPathParams(fullPath)
							params := extractPathParams(fullPath)
							tags := inferTags(fullPath)
							operationID := generateOperationID(httpMethod, fullPath, "")

							var requestBody *types.RequestBody
							if len(innerArgs) >= 2 {
								lastArg := innerArgs[len(innerArgs)-1]
								if lastArg.Type() == "object" {
									requestBody = p.extractValidationSchema(lastArg, content, zodSchemas)
								}
							}

							route := types.Route{
								Method:      httpMethod,
								Path:        fullPath,
								OperationID: operationID,
								Tags:        tags,
								Parameters:  params,
								RequestBody: requestBody,
								SourceLine:  int(n.StartPoint().Row) + 1,
							}
							routes = append(routes, route)
						}
					}
				}
			}
		}
		return true
	})

	return routes
}

// extractValidationSchema extracts validation schema from Elysia's options object.
// TODO: Use zodSchemas to resolve and validate schema references.
func (p *Plugin) extractValidationSchema(
	optionsNode *sitter.Node,
	content []byte,
	_ map[string]*sitter.Node,
) *types.RequestBody {
	var requestBody *types.RequestBody

	p.walkNodes(optionsNode, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			key := ""
			var valueNode *sitter.Node

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "property_identifier" {
					key = child.Content(content)
				} else if child.Type() != ":" {
					valueNode = child
				}
			}

			if key == "body" && valueNode != nil {
				bodySchema := p.extractTypeBoxOrZodSchema(valueNode, content)
				if bodySchema != nil {
					requestBody = &types.RequestBody{
						Required: true,
						Content: map[string]types.MediaType{
							"application/json": {Schema: bodySchema},
						},
					}
				}
				return false
			}
		}
		return true
	})

	return requestBody
}

// extractTypeBoxOrZodSchema extracts a schema from TypeBox (t.Object) or Zod expressions.
func (p *Plugin) extractTypeBoxOrZodSchema(node *sitter.Node, content []byte) *types.Schema {
	if node == nil {
		return nil
	}

	// Handle identifier reference
	if node.Type() == "identifier" {
		schemaName := node.Content(content)
		return schema.SchemaRef(schemaName)
	}

	// Handle call expression (t.Object(...) or z.object(...))
	if node.Type() == "call_expression" {
		calleeText := p.tsParser.GetCalleeText(node, content)

		// TypeBox patterns: t.Object, t.String, t.Number, etc.
		if strings.HasPrefix(calleeText, "t.") {
			return p.parseTypeBoxSchema(node, content)
		}

		// Zod patterns: z.object, z.string, etc.
		if strings.HasPrefix(calleeText, "z.") {
			parsedSchema, _ := p.zodParser.ParseZodSchema(node, content)
			return parsedSchema
		}
	}

	return nil
}

// parseTypeBoxSchema parses TypeBox schema definitions.
func (p *Plugin) parseTypeBoxSchema(node *sitter.Node, content []byte) *types.Schema {
	calleeText := p.tsParser.GetCalleeText(node, content)
	args := p.tsParser.GetCallArguments(node, content)

	// Extract TypeBox method name
	method := strings.TrimPrefix(calleeText, "t.")
	// Handle chained calls like t.String().optional()
	if idx := strings.Index(method, "."); idx > 0 {
		method = method[:idx]
	}

	switch method {
	case "String":
		return &types.Schema{Type: "string"}
	case "Number":
		return &types.Schema{Type: "number"}
	case "Integer":
		return &types.Schema{Type: "integer"}
	case "Boolean":
		return &types.Schema{Type: "boolean"}
	case "Array":
		schema := &types.Schema{Type: "array"}
		if len(args) > 0 {
			schema.Items = p.extractTypeBoxOrZodSchema(args[0], content)
		}
		return schema
	case "Object":
		schema := &types.Schema{
			Type:       "object",
			Properties: make(map[string]*types.Schema),
		}
		if len(args) > 0 && args[0].Type() == "object" {
			schema.Properties = p.extractTypeBoxProperties(args[0], content)
		}
		return schema
	case "Optional":
		if len(args) > 0 {
			return p.extractTypeBoxOrZodSchema(args[0], content)
		}
		return &types.Schema{}
	case "Nullable":
		schema := &types.Schema{Nullable: true}
		if len(args) > 0 {
			inner := p.extractTypeBoxOrZodSchema(args[0], content)
			if inner != nil {
				inner.Nullable = true
				return inner
			}
		}
		return schema
	case "Literal":
		if len(args) > 0 {
			literalValue := args[0].Content(content)
			literalValue = strings.Trim(literalValue, `"'`)
			return &types.Schema{
				Type: "string",
				Enum: []any{literalValue},
			}
		}
		return &types.Schema{}
	case "Union":
		if len(args) > 0 && args[0].Type() == "array" {
			var oneOf []*types.Schema
			p.walkNodes(args[0], func(n *sitter.Node) bool {
				if n.Type() == "call_expression" {
					itemSchema := p.extractTypeBoxOrZodSchema(n, content)
					if itemSchema != nil {
						oneOf = append(oneOf, itemSchema)
					}
					return false
				}
				return true
			})
			return &types.Schema{OneOf: oneOf}
		}
		return &types.Schema{}
	default:
		return &types.Schema{}
	}
}

// extractTypeBoxProperties extracts properties from a TypeBox Object definition.
func (p *Plugin) extractTypeBoxProperties(node *sitter.Node, content []byte) map[string]*types.Schema {
	props := make(map[string]*types.Schema)

	p.walkNodes(node, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			propName := ""
			var propValueNode *sitter.Node

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "property_identifier" || child.Type() == "string" {
					if propName == "" {
						propName = strings.Trim(child.Content(content), `"'`)
					}
				} else if child.Type() != ":" {
					propValueNode = child
				}
			}

			if propName != "" && propValueNode != nil {
				props[propName] = p.extractTypeBoxOrZodSchema(propValueNode, content)
			}
			return false
		}
		return true
	})

	return props
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

// colonParamRegex matches path parameters in the format :param.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts Elysia-style path params (:id) to OpenAPI format ({id}).
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

// Register registers the Elysia plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
