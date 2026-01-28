// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package phoenix provides a plugin for extracting routes from Phoenix framework applications.
package phoenix

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

// Plugin implements the FrameworkPlugin interface for Phoenix framework.
type Plugin struct {
	elixirParser *parser.ElixirParser
}

// New creates a new Phoenix plugin instance.
func New() *Plugin {
	return &Plugin{
		elixirParser: parser.NewElixirParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "phoenix"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".ex", ".exs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "phoenix",
		Version:     "1.0.0",
		Description: "Extracts routes from Phoenix framework applications",
		SupportedFrameworks: []string{
			":phoenix",
			"Phoenix",
		},
	}
}

// Detect checks if Phoenix is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check mix.exs for :phoenix dependency
	mixPath := filepath.Join(projectRoot, "mix.exs")
	if found, _ := p.checkFileForDependency(mixPath, ":phoenix"); found {
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
	for scanr.Scan() {
		line := scanr.Text()
		if strings.Contains(line, dep) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Phoenix route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "elixir" {
			continue
		}

		// Only process router files
		if !strings.Contains(file.Path, "router") {
			continue
		}

		pf := p.elixirParser.Parse(file.Path, file.Content)

		// Extract routes from scopes
		for _, scope := range pf.Scopes {
			scopeRoutes := p.extractRoutesFromScope(scope, file.Path)
			routes = append(routes, scopeRoutes...)
		}

		// Extract direct routes
		for _, route := range pf.Routes {
			r := p.convertRoute(route, "", file.Path)
			if r != nil {
				routes = append(routes, *r)
			}
		}

		// Expand and extract resource routes
		for _, resource := range pf.Resources {
			expandedRoutes := parser.ExpandElixirResources(resource)
			for _, route := range expandedRoutes {
				r := p.convertRoute(route, "", file.Path)
				if r != nil {
					routes = append(routes, *r)
				}
			}
		}
	}

	return routes, nil
}

// extractRoutesFromScope extracts routes from a Phoenix scope.
func (p *Plugin) extractRoutesFromScope(scope parser.ElixirScope, filePath string) []types.Route {
	var routes []types.Route

	// Extract routes within scope
	for _, route := range scope.Routes {
		r := p.convertRoute(route, scope.Path, filePath)
		if r != nil {
			routes = append(routes, *r)
		}
	}

	return routes
}

// convertRoute converts an Elixir route to a types.Route.
func (p *Plugin) convertRoute(route parser.ElixirRoute, prefix, filePath string) *types.Route {
	fullPath := combinePaths(prefix, route.Path)

	// Convert :param to {param} format
	fullPath = convertPhoenixPathParams(fullPath)

	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, route.Action)
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      route.Method,
		Path:        fullPath,
		Handler:     route.Controller + "." + route.Action,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// phoenixParamRegex matches Phoenix path parameters like :param.
var phoenixParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPhoenixPathParams converts Phoenix-style path params (:id) to OpenAPI format ({id}).
func convertPhoenixPathParams(path string) string {
	return phoenixParamRegex.ReplaceAllString(path, "{$1}")
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

// combinePaths combines a prefix and a path.
func combinePaths(prefix, path string) string {
	if prefix == "" {
		if !strings.HasPrefix(path, "/") {
			return "/" + path
		}
		return path
	}

	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	prefix = strings.TrimSuffix(prefix, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
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
		if strings.HasPrefix(part, "{") || strings.HasPrefix(part, ":") {
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

// ExtractSchemas extracts schema definitions from Ecto schemas.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "elixir" {
			continue
		}

		pf := p.elixirParser.Parse(file.Path, file.Content)

		for _, ectoSchema := range pf.Schemas {
			schema := p.convertEctoSchema(ectoSchema)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// convertEctoSchema converts an Ecto schema to an OpenAPI schema.
func (p *Plugin) convertEctoSchema(ecto parser.EctoSchema) *types.Schema {
	if len(ecto.Fields) == 0 {
		return nil
	}

	// Extract schema name from module name (e.g., "App.Schemas.User" -> "User")
	schemaName := ecto.ModuleName
	if parts := strings.Split(ecto.ModuleName, "."); len(parts) > 0 {
		schemaName = parts[len(parts)-1]
	}

	properties := make(map[string]*types.Schema)
	var required []string

	for _, field := range ecto.Fields {
		propSchema := p.ectoTypeToJSONSchema(field.Type)

		// Set default value if present
		if field.HasDefault && field.Default != "" {
			propSchema.Default = p.parseEctoDefault(field.Default, field.Type)
		}

		properties[field.Name] = propSchema

		// Fields without defaults are considered required
		if !field.HasDefault {
			required = append(required, field.Name)
		}
	}

	return &types.Schema{
		Type:       "object",
		Title:      schemaName,
		Properties: properties,
		Required:   required,
	}
}

// ectoTypeToJSONSchema converts an Ecto type to a JSON Schema type.
func (p *Plugin) ectoTypeToJSONSchema(ectoType string) *types.Schema {
	switch ectoType {
	case "string":
		return &types.Schema{Type: "string"}
	case "integer", "id":
		return &types.Schema{Type: "integer"}
	case "float", "decimal":
		return &types.Schema{Type: "number"}
	case "boolean":
		return &types.Schema{Type: "boolean"}
	case "date":
		return &types.Schema{Type: "string", Format: "date"}
	case "time":
		return &types.Schema{Type: "string", Format: "time"}
	case "datetime", "utc_datetime", "naive_datetime":
		return &types.Schema{Type: "string", Format: "date-time"}
	case "uuid", "binary_id":
		return &types.Schema{Type: "string", Format: "uuid"}
	case "map":
		return &types.Schema{Type: "object"}
	case "array":
		return &types.Schema{Type: "array", Items: &types.Schema{Type: "string"}}
	default:
		// Unknown type, default to string
		return &types.Schema{Type: "string"}
	}
}

// parseEctoDefault parses an Ecto default value.
func (p *Plugin) parseEctoDefault(defaultVal, ectoType string) interface{} {
	defaultVal = strings.TrimSpace(defaultVal)

	switch ectoType {
	case "boolean":
		return defaultVal == "true"
	case "integer", "id":
		// Try to parse as integer
		if defaultVal == "0" {
			return 0
		}
		return defaultVal
	case "float", "decimal":
		if defaultVal == "0" || defaultVal == "0.0" {
			return 0.0
		}
		return defaultVal
	default:
		// Return as string, removing quotes if present
		defaultVal = strings.Trim(defaultVal, "\"'")
		return defaultVal
	}
}

// Register registers the Phoenix plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
