// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package fastify provides a plugin for extracting routes from Fastify framework applications.
package fastify

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

// Plugin implements the FrameworkPlugin interface for Fastify framework.
type Plugin struct {
	tsParser  *parser.TypeScriptParser
	zodParser *schema.ZodParser
}

// New creates a new Fastify plugin instance.
func New() *Plugin {
	tsParser := parser.NewTypeScriptParser()
	return &Plugin{
		tsParser:  tsParser,
		zodParser: schema.NewZodParser(tsParser),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "fastify"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "fastify",
		Version:     "1.0.0",
		Description: "Extracts routes from Fastify framework applications",
		SupportedFrameworks: []string{
			"fastify",
		},
	}
}

// Detect checks if Fastify is used in the project by looking at package.json.
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

	// Check for fastify in dependencies or devDependencies
	if _, ok := pkg.Dependencies["fastify"]; ok {
		return true, nil
	}
	if _, ok := pkg.DevDependencies["fastify"]; ok {
		return true, nil
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Fastify route definitions.
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

// fastifyInfo tracks information about a Fastify instance variable.
type fastifyInfo struct {
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

	// Check if this file imports Fastify
	if !p.hasFastifyImport(pf.RootNode, file.Content) {
		return nil, nil
	}

	var routes []types.Route

	// Build a map of Zod schema names to their nodes
	zodSchemas := make(map[string]*sitter.Node)
	for _, zs := range pf.ZodSchemas {
		zodSchemas[zs.Name] = zs.Node
	}

	// Track fastify instances and their prefixes
	fastifyInstances := p.findFastifyInstances(pf.RootNode, file.Content)

	// Track plugin registrations with prefixes
	pluginPrefixes := p.findPluginPrefixes(pf.RootNode, file.Content, fastifyInstances)

	// Find all call expressions
	calls := p.tsParser.FindCallExpressions(pf.RootNode, file.Content)

	for _, call := range calls {
		extractedRoutes := p.extractRoutesFromCall(call, file.Content, fastifyInstances, pluginPrefixes, zodSchemas)
		for i := range extractedRoutes {
			extractedRoutes[i].SourceFile = file.Path
			routes = append(routes, extractedRoutes[i])
		}
	}

	return routes, nil
}

// hasFastifyImport checks if the file imports from 'fastify'.
func (p *Plugin) hasFastifyImport(rootNode *sitter.Node, content []byte) bool {
	hasImport := false

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check for ES6 import statements
		if node.Type() == "import_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "string" {
					source := child.Content(content)
					source = strings.Trim(source, `"'`)
					if source == "fastify" || strings.HasPrefix(source, "fastify/") ||
						strings.HasPrefix(source, "@fastify/") {
						hasImport = true
						return false
					}
				}
			}
		}

		// Check for require() calls
		if node.Type() == "call_expression" {
			calleeText := p.tsParser.GetCalleeText(node, content)
			if calleeText == "require" {
				args := p.tsParser.GetCallArguments(node, content)
				if len(args) > 0 {
					argText := args[0].Content(content)
					argText = strings.Trim(argText, `"'`)
					if argText == "fastify" || strings.HasPrefix(argText, "fastify/") ||
						strings.HasPrefix(argText, "@fastify/") {
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

// findFastifyInstances finds variables that are Fastify instances.
func (p *Plugin) findFastifyInstances(rootNode *sitter.Node, content []byte) map[string]*fastifyInfo {
	instances := make(map[string]*fastifyInfo)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "lexical_declaration" || node.Type() == "variable_declaration" {
			p.walkNodes(node, func(n *sitter.Node) bool {
				if n.Type() == "variable_declarator" {
					var name string
					var isFastify bool

					for i := 0; i < int(n.ChildCount()); i++ {
						child := n.Child(i)
						if child.Type() == "identifier" && name == "" {
							name = child.Content(content)
						}
						if child.Type() == "call_expression" {
							calleeText := p.tsParser.GetCalleeText(child, content)
							// Check for Fastify(), fastify(), or Fastify.default()
							if calleeText == "Fastify" || calleeText == "fastify" ||
								calleeText == "Fastify.default" || calleeText == "fastify.default" {
								isFastify = true
							}
						}
						if child.Type() == "await_expression" {
							// Handle await fastify() pattern
							p.walkNodes(child, func(c *sitter.Node) bool {
								if c.Type() == "call_expression" {
									calleeText := p.tsParser.GetCalleeText(c, content)
									if calleeText == "Fastify" || calleeText == "fastify" {
										isFastify = true
									}
								}
								return true
							})
						}
					}

					if name != "" && isFastify {
						instances[name] = &fastifyInfo{name: name}
					}
				}
				return true
			})
		}
		return true
	})

	// Default to common names if not found
	if len(instances) == 0 {
		instances["fastify"] = &fastifyInfo{name: "fastify"}
		instances["app"] = &fastifyInfo{name: "app"}
		instances["server"] = &fastifyInfo{name: "server"}
	}

	return instances
}

// findPluginPrefixes finds fastify.register(routes, { prefix: '/api' }) calls.
func (p *Plugin) findPluginPrefixes(rootNode *sitter.Node, content []byte, instances map[string]*fastifyInfo) map[string]string {
	prefixes := make(map[string]string)

	calls := p.tsParser.FindCallExpressions(rootNode, content)

	for _, call := range calls {
		callee := call.Child(0)
		if callee == nil || callee.Type() != "member_expression" {
			continue
		}

		object, method := p.tsParser.GetMemberExpressionParts(callee, content)
		if method != "register" {
			continue
		}

		// Check if the object is a known Fastify instance
		if _, ok := instances[object]; !ok {
			continue
		}

		args := p.tsParser.GetCallArguments(call, content)
		if len(args) < 2 {
			continue
		}

		// Second arg should be options object with prefix
		optionsArg := args[1]
		if optionsArg.Type() == "object" {
			prefix := p.extractPrefixFromObject(optionsArg, content)
			if prefix != "" {
				// First arg is the plugin/routes
				pluginArg := args[0]
				if pluginArg.Type() == "identifier" {
					pluginName := pluginArg.Content(content)
					prefixes[pluginName] = prefix
				}
			}
		}
	}

	return prefixes
}

// extractPrefixFromObject extracts the prefix property from an options object.
func (p *Plugin) extractPrefixFromObject(objNode *sitter.Node, content []byte) string {
	var prefix string

	p.walkNodes(objNode, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			var key, value string
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "property_identifier" {
					key = child.Content(content)
				}
				if child.Type() == "string" {
					value = child.Content(content)
					value = strings.Trim(value, `"'`)
				}
			}
			if key == "prefix" {
				prefix = value
				return false
			}
		}
		return true
	})

	return prefix
}

