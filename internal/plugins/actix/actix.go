// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package actix provides a plugin for extracting routes from Actix-web framework applications.
package actix

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
	"trace":   "TRACE",
}

// Plugin implements the FrameworkPlugin interface for Actix-web framework.
type Plugin struct {
	rustParser *parser.RustParser
}

// New creates a new Actix plugin instance.
func New() *Plugin {
	return &Plugin{
		rustParser: parser.NewRustParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "actix"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".rs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "actix",
		Version:     "1.0.0",
		Description: "Extracts routes from Actix-web framework applications",
		SupportedFrameworks: []string{
			"actix-web",
		},
	}
}

// Detect checks if Actix-web is used in the project by examining Cargo.toml.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	cargoPath := filepath.Join(projectRoot, "Cargo.toml")
	return p.checkCargoForDependency(cargoPath, "actix-web")
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

// ExtractRoutes parses source files and extracts Actix-web route definitions.
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

	// Check if this file uses Actix-web
	if !p.hasActixImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Extract routes from attribute macros on functions (#[get("/path")])
	for _, fn := range pf.Functions {
		fnRoutes := p.extractRoutesFromFunction(fn)
		for i := range fnRoutes {
			fnRoutes[i].SourceFile = file.Path
			routes = append(routes, fnRoutes[i])
		}
	}

	// TODO: Extract routes from web::resource() and App::route() calls

	return routes, nil
}

// hasActixImport checks if the file imports Actix-web.
func (p *Plugin) hasActixImport(pf *parser.ParsedRustFile) bool {
	for _, use := range pf.Uses {
		if strings.Contains(use.Path, "actix_web") || strings.Contains(use.Path, "actix-web") {
			return true
		}
	}
	return false
}

// extractRoutesFromFunction extracts routes from function attributes.
func (p *Plugin) extractRoutesFromFunction(fn parser.RustFunction) []types.Route {
	var routes []types.Route

	for _, attr := range fn.Attributes {
		// Check if attribute is an HTTP method macro
		httpMethod, ok := httpMethods[strings.ToLower(attr.Name)]
		if !ok {
			continue
		}

		// Extract path from attribute arguments
		path := p.extractPathFromAttribute(attr)
		if path == "" {
			continue
		}

		// Convert to OpenAPI format (Actix uses {param} already)
		fullPath := normalizePath(path)

		// Extract path parameters
		params := extractPathParams(fullPath)

		// Generate operation ID
		operationID := generateOperationID(httpMethod, fullPath, fn.Name)

		// Infer tags from path
		tags := inferTags(fullPath)

		route := types.Route{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     fn.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceLine:  fn.Line,
		}

		// Extract request body type from function parameters
		requestBody := p.extractRequestBody(fn)
		if requestBody != nil {
			route.RequestBody = requestBody
		}

		routes = append(routes, route)
	}

	return routes
}

// extractPathFromAttribute extracts the path from an attribute.
func (p *Plugin) extractPathFromAttribute(attr parser.RustAttribute) string {
	// The path is typically the first argument: #[get("/path")]
	if len(attr.Arguments) == 0 {
		return ""
	}

	// Extract path from the raw attribute or arguments
	// Arguments might be like: "/users/{id}"
	for _, arg := range attr.Arguments {
		// Try to extract a string literal
		pathRegex := regexp.MustCompile(`"([^"]+)"`)
		matches := pathRegex.FindStringSubmatch(arg)
		if len(matches) > 1 {
			return matches[1]
		}

		// If no quotes, the argument itself might be the path
		if strings.HasPrefix(arg, "/") {
			return arg
		}
	}

	// Also check raw attribute text
	pathRegex := regexp.MustCompile(`\[\s*\w+\s*\(\s*"([^"]+)"`)
	matches := pathRegex.FindStringSubmatch(attr.Raw)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// extractRequestBody extracts request body type from function parameters.
func (p *Plugin) extractRequestBody(fn parser.RustFunction) *types.RequestBody {
	for _, param := range fn.Parameters {
		if param.IsSelf {
			continue
		}

		// Check for web::Json<T>, Json<T>
		if strings.Contains(param.Type, "Json<") {
			innerType := extractGenericType(param.Type)
			if innerType != "" {
				return &types.RequestBody{
					Required: true,
					Content: map[string]types.MediaType{
						"application/json": {
							Schema: &types.Schema{
								Ref: "#/components/schemas/" + innerType,
							},
						},
					},
				}
			}
		}
	}

	return nil
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

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

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

// extractGenericType extracts the inner type from a generic like Vec<String>.
func extractGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// Register registers the Actix plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
