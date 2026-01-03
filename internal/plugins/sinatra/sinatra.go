// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package sinatra provides a plugin for extracting routes from Sinatra framework applications.
package sinatra

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

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

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
func (p *Plugin) ExtractSchemas(_ []scanner.SourceFile) ([]types.Schema, error) {
	// Sinatra doesn't have a standard schema definition pattern
	return []types.Schema{}, nil
}

// Register registers the Sinatra plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
