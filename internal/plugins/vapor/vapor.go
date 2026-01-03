// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package vapor provides a plugin for extracting routes from Vapor Swift framework applications.
package vapor

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

// Plugin implements the FrameworkPlugin interface for Vapor.
type Plugin struct {
	swiftParser *parser.SwiftParser
}

// New creates a new Vapor plugin instance.
func New() *Plugin {
	return &Plugin{
		swiftParser: parser.NewSwiftParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "vapor"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".swift"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "vapor",
		Version:     "1.0.0",
		Description: "Extracts routes from Vapor Swift framework applications",
		SupportedFrameworks: []string{
			"vapor",
			"Vapor",
		},
	}
}

// Detect checks if Vapor is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check Package.swift for Vapor dependency
	packageSwiftPath := filepath.Join(projectRoot, "Package.swift")
	if found, _ := p.checkFileForDependency(packageSwiftPath, "vapor"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(packageSwiftPath, "github.com/vapor/vapor"); found {
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

// ExtractRoutes parses source files and extracts Vapor route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "swift" {
			continue
		}

		fileRoutes := p.extractVaporRoutes(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// Regex patterns for Vapor route extraction
var (
	// Matches Vapor app.get/post/put/delete/patch routes
	vaporRouteRegex = regexp.MustCompile(`(?m)(\w+)\.(get|post|put|delete|patch|head|options)\s*\(\s*([^)]*)\s*\)\s*\{`)

	// Matches Vapor grouped routes: app.grouped("api").get("users") { ... }
	vaporGroupedRouteRegex = regexp.MustCompile(`(?m)(\w+)\.grouped\s*\(\s*"([^"]+)"\s*\)\s*\.\s*(get|post|put|delete|patch|head|options)\s*\(\s*([^)]*)\s*\)\s*\{`)

	// Matches route group assignment: let users = app.grouped("users")
	vaporRouteGroupRegex = regexp.MustCompile(`(?m)let\s+(\w+)\s*=\s*(\w+)\.grouped\s*\(\s*"([^"]+)"\s*\)`)

	// Matches controller routes: routes.get("path", use: handler)
	vaporControllerRouteRegex = regexp.MustCompile(`(?m)(\w+)\.(get|post|put|delete|patch|head|options)\s*\(\s*([^,)]*)\s*,\s*use\s*:\s*(\w+)\s*\)`)

	// Matches path segments in route definitions
	vaporPathSegmentRegex = regexp.MustCompile(`"([^"]+)"`)

	// Matches path parameters like ":id", ":userId"
	vaporPathParamRegex = regexp.MustCompile(`:(\w+)`)

	// Matches path parameter in brace format
	braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)
)

// extractVaporRoutes extracts Vapor route definitions from Swift source code.
func (p *Plugin) extractVaporRoutes(src, filePath string) []types.Route {
	var routes []types.Route

	// Extract route groups first
	groupMap := p.extractRouteGroups(src)

	// Extract grouped routes (app.grouped("api").get("users"))
	routes = append(routes, p.extractGroupedRoutes(src, filePath)...)

	// Extract controller routes (routes.get("path", use: handler))
	routes = append(routes, p.extractControllerRoutes(src, filePath, groupMap)...)

	// Extract simple routes (app.get("users"))
	routes = append(routes, p.extractSimpleRoutes(src, filePath, groupMap)...)

	return routes
}

// extractRouteGroups extracts route group definitions.
func (p *Plugin) extractRouteGroups(src string) map[string]string {
	groupMap := make(map[string]string)

	matches := vaporRouteGroupRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			groupName := match[1]
			groupPrefix := match[3]
			groupMap[groupName] = groupPrefix
		}
	}

	return groupMap
}

// extractGroupedRoutes extracts routes with inline grouping.
func (p *Plugin) extractGroupedRoutes(src, filePath string) []types.Route {
	var routes []types.Route

	matches := vaporGroupedRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := types.Route{
			SourceFile: filePath,
			SourceLine: line,
		}

		// Extract group prefix (group 2)
		var groupPrefix string
		if match[4] >= 0 && match[5] >= 0 {
			groupPrefix = src[match[4]:match[5]]
		}

		// Extract HTTP method (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Method = strings.ToUpper(src[match[6]:match[7]])
		}

		// Extract path segments (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			pathStr := src[match[8]:match[9]]
			route.Path = buildPath(groupPrefix, pathStr)
		} else {
			route.Path = "/" + groupPrefix
		}

		// Extract parameters from path
		route.Parameters = extractPathParameters(route.Path)

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

