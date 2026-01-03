// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package fastendpoints provides a plugin for extracting routes from FastEndpoints framework applications.
package fastendpoints

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

// Plugin implements the FrameworkPlugin interface for FastEndpoints framework.
type Plugin struct {
	csParser *parser.CSharpParser
}

// New creates a new FastEndpoints plugin instance.
func New() *Plugin {
	return &Plugin{
		csParser: parser.NewCSharpParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "fastendpoints"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "fastendpoints",
		Version:     "1.0.0",
		Description: "Extracts routes from FastEndpoints framework applications",
		SupportedFrameworks: []string{
			"FastEndpoints",
		},
	}
}

// Detect checks if FastEndpoints is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check for *.csproj files containing FastEndpoints
	csprojFiles, err := filepath.Glob(filepath.Join(projectRoot, "*.csproj"))
	if err != nil {
		return false, nil
	}

	for _, csprojPath := range csprojFiles {
		if found, _ := p.checkFileForDependency(csprojPath, "FastEndpoints"); found {
			return true, nil
		}
	}

	return false, nil
}

// checkFileForDependency checks if a file contains a dependency.
func (p *Plugin) checkFileForDependency(path, dep string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, nil
	}
	defer func() { _ = file.Close() }()

	scanr := bufio.NewScanner(file)
	for scanr.Scan() {
		line := scanr.Text()
		if strings.Contains(line, dep) {
			return true, nil
		}
	}

	return false, nil
}

// Regex patterns for FastEndpoints route extraction
var (
	// Matches class inheriting from Endpoint<TRequest, TResponse> or EndpointWithoutRequest
	endpointClassRegex = regexp.MustCompile(`class\s+(\w+)\s*:\s*(?:Endpoint(?:WithoutRequest)?(?:<([^>]+)>)?|EndpointWithMapping)`)

	// Matches Endpoint<TRequest> without response
	endpointNoResponseRegex = regexp.MustCompile(`class\s+(\w+)\s*:\s*Endpoint<(\w+)>`)

	// Matches Endpoint<TRequest, TResponse>
	endpointWithResponseRegex = regexp.MustCompile(`class\s+(\w+)\s*:\s*Endpoint<(\w+),\s*(\w+)>`)

	// Matches Get/Post/Put/Delete/Patch route configuration in Configure()
	httpMethodRegex = regexp.MustCompile(`(Get|Post|Put|Delete|Patch)\s*\(\s*"([^"]+)"`)

	// Matches Routes() method calls
	routesRegex = regexp.MustCompile(`Routes\s*\(\s*"([^"]+)"`)

	// Matches Verbs() method calls
	verbsRegex = regexp.MustCompile(`Verbs\s*\(\s*([^)]+)\)`)

	// Matches Http.GET, Http.POST etc.
	httpVerbRegex = regexp.MustCompile(`Http\.(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)`)

	// Matches AllowAnonymous() which often indicates public endpoints
	allowAnonymousRegex = regexp.MustCompile(`AllowAnonymous\s*\(\s*\)`)

	// Matches Summary() which provides documentation
	summaryRegex = regexp.MustCompile(`Summary\s*\(\s*s\s*=>\s*\{[^}]*s\.Summary\s*=\s*"([^"]+)"`)
)

// ExtractRoutes parses source files and extracts FastEndpoints route definitions.
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
	var routes []types.Route
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	// Find endpoint class
	endpointName := ""
	requestType := ""
	responseType := ""

	if match := endpointWithResponseRegex.FindStringSubmatch(content); len(match) > 3 {
		endpointName = match[1]
		requestType = match[2]
		responseType = match[3]
	} else if match := endpointNoResponseRegex.FindStringSubmatch(content); len(match) > 2 {
		endpointName = match[1]
		requestType = match[2]
	} else if match := endpointClassRegex.FindStringSubmatch(content); len(match) > 1 {
		endpointName = match[1]
		if len(match) > 2 && match[2] != "" {
			// Parse generic types
			types := strings.Split(match[2], ",")
			if len(types) > 0 {
				requestType = strings.TrimSpace(types[0])
			}
			if len(types) > 1 {
				responseType = strings.TrimSpace(types[1])
			}
		}
	}

	if endpointName == "" {
		return routes
	}

	// Find HTTP methods and paths
	for i, line := range lines {
		lineNum := i + 1

		// Check for Http method configuration: Get("/path"), Post("/path"), etc.
		if match := httpMethodRegex.FindStringSubmatch(line); len(match) > 2 {
			method := strings.ToUpper(match[1])
			path := match[2]

			route := p.createRoute(method, path, endpointName, requestType, responseType, file.Path, lineNum)
			routes = append(routes, route)
		}

		// Check for Routes() configuration
		if match := routesRegex.FindStringSubmatch(line); len(match) > 1 {
			path := match[1]

			// Look for Verbs() to determine HTTP methods
			methods := []string{"GET"} // Default
			if verbMatch := verbsRegex.FindStringSubmatch(content); len(verbMatch) > 1 {
				methods = parseVerbsMethods(verbMatch[1])
			}

			for _, method := range methods {
				route := p.createRoute(method, path, endpointName, requestType, responseType, file.Path, lineNum)
				routes = append(routes, route)
			}
		}
	}

	return routes
}

