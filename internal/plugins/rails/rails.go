// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package rails provides a plugin for extracting routes from Ruby on Rails applications.
package rails

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
	"github.com/api2spec/api2spec/internal/plugins/ruby"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// Plugin implements the FrameworkPlugin interface for Ruby on Rails.
type Plugin struct {
	rubyParser *parser.RubyParser
}

// New creates a new Rails plugin instance.
func New() *Plugin {
	return &Plugin{
		rubyParser: parser.NewRubyParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "rails"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".rb"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "rails",
		Version:     "1.0.0",
		Description: "Extracts routes from Ruby on Rails applications",
		SupportedFrameworks: []string{
			"rails",
			"Ruby on Rails",
		},
	}
}

// Detect checks if Rails is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check Gemfile for rails
	gemfilePath := filepath.Join(projectRoot, "Gemfile")
	if found, _ := p.checkFileForDependency(gemfilePath, "rails"); found {
		return true, nil
	}

	// Check for config/routes.rb (Rails signature file)
	routesPath := filepath.Join(projectRoot, "config", "routes.rb")
	if _, err := os.Stat(routesPath); err == nil {
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
		// Match gem 'rails' or gem "rails"
		if strings.Contains(line, `'`+dep+`'`) || strings.Contains(line, `"`+dep+`"`) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Rails route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "ruby" {
			continue
		}

		// Focus on routes.rb files
		if !strings.Contains(file.Path, "routes") {
			continue
		}

		pf := p.rubyParser.Parse(file.Path, file.Content)

		// Extract routes from namespaces
		for _, ns := range pf.Namespaces {
			nsRoutes := p.extractRoutesFromNamespace(ns, file.Path)
			routes = append(routes, nsRoutes...)
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
			expandedRoutes := parser.ExpandRubyResources(resource)
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

// extractRoutesFromNamespace extracts routes from a Rails namespace.
func (p *Plugin) extractRoutesFromNamespace(ns parser.RubyNamespace, filePath string) []types.Route {
	var routes []types.Route

	prefix := "/" + ns.Name

	// Extract routes within namespace
	for _, route := range ns.Routes {
		r := p.convertRoute(route, prefix, filePath)
		if r != nil {
			routes = append(routes, *r)
		}
	}

	// Expand and extract resource routes within namespace
	for _, resource := range ns.Resources {
		expandedRoutes := parser.ExpandRubyResources(resource)
		for _, route := range expandedRoutes {
			r := p.convertRoute(route, prefix, filePath)
			if r != nil {
				routes = append(routes, *r)
			}
		}
	}

	return routes
}

// convertRoute converts a Ruby route to a types.Route.
func (p *Plugin) convertRoute(route parser.RubyRoute, prefix, filePath string) *types.Route {
	fullPath := combinePaths(prefix, route.Path)

	// Convert :param to {param} format
	fullPath = convertRailsPathParams(fullPath)

	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, route.Action)
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      route.Method,
		Path:        fullPath,
		Handler:     route.Controller + "#" + route.Action,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// railsParamRegex matches Rails path parameters like :param.
var railsParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)


// convertRailsPathParams converts Rails-style path params (:id) to OpenAPI format ({id}).
func convertRailsPathParams(path string) string {
	return railsParamRegex.ReplaceAllString(path, "{$1}")
}

// extractPathParams extracts path parameters from a route path.
func extractPathParams(path string) []types.Parameter {
	var params []types.Parameter

	matches := ruby.BraceParamRegex.FindAllStringSubmatch(path, -1)
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

	cleanPath := ruby.BraceParamRegex.ReplaceAllString(path, "By${1}")
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

// ExtractSchemas extracts schema definitions from Ruby files.
// For Rails, we extract schemas from:
// - render json: calls with inline hash structures
// - params.permit() calls for request body schemas
// - Constant hash definitions that represent data models
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	schemas := make(map[string]*types.Schema)

	for _, file := range files {
		if file.Language != "ruby" {
			continue
		}

		content := string(file.Content)

		// Extract schemas from controller files
		if strings.Contains(file.Path, "controller") {
			p.extractControllerSchemas(content, schemas)
		}
	}

	// Convert map to slice
	var result []types.Schema
	for _, schema := range schemas {
		result = append(result, *schema)
	}

	return result, nil
}

// extractControllerSchemas extracts schemas from Rails controller files.
func (p *Plugin) extractControllerSchemas(content string, schemas map[string]*types.Schema) {
	// Extract schemas from render json: calls
	p.extractRenderJSONSchemas(content, schemas)

	// Extract schemas from params.permit() calls
	p.extractStrongParamsSchemas(content, schemas)
}

// renderJSONRegex matches render json: with hash literals or arrays of hashes.
var renderJSONHashRegex = regexp.MustCompile(`render\s+json:\s*\{([^}]+)\}`)
var renderJSONArrayRegex = regexp.MustCompile(`render\s+json:\s*\[\s*\{([^}]+)\}`)

// railsSkipKeys defines keys to skip when extracting hash properties in Rails context.
var railsSkipKeys = map[string]bool{
	"status": true,
	"to":     true,
}

// extractRenderJSONSchemas extracts schemas from render json: calls.
func (p *Plugin) extractRenderJSONSchemas(content string, schemas map[string]*types.Schema) {
	// Find render json: with hash literals
	matches := renderJSONHashRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		hashContent := match[1]
		props := ruby.ExtractHashProperties(hashContent, railsSkipKeys)
		if len(props) == 0 {
			continue
		}

		// Try to infer schema name from controller context
		schemaName := p.inferSchemaName(content, props)
		if schemaName == "" {
			continue
		}

		if _, exists := schemas[schemaName]; !exists {
			schemas[schemaName] = &types.Schema{
				Title:      schemaName,
				Type:       "object",
				Properties: props,
			}
		}
	}

	// Find render json: with arrays of hashes
	arrayMatches := renderJSONArrayRegex.FindAllStringSubmatch(content, -1)
	for _, match := range arrayMatches {
		if len(match) < 2 {
			continue
		}

		hashContent := match[1]
		props := ruby.ExtractHashProperties(hashContent, railsSkipKeys)
		if len(props) == 0 {
			continue
		}

		schemaName := p.inferSchemaName(content, props)
		if schemaName == "" {
			continue
		}

		if _, exists := schemas[schemaName]; !exists {
			schemas[schemaName] = &types.Schema{
				Title:      schemaName,
				Type:       "object",
				Properties: props,
			}
		}
	}
}

