// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package rocket provides a plugin for extracting routes from Rocket framework applications.
package rocket

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

// httpMethods maps HTTP method attribute names to their uppercase forms.
var httpMethods = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"delete":  "DELETE",
	"patch":   "PATCH",
	"head":    "HEAD",
	"options": "OPTIONS",
}

// Plugin implements the FrameworkPlugin interface for Rocket framework.
type Plugin struct {
	rustParser *parser.RustParser
}

// New creates a new Rocket plugin instance.
func New() *Plugin {
	return &Plugin{
		rustParser: parser.NewRustParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "rocket"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".rs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "rocket",
		Version:     "1.0.0",
		Description: "Extracts routes from Rocket framework applications",
		SupportedFrameworks: []string{
			"rocket",
		},
	}
}

// Detect checks if Rocket is used in the project by examining Cargo.toml.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	cargoPath := filepath.Join(projectRoot, "Cargo.toml")
	return p.checkCargoForDependency(cargoPath, "rocket")
}

// checkCargoForDependency checks if Cargo.toml contains a dependency.
func (p *Plugin) checkCargoForDependency(path, dep string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer func() { _ = file.Close() }()

	scanr := bufio.NewScanner(file)
	depLower := strings.ToLower(dep)
	inDependencies := false

	for scanr.Scan() {
		line := strings.ToLower(scanr.Text())

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

// ExtractRoutes parses source files and extracts Rocket route definitions.
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

	// Check if this file uses Rocket
	if !p.hasRocketImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Track mount prefixes
	mountPrefixes := p.extractMountPrefixes(pf.RootNode, file.Content)
	_ = mountPrefixes // TODO: Apply mount prefixes to routes

	// Extract routes from function attributes
	for _, fn := range pf.Functions {
		fnRoutes := p.extractRoutesFromFunction(fn)
		for i := range fnRoutes {
			fnRoutes[i].SourceFile = file.Path
			routes = append(routes, fnRoutes[i])
		}
	}

	return routes, nil
}

// hasRocketImport checks if the file imports Rocket.
func (p *Plugin) hasRocketImport(pf *parser.ParsedRustFile) bool {
	for _, use := range pf.Uses {
		if strings.Contains(use.Path, "rocket") {
			return true
		}
	}
	return false
}

// extractMountPrefixes extracts mount prefixes from rocket::build().mount() calls.
func (p *Plugin) extractMountPrefixes(rootNode *sitter.Node, content []byte) map[string][]string {
	prefixes := make(map[string][]string)

	// Look for .mount("/prefix", routes![...]) patterns
	mountRegex := regexp.MustCompile(`\.mount\s*\(\s*"([^"]+)"\s*,\s*routes!\s*\[([^\]]+)\]`)
	matches := mountRegex.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) >= 3 {
			prefix := match[1]
			routeNames := strings.Split(match[2], ",")
			for _, name := range routeNames {
				name = strings.TrimSpace(name)
				if name != "" {
					prefixes[name] = append(prefixes[name], prefix)
				}
			}
		}
	}

	return prefixes
}

// extractRoutesFromFunction extracts routes from a function's attributes.
func (p *Plugin) extractRoutesFromFunction(fn parser.RustFunction) []types.Route {
	var routes []types.Route

	for _, attr := range fn.Attributes {
		route := p.parseRouteAttribute(attr, fn)
		if route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

// parseRouteAttribute parses a route attribute like #[get("/path")].
func (p *Plugin) parseRouteAttribute(attr parser.RustAttribute, fn parser.RustFunction) *types.Route {
	// Check for HTTP method attributes
	attrName := strings.ToLower(attr.Name)
	httpMethod, ok := httpMethods[attrName]
	if !ok {
		return nil
	}

	// Extract path from attribute arguments
	if len(attr.Arguments) == 0 {
		return nil
	}

	// Parse the attribute arguments
	// Format: "/path" or "/path", data = "<user>"
	argStr := attr.Arguments[0]
	path := extractPathFromArgs(argStr)
	if path == "" {
		return nil
	}

	// Convert Rocket <param> syntax to OpenAPI {param} syntax
	fullPath := convertRocketPathParams(path)

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Check for data parameter - indicates request body
	// Body type extraction is a future enhancement
	_ = strings.Contains(argStr, "data")

	// Generate operation ID
	operationID := generateOperationID(httpMethod, fullPath, fn.Name)

	// Infer tags from path
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      httpMethod,
		Path:        fullPath,
		Handler:     fn.Name,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceLine:  fn.Line,
	}
}

// extractPathFromArgs extracts the path from attribute arguments.
func extractPathFromArgs(args string) string {
	// Remove outer parentheses if present
	args = strings.TrimPrefix(args, "(")
	args = strings.TrimSuffix(args, ")")
	args = strings.TrimSpace(args)

	// Find the first quoted string
	if idx := strings.Index(args, `"`); idx >= 0 {
		end := strings.Index(args[idx+1:], `"`)
		if end >= 0 {
			return args[idx+1 : idx+1+end]
		}
	}

	return ""
}

// rocketParamRegex matches Rocket path parameters like <param> or <param..>.
var rocketParamRegex = regexp.MustCompile(`<([a-zA-Z_][a-zA-Z0-9_]*)(\.\.)?>`);

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertRocketPathParams converts Rocket-style path params (<id>) to OpenAPI format ({id}).
func convertRocketPathParams(path string) string {
	return rocketParamRegex.ReplaceAllString(path, "{$1}")
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
		return strings.ToLower(method) + toTitleCase(handler)
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
		if strings.HasPrefix(part, "{") || strings.HasPrefix(part, "<") {
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

// extractGenericType extracts the inner type from a generic like Vec<String>.
func extractGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// Register registers the Rocket plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
