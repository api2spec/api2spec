// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package aspnet provides a plugin for extracting routes from ASP.NET Core applications.
package aspnet

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

// httpMethods maps HTTP attribute names to their uppercase forms.
var httpMethods = map[string]string{
	"HttpGet":     "GET",
	"HttpPost":    "POST",
	"HttpPut":     "PUT",
	"HttpDelete":  "DELETE",
	"HttpPatch":   "PATCH",
	"HttpHead":    "HEAD",
	"HttpOptions": "OPTIONS",
}

// Plugin implements the FrameworkPlugin interface for ASP.NET Core.
type Plugin struct {
	csParser *parser.CSharpParser
}

// New creates a new ASP.NET Core plugin instance.
func New() *Plugin {
	return &Plugin{
		csParser: parser.NewCSharpParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "aspnet"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "aspnet",
		Version:     "1.0.0",
		Description: "Extracts routes from ASP.NET Core applications",
		SupportedFrameworks: []string{
			"Microsoft.AspNetCore",
			"ASP.NET Core",
		},
	}
}

// Detect checks if ASP.NET Core is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Look for .csproj files with AspNetCore reference
	csprojFiles, err := filepath.Glob(filepath.Join(projectRoot, "*.csproj"))
	if err != nil {
		return false, err
	}

	for _, csproj := range csprojFiles {
		if found, _ := p.checkCsprojForAspNet(csproj); found {
			return true, nil
		}
	}

	// Also check subdirectories
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return false, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subCsproj := filepath.Join(projectRoot, entry.Name(), entry.Name()+".csproj")
			if _, err := os.Stat(subCsproj); err == nil {
				if found, _ := p.checkCsprojForAspNet(subCsproj); found {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// checkCsprojForAspNet checks if a .csproj file references ASP.NET Core.
func (p *Plugin) checkCsprojForAspNet(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer func() { _ = file.Close() }()

	scanr := bufio.NewScanner(file)
	for scanr.Scan() {
		line := scanr.Text()
		if strings.Contains(line, "Microsoft.AspNetCore") ||
			strings.Contains(line, "Microsoft.NET.Sdk.Web") {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts ASP.NET Core route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "csharp" {
			continue
		}

		fileRoutes := p.extractRoutesFromFile(file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single C# file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) []types.Route {
	pf := p.csParser.Parse(file.Path, file.Content)

	var routes []types.Route

	// Extract routes from controllers
	for _, class := range pf.Classes {
		if !p.isController(class) {
			continue
		}

		classRoutes := p.extractRoutesFromController(class, file.Path)
		routes = append(routes, classRoutes...)
	}

	// Extract minimal API routes
	for _, route := range pf.MinimalAPIRoutes {
		r := p.convertMinimalRoute(route, file.Path)
		if r != nil {
			routes = append(routes, *r)
		}
	}

	return routes
}

// isController checks if a class is an ASP.NET Core controller.
func (p *Plugin) isController(class parser.CSharpClass) bool {
	// Check for [ApiController] or [Controller] attribute
	if class.HasAttribute("ApiController") || class.HasAttribute("Controller") {
		return true
	}

	// Check if inherits from ControllerBase or Controller
	return class.IsControllerBase()
}

// extractRoutesFromController extracts routes from a controller class.
func (p *Plugin) extractRoutesFromController(class parser.CSharpClass, filePath string) []types.Route {
	var routes []types.Route

	// Get base route from [Route] attribute on class
	baseRoute := ""
	if routeAttr := class.GetAttribute("Route"); routeAttr != nil {
		if len(routeAttr.Arguments) > 0 {
			baseRoute = routeAttr.Arguments[0]
		}
	}

	// Replace [controller] placeholder with controller name
	controllerName := strings.TrimSuffix(class.Name, "Controller")
	baseRoute = strings.ReplaceAll(baseRoute, "[controller]", strings.ToLower(controllerName))

	// Extract routes from methods
	for _, method := range class.Methods {
		methodRoutes := p.extractRoutesFromMethod(method, baseRoute, controllerName, filePath)
		routes = append(routes, methodRoutes...)
	}

	return routes
}

// extractRoutesFromMethod extracts routes from a controller method.
func (p *Plugin) extractRoutesFromMethod(method parser.CSharpMethod, baseRoute, controllerName, filePath string) []types.Route {
	var routes []types.Route

	for _, attr := range method.Attributes {
		httpMethod, ok := httpMethods[attr.Name]
		if !ok {
			continue
		}

		// Get path from attribute or use method name
		path := ""
		if len(attr.Arguments) > 0 {
			path = attr.Arguments[0]
		}

		// Combine base route and method path
		fullPath := combinePaths(baseRoute, path)

		// Replace [action] placeholder
		fullPath = strings.ReplaceAll(fullPath, "[action]", strings.ToLower(method.Name))

		// Convert to OpenAPI format
		fullPath = convertAspNetPathParams(fullPath)

		// Extract path parameters
		params := extractPathParams(fullPath)

		// Check for body parameters
		for _, param := range method.Parameters {
			if hasFromBodyAttribute(param) {
				// Request body parameter found - type extraction is a future enhancement
				_ = param // Mark as intentionally unused
			}
		}

		// Generate operation ID
		operationID := generateOperationID(httpMethod, fullPath, method.Name)

		// Infer tags
		tags := []string{controllerName}

		routes = append(routes, types.Route{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     controllerName + "." + method.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceFile:  filePath,
			SourceLine:  method.Line,
		})
	}

	return routes
}

// convertMinimalRoute converts a minimal API route to a types.Route.
func (p *Plugin) convertMinimalRoute(route parser.CSharpMinimalRoute, filePath string) *types.Route {
	fullPath := convertAspNetPathParams(route.Path)
	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, "")
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      route.Method,
		Path:        fullPath,
		Handler:     route.Handler,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// hasFromBodyAttribute checks if a parameter has [FromBody] attribute.
func hasFromBodyAttribute(param parser.CSharpParameter) bool {
	for _, attr := range param.Attributes {
		if attr.Name == "FromBody" {
			return true
		}
	}
	return false
}

// aspnetParamRegex matches ASP.NET route parameters like {id} or {id:int}.
var aspnetParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]+)?\}`)

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}:]+)\}`)

// convertAspNetPathParams converts ASP.NET-style path params ({id:int}) to OpenAPI format ({id}).
func convertAspNetPathParams(path string) string {
	// Remove type constraints
	return aspnetParamRegex.ReplaceAllString(path, "{$1}")
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

// combinePaths combines a base path and a relative path.
func combinePaths(base, relative string) string {
	if base == "" {
		if relative == "" {
			return "/"
		}
		if !strings.HasPrefix(relative, "/") {
			return "/" + relative
		}
		return relative
	}

	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}

	if relative == "" {
		return base
	}

	base = strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(relative, "/") {
		relative = "/" + relative
	}

	return base + relative
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" {
		return strings.ToLower(method) + handler
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

// ExtractSchemas extracts schema definitions from C# DTOs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "csharp" {
			continue
		}

		pf := p.csParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			// Skip controllers
			if p.isController(class) {
				continue
			}

			// Check if this looks like a DTO
			if strings.HasSuffix(class.Name, "Dto") ||
				strings.HasSuffix(class.Name, "DTO") ||
				strings.HasSuffix(class.Name, "Request") ||
				strings.HasSuffix(class.Name, "Response") ||
				strings.HasSuffix(class.Name, "Model") {

				schema := p.classToSchema(class)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// classToSchema converts a C# class to an OpenAPI schema.
func (p *Plugin) classToSchema(class parser.CSharpClass) *types.Schema {
	schema := &types.Schema{
		Title:      class.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Extract properties from methods that look like getters
	// In a real implementation, we'd parse properties directly
	for _, method := range class.Methods {
		if strings.HasPrefix(method.Name, "get_") {
			propName := strings.TrimPrefix(method.Name, "get_")
			openAPIType, format := parser.CSharpTypeToOpenAPI(method.ReturnType)

			propSchema := &types.Schema{
				Type:   openAPIType,
				Format: format,
			}

			schema.Properties[propName] = propSchema
		}
	}

	return schema
}

// Register registers the ASP.NET Core plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
