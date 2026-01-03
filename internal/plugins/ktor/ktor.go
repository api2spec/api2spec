// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package ktor provides a plugin for extracting routes from Ktor framework applications.
package ktor

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


// Plugin implements the FrameworkPlugin interface for Ktor framework.
type Plugin struct {
	kotlinParser *parser.KotlinParser
}

// New creates a new Ktor plugin instance.
func New() *Plugin {
	return &Plugin{
		kotlinParser: parser.NewKotlinParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "ktor"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".kt", ".kts"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "ktor",
		Version:     "1.0.0",
		Description: "Extracts routes from Ktor framework applications",
		SupportedFrameworks: []string{
			"io.ktor",
			"Ktor",
		},
	}
}

// Detect checks if Ktor is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check build.gradle.kts for Ktor
	gradleKtsPath := filepath.Join(projectRoot, "build.gradle.kts")
	if found, _ := p.checkFileForDependency(gradleKtsPath, "io.ktor"); found {
		return true, nil
	}

	// Check build.gradle for Ktor
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	if found, _ := p.checkFileForDependency(gradlePath, "io.ktor"); found {
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

// ExtractRoutes parses source files and extracts Ktor route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "kotlin" {
			continue
		}

		pf := p.kotlinParser.Parse(file.Path, file.Content)

		// Extract routes from parsed routes
		for _, route := range pf.Routes {
			r := p.convertRoute(route, file.Path)
			if r != nil {
				routes = append(routes, *r)
			}
		}

		// Also extract routes from raw content using regex
		rawRoutes := p.extractRoutesFromContent(string(file.Content), file.Path)
		routes = append(routes, rawRoutes...)
	}

	return routes, nil
}

// convertRoute converts a Kotlin route to a types.Route.
func (p *Plugin) convertRoute(route parser.KotlinRoute, filePath string) *types.Route {
	fullPath := convertKtorPathParams(route.Path)
	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, route.Handler)
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

// extractRoutesFromContent extracts routes from raw Kotlin content.
func (p *Plugin) extractRoutesFromContent(content, filePath string) []types.Route {
	var routes []types.Route

	// Track route blocks for nested prefixes
	prefixes := p.extractRoutePrefixes(content)
	_ = prefixes // TODO: Apply nested prefixes

	// Match routes with HTTP method calls
	routeRegex := regexp.MustCompile(`(?m)(get|post|put|delete|patch|head|options)\s*\(\s*"([^"]+)"\s*\)\s*\{`)
	matches := routeRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(content[:match[0]])
		method := strings.ToUpper(content[match[2]:match[3]])
		path := content[match[4]:match[5]]

		fullPath := convertKtorPathParams(path)
		params := extractPathParams(fullPath)
		operationID := generateOperationID(method, fullPath, "")
		tags := inferTags(fullPath)

		routes = append(routes, types.Route{
			Method:      method,
			Path:        fullPath,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceFile:  filePath,
			SourceLine:  line,
		})
	}

	return routes
}

// extractRoutePrefixes extracts route block prefixes from content.
func (p *Plugin) extractRoutePrefixes(content string) []string {
	var prefixes []string

	routeBlockRegex := regexp.MustCompile(`route\s*\(\s*"([^"]+)"\s*\)`)
	matches := routeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			prefixes = append(prefixes, match[1])
		}
	}

	return prefixes
}

// countLines counts the number of lines up to a position.
func countLines(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertKtorPathParams converts Ktor-style path params to OpenAPI format.
// Ktor uses {param} format which is already OpenAPI compatible.
func convertKtorPathParams(path string) string {
	// Ktor uses {param} format, same as OpenAPI
	// Handle optional parameters like {param?}
	optionalRegex := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\?\}`)
	return optionalRegex.ReplaceAllString(path, "{$1}")
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

// ExtractSchemas extracts schema definitions from Kotlin data classes.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "kotlin" {
			continue
		}

		pf := p.kotlinParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			// Extract data classes that look like DTOs
			if strings.HasSuffix(class.Name, "Dto") ||
				strings.HasSuffix(class.Name, "DTO") ||
				strings.HasSuffix(class.Name, "Request") ||
				strings.HasSuffix(class.Name, "Response") {

				schema := p.classToSchema(class)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// classToSchema converts a Kotlin class to an OpenAPI schema.
func (p *Plugin) classToSchema(class parser.KotlinClass) *types.Schema {
	schema := &types.Schema{
		Title:      class.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Extract from primary constructor parameters for data classes
	for _, fn := range class.Functions {
		if fn.Name == class.Name {
			// Constructor
			for _, param := range fn.Parameters {
				openAPIType, format := parser.KotlinTypeToOpenAPI(param.Type)
				propSchema := &types.Schema{
					Type:   openAPIType,
					Format: format,
				}

				isOptional := strings.HasSuffix(param.Type, "?")
				if isOptional {
					propSchema.Nullable = true
				}

				schema.Properties[param.Name] = propSchema

				if !isOptional && param.Default == "" {
					schema.Required = append(schema.Required, param.Name)
				}
			}
		}
	}

	return schema
}

// Register registers the Ktor plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
