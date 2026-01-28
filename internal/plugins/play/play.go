// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package play provides a plugin for extracting routes from Play Framework Scala applications.
package play

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

// Plugin implements the FrameworkPlugin interface for Play Framework.
type Plugin struct {
	scalaParser *parser.ScalaParser
}

// New creates a new Play Framework plugin instance.
func New() *Plugin {
	return &Plugin{
		scalaParser: parser.NewScalaParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "play"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".scala", ".sc"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "play",
		Version:     "1.0.0",
		Description: "Extracts routes from Play Framework Scala applications",
		SupportedFrameworks: []string{
			"play",
			"Play Framework",
			"com.typesafe.play",
		},
	}
}

// Detect checks if Play Framework is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check build.sbt for Play dependency
	buildSbtPath := filepath.Join(projectRoot, "build.sbt")
	if found, _ := p.checkFileForDependency(buildSbtPath, "com.typesafe.play"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(buildSbtPath, "play-"); found {
		return true, nil
	}

	// Check for conf/routes file (Play's signature file)
	routesPath := filepath.Join(projectRoot, "conf", "routes")
	if _, err := os.Stat(routesPath); err == nil {
		return true, nil
	}

	// Check project/plugins.sbt for Play plugin
	pluginsSbtPath := filepath.Join(projectRoot, "project", "plugins.sbt")
	if found, _ := p.checkFileForDependency(pluginsSbtPath, "sbt-plugin"); found {
		if found2, _ := p.checkFileForDependency(pluginsSbtPath, "play"); found2 {
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

// ExtractRoutes parses source files and extracts Play Framework route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	// Try to find and read conf/routes directly from project root
	// Derive project root from the first scanned file's path
	if len(files) > 0 {
		routesFile := p.findAndReadRoutesFile(files[0].Path)
		if routesFile != nil {
			pf := p.scalaParser.ParsePlayRoutes(routesFile.Path, routesFile.Content)
			for _, route := range pf.PlayRoutes {
				r := p.convertPlayRoute(route, routesFile.Path)
				if r != nil {
					routes = append(routes, *r)
				}
			}
		}
	}

	// Also check any routes files that might have been passed in directly
	for _, file := range files {
		// Check for conf/routes file (in case it was included)
		if strings.HasSuffix(file.Path, "conf/routes") || strings.HasSuffix(file.Path, "/routes") {
			// Skip if we already processed this file
			if len(routes) > 0 {
				continue
			}
			pf := p.scalaParser.ParsePlayRoutes(file.Path, file.Content)
			for _, route := range pf.PlayRoutes {
				r := p.convertPlayRoute(route, file.Path)
				if r != nil {
					routes = append(routes, *r)
				}
			}
		}
	}

	return routes, nil
}

// findAndReadRoutesFile finds the conf/routes file from a source file path.
func (p *Plugin) findAndReadRoutesFile(sourcePath string) *scanner.SourceFile {
	// Try to find project root by looking for common Play markers
	dir := filepath.Dir(sourcePath)

	// Walk up the directory tree to find project root
	for i := 0; i < 10; i++ { // Limit depth to avoid infinite loop
		routesPath := filepath.Join(dir, "conf", "routes")
		if content, err := os.ReadFile(routesPath); err == nil {
			return &scanner.SourceFile{
				Path:     routesPath,
				Language: "routes",
				Content:  content,
			}
		}

		// Check if we're at filesystem root
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return nil
}

// convertPlayRoute converts a parsed Play route to a types.Route.
func (p *Plugin) convertPlayRoute(route parser.PlayRoute, filePath string) *types.Route {
	// Convert Play path params to OpenAPI format
	path := parser.ConvertPlayPathParams(route.Path)

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Extract path parameters
	params := extractPathParams(path)

	// Generate operation ID
	handler := route.Action
	if route.Controller != "" {
		handler = route.Controller + "." + route.Action
	}
	operationID := generateOperationID(route.Method, path, route.Action)

	// Infer tags from controller name
	tags := inferTags(path)
	if route.Controller != "" {
		controllerName := route.Controller
		// Extract just the class name from full path
		if lastDot := strings.LastIndex(controllerName, "."); lastDot != -1 {
			controllerName = controllerName[lastDot+1:]
		}
		controllerName = strings.TrimSuffix(controllerName, "Controller")
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
		if strings.HasPrefix(part, "{") || strings.HasPrefix(part, ":") || strings.HasPrefix(part, "$") {
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

// ExtractSchemas extracts schema definitions from Scala case classes.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "scala" {
			continue
		}

		pf := p.scalaParser.Parse(file.Path, file.Content)

		for _, caseClass := range pf.CaseClasses {
			// Check if this looks like a DTO/model
			if strings.HasSuffix(caseClass.Name, "Dto") ||
				strings.HasSuffix(caseClass.Name, "DTO") ||
				strings.HasSuffix(caseClass.Name, "Request") ||
				strings.HasSuffix(caseClass.Name, "Response") ||
				strings.HasSuffix(caseClass.Name, "Model") ||
				strings.HasSuffix(caseClass.Name, "Form") {

				schema := p.caseClassToSchema(caseClass)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
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

// Register registers the Play Framework plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