// extractRoutesFromCall extracts routes from a call expression.
func (p *Plugin) extractRoutesFromCall(
	node *sitter.Node,
	content []byte,
	instances map[string]*fastifyInfo,
	pluginPrefixes map[string]string,
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

	// Check for fastify.route() method
	if method == "route" {
		return p.extractRouteFromRouteMethod(node, content, instances, pluginPrefixes, zodSchemas)
	}

	// Check if method is an HTTP method
	httpMethod, isHTTPMethod := httpMethods[strings.ToLower(method)]
	if !isHTTPMethod {
		return nil
	}

	// Check if object is a known Fastify instance
	if _, ok := instances[object]; !ok {
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

	// Convert Fastify path parameters (:param) to OpenAPI format ({param})
	path = convertPathParams(path)

	// Extract path parameters
	params := extractPathParams(path)

	// Look for schema option in second argument (options object)
	var requestBody *types.RequestBody
	var responseSchemas map[int]*types.Schema

	if len(args) >= 2 {
		optionsArg := args[1]
		if optionsArg.Type() == "object" {
			requestBody, responseSchemas = p.extractSchemasFromOptions(optionsArg, content, zodSchemas)
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

	// Add response schemas if available
	if len(responseSchemas) > 0 {
		route.Responses = make(map[string]types.Response)
		for status, s := range responseSchemas {
			route.Responses[fmt.Sprintf("%d", status)] = types.Response{
				Description: fmt.Sprintf("Response %d", status),
				Content: map[string]types.MediaType{
					"application/json": {Schema: s},
				},
			}
		}
	}

	return []types.Route{route}
}

// extractRouteFromRouteMethod handles fastify.route({ method, url, schema, handler }) pattern.
func (p *Plugin) extractRouteFromRouteMethod(
	node *sitter.Node,
	content []byte,
	_ map[string]*fastifyInfo,
	_ map[string]string,
	zodSchemas map[string]*sitter.Node,
) []types.Route {
	args := p.tsParser.GetCallArguments(node, content)
	if len(args) == 0 {
		return nil
	}

	// First argument should be the route options object
	optionsArg := args[0]
	if optionsArg.Type() != "object" {
		return nil
	}

	var method, url string
	var methods []string
	var requestBody *types.RequestBody
	var responseSchemas map[int]*types.Schema

	p.walkNodes(optionsArg, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			key := ""
			var valueNode *sitter.Node

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				if child.Type() == "property_identifier" {
					key = child.Content(content)
				} else if child.Type() == "string" && key == "" {
					// Skip, this might be a value
				} else if child.Type() != ":" {
					valueNode = child
				}
			}

			switch key {
			case "method":
				if valueNode != nil {
					if valueNode.Type() == "string" {
						method = strings.Trim(valueNode.Content(content), `"'`)
					} else if valueNode.Type() == "array" {
						// Handle array of methods
						p.walkNodes(valueNode, func(m *sitter.Node) bool {
							if m.Type() == "string" {
								methods = append(methods, strings.Trim(m.Content(content), `"'`))
							}
							return true
						})
					}
				}
			case "url":
				if valueNode != nil && valueNode.Type() == "string" {
					url = strings.Trim(valueNode.Content(content), `"'`)
				}
			case "schema":
				if valueNode != nil && valueNode.Type() == "object" {
					requestBody, responseSchemas = p.extractSchemasFromSchemaObject(valueNode, content, zodSchemas)
				}
			}
			return false
		}
		return true
	})

	if url == "" {
		return nil
	}

	// Convert path parameters
	url = convertPathParams(url)
	params := extractPathParams(url)
	tags := inferTags(url)

	var routes []types.Route

	// Handle single method or array of methods
	if method != "" {
		methods = append(methods, method)
	}

	for _, m := range methods {
		httpMethod := strings.ToUpper(m)
		operationID := generateOperationID(httpMethod, url, "")

		route := types.Route{
			Method:      httpMethod,
			Path:        url,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			RequestBody: requestBody,
			SourceLine:  int(node.StartPoint().Row) + 1,
		}

		if len(responseSchemas) > 0 {
			route.Responses = make(map[string]types.Response)
			for status, s := range responseSchemas {
				route.Responses[fmt.Sprintf("%d", status)] = types.Response{
					Description: fmt.Sprintf("Response %d", status),
					Content: map[string]types.MediaType{
						"application/json": {Schema: s},
					},
				}
			}
		}

		routes = append(routes, route)
	}

	return routes
}