// createRoute creates a route from the extracted information.
func (p *Plugin) createRoute(method, path, endpointName, requestType, responseType, filePath string, lineNum int) types.Route {
	fullPath := ensureLeadingSlash(path)
	fullPath = convertFastEndpointsPathParams(fullPath)
	params := extractPathParams(fullPath)
	operationID := generateOperationID(method, fullPath, endpointName)
	tags := inferTags(fullPath, endpointName)

	route := types.Route{
		Method:      method,
		Path:        fullPath,
		Handler:     endpointName,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  lineNum,
	}

	// Add request body reference if POST/PUT/PATCH and request type exists
	if requestType != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		route.RequestBody = &types.RequestBody{
			Required: true,
			Content: map[string]types.MediaType{
				"application/json": {
					Schema: &types.Schema{
						Ref: "#/components/schemas/" + requestType,
					},
				},
			},
		}
	}

	// Add response reference if response type exists
	if responseType != "" {
		route.Responses = map[string]types.Response{
			"200": {
				Description: "Success",
				Content: map[string]types.MediaType{
					"application/json": {
						Schema: &types.Schema{
							Ref: "#/components/schemas/" + responseType,
						},
					},
				},
			},
		}
	}

	return route
}

// parseVerbsMethods parses HTTP methods from Verbs() call.
func parseVerbsMethods(s string) []string {
	var methods []string

	matches := httpVerbRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) > 1 {
			methods = append(methods, match[1])
		}
	}

	if len(methods) == 0 {
		return []string{"GET"}
	}

	return methods
}

// feParamRegex matches FastEndpoints path parameters like {param} or {param:int}.
var feParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]+)?\}`)

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}:]+)\}`)

// convertFastEndpointsPathParams converts FastEndpoints path params to OpenAPI format.
func convertFastEndpointsPathParams(path string) string {
	// Remove type constraints: {id:int} -> {id}
	return feParamRegex.ReplaceAllString(path, "{$1}")
}

// ensureLeadingSlash ensures the path has a leading slash.
func ensureLeadingSlash(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
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
	if handler != "" {
		// Remove "Endpoint" suffix for cleaner operation ID
		cleanHandler := strings.TrimSuffix(handler, "Endpoint")
		return strings.ToLower(method) + toTitleCase(cleanHandler)
	}

	cleanPath := braceParamRegex.ReplaceAllString(path, "By${1}")
	cleanPath = strings.ReplaceAll(cleanPath, "/", " ")
	cleanPath = strings.TrimSpace(cleanPath)

	words := strings.Fields(cleanPath)
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

// toTitleCase converts the first character to uppercase.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// inferTags infers tags from the route path and endpoint name.
func inferTags(path, endpointName string) []string {
	// Use endpoint name as tag if available
	if endpointName != "" {
		// Remove Endpoint suffix and extract resource name
		tag := strings.TrimSuffix(endpointName, "Endpoint")

		// Try to extract resource name from patterns like "GetUserEndpoint" -> "User"
		for _, prefix := range []string{"Get", "Create", "Update", "Delete", "List"} {
			if strings.HasPrefix(tag, prefix) {
				tag = strings.TrimPrefix(tag, prefix)
				break
			}
		}

		if tag != "" {
			return []string{tag}
		}
	}

	// Fall back to path-based tag
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		return nil
	}

	skipPrefixes := map[string]bool{
		"api": true,
		"v1":  true,
		"v2":  true,
		"v3":  true,
	}

	var tagPart string
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
		tagPart = part
		break
	}

	if tagPart == "" {
		return nil
	}

	return []string{tagPart}
}

// ExtractSchemas extracts schema definitions from Request/Response DTOs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "csharp" {
			continue
		}

		pf := p.csParser.Parse(file.Path, file.Content)

		for _, cls := range pf.Classes {
			// Look for Request/Response DTO classes
			if p.isSchemaClass(cls) {
				schema := p.classToSchema(cls)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// isSchemaClass checks if a class is likely a schema/DTO class.
func (p *Plugin) isSchemaClass(cls parser.CSharpClass) bool {
	name := cls.Name
	if strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Dto") ||
		strings.HasSuffix(name, "DTO") ||
		strings.HasSuffix(name, "Model") {
		return true
	}
	return false
}

// classToSchema converts a C# class to an OpenAPI schema.
func (p *Plugin) classToSchema(cls parser.CSharpClass) *types.Schema {
	schema := &types.Schema{
		Title:      cls.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// The CSharpParser extracts methods but not properties directly
	// This would need enhancement to the parser for full property extraction
	return schema
}

// Register registers the FastEndpoints plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
