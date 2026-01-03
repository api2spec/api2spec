// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package crow provides a plugin for extracting routes from Crow C++ framework applications.
package crow

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

// Regex patterns for Crow parsing
var (
	// Matches CROW_ROUTE macro
	crowRouteRegex = regexp.MustCompile(`(?m)CROW_ROUTE\s*\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)(?:\s*\.methods\s*\(([^)]+)\))?`)

	// Matches CROW_BP_ROUTE macro (blueprint routes)
	crowBPRouteRegex = regexp.MustCompile(`(?m)CROW_BP_ROUTE\s*\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)(?:\s*\.methods\s*\(([^)]+)\))?`)

	// Matches Crow method like "GET"_method or crow::HTTPMethod::GET
	crowMethodRegex = regexp.MustCompile(`"(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)"_method|crow::HTTPMethod::(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)`)

	// Matches Crow path parameters like <int>, <uint>, <double>, <string>
	crowPathParamRegex = regexp.MustCompile(`<(int|uint|double|string)>`)

	// Matches named Crow path parameters like <int: id> or <string: name>
	crowNamedParamRegex = regexp.MustCompile(`<(int|uint|double|string)\s*:\s*(\w+)>`)
)

// Plugin implements the FrameworkPlugin interface for Crow.
type Plugin struct {
	cppParser *parser.CppParser
}

// New creates a new Crow plugin instance.
func New() *Plugin {
	return &Plugin{
		cppParser: parser.NewCppParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "crow"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cpp", ".hpp", ".h", ".cc", ".cxx"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "crow",
		Version:     "1.0.0",
		Description: "Extracts routes from Crow C++ framework applications",
		SupportedFrameworks: []string{
			"crow",
			"Crow",
		},
	}
}

// Detect checks if Crow is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check CMakeLists.txt for Crow
	cmakePath := filepath.Join(projectRoot, "CMakeLists.txt")
	if found, _ := p.checkFileForDependency(cmakePath, "crow"); found {
		return true, nil
	}

	// Check conanfile.txt for crow
	conanPath := filepath.Join(projectRoot, "conanfile.txt")
	if found, _ := p.checkFileForDependency(conanPath, "crow"); found {
		return true, nil
	}

	// Check vcpkg.json for crow
	vcpkgPath := filepath.Join(projectRoot, "vcpkg.json")
	if found, _ := p.checkFileForDependency(vcpkgPath, "crow"); found {
		return true, nil
	}

	// Check for crow.h or crow_all.h header
	headerPaths := []string{
		filepath.Join(projectRoot, "include", "crow.h"),
		filepath.Join(projectRoot, "include", "crow_all.h"),
		filepath.Join(projectRoot, "crow.h"),
		filepath.Join(projectRoot, "crow_all.h"),
	}
	for _, headerPath := range headerPaths {
		if _, err := os.Stat(headerPath); err == nil {
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
	depLower := strings.ToLower(dep)
	for scanr.Scan() {
		line := strings.ToLower(scanr.Text())
		if strings.Contains(line, depLower) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Crow route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "cpp" {
			continue
		}

		fileRoutes := p.extractCrowRoutes(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)

		bpRoutes := p.extractCrowBPRoutes(string(file.Content), file.Path)
		routes = append(routes, bpRoutes...)
	}

	return routes, nil
}

// extractCrowRoutes extracts CROW_ROUTE definitions from Crow source code.
func (p *Plugin) extractCrowRoutes(src, filePath string) []types.Route {
	var routes []types.Route

	matches := crowRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		// Extract path (group 2)
		path := ""
		if match[4] >= 0 && match[5] >= 0 {
			path = src[match[4]:match[5]]
		}

		if path == "" {
			continue
		}

		// Convert Crow path params to OpenAPI format
		openAPIPath := convertCrowPathParams(path)

		// Ensure path starts with /
		if !strings.HasPrefix(openAPIPath, "/") {
			openAPIPath = "/" + openAPIPath
		}

		// Extract methods if present (group 3)
		methods := []string{"GET"} // Default
		if len(match) >= 8 && match[6] >= 0 && match[7] >= 0 {
			methodsStr := src[match[6]:match[7]]
			parsedMethods := parseCrowMethods(methodsStr)
			if len(parsedMethods) > 0 {
				methods = parsedMethods
			}
		}

		// Create a route for each method
		for _, method := range methods {
			route := types.Route{
				Method:     method,
				Path:       openAPIPath,
				Handler:    "lambda",
				SourceFile: filePath,
				SourceLine: line,
			}

			// Extract path parameters
			route.Parameters = extractPathParams(openAPIPath, path)

			// Generate operation ID
			route.OperationID = generateOperationID(method, openAPIPath, "")

			// Infer tags
			route.Tags = inferTags(openAPIPath)

			routes = append(routes, route)
		}
	}

	return routes
}