// extractSchemasFromOptions extracts body and response schemas from route options.
func (p *Plugin) extractSchemasFromOptions(
	optionsNode *sitter.Node,
	content []byte,
	zodSchemas map[string]*sitter.Node,
) (*types.RequestBody, map[int]*types.Schema) {
	var requestBody *types.RequestBody
	responseSchemas := make(map[int]*types.Schema)

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

			if key == "schema" && valueNode != nil && valueNode.Type() == "object" {
				requestBody, responseSchemas = p.extractSchemasFromSchemaObject(valueNode, content, zodSchemas)
				return false
			}
		}
		return true
	})

	return requestBody, responseSchemas
}

// extractSchemasFromSchemaObject extracts schemas from Fastify's schema option object.
// TODO: Use zodSchemas to resolve and validate schema references.
func (p *Plugin) extractSchemasFromSchemaObject(
	schemaNode *sitter.Node,
	content []byte,
	_ map[string]*sitter.Node,
) (*types.RequestBody, map[int]*types.Schema) {
	var requestBody *types.RequestBody
	responseSchemas := make(map[int]*types.Schema)

	p.walkNodes(schemaNode, func(n *sitter.Node) bool {
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

			switch key {
			case "body":
				if valueNode != nil {
					bodySchema := p.extractJSONSchema(valueNode, content)
					if bodySchema != nil {
						requestBody = &types.RequestBody{
							Required: true,
							Content: map[string]types.MediaType{
								"application/json": {Schema: bodySchema},
							},
						}
					}
				}
			case "response":
				if valueNode != nil && valueNode.Type() == "object" {
					// Parse response schemas by status code
					p.walkNodes(valueNode, func(r *sitter.Node) bool {
						if r.Type() == "pair" || r.Type() == "property" {
							var statusCode int
							var respValueNode *sitter.Node

							for i := 0; i < int(r.ChildCount()); i++ {
								child := r.Child(i)
								if child.Type() == "property_identifier" || child.Type() == "number" {
									// Parse status code
									statusStr := child.Content(content)
									_, err := fmt.Sscanf(statusStr, "%d", &statusCode)
									if err != nil {
										continue
									}
								} else if child.Type() != ":" {
									respValueNode = child
								}
							}

							if statusCode > 0 && respValueNode != nil {
								respSchema := p.extractJSONSchema(respValueNode, content)
								if respSchema != nil {
									responseSchemas[statusCode] = respSchema
								}
							}
							return false
						}
						return true
					})
				}
			}
			return false
		}
		return true
	})

	return requestBody, responseSchemas
}

