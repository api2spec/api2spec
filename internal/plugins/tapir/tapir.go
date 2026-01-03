// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package tapir provides a plugin for extracting routes from Tapir Scala framework applications.
package tapir

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

// Regex patterns for Tapir parsing
var (
	// Matches Tapir endpoint definitions
	tapirEndpointRegex = regexp.MustCompile(`(?m)endpoint\s*\.\s*(get|post|put|delete|patch|head|options)`)

	// Matches Tapir .in("path") segments
	tapirInRegex = regexp.MustCompile(`\.in\s*\(\s*"([^"]+)"`)

	// Matches Tapir path[Type]("name") segments
	tapirPathParamRegex = regexp.MustCompile(`path\s*\[\s*(\w+)\s*\]\s*\(\s*"([^"]+)"\s*\)`)

	// Matches Tapir query[Type]("name") segments
	tapirQueryParamRegex = regexp.MustCompile(`query\s*\[\s*(\w+)\s*\]\s*\(\s*"([^"]+)"\s*\)`)

	// Matches Tapir jsonBody[Type]
	tapirJsonBodyRegex = regexp.MustCompile(`jsonBody\s*\[\s*([^\]]+)\s*\]`)

	// Matches Tapir .out(jsonBody[Type])
	tapirOutJsonBodyRegex = regexp.MustCompile(`\.out\s*\(\s*jsonBody\s*\[\s*([^\]]+)\s*\]`)

	// Matches Tapir .serverLogic or .serverLogicSuccess
	tapirServerLogicRegex = regexp.MustCompile(`\.serverLogic(?:Success|RecoverErrors)?\s*[({]`)

	// Matches Tapir endpoint name (val endpointName = endpoint...)
	tapirEndpointNameRegex = regexp.MustCompile(`(?m)(?:val|def)\s+(\w+)\s*(?::\s*[^=]+)?\s*=\s*(?:baseEndpoint\.)?endpoint`)

	// Matches path segments like "users" / "list"
	tapirPathSegmentRegex = regexp.MustCompile(`"([^"]+)"(?:\s*/\s*"([^"]+)")*`)

	// Matches path concatenation with /
	tapirPathConcatRegex = regexp.MustCompile(`"([^"]+)"\s*/\s*`)
)

// Plugin implements the FrameworkPlugin interface for Tapir.
type Plugin struct {
	scalaParser *parser.ScalaParser
}

// New creates a new Tapir plugin instance.
func New() *Plugin {
	return &Plugin{
		scalaParser: parser.NewScalaParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "tapir"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".scala", ".sc"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "tapir",
		Version:     "1.0.0",
		Description: "Extracts routes from Tapir Scala framework applications",
		SupportedFrameworks: []string{
			"tapir",
			"sttp.tapir",
			"Tapir",
		},
	}
}

