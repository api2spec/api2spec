// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package sinatra provides a plugin for extracting routes from Sinatra framework applications.
package sinatra

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/plugins/ruby"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// Plugin implements the FrameworkPlugin interface for Sinatra framework.
type Plugin struct {
	rubyParser *parser.RubyParser
}

// New creates a new Sinatra plugin instance.
func New() *Plugin {
	return &Plugin{
		rubyParser: parser.NewRubyParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "sinatra"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".rb"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "sinatra",
		Version:     "1.0.0",
		Description: "Extracts routes from Sinatra framework applications",
		SupportedFrameworks: []string{
			"sinatra",
			"Sinatra",
		},
	}
}

// Detect checks if Sinatra is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check Gemfile for sinatra
	gemfilePath := filepath.Join(projectRoot, "Gemfile")
	if found, _ := p.checkFileForDependency(gemfilePath, "sinatra"); found {
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
		// Match gem 'sinatra' or gem "sinatra"
		if strings.Contains(line, `'`+dep+`'`) || strings.Contains(line, `"`+dep+`"`) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Sinatra route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "ruby" {
			continue
		}

		// Skip spec/test files - they contain test helper calls that look like routes
		if p.isTestFile(file.Path) {
			continue
		}

		pf := p.rubyParser.Parse(file.Path, file.Content)

		// Check if this file uses Sinatra
		hasSinatra := false
		for _, class := range pf.Classes {
			for _, mod := range class.Modules {
				if mod == "Sinatra" || strings.Contains(mod, "Sinatra") {
					hasSinatra = true
					break
				}
			}
			if class.SuperClass == "Sinatra::Base" || class.SuperClass == "Sinatra::Application" {
				hasSinatra = true
			}
		}

		// Also check for require 'sinatra' in raw content
		if strings.Contains(string(file.Content), "require 'sinatra'") ||
			strings.Contains(string(file.Content), `require "sinatra"`) {
			hasSinatra = true
		}

		if !hasSinatra {
			// Still process routes if found - Sinatra can be used without explicit require
			if len(pf.Routes) == 0 {
				continue
			}
		}

		// Extract routes
		for _, route := range pf.Routes {
			r := p.convertRoute(route, file.Path)
			if r != nil {
				routes = append(routes, *r)
			}
		}
	}

	return routes, nil
}

// isTestFile checks if a file path indicates a test/spec file.
func (p *Plugin) isTestFile(path string) bool {
	// Check for spec/ or test/ directory
	if strings.Contains(path, "/spec/") || strings.Contains(path, "/test/") {
		return true
	}

	// Check for _spec.rb or _test.rb suffix
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_spec.rb") || strings.HasSuffix(base, "_test.rb") {
		return true
	}

	// Check for spec_helper.rb or test_helper.rb
	if base == "spec_helper.rb" || base == "test_helper.rb" {
		return true
	}

	return false
}

// convertRoute converts a Ruby route to a types.Route.
func (p *Plugin) convertRoute(route parser.RubyRoute, filePath string) *types.Route {
	fullPath := route.Path
	if !strings.HasPrefix(fullPath, "/") {
		fullPath = "/" + fullPath
	}

	// Convert :param to {param} format
	fullPath = convertSinatraPathParams(fullPath)

	params := extractPathParams(fullPath)
	operationID := generateOperationID(route.Method, fullPath, "")
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      route.Method,
		Path:        fullPath,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  route.Line,
	}
}

// sinatraParamRegex matches Sinatra path parameters like :param.
var sinatraParamRegex = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)

// splatParamRegex matches Sinatra splat parameters like *.
var splatParamRegex = regexp.MustCompile(`\*`)

// convertSinatraPathParams converts Sinatra-style path params (:id) to OpenAPI format ({id}).
func convertSinatraPathParams(path string) string {
	// Convert :param to {param}
	path = sinatraParamRegex.ReplaceAllString(path, "{$1}")

	// Convert * (splat) to {splat}
	path = splatParamRegex.ReplaceAllString(path, "{splat}")

	return path
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

// ExtractSchemas extracts schema definitions from Sinatra files.
// For Sinatra, we extract schemas from:
// - Constant hash/array definitions (e.g., USERS = [{ id: 1, name: 'Alice' }])
// - Inline response hashes (e.g., { status: 'ok' }.to_json)
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	schemas := make(map[string]*types.Schema)

	for _, file := range files {
		if file.Language != "ruby" {
			continue
		}

		// Skip test files
		if p.isTestFile(file.Path) {
			continue
		}

		content := string(file.Content)

		// Extract schemas from constant definitions
		p.extractConstantSchemas(content, schemas)

		// Extract schemas from inline responses
		p.extractResponseSchemas(content, schemas)
	}

	// Convert map to slice
	var result []types.Schema
	for _, schema := range schemas {
		result = append(result, *schema)
	}

	return result, nil
}

// constantArrayHashRegex matches CONSTANT = [{ key: value }] patterns.
var constantArrayHashRegex = regexp.MustCompile(`([A-Z][A-Z_0-9]*)\s*=\s*\[\s*\{([^}]+)\}`)

