// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package axum provides a plugin for extracting routes from Axum framework applications.
package axum

import (
	"bufio"
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
	"github.com/api2spec/api2spec/pkg/types"
)

// httpMethods maps HTTP method function names to their uppercase forms.
var httpMethods = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"delete":  "DELETE",
	"patch":   "PATCH",
	"head":    "HEAD",
	"options": "OPTIONS",
	"trace":   "TRACE",
}

// Plugin implements the FrameworkPlugin interface for Axum framework.
type Plugin struct {
	rustParser *parser.RustParser
}

// New creates a new Axum plugin instance.
func New() *Plugin {
	return &Plugin{
		rustParser: parser.NewRustParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "axum"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".rs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "axum",
		Version:     "1.0.0",
		Description: "Extracts routes from Axum framework applications",
		SupportedFrameworks: []string{
			"axum",
		},
	}
}

// Detect checks if Axum is used in the project by examining Cargo.toml.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	cargoPath := filepath.Join(projectRoot, "Cargo.toml")
	return p.checkCargoForDependency(cargoPath, "axum")
}

// checkCargoForDependency checks if Cargo.toml contains a dependency.
func (p *Plugin) checkCargoForDependency(path, dep string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	depLower := strings.ToLower(dep)
	inDependencies := false

	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())

		// Track if we're in a dependencies section
		if strings.Contains(line, "[dependencies]") || strings.Contains(line, "[dev-dependencies]") {
			inDependencies = true
			continue
		}
		if strings.HasPrefix(line, "[") && !strings.Contains(line, "dependencies") {
			inDependencies = false
			continue
		}

		// Check for the dependency
		if inDependencies && strings.Contains(line, depLower) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Axum route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "rust" {
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

// extractRoutesFromFile extracts routes from a single Rust file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.rustParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file uses Axum
	if !p.hasAxumImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Extract routes from Router::new().route() chains
	routerRoutes := p.extractRouterRoutes(pf.RootNode, file.Content)
	for i := range routerRoutes {
		routerRoutes[i].SourceFile = file.Path
		routes = append(routes, routerRoutes[i])
	}

	return routes, nil
}

// hasAxumImport checks if the file imports Axum.
func (p *Plugin) hasAxumImport(pf *parser.ParsedRustFile) bool {
	for _, use := range pf.Uses {
		if strings.Contains(use.Path, "axum") {
			return true
		}
	}
	return false
}

// extractRouterRoutes extracts routes from Router building patterns.
func (p *Plugin) extractRouterRoutes(rootNode *sitter.Node, content []byte) []types.Route {
	var routes []types.Route

	// Look for .route() method calls
	p.rustParser.WalkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "call_expression" {
			nodeRoutes := p.parseRouteCall(node, content)
			routes = append(routes, nodeRoutes...)
		}
		return true
	})

	return routes
}

// parseRouteCall parses a .route() or HTTP method call.
func (p *Plugin) parseRouteCall(node *sitter.Node, content []byte) []types.Route {
	var routes []types.Route

	nodeText := node.Content(content)

	// Parse .route() calls by finding the start and extracting arguments
	// We need to handle nested parentheses properly
	routeStarts := findRouteStarts(nodeText)

	for _, start := range routeStarts {
		path, methodsStr := extractRouteArgs(nodeText[start:])
		if path == "" {
			continue
		}

		// Parse the methods from the route handler
		// e.g., get(handler).post(handler2)
		methodRoutes := p.parseMethodHandlers(path, methodsStr, node, content)
		routes = append(routes, methodRoutes...)
	}

	// Check for .nest("/prefix", router) pattern
	nestRegex := regexp.MustCompile(`\.nest\s*\(\s*"([^"]+)"\s*,`)
	nestMatches := nestRegex.FindAllStringSubmatch(nodeText, -1)
	for _, match := range nestMatches {
		if len(match) >= 2 {
			// TODO: Track nested router prefixes
			_ = match[1]
		}
	}

	return routes
}

// findRouteStarts finds all positions where ".route(" appears.
func findRouteStarts(text string) []int {
	var positions []int
	routePattern := regexp.MustCompile(`\.route\s*\(`)
	matches := routePattern.FindAllStringIndex(text, -1)
	for _, match := range matches {
		positions = append(positions, match[0])
	}
	return positions
}

