// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package oatpp provides a plugin for extracting routes from Oat++ C++ framework applications.
package oatpp

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

// Regex patterns for Oat++ parsing
var (
	// Matches ENDPOINT macro with various parameter types
	endpointRegex = regexp.MustCompile(`(?m)ENDPOINT\s*\(\s*"(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)"\s*,\s*"([^"]+)"\s*,\s*(\w+)`)

	// Matches ENDPOINT_INFO macro for metadata
	endpointInfoRegex = regexp.MustCompile(`(?m)ENDPOINT_INFO\s*\(\s*(\w+)\s*\)`)

	// Matches DTO_FIELD macro for DTO field extraction
	dtoFieldRegex = regexp.MustCompile(`(?m)DTO_FIELD\s*\(\s*([^,]+)\s*,\s*(\w+)\s*\)`)

	// Matches PATH parameter in ENDPOINT
	pathParamRegex = regexp.MustCompile(`PATH\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)`)

	// Matches BODY_DTO parameter in ENDPOINT
	bodyDtoRegex = regexp.MustCompile(`BODY_DTO\s*\(\s*Object\s*<\s*(\w+)\s*>\s*,\s*(\w+)\s*\)`)

	// Matches QUERY parameter in ENDPOINT
	queryParamRegex = regexp.MustCompile(`QUERY\s*\(\s*(\w+)\s*,\s*(\w+)\s*\)`)

	// Matches DTO class definitions
	dtoClassRegex = regexp.MustCompile(`(?m)class\s+(\w+)\s*:\s*public\s+oatpp::DTO`)
)

// Plugin implements the FrameworkPlugin interface for Oat++.
type Plugin struct {
	cppParser *parser.CppParser
}

// New creates a new Oat++ plugin instance.
func New() *Plugin {
	return &Plugin{
		cppParser: parser.NewCppParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "oatpp"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cpp", ".hpp", ".h", ".cc", ".cxx"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "oatpp",
		Version:     "1.0.0",
		Description: "Extracts routes from Oat++ C++ framework applications",
		SupportedFrameworks: []string{
			"oatpp",
			"Oat++",
		},
	}
}

// Detect checks if Oat++ is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check CMakeLists.txt for oatpp
	cmakePath := filepath.Join(projectRoot, "CMakeLists.txt")
	if found, _ := p.checkFileForDependency(cmakePath, "oatpp"); found {
		return true, nil
	}

	// Check conanfile.txt for oatpp
	conanPath := filepath.Join(projectRoot, "conanfile.txt")
	if found, _ := p.checkFileForDependency(conanPath, "oatpp"); found {
		return true, nil
	}

	// Check vcpkg.json for oatpp
	vcpkgPath := filepath.Join(projectRoot, "vcpkg.json")
	if found, _ := p.checkFileForDependency(vcpkgPath, "oatpp"); found {
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

// ExtractRoutes parses source files and extracts Oat++ route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "cpp" {
			continue
		}

		fileRoutes := p.extractEndpoints(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractEndpoints extracts ENDPOINT definitions from Oat++ source code.
func (p *Plugin) extractEndpoints(src, filePath string) []types.Route {
	var routes []types.Route

	matches := endpointRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := types.Route{
			SourceFile: filePath,
			SourceLine: line,
		}

		// Extract HTTP method (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.Method = strings.ToUpper(src[match[2]:match[3]])
		}

		// Extract path (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			path := src[match[4]:match[5]]
			// Convert Oat++ path params to OpenAPI format
			route.Path = convertOatppPathParams(path)
		}

		// Extract handler name (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Handler = src[match[6]:match[7]]
		}

		// Ensure path starts with /
		if !strings.HasPrefix(route.Path, "/") {
			route.Path = "/" + route.Path
		}

		// Extract path parameters
		route.Parameters = extractPathParams(route.Path)

		// Look ahead for additional parameters in the ENDPOINT macro
		endpointEnd := findEndpointEnd(src[match[0]:])
		if endpointEnd > 0 {
			endpointContent := src[match[0] : match[0]+endpointEnd]

			// Extract QUERY parameters
			queryParams := p.extractQueryParams(endpointContent)
			route.Parameters = append(route.Parameters, queryParams...)
		}

		// Generate operation ID
		route.OperationID = generateOperationID(route.Method, route.Path, route.Handler)

		// Infer tags
		route.Tags = inferTags(route.Path)

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// findEndpointEnd finds the end of an ENDPOINT macro (matching parenthesis).
func findEndpointEnd(src string) int {
	depth := 0
	started := false

	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
			started = true
		case ')':
			depth--
			if started && depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// extractQueryParams extracts QUERY parameters from an ENDPOINT definition.
func (p *Plugin) extractQueryParams(endpointContent string) []types.Parameter {
	var params []types.Parameter

	matches := queryParamRegex.FindAllStringSubmatch(endpointContent, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		paramType := match[1]
		paramName := match[2]

		openAPIType, format := oatppTypeToOpenAPI(paramType)

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "query",
			Required: false,
			Schema: &types.Schema{
				Type:   openAPIType,
				Format: format,
			},
		})
	}

	return params
}