// Detect checks if Tapir is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check build.sbt for Tapir dependency
	buildSbtPath := filepath.Join(projectRoot, "build.sbt")
	if found, _ := p.checkFileForDependency(buildSbtPath, "sttp.tapir"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(buildSbtPath, "tapir-core"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(buildSbtPath, "tapir-"); found {
		return true, nil
	}

	// Check build.sc (Mill) for Tapir
	buildScPath := filepath.Join(projectRoot, "build.sc")
	if found, _ := p.checkFileForDependency(buildScPath, "tapir"); found {
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

// ExtractRoutes parses source files and extracts Tapir route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "scala" {
			continue
		}

		fileRoutes := p.extractTapirEndpoints(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractTapirEndpoints extracts Tapir endpoint definitions from Scala source code.
func (p *Plugin) extractTapirEndpoints(src, filePath string) []types.Route {
	var routes []types.Route

	// Find all endpoint definitions with HTTP methods
	methodMatches := tapirEndpointRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range methodMatches {
		if len(match) < 4 {
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

		// Find the endpoint chain (from start to .serverLogic or end of statement)
		chainStart := match[0]
		chain := p.extractEndpointChain(src[chainStart:])

		// Extract path from the chain
		route.Path = p.extractPath(chain)

		// Ensure path starts with /
		if !strings.HasPrefix(route.Path, "/") {
			route.Path = "/" + route.Path
		}

		// Extract parameters
		route.Parameters = p.extractParameters(chain)

		// Try to find endpoint name
		endpointName := p.findEndpointName(src[:chainStart])
		if endpointName != "" {
			route.Handler = endpointName
		}

		// Generate operation ID
		route.OperationID = generateOperationID(route.Method, route.Path, route.Handler)

		// Infer tags
		route.Tags = inferTags(route.Path)

		if route.Path != "" && route.Path != "/" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractEndpointChain extracts the full endpoint chain until .serverLogic or end.
func (p *Plugin) extractEndpointChain(src string) string {
	// Look for .serverLogic
	if match := tapirServerLogicRegex.FindStringIndex(src); match != nil {
		return src[:match[0]]
	}

	// Look for common chain terminators (newline followed by non-continuation)
	depth := 0
	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '\n':
			if depth == 0 && i+1 < len(src) {
				// Check if next non-space char continues the chain
				nextChar := strings.TrimLeft(src[i+1:], " \t")
				if len(nextChar) == 0 || (nextChar[0] != '.' && nextChar[0] != '/') {
					return src[:i]
				}
			}
		}
	}

	// Return up to a reasonable length
	if len(src) > 500 {
		return src[:500]
	}
	return src
}

// extractPath extracts the path from a Tapir endpoint chain.
func (p *Plugin) extractPath(chain string) string {
	var pathParts []string

	// Extract static path segments from .in("segment")
	inMatches := tapirInRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range inMatches {
		if len(match) > 1 {
			segment := match[1]
			// Handle path concatenation like "users" / "list"
			if strings.Contains(segment, "/") {
				parts := strings.Split(segment, "/")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					part = strings.Trim(part, "\"")
					if part != "" {
						pathParts = append(pathParts, part)
					}
				}
			} else {
				pathParts = append(pathParts, segment)
			}
		}
	}

	// Extract path parameters from path[Type]("name")
	paramMatches := tapirPathParamRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range paramMatches {
		if len(match) > 2 {
			paramName := match[2]
			pathParts = append(pathParts, "{"+paramName+"}")
		}
	}

	if len(pathParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(pathParts, "/")
}

// extractParameters extracts path and query parameters from a Tapir endpoint chain.
func (p *Plugin) extractParameters(chain string) []types.Parameter {
	var params []types.Parameter

	// Extract path parameters
	pathMatches := tapirPathParamRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range pathMatches {
		if len(match) < 3 {
			continue
		}

		paramType := match[1]
		paramName := match[2]

		openAPIType, format := scalaTypeToOpenAPI(paramType)

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

	// Extract query parameters
	queryMatches := tapirQueryParamRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range queryMatches {
		if len(match) < 3 {
			continue
		}

		paramType := match[1]
		paramName := match[2]

		openAPIType, format := scalaTypeToOpenAPI(paramType)

		// Check if optional
		required := true
		if strings.HasPrefix(paramType, "Option[") {
			required = false
			paramType = extractScalaGenericType(paramType)
			openAPIType, format = scalaTypeToOpenAPI(paramType)
		}

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "query",
			Required: required,
			Schema: &types.Schema{
				Type:   openAPIType,
				Format: format,
			},
		})
	}

	return params
}

// findEndpointName looks for the endpoint val/def name before the endpoint definition.
func (p *Plugin) findEndpointName(src string) string {
	// Search backwards for val/def name = endpoint
	lines := strings.Split(src, "\n")
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		if match := tapirEndpointNameRegex.FindStringSubmatch(line); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// scalaTypeToOpenAPI converts a Scala type to an OpenAPI type.
func scalaTypeToOpenAPI(scalaType string) (openAPIType string, format string) {
	// Trim whitespace
	scalaType = strings.TrimSpace(scalaType)

	// Handle Option types
	if strings.HasPrefix(scalaType, "Option[") {
		scalaType = extractScalaGenericType(scalaType)
	}

	switch scalaType {
	case "String":
		return "string", ""
	case "Int", "Integer":
		return "integer", ""
	case "Long":
		return "integer", "int64"
	case "Float":
		return "number", "float"
	case "Double":
		return "number", "double"
	case "Boolean":
		return "boolean", ""
	case "UUID":
		return "string", "uuid"
	default:
		return "string", ""
	}
}

// extractScalaGenericType extracts the inner type from a generic like Option[String].
func extractScalaGenericType(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

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

// ExtractSchemas extracts schema definitions from Scala case classes.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "scala" {
			continue
		}

		pf := p.scalaParser.Parse(file.Path, file.Content)

		for _, caseClass := range pf.CaseClasses {
			schema := p.caseClassToSchema(caseClass)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// caseClassToSchema converts a Scala case class to an OpenAPI schema.
func (p *Plugin) caseClassToSchema(caseClass parser.ScalaCaseClass) *types.Schema {
	schema := &types.Schema{
		Title:      caseClass.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range caseClass.Fields {
		openAPIType, format := parser.ScalaTypeToOpenAPI(field.Type)
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

// Register registers the Tapir plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