// strongParamsRegex matches params.permit(:field1, :field2) calls.
var strongParamsRegex = regexp.MustCompile(`params\.permit\(([^)]+)\)`)

// extractStrongParamsSchemas extracts schemas from params.permit() calls.
func (p *Plugin) extractStrongParamsSchemas(content string, schemas map[string]*types.Schema) {
	matches := strongParamsRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		paramsContent := match[1]
		props := p.extractSymbolProperties(paramsContent)
		if len(props) == 0 {
			continue
		}

		// Try to infer schema name from controller name
		schemaName := p.inferRequestSchemaName(content)
		if schemaName == "" {
			continue
		}

		if _, exists := schemas[schemaName]; !exists {
			schemas[schemaName] = &types.Schema{
				Title:      schemaName,
				Type:       "object",
				Properties: props,
			}
		}
	}
}

// symbolRegex matches Ruby symbol patterns.
var symbolRegex = regexp.MustCompile(`:(\w+)`)

// extractSymbolProperties extracts properties from params.permit(:field1, :field2).
func (p *Plugin) extractSymbolProperties(content string) map[string]*types.Schema {
	props := make(map[string]*types.Schema)

	matches := symbolRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		key := match[1]

		// Infer type from common naming conventions
		propSchema := ruby.InferPropertyType(key, "")
		props[key] = propSchema
	}

	return props
}

// controllerNameRegex extracts the controller class name.
var controllerNameRegex = regexp.MustCompile(`class\s+(\w+)Controller`)

// inferSchemaName infers a schema name from controller context and properties.
func (p *Plugin) inferSchemaName(content string, props map[string]*types.Schema) string {
	// Try to extract from controller name
	matches := controllerNameRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		// Convert UsersController to User
		name := matches[1]
		// Singularize if plural
		if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
			name = strings.TrimSuffix(name, "s")
		}
		return name
	}

	// Try to infer from common property patterns
	if _, hasID := props["id"]; hasID {
		if _, hasName := props["name"]; hasName {
			return "Resource"
		}
		if _, hasTitle := props["title"]; hasTitle {
			return "Item"
		}
	}

	return ""
}

// inferRequestSchemaName infers a request schema name from controller name.
func (p *Plugin) inferRequestSchemaName(content string) string {
	matches := controllerNameRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		name := matches[1]
		// Singularize if plural
		if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
			name = strings.TrimSuffix(name, "s")
		}
		return name + "Request"
	}
	return ""
}

// Register registers the Rails plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
