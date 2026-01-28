// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package drogon provides a plugin for extracting routes from Drogon C++ framework applications.
package drogon

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

// Plugin implements the FrameworkPlugin interface for Drogon.
type Plugin struct {
	cppParser *parser.CppParser
}

// New creates a new Drogon plugin instance.
func New() *Plugin {
	return &Plugin{
		cppParser: parser.NewCppParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "drogon"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cpp", ".hpp", ".h", ".cc", ".cxx"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "drogon",
		Version:     "1.0.0",
		Description: "Extracts routes from Drogon C++ framework applications",
		SupportedFrameworks: []string{
			"drogon",
			"Drogon",
		},
	}
}

// Detect checks if Drogon is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check CMakeLists.txt for Drogon
	cmakePath := filepath.Join(projectRoot, "CMakeLists.txt")
	if found, _ := p.checkFileForDependency(cmakePath, "drogon"); found {
		return true, nil
	}

	// Check conanfile.txt for drogon
	conanPath := filepath.Join(projectRoot, "conanfile.txt")
	if found, _ := p.checkFileForDependency(conanPath, "drogon"); found {
		return true, nil
	}

	// Check vcpkg.json for drogon
	vcpkgPath := filepath.Join(projectRoot, "vcpkg.json")
	if found, _ := p.checkFileForDependency(vcpkgPath, "drogon"); found {
		return true, nil
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
	depLower := strings.ToLower(dep)
	for scanr.Scan() {
		line := strings.ToLower(scanr.Text())
		if strings.Contains(line, depLower) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Drogon route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route
	seen := make(map[string]bool)

	for _, file := range files {
		if file.Language != "cpp" {
			continue
		}

		pf := p.cppParser.Parse(file.Path, file.Content)

		for _, route := range pf.Routes {
			r := p.convertRoute(route, file.Path)
			if r != nil {
				// Deduplicate routes by method+path
				// This prevents double-counting when routes are defined in both
				// main source files and test files
				key := r.Method + " " + r.Path
				if seen[key] {
					continue
				}
				seen[key] = true
				routes = append(routes, *r)
			}
		}
	}

	return routes, nil
}

// convertRoute converts a parsed C++ route to a types.Route.
func (p *Plugin) convertRoute(route parser.CppRoute, filePath string) *types.Route {
	// Convert path params to OpenAPI format
	path := parser.ConvertCppPathParams(route.Path)

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Extract path parameters
	params := extractPathParams(path)

	// Generate operation ID
	handler := route.Handler
	if route.ControllerClass != "" {
		handler = route.ControllerClass + "." + route.Handler
	}
	operationID := generateOperationID(route.Method, path, route.Handler)

	// Infer tags
	tags := inferTags(path)
	if route.ControllerClass != "" {
		controllerName := strings.TrimSuffix(route.ControllerClass, "Controller")
		controllerName = strings.TrimSuffix(controllerName, "Ctrl")
		if controllerName != "" {
			tags = []string{controllerName}
		}
	}

	return &types.Route{
		Method:      route.Method,
		Path:        path,
		Handler:     handler,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

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
	if handler != "" && handler != "lambda" {
		return strings.ToLower(method) + toTitleCase(handler)
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

// inferTags infers tags from the route path.
func inferTags(path string) []string {
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

// ExtractSchemas extracts schema definitions from C++ structs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "cpp" {
			continue
		}

		pf := p.cppParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			// Only extract structs that look like DTOs
			if !class.IsStruct {
				continue
			}

			// Check if this looks like a DTO
			if strings.HasSuffix(class.Name, "Dto") ||
				strings.HasSuffix(class.Name, "DTO") ||
				strings.HasSuffix(class.Name, "Request") ||
				strings.HasSuffix(class.Name, "Response") ||
				strings.HasSuffix(class.Name, "Model") {

				schema := p.structToSchema(class)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// structToSchema converts a C++ struct to an OpenAPI schema.
func (p *Plugin) structToSchema(class parser.CppClass) *types.Schema {
	schema := &types.Schema{
		Title:      class.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range class.Fields {
		openAPIType, format := parser.CppTypeToOpenAPI(field.Type)
		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		schema.Properties[field.Name] = propSchema
	}

	return schema
}

// Register registers the Drogon plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
