// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package servant provides a plugin for extracting routes from Servant Haskell framework applications.
package servant

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

// Plugin implements the FrameworkPlugin interface for Servant.
type Plugin struct {
	haskellParser *parser.HaskellParser
}

// New creates a new Servant plugin instance.
func New() *Plugin {
	return &Plugin{
		haskellParser: parser.NewHaskellParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "servant"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".hs", ".lhs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "servant",
		Version:     "1.0.0",
		Description: "Extracts routes from Servant Haskell framework applications",
		SupportedFrameworks: []string{
			"servant",
			"servant-server",
			"Servant",
		},
	}
}

// Detect checks if Servant is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check package.yaml for Servant dependency
	packageYamlPath := filepath.Join(projectRoot, "package.yaml")
	if found, _ := p.checkFileForDependency(packageYamlPath, "servant"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(packageYamlPath, "servant-server"); found {
		return true, nil
	}

	// Check *.cabal files for Servant dependency
	cabalFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.cabal"))
	for _, cabalFile := range cabalFiles {
		if found, _ := p.checkFileForDependency(cabalFile, "servant"); found {
			return true, nil
		}
		if found, _ := p.checkFileForDependency(cabalFile, "servant-server"); found {
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

// ExtractRoutes parses source files and extracts Servant API definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "haskell" {
			continue
		}

		fileRoutes := p.extractServantEndpoints(string(file.Content), file.Path)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// Regex patterns for Servant API extraction
var (
	// Matches type alias definitions (type API = ...)
	servantTypeAliasRegex = regexp.MustCompile(`(?m)^type\s+(\w+)\s*=\s*`)

	// Matches Servant HTTP method combinators
	servantMethodRegex = regexp.MustCompile(`(Get|Post|Put|Delete|Patch|Head|Options)\s*'\[\s*(\w+)\s*\]`)

	// Matches Servant path segment: "users" :>
	servantPathSegmentRegex = regexp.MustCompile(`"([^"]+)"\s*:>`)

	// Matches Servant Capture: Capture "id" Int :>
	servantCaptureRegex = regexp.MustCompile(`Capture\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant QueryParam: QueryParam "name" Text :>
	servantQueryParamRegex = regexp.MustCompile(`QueryParam\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant QueryParam': QueryParam' '[Required] "name" Text :>
	servantQueryParamModRegex = regexp.MustCompile(`QueryParam'\s*'\[([^\]]+)\]\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant ReqBody: ReqBody '[JSON] CreateUser :>
	servantReqBodyRegex = regexp.MustCompile(`ReqBody\s*'\[\s*(\w+)\s*\]\s+(\w+)\s*:>`)

	// Matches Servant combined APIs: UserAPI :<|> ProductAPI
	servantCombinedAPIRegex = regexp.MustCompile(`:<\|>`)

	// Matches response type after method
	servantResponseTypeRegex = regexp.MustCompile(`(?:Get|Post|Put|Delete|Patch|Head|Options)\s*'\[\s*\w+\s*\]\s+(\[?\w+\]?)`)

	// Matches brace-style path parameters
	braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)
)

// extractServantEndpoints extracts Servant API endpoint definitions from Haskell source code.
func (p *Plugin) extractServantEndpoints(src, filePath string) []types.Route {
	var routes []types.Route

	// Find all type definitions that look like Servant APIs
	typeAliases := p.extractTypeAliases(src)

	for _, alias := range typeAliases {
		// Skip combined APIs (they reference other types)
		if servantCombinedAPIRegex.MatchString(alias.Definition) {
			continue
		}

		// Check if this looks like a Servant endpoint
		if !servantMethodRegex.MatchString(alias.Definition) {
			continue
		}

		route := p.parseServantEndpoint(alias, filePath)
		if route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

// typeAlias represents a Haskell type alias.
type typeAlias struct {
	Name       string
	Definition string
	Line       int
}

// extractTypeAliases extracts type alias definitions from Haskell source.
func (p *Plugin) extractTypeAliases(src string) []typeAlias {
	var aliases []typeAlias

	// Find type declarations
	lines := strings.Split(src, "\n")
	var currentAlias *typeAlias
	var defBuilder strings.Builder

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for new type alias
		if matches := servantTypeAliasRegex.FindStringSubmatch(line); len(matches) > 1 {
			// Save previous alias if exists
			if currentAlias != nil {
				currentAlias.Definition = strings.TrimSpace(defBuilder.String())
				aliases = append(aliases, *currentAlias)
			}

			currentAlias = &typeAlias{
				Name: matches[1],
				Line: i + 1,
			}
			defBuilder.Reset()

			// Get the rest of the line after the =
			eqIdx := strings.Index(line, "=")
			if eqIdx != -1 && eqIdx+1 < len(line) {
				defBuilder.WriteString(strings.TrimSpace(line[eqIdx+1:]))
			}
			continue
		}

		// Continue building definition if we're in a type alias
		if currentAlias != nil {
			// Check if this is a continuation line (starts with whitespace)
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				defBuilder.WriteString(" ")
				defBuilder.WriteString(trimmedLine)
			} else if trimmedLine == "" {
				// Empty line might end the definition
				continue
			} else {
				// Non-indented line ends the current definition
				currentAlias.Definition = strings.TrimSpace(defBuilder.String())
				aliases = append(aliases, *currentAlias)
				currentAlias = nil
				defBuilder.Reset()
			}
		}
	}

	// Don't forget the last alias
	if currentAlias != nil {
		currentAlias.Definition = strings.TrimSpace(defBuilder.String())
		aliases = append(aliases, *currentAlias)
	}

	return aliases
}

// parseServantEndpoint parses a Servant type definition into a Route.
func (p *Plugin) parseServantEndpoint(alias typeAlias, filePath string) *types.Route {
	def := alias.Definition

	route := &types.Route{
		SourceFile: filePath,
		SourceLine: alias.Line,
	}

	// Build path from segments and captures
	var pathParts []string

	// Extract path segments
	segmentMatches := servantPathSegmentRegex.FindAllStringSubmatch(def, -1)
	for _, segMatch := range segmentMatches {
		if len(segMatch) > 1 {
			pathParts = append(pathParts, segMatch[1])
		}
	}

	// Extract capture parameters (path params) and add them to path
	captureMatches := servantCaptureRegex.FindAllStringSubmatch(def, -1)
	captureIdx := 0
	for _, capMatch := range captureMatches {
		if len(capMatch) > 2 {
			paramName := capMatch[1]
			paramType := capMatch[2]

			// Insert capture at the right position (after segments before it)
			// This is a simplification - captures go at the end of current path parts
			pathParts = append(pathParts, "{"+paramName+"}")

			openAPIType, format := parser.HaskellTypeToOpenAPI(paramType)
			route.Parameters = append(route.Parameters, types.Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema: &types.Schema{
					Type:   openAPIType,
					Format: format,
				},
			})
			captureIdx++
		}
	}

	// Extract query parameters
	queryMatches := servantQueryParamRegex.FindAllStringSubmatch(def, -1)
	for _, qMatch := range queryMatches {
		if len(qMatch) > 2 {
			openAPIType, format := parser.HaskellTypeToOpenAPI(qMatch[2])
			route.Parameters = append(route.Parameters, types.Parameter{
				Name:     qMatch[1],
				In:       "query",
				Required: false, // QueryParam is optional by default
				Schema: &types.Schema{
					Type:   openAPIType,
					Format: format,
				},
			})
		}
	}

	// Extract required query parameters (QueryParam')
	queryModMatches := servantQueryParamModRegex.FindAllStringSubmatch(def, -1)
	for _, qMatch := range queryModMatches {
		if len(qMatch) > 3 {
			required := strings.Contains(qMatch[1], "Required")
			openAPIType, format := parser.HaskellTypeToOpenAPI(qMatch[3])
			route.Parameters = append(route.Parameters, types.Parameter{
				Name:     qMatch[2],
				In:       "query",
				Required: required,
				Schema: &types.Schema{
					Type:   openAPIType,
					Format: format,
				},
			})
		}
	}

	// Extract HTTP method and content type
	methodMatch := servantMethodRegex.FindStringSubmatch(def)
	if len(methodMatch) > 2 {
		route.Method = strings.ToUpper(methodMatch[1])
	}

	// Build path
	if len(pathParts) == 0 {
		route.Path = "/"
	} else {
		route.Path = "/" + strings.Join(pathParts, "/")
	}

	// Reorder path parts to intersperse captures correctly
	route.Path = p.reorderPathWithCaptures(def)

	// Generate operation ID from type name
	route.Handler = alias.Name
	route.OperationID = generateOperationID(route.Method, route.Path, alias.Name)

	// Infer tags
	route.Tags = inferTags(route.Path)

	if route.Method == "" {
		return nil
	}

	return route
}

// reorderPathWithCaptures properly orders path segments and captures.
func (p *Plugin) reorderPathWithCaptures(def string) string {
	var pathParts []string

	// Split by :> to get segments in order
	parts := strings.Split(def, ":>")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Check if it's a path segment
		if segMatch := regexp.MustCompile(`^"([^"]+)"$`).FindStringSubmatch(part); len(segMatch) > 1 {
			pathParts = append(pathParts, segMatch[1])
			continue
		}

		// Check if it's a Capture
		if capMatch := servantCaptureRegex.FindStringSubmatch(part + " :>"); len(capMatch) > 2 {
			pathParts = append(pathParts, "{"+capMatch[1]+"}")
			continue
		}

		// Skip other combinators (QueryParam, ReqBody, etc.)
	}

	if len(pathParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(pathParts, "/")
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	if handler != "" {
		// Convert CamelCase type name to camelCase operation ID
		return strings.ToLower(method) + handler
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

// ExtractSchemas extracts schema definitions from Haskell data types.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "haskell" {
			continue
		}

		pf := p.haskellParser.Parse(file.Path, file.Content)

		for _, dataType := range pf.DataTypes {
			schema := p.dataTypeToSchema(dataType)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// dataTypeToSchema converts a Haskell data type to an OpenAPI schema.
func (p *Plugin) dataTypeToSchema(dataType parser.HaskellDataType) *types.Schema {
	schema := &types.Schema{
		Title:      dataType.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range dataType.Fields {
		openAPIType, format := parser.HaskellTypeToOpenAPI(field.Type)
		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		// Handle nullable/optional fields (Maybe T)
		if field.IsOptional {
			propSchema.Nullable = true
		} else {
			schema.Required = append(schema.Required, field.Name)
		}

		schema.Properties[field.Name] = propSchema
	}

	return schema
}

// Register registers the Servant plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