// constantHashRegex matches CONSTANT = { key: value } patterns.
var constantHashRegex = regexp.MustCompile(`([A-Z][A-Z_0-9]*)\s*=\s*\{([^}]+)\}`)

// extractConstantSchemas extracts schemas from Ruby constant definitions.
func (p *Plugin) extractConstantSchemas(content string, schemas map[string]*types.Schema) {
	// Find array of hashes: USERS = [{ id: 1, name: 'Alice' }]
	arrayMatches := constantArrayHashRegex.FindAllStringSubmatch(content, -1)
	for _, match := range arrayMatches {
		if len(match) < 3 {
			continue
		}

		constantName := match[1]
		hashContent := match[2]

		props := p.extractHashProperties(hashContent)
		if len(props) == 0 {
			continue
		}

		// Convert USERS to User (singular)
		schemaName := p.singularize(constantName)

		if _, exists := schemas[schemaName]; !exists {
			schemas[schemaName] = &types.Schema{
				Title:      schemaName,
				Type:       "object",
				Properties: props,
			}
		}
	}

	// Find single hashes: STATUS = { code: 200 }
	hashMatches := constantHashRegex.FindAllStringSubmatch(content, -1)
	for _, match := range hashMatches {
		if len(match) < 3 {
			continue
		}

		constantName := match[1]
		hashContent := match[2]

		props := p.extractHashProperties(hashContent)
		if len(props) == 0 {
			continue
		}

		// Use constant name as schema name
		schemaName := p.titleCase(constantName)

		if _, exists := schemas[schemaName]; !exists {
			schemas[schemaName] = &types.Schema{
				Title:      schemaName,
				Type:       "object",
				Properties: props,
			}
		}
	}
}

// inlineHashToJSONRegex matches { key: value }.to_json patterns in route handlers.
var inlineHashToJSONRegex = regexp.MustCompile(`\{\s*([^}]+)\s*\}\.to_json`)

// extractResponseSchemas extracts schemas from inline response hashes.
func (p *Plugin) extractResponseSchemas(content string, schemas map[string]*types.Schema) {
	matches := inlineHashToJSONRegex.FindAllStringSubmatch(content, -1)

	// Track unique property sets to avoid duplicates
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		hashContent := match[1]
		props := p.extractHashProperties(hashContent)
		if len(props) == 0 {
			continue
		}

		// Create a key based on properties to detect duplicates
		var propKeys []string
		for key := range props {
			propKeys = append(propKeys, key)
		}
		sort.Strings(propKeys) // Ensure deterministic key order
		propKey := strings.Join(propKeys, ",")

		if seen[propKey] {
			continue
		}
		seen[propKey] = true

		// Try to infer a meaningful name from the properties
		schemaName := p.inferSchemaNameFromProps(props)
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

// symbolKeyRegex matches modern Ruby hash syntax: key: value.
var symbolKeyRegex = regexp.MustCompile(`(\w+):\s*([^,}\n]+)`)

// extractHashProperties extracts properties from a Ruby hash literal.
// Sinatra has special handling for "status" key - only skip if value is not a string literal.
func (p *Plugin) extractHashProperties(hashContent string) map[string]*types.Schema {
	props := make(map[string]*types.Schema)

	matches := symbolKeyRegex.FindAllStringSubmatch(hashContent, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		key := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])

		// Skip if key is a common Ruby keyword that's not a property
		// Only skip "status" if it's not a string literal (i.e., it's a symbol like :ok)
		if key == "status" && !strings.Contains(value, "'") && !strings.Contains(value, "\"") {
			continue
		}

		propSchema := ruby.InferPropertyType(key, value)
		props[key] = propSchema
	}

	return props
}

// singularize converts a plural constant name to a singular schema name.
func (p *Plugin) singularize(name string) string {
	// Convert to title case first
	name = p.titleCase(name)

	// Common pluralization patterns
	if strings.HasSuffix(name, "ies") {
		return strings.TrimSuffix(name, "ies") + "y"
	}
	if strings.HasSuffix(name, "es") && !strings.HasSuffix(name, "ses") {
		return strings.TrimSuffix(name, "es")
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		return strings.TrimSuffix(name, "s")
	}
	return name
}

// titleCase converts UPPER_CASE to TitleCase.
func (p *Plugin) titleCase(name string) string {
	// Split by underscore
	parts := strings.Split(strings.ToLower(name), "_")

	titleCaser := cases.Title(language.English)
	for i, part := range parts {
		parts[i] = titleCaser.String(part)
	}

	return strings.Join(parts, "")
}

// inferSchemaNameFromProps infers a schema name from property patterns.
func (p *Plugin) inferSchemaNameFromProps(props map[string]*types.Schema) string {
	// Check for common patterns
	_, hasStatus := props["status"]
	_, hasVersion := props["version"]
	if hasStatus && hasVersion {
		return "HealthCheck"
	}

	_, hasID := props["id"]
	_, hasName := props["name"]
	_, hasEmail := props["email"]
	if hasID && hasName && hasEmail {
		return "User"
	}

	_, hasTitle := props["title"]
	_, hasBody := props["body"]
	if hasID && hasTitle && hasBody {
		return "Post"
	}

	_, hasError := props["error"]
	if hasError {
		return "Error"
	}

	return ""
}

// Register registers the Sinatra plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