// extractRouteArgs extracts the path and method handlers from a .route() call.
// It handles nested parentheses properly.
func extractRouteArgs(text string) (path string, methodsStr string) {
	// Find the opening parenthesis
	openParen := strings.Index(text, "(")
	if openParen == -1 {
		return "", ""
	}

	// Find the path string (first argument)
	pathStart := strings.Index(text[openParen:], "\"")
	if pathStart == -1 {
		return "", ""
	}
	pathStart += openParen + 1 // Move past the opening quote

	pathEnd := strings.Index(text[pathStart:], "\"")
	if pathEnd == -1 {
		return "", ""
	}

	path = text[pathStart : pathStart+pathEnd]

	// Find the comma after the path
	commaPos := pathStart + pathEnd + 1
	for commaPos < len(text) && text[commaPos] != ',' {
		commaPos++
	}
	if commaPos >= len(text) {
		return "", ""
	}

	// Now extract the method handlers - need to balance parentheses
	methodStart := commaPos + 1
	// Skip whitespace
	for methodStart < len(text) && (text[methodStart] == ' ' || text[methodStart] == '\t' || text[methodStart] == '\n') {
		methodStart++
	}

	// Find the matching closing parenthesis
	depth := 1 // We're inside the .route() call
	pos := methodStart
	for pos < len(text) && depth > 0 {
		if text[pos] == '(' {
			depth++
		} else if text[pos] == ')' {
			depth--
		}
		pos++
	}

	if depth == 0 {
		methodsStr = strings.TrimSpace(text[methodStart : pos-1])
	}

	return path, methodsStr
}

// parseMethodHandlers parses method handlers like get(handler).post(handler2).
func (p *Plugin) parseMethodHandlers(path, methodsStr string, node *sitter.Node, _ []byte) []types.Route {
	var routes []types.Route

	// Convert Axum :param to OpenAPI {param}
	fullPath := convertPathParams(path)
	params := extractPathParams(fullPath)
	tags := inferTags(fullPath)
	line := int(node.StartPoint().Row) + 1

	// Match method(handler) patterns
	methodRegex := regexp.MustCompile(`(get|post|put|delete|patch|head|options|trace)\s*\(\s*([^)]+)\)`)
	methodMatches := methodRegex.FindAllStringSubmatch(methodsStr, -1)

	for _, match := range methodMatches {
		if len(match) < 3 {
			continue
		}

		methodName := strings.ToLower(match[1])
		handlerName := strings.TrimSpace(match[2])

		httpMethod, ok := httpMethods[methodName]
		if !ok {
			continue
		}

		operationID := generateOperationID(httpMethod, fullPath, handlerName)

		route := types.Route{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     handlerName,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceLine:  line,
		}

		routes = append(routes, route)
	}

	return routes
}

// ExtractSchemas extracts schema definitions from Rust structs with serde.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "rust" {
			continue
		}

		pf, err := p.rustParser.Parse(file.Path, file.Content)
		if err != nil {
			continue
		}

		for _, s := range pf.Structs {
			// Only extract structs with Serialize or Deserialize derives
			if s.HasDeriveAttribute("Serialize") || s.HasDeriveAttribute("Deserialize") {
				schema := p.structToSchema(s)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}

		pf.Close()
	}

	return schemas, nil
}

// structToSchema converts a Rust struct to an OpenAPI schema.
func (p *Plugin) structToSchema(s parser.RustStruct) *types.Schema {
	schema := &types.Schema{
		Title:      s.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range s.Fields {
		propSchema := &types.Schema{}

		// Get the field name (possibly renamed by serde)
		fieldName := field.Name
		if rename := field.GetSerdeRename(); rename != "" {
			fieldName = rename
		}

		// Convert Rust type to OpenAPI type
		isOptional := strings.HasPrefix(field.Type, "Option<")
		openAPIType, format := parser.RustTypeToOpenAPI(field.Type)
		propSchema.Type = openAPIType
		if format != "" {
			propSchema.Format = format
		}

		// Handle Vec<T> types
		if strings.HasPrefix(field.Type, "Vec<") {
			propSchema.Type = "array"
			innerType := extractGenericType(field.Type)
			innerOpenAPIType, innerFormat := parser.RustTypeToOpenAPI(innerType)
			propSchema.Items = &types.Schema{
				Type:   innerOpenAPIType,
				Format: innerFormat,
			}
		}

		// Handle Option<T> types
		if isOptional {
			propSchema.Nullable = true
			innerType := extractGenericType(field.Type)
			innerOpenAPIType, innerFormat := parser.RustTypeToOpenAPI(innerType)
			propSchema.Type = innerOpenAPIType
			propSchema.Format = innerFormat
		}

		schema.Properties[fieldName] = propSchema

		if !isOptional {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return schema
}

// --- Helper Functions ---

// colonParamRegex matches Axum path parameters like :param.
var colonParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts Axum-style path params (:id) to OpenAPI format ({id}).
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

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" && handler != "<anonymous>" {
		// Clean up the handler name
		handler = strings.TrimSpace(handler)
		parts := strings.Split(handler, "::")
		handlerName := parts[len(parts)-1]
		return strings.ToLower(method) + toTitleCase(handlerName)
	}

	// Generate from path
	// Remove parameter syntax and convert to camelCase
	cleanPath := braceParamRegex.ReplaceAllString(path, "By${1}")
	cleanPath = strings.ReplaceAll(cleanPath, "/", " ")
	cleanPath = strings.TrimSpace(cleanPath)

	words := strings.Fields(cleanPath)
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

// toTitleCase converts the first character to uppercase.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
		if strings.HasPrefix(part, "{") || strings.HasPrefix(part, ":") {
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

// extractGenericType extracts the inner type from a generic like Vec<String>.
func extractGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// Register registers the Axum plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