// convertOatppPathParams converts Oat++ path parameters to OpenAPI format.
func convertOatppPathParams(path string) string {
	// Oat++ uses {param} format which is already OpenAPI compatible
	return path
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
	if handler != "" {
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

// countLines counts the number of lines in a string.
func countLines(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

// ExtractSchemas extracts schema definitions from Oat++ DTOs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "cpp" {
			continue
		}

		fileSchemas := p.extractDTOs(string(file.Content))
		schemas = append(schemas, fileSchemas...)
	}

	return schemas, nil
}

// extractDTOs extracts DTO class definitions from Oat++ source code.
func (p *Plugin) extractDTOs(src string) []types.Schema {
	var schemas []types.Schema

	// Find all DTO class definitions
	classMatches := dtoClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range classMatches {
		if len(match) < 4 {
			continue
		}

		className := ""
		if match[2] >= 0 && match[3] >= 0 {
			className = src[match[2]:match[3]]
		}

		if className == "" {
			continue
		}

		schema := types.Schema{
			Title:      className,
			Type:       "object",
			Properties: make(map[string]*types.Schema),
			Required:   []string{},
		}

		// Find the class body
		classStart := match[0]
		classBody := findClassBody(src[classStart:])
		if classBody != "" {
			// Extract DTO_FIELD definitions
			fieldMatches := dtoFieldRegex.FindAllStringSubmatch(classBody, -1)
			for _, fieldMatch := range fieldMatches {
				if len(fieldMatch) < 3 {
					continue
				}

				fieldType := strings.TrimSpace(fieldMatch[1])
				fieldName := fieldMatch[2]

				openAPIType, format := oatppTypeToOpenAPI(fieldType)
				schema.Properties[fieldName] = &types.Schema{
					Type:   openAPIType,
					Format: format,
				}
			}
		}

		schemas = append(schemas, schema)
	}

	return schemas
}

// findClassBody finds the body of a class (between { and }).
func findClassBody(src string) string {
	openBrace := strings.Index(src, "{")
	if openBrace == -1 {
		return ""
	}

	depth := 0
	for i := openBrace; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[openBrace+1 : i]
			}
		}
	}
	return ""
}

// oatppTypeToOpenAPI converts an Oat++ type to an OpenAPI type.
func oatppTypeToOpenAPI(oatppType string) (openAPIType string, format string) {
	// Trim whitespace
	oatppType = strings.TrimSpace(oatppType)

	// Handle Oat++ wrapper types
	if strings.HasPrefix(oatppType, "oatpp::") {
		oatppType = strings.TrimPrefix(oatppType, "oatpp::")
	}

	// Handle Object<T> types
	if strings.HasPrefix(oatppType, "Object<") {
		return "object", ""
	}

	// Handle List<T> types
	if strings.HasPrefix(oatppType, "List<") || strings.HasPrefix(oatppType, "Vector<") {
		return "array", ""
	}

	switch oatppType {
	case "String", "string":
		return "string", ""
	case "Int8", "Int16", "Int32", "Int64", "UInt8", "UInt16", "UInt32", "UInt64":
		return "integer", ""
	case "Float32", "Float64":
		return "number", ""
	case "Boolean", "bool":
		return "boolean", ""
	default:
		return "object", ""
	}
}

// Register registers the Oat++ plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
