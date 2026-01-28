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

		// Extract routes with proper nested route handling
		fileRoutes := p.extractRoutesFromContent(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)
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

// extractRoutesFromContent extracts routes from Kotlin content with proper nested route handling.
// It handles Ktor's nested routing blocks like route("/prefix") { get { } }
func (p *Plugin) extractRoutesFromContent(content, filePath string) []types.Route {
	var routes []types.Route

	// First, find all route() blocks and their prefixes
	routeBlocks := p.findRouteBlocks(content)

	// Match HTTP method routes with explicit paths: get("/path") {
	// The \s*\{ at the end ensures we're matching route definitions, not client calls
	// The negative lookbehind (?<!\.) doesn't work in Go regex, so we check context manually
	routeWithPathRegex := regexp.MustCompile(`(get|post|put|delete|patch|head|options)\s*\(\s*"([^"]+)"\s*\)\s*\{`)
	matches := routeWithPathRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		pos := match[0]

		// Skip if preceded by a dot (e.g., client.get) - this filters out test client calls
		if pos > 0 && content[pos-1] == '.' {
			continue
		}

		line := countLines(content[:pos])
		method := strings.ToUpper(content[match[2]:match[3]])
		path := content[match[4]:match[5]]

		// Find the containing route block prefix
		prefix := p.findContainingRoutePrefix(routeBlocks, pos)
		fullPath := prefix + path

		fullPath = convertKtorPathParams(fullPath)
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

	// Match HTTP method routes without paths inside route blocks: get {
	// These inherit their path from the containing route() block
	routeNoPathRegex := regexp.MustCompile(`(get|post|put|delete|patch|head|options)\s*\{`)
	noPathMatches := routeNoPathRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range noPathMatches {
		if len(match) < 4 {
			continue
		}

		pos := match[0]

		// Skip if preceded by a dot (e.g., client.get)
		if pos > 0 && content[pos-1] == '.' {
			continue
		}

		// Skip if this looks like a route with a path (already handled above)
		// Check if there's a ( before the {
		beforeBrace := content[match[0]:match[1]]
		if strings.Contains(beforeBrace, "(") {
			continue
		}

		line := countLines(content[:pos])
		method := strings.ToUpper(content[match[2]:match[3]])

		// Must be inside a route block to be valid
		prefix := p.findContainingRoutePrefix(routeBlocks, pos)
		if prefix == "" {
			continue // Skip - not inside a route block
		}

		fullPath := convertKtorPathParams(prefix)
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

// routeBlock represents a route() block with its prefix and position range.
type routeBlock struct {
	prefix   string
	startPos int
	endPos   int
}

// findRouteBlocks finds all route("/prefix") { ... } blocks and their positions.
func (p *Plugin) findRouteBlocks(content string) []routeBlock {
	var blocks []routeBlock

	// Match route("/prefix") and find the corresponding block
	routeBlockRegex := regexp.MustCompile(`route\s*\(\s*"([^"]+)"\s*\)\s*\{`)
	matches := routeBlockRegex.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		prefix := content[match[2]:match[3]]
		blockStart := match[0]

		// Find the opening brace position
		braceStart := strings.Index(content[match[0]:], "{")
		if braceStart == -1 {
			continue
		}
		braceStart += match[0]

		// Find the matching closing brace
		blockEnd := p.findMatchingBrace(content, braceStart)
		if blockEnd == -1 {
			continue
		}

		blocks = append(blocks, routeBlock{
			prefix:   prefix,
			startPos: blockStart,
			endPos:   blockEnd,
		})
	}

	return blocks
}

// findMatchingBrace finds the position of the closing brace matching the opening brace at pos.
func (p *Plugin) findMatchingBrace(content string, pos int) int {
	if pos >= len(content) || content[pos] != '{' {
		return -1
	}

	depth := 0
	inString := false
	for i := pos; i < len(content); i++ {
		ch := content[i]

		// Handle string literals
		if ch == '"' && (i == 0 || content[i-1] != '\\') {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// findContainingRoutePrefix finds the route prefix for a position, handling nested blocks.
func (p *Plugin) findContainingRoutePrefix(blocks []routeBlock, pos int) string {
	var prefix string

	// Find all blocks containing this position and concatenate their prefixes
	// (for nested route blocks)
	for _, block := range blocks {
		if pos > block.startPos && pos < block.endPos {
			prefix += block.prefix
		}
	}

	return prefix
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
