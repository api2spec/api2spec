// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package gleam provides a plugin for extracting routes from Gleam/Wisp applications.
package gleam

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

// Plugin implements the FrameworkPlugin interface for Gleam/Wisp framework.
type Plugin struct {
	gleamParser *parser.GleamParser
}

// New creates a new Gleam/Wisp plugin instance.
func New() *Plugin {
	return &Plugin{
		gleamParser: parser.NewGleamParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "gleam"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".gleam"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "gleam",
		Version:     "1.0.0",
		Description: "Extracts routes from Gleam/Wisp framework applications",
		SupportedFrameworks: []string{
			"wisp",
			"gleam_http",
			"mist",
		},
	}
}

// Detect checks if Wisp/Gleam HTTP is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check gleam.toml for wisp or gleam_http
	gleamTomlPath := filepath.Join(projectRoot, "gleam.toml")
	if found, _ := p.checkFileForDependency(gleamTomlPath, "wisp"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(gleamTomlPath, "gleam_http"); found {
		return true, nil
	}
	if found, _ := p.checkFileForDependency(gleamTomlPath, "mist"); found {
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

// ExtractRoutes parses source files and extracts Gleam/Wisp route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "gleam" {
			continue
		}

		pf := p.gleamParser.Parse(file.Path, file.Content)

		// Check if this file uses wisp or gleam_http
		hasHTTP := false
		for _, imp := range pf.Module.Imports {
			if strings.Contains(imp.Module, "wisp") ||
				strings.Contains(imp.Module, "gleam/http") ||
				strings.Contains(imp.Module, "mist") {
				hasHTTP = true
				break
			}
		}

		if !hasHTTP {
			continue
		}

		// Extract routes from parsed routes
		for _, route := range pf.Routes {
			r := p.convertRoute(route, file.Path)
			if r != nil {
				routes = append(routes, *r)
			}
		}
	}

	return routes, nil
}

// convertRoute converts a Gleam route to a types.Route.
func (p *Plugin) convertRoute(route parser.GleamRoute, filePath string) *types.Route {
	fullPath := route.Path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

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

// ExtractSchemas extracts schema definitions from Gleam types.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "gleam" {
			continue
		}

		pf := p.gleamParser.Parse(file.Path, file.Content)

		for _, t := range pf.Module.Types {
			if !t.IsPublic {
				continue
			}

			schema := p.typeToSchema(t)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// typeToSchema converts a Gleam type to an OpenAPI schema.
func (p *Plugin) typeToSchema(t parser.GleamType) *types.Schema {
	schema := &types.Schema{
		Title:      t.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range t.Fields {
		openAPIType, format := parser.GleamTypeToOpenAPI(field.Type)
		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		schema.Properties[field.Name] = propSchema
		schema.Required = append(schema.Required, field.Name)
	}

	return schema
}

// Register registers the Gleam/Wisp plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