// extractJSONSchema extracts a JSON Schema from a Fastify schema definition.
func (p *Plugin) extractJSONSchema(node *sitter.Node, content []byte) *types.Schema {
	if node == nil {
		return nil
	}

	// Handle identifier reference
	if node.Type() == "identifier" {
		schemaName := node.Content(content)
		return schema.SchemaRef(schemaName)
	}

	// Handle inline object schema
	if node.Type() != "object" {
		return nil
	}

	result := &types.Schema{
		Properties: make(map[string]*types.Schema),
	}

	p.walkNodes(node, func(n *sitter.Node) bool {
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

			switch key {
			case "type":
				if valueNode != nil && valueNode.Type() == "string" {
					result.Type = strings.Trim(valueNode.Content(content), `"'`)
				}
			case "properties":
				if valueNode != nil && valueNode.Type() == "object" {
					result.Properties = p.extractJSONSchemaProperties(valueNode, content)
				}
			case "required":
				if valueNode != nil && valueNode.Type() == "array" {
					var required []string
					p.walkNodes(valueNode, func(r *sitter.Node) bool {
						if r.Type() == "string" {
							required = append(required, strings.Trim(r.Content(content), `"'`))
						}
						return true
					})
					result.Required = required
				}
			case "items":
				if valueNode != nil {
					result.Items = p.extractJSONSchema(valueNode, content)
				}
			case "description":
				if valueNode != nil && valueNode.Type() == "string" {
					result.Description = strings.Trim(valueNode.Content(content), `"'`)
				}
			case "format":
				if valueNode != nil && valueNode.Type() == "string" {
					result.Format = strings.Trim(valueNode.Content(content), `"'`)
				}
			case "enum":
				if valueNode != nil && valueNode.Type() == "array" {
					var enumVals []any
					p.walkNodes(valueNode, func(e *sitter.Node) bool {
						if e.Type() == "string" {
							enumVals = append(enumVals, strings.Trim(e.Content(content), `"'`))
						} else if e.Type() == "number" {
							enumVals = append(enumVals, e.Content(content))
						}
						return true
					})
					result.Enum = enumVals
				}
			}
			return false
		}
		return true
	})

	return result
}

// extractJSONSchemaProperties extracts properties from a JSON Schema properties object.
func (p *Plugin) extractJSONSchemaProperties(node *sitter.Node, content []byte) map[string]*types.Schema {
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
				props[propName] = p.extractJSONSchema(propValueNode, content)
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

// convertPathParams converts Fastify-style path params (:id) to OpenAPI format ({id}).
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

// Register registers the Fastify plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