// extractControllerRoutes extracts routes from controller definitions.
func (p *Plugin) extractControllerRoutes(src, filePath string, groupMap map[string]string) []types.Route {
	var routes []types.Route

	matches := vaporControllerRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := types.Route{
			SourceFile: filePath,
			SourceLine: line,
		}

		// Extract router variable name (group 1)
		var routerName string
		if match[2] >= 0 && match[3] >= 0 {
			routerName = src[match[2]:match[3]]
		}

		// Extract HTTP method (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Method = strings.ToUpper(src[match[4]:match[5]])
		}

		// Extract path segments (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			pathStr := src[match[6]:match[7]]
			prefix := groupMap[routerName]
			route.Path = buildPath(prefix, pathStr)
		}

		// Extract handler name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			route.Handler = src[match[8]:match[9]]
		}

		// Extract parameters from path
		route.Parameters = extractPathParameters(route.Path)

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

// extractSimpleRoutes extracts simple route definitions.
func (p *Plugin) extractSimpleRoutes(src, filePath string, groupMap map[string]string) []types.Route {
	var routes []types.Route

	matches := vaporRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := types.Route{
			SourceFile: filePath,
			SourceLine: line,
		}

		// Extract router variable name (group 1)
		var routerName string
		if match[2] >= 0 && match[3] >= 0 {
			routerName = src[match[2]:match[3]]
		}

		// Skip if this looks like a grouped route (already handled)
		startIdx := match[0] - 50
		if startIdx < 0 {
			startIdx = 0
		}
		if strings.Contains(src[startIdx:match[0]], ".grouped") {
			continue
		}

		// Extract HTTP method (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Method = strings.ToUpper(src[match[4]:match[5]])
		}

		// Extract path segments (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			pathStr := src[match[6]:match[7]]
			prefix := groupMap[routerName]
			route.Path = buildPath(prefix, pathStr)
		}

		// Extract parameters from path
		route.Parameters = extractPathParameters(route.Path)

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

// buildPath builds a complete path from a prefix and path segments.
func buildPath(prefix, pathStr string) string {
	var segments []string

	// Add prefix if present
	if prefix != "" {
		segments = append(segments, prefix)
	}

	// Extract path segments from the string
	segmentMatches := vaporPathSegmentRegex.FindAllStringSubmatch(pathStr, -1)
	for _, segMatch := range segmentMatches {
		if len(segMatch) > 1 {
			segment := segMatch[1]
			// Convert :param to {param}
			segment = vaporPathParamRegex.ReplaceAllString(segment, "{$1}")
			if segment != "" {
				segments = append(segments, segment)
			}
		}
	}

	if len(segments) == 0 {
		return "/"
	}

	path := "/" + strings.Join(segments, "/")
	// Convert any remaining :param to {param}
	path = vaporPathParamRegex.ReplaceAllString(path, "{$1}")
	return path
}

// extractPathParameters extracts path parameters from a route path.
func extractPathParameters(path string) []types.Parameter {
	var params []types.Parameter

	matches := braceParamRegex.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, types.Parameter{
				Name:     match[1],
				In:       "path",
				Required: true,
				Schema: &types.Schema{
					Type: "string",
				},
			})
		}
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

// ExtractSchemas extracts schema definitions from Swift structs conforming to Content.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "swift" {
			continue
		}

		pf := p.swiftParser.Parse(file.Path, file.Content)

		for _, swiftStruct := range pf.Structs {
			// Only include structs that conform to Content protocol
			if !swiftStruct.ConformsToContent {
				continue
			}

			schema := p.structToSchema(swiftStruct)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// structToSchema converts a Swift struct to an OpenAPI schema.
func (p *Plugin) structToSchema(swiftStruct parser.SwiftStruct) *types.Schema {
	schema := &types.Schema{
		Title:      swiftStruct.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range swiftStruct.Fields {
		openAPIType, format := parser.SwiftTypeToOpenAPI(field.Type)
		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		// Handle nullable/optional fields
		if field.IsOptional {
			propSchema.Nullable = true
		} else {
			schema.Required = append(schema.Required, field.Name)
		}

		schema.Properties[field.Name] = propSchema
	}

	return schema
}

// Register registers the Vapor plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