// extractCrowBPRoutes extracts CROW_BP_ROUTE definitions from Crow source code.
func (p *Plugin) extractCrowBPRoutes(src, filePath string) []types.Route {
	var routes []types.Route

	matches := crowBPRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		// Extract path (group 2)
		path := ""
		if match[4] >= 0 && match[5] >= 0 {
			path = src[match[4]:match[5]]
		}

		if path == "" {
			continue
		}

		// Convert Crow path params to OpenAPI format
		openAPIPath := convertCrowPathParams(path)

		// Ensure path starts with /
		if !strings.HasPrefix(openAPIPath, "/") {
			openAPIPath = "/" + openAPIPath
		}

		// Extract methods if present (group 3)
		methods := []string{"GET"} // Default
		if len(match) >= 8 && match[6] >= 0 && match[7] >= 0 {
			methodsStr := src[match[6]:match[7]]
			parsedMethods := parseCrowMethods(methodsStr)
			if len(parsedMethods) > 0 {
				methods = parsedMethods
			}
		}

		// Create a route for each method
		for _, method := range methods {
			route := types.Route{
				Method:     method,
				Path:       openAPIPath,
				Handler:    "lambda",
				SourceFile: filePath,
				SourceLine: line,
			}

			// Extract path parameters
			route.Parameters = extractPathParams(openAPIPath, path)

			// Generate operation ID
			route.OperationID = generateOperationID(method, openAPIPath, "")

			// Infer tags
			route.Tags = inferTags(openAPIPath)

			routes = append(routes, route)
		}
	}

	return routes
}

// parseCrowMethods parses Crow method specifications.
func parseCrowMethods(methodsStr string) []string {
	var methods []string

	matches := crowMethodRegex.FindAllStringSubmatch(methodsStr, -1)
	for _, match := range matches {
		if len(match) >= 2 && match[1] != "" {
			methods = append(methods, strings.ToUpper(match[1]))
		} else if len(match) >= 3 && match[2] != "" {
			methods = append(methods, strings.ToUpper(match[2]))
		}
	}

	return methods
}

// convertCrowPathParams converts Crow path parameters to OpenAPI format.
// Crow uses <int>, <uint>, <double>, <string> or <type: name> format.
func convertCrowPathParams(path string) string {
	// First, convert named parameters <type: name>
	result := crowNamedParamRegex.ReplaceAllString(path, "{$2}")

	// Then, convert unnamed parameters <type> to {param1}, {param2}, etc.
	paramCount := 0
	result = crowPathParamRegex.ReplaceAllStringFunc(result, func(match string) string {
		paramCount++
		return "{param" + string(rune('0'+paramCount)) + "}"
	})

	return result
}

// extractPathParams extracts path parameters from a route path.
func extractPathParams(openAPIPath, originalPath string) []types.Parameter {
	var params []types.Parameter

	// Extract named parameters
	namedMatches := crowNamedParamRegex.FindAllStringSubmatch(originalPath, -1)
	for _, match := range namedMatches {
		if len(match) < 3 {
			continue
		}

		paramType := match[1]
		paramName := match[2]

		openAPIType, format := crowTypeToOpenAPI(paramType)

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "path",
			Required: true,
			Schema: &types.Schema{
				Type:   openAPIType,
				Format: format,
			},
		})
	}

	// Extract unnamed parameters (from the converted path)
	braceParamRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := braceParamRegex.FindAllStringSubmatch(openAPIPath, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		paramName := match[1]

		// Skip if already added as named parameter
		alreadyAdded := false
		for _, p := range params {
			if p.Name == paramName {
				alreadyAdded = true
				break
			}
		}
		if alreadyAdded {
			continue
		}

		// Determine type from original path
		openAPIType := "string"
		format := ""

		// Check the original path for type hints
		if strings.Contains(originalPath, "<int>") {
			openAPIType = "integer"
		} else if strings.Contains(originalPath, "<uint>") {
			openAPIType = "integer"
		} else if strings.Contains(originalPath, "<double>") {
			openAPIType = "number"
		}

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "path",
			Required: true,
			Schema: &types.Schema{
				Type:   openAPIType,
				Format: format,
			},
		})
	}

	return params
}

// crowTypeToOpenAPI converts a Crow type to an OpenAPI type.
func crowTypeToOpenAPI(crowType string) (openAPIType string, format string) {
	switch crowType {
	case "int":
		return "integer", ""
	case "uint":
		return "integer", ""
	case "double":
		return "number", ""
	case "string":
		return "string", ""
	default:
		return "string", ""
	}
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	if handler != "" && handler != "lambda" {
		return strings.ToLower(method) + toTitleCase(handler)
	}

	braceParamRegex := regexp.MustCompile(`\{([^}]+)\}`)
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

// ExtractSchemas extracts schema definitions from Crow source code.
// Crow doesn't have a built-in schema system, so this returns empty.
func (p *Plugin) ExtractSchemas(_ []scanner.SourceFile) ([]types.Schema, error) {
	// Crow uses plain C++ without a schema system
	return []types.Schema{}, nil
}

// Register registers the Crow plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
