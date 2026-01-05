// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package laravel provides a plugin for extracting routes from Laravel framework applications.
package laravel

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

// Plugin implements the FrameworkPlugin interface for Laravel framework.
type Plugin struct {
	phpParser *parser.PHPParser
}

// New creates a new Laravel plugin instance.
func New() *Plugin {
	return &Plugin{
		phpParser: parser.NewPHPParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "laravel"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".php"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "laravel",
		Version:     "1.0.0",
		Description: "Extracts routes from Laravel framework applications",
		SupportedFrameworks: []string{
			"laravel/framework",
			"Laravel",
		},
	}
}

// Detect checks if Laravel is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check composer.json for laravel/framework
	composerPath := filepath.Join(projectRoot, "composer.json")
	if found, _ := p.checkFileForDependency(composerPath, "laravel/framework"); found {
		return true, nil
	}

	// Check for artisan file (Laravel signature file)
	artisanPath := filepath.Join(projectRoot, "artisan")
	if _, err := os.Stat(artisanPath); err == nil {
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

// ExtractRoutes parses source files and extracts Laravel route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	// Track route groups for prefix application
	var currentPrefixes []string

	for _, file := range files {
		if file.Language != "php" {
			continue
		}

		pf := p.phpParser.Parse(file.Path, file.Content)

		// Extract route group prefixes
		for _, group := range pf.RouteGroups {
			currentPrefixes = append(currentPrefixes, group.Prefix)
		}

		// Extract explicit routes
		for _, route := range pf.Routes {
			r := p.convertRoute(route, currentPrefixes, file.Path)
			if r != nil {
				routes = append(routes, *r)
			}
		}

		// Expand and extract resource routes
		for _, resource := range pf.ResourceRoutes {
			expandedRoutes := parser.ExpandResourceRoutes(resource)
			for _, route := range expandedRoutes {
				r := p.convertRoute(route, currentPrefixes, file.Path)
				if r != nil {
					routes = append(routes, *r)
				}
			}
		}
	}

	return routes, nil
}

// convertRoute converts a PHP route to a types.Route.
func (p *Plugin) convertRoute(route parser.PHPRoute, prefixes []string, filePath string) *types.Route {
	// Apply prefixes
	fullPath := route.Path
	for i := len(prefixes) - 1; i >= 0; i-- {
		fullPath = combinePaths(prefixes[i], fullPath)
	}

	// Ensure path starts with /
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	// Laravel uses {param} format which is already OpenAPI compatible
	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, route.Action)
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      route.Method,
		Path:        fullPath,
		Handler:     route.Controller + "@" + route.Action,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// braceParamRegex matches path parameters like {param}.
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
		// Remove optional marker
		paramName = strings.TrimSuffix(paramName, "?")

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "path",
			Required: !strings.Contains(match[0], "?"),
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
		return path
	}

	prefix = strings.TrimSuffix(prefix, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" {
		return strings.ToLower(method) + toTitleCase(handler)
	}

	// Generate from path
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

// ExtractSchemas extracts schema definitions from PHP classes (models, DTOs, etc).
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "php" {
			continue
		}

		pf := p.phpParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			// Skip classes that look like controllers/services/etc
			if isImplementationClass(class.Name) {
				continue
			}

			schema := p.classToSchema(class)
			if schema != nil && len(schema.Properties) > 0 {
				schemas = append(schemas, *schema)
			}
		}
	}

	return schemas, nil
}

// classToSchema converts a PHP class to an OpenAPI schema.
func (p *Plugin) classToSchema(class parser.PHPClass) *types.Schema {
	schema := &types.Schema{
		Title:      class.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Handle Eloquent models via $fillable and $casts
	if class.IsEloquentModel {
		for _, field := range class.Fillable {
			propSchema := &types.Schema{Type: "string"} // Default type

			// Check if there's a cast that specifies the type
			if castType, ok := class.Casts[field]; ok {
				openAPIType, format := castTypeToOpenAPI(castType)
				propSchema.Type = openAPIType
				if format != "" {
					propSchema.Format = format
				}
			}

			schema.Properties[field] = propSchema
		}

		// Also add casted fields that aren't in fillable (like timestamps)
		for field, castType := range class.Casts {
			if _, exists := schema.Properties[field]; !exists {
				openAPIType, format := castTypeToOpenAPI(castType)
				propSchema := &types.Schema{Type: openAPIType}
				if format != "" {
					propSchema.Format = format
				}
				schema.Properties[field] = propSchema
			}
		}
	}

	// Handle plain PHP classes with properties (including constructor promoted)
	for _, prop := range class.Properties {
		// Only include public properties
		if prop.Visibility != "public" {
			continue
		}

		openAPIType, format := parser.PHPTypeToOpenAPI(prop.Type)
		propSchema := &types.Schema{
			Type: openAPIType,
		}
		if format != "" {
			propSchema.Format = format
		}
		if prop.IsNullable {
			propSchema.Nullable = true
		}

		schema.Properties[prop.Name] = propSchema

		if !prop.IsNullable {
			schema.Required = append(schema.Required, prop.Name)
		}
	}

	return schema
}

// castTypeToOpenAPI converts a Laravel cast type to OpenAPI type.
func castTypeToOpenAPI(castType string) (string, string) {
	castType = strings.ToLower(castType)

	switch castType {
	case "string", "encrypted":
		return "string", ""
	case "int", "integer":
		return "integer", ""
	case "real", "float", "double", "decimal":
		return "number", ""
	case "bool", "boolean":
		return "boolean", ""
	case "array", "collection":
		return "array", ""
	case "object":
		return "object", ""
	case "date":
		return "string", "date"
	case "datetime", "immutable_date", "immutable_datetime", "timestamp":
		return "string", "date-time"
	default:
		// Could be a custom cast class
		return "string", ""
	}
}

// isImplementationClass checks if a class name suggests it's implementation rather than a schema.
func isImplementationClass(name string) bool {
	implementationSuffixes := []string{
		"Controller",
		"Service",
		"Repository",
		"Factory",
		"Seeder",
		"Middleware",
		"Provider",
		"Request",
		"Resource",
		"Policy",
		"Observer",
		"Event",
		"Listener",
		"Job",
		"Mail",
		"Notification",
		"Console",
		"Command",
		"Exception",
		"Handler",
		"Kernel",
		"Test",
	}

	for _, suffix := range implementationSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// Register registers the Laravel plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
