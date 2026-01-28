// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package slim provides a plugin for extracting routes from Slim framework applications.
package slim

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// Plugin implements the FrameworkPlugin interface for Slim framework.
type Plugin struct{}

// New creates a new Slim plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "slim"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".php"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "slim",
		Version:     "1.0.0",
		Description: "Extracts routes from Slim framework applications",
		SupportedFrameworks: []string{
			"slim/slim",
			"Slim Framework",
		},
	}
}

// Detect checks if Slim is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check composer.json for slim/slim
	composerPath := filepath.Join(projectRoot, "composer.json")
	if found, _ := p.checkFileForDependency(composerPath, "slim/slim"); found {
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

// Regex patterns for Slim route extraction
var (
	// Matches $app->get('/path', ...), $app->post('/path', ...), etc.
	slimRouteRegex = regexp.MustCompile(`\$(app|group)\s*->\s*(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]`)

	// Matches $group->get('/path', ...)
	slimGroupRouteRegex = regexp.MustCompile(`\$(\w+)\s*->\s*(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]`)

	// Matches ->group('/prefix', function ($group) { ... })
	slimGroupRegex = regexp.MustCompile(`->\s*group\s*\(\s*['"]([^'"]+)['"]`)

	// Matches map(['GET', 'POST'], '/path', ...)
	slimMapRegex = regexp.MustCompile(`->\s*map\s*\(\s*\[([^\]]+)\]\s*,\s*['"]([^'"]+)['"]`)
)

// ExtractRoutes parses source files and extracts Slim route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "php" {
			continue
		}

		fileRoutes := p.extractRoutesFromFile(file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single PHP file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) []types.Route {
	var routes []types.Route
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	// Track group prefixes
	groupPrefixes := p.findGroupPrefixes(content)
	currentPrefix := ""

	for i, line := range lines {
		lineNum := i + 1

		// Check for group definitions to track prefix
		if match := slimGroupRegex.FindStringSubmatch(line); len(match) > 1 {
			currentPrefix = match[1]
		}

		// Check for $app->method('/path', ...)
		if match := slimRouteRegex.FindStringSubmatch(line); len(match) > 3 {
			method := strings.ToUpper(match[2])
			path := match[3]

			// Handle 'any' method
			if method == "ANY" {
				// Create routes for common methods
				for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
					route := p.createRoute(m, path, currentPrefix, file.Path, lineNum)
					routes = append(routes, route)
				}
			} else {
				route := p.createRoute(method, path, currentPrefix, file.Path, lineNum)
				routes = append(routes, route)
			}
		}

		// Check for $group->method('/path', ...)
		if match := slimGroupRouteRegex.FindStringSubmatch(line); len(match) > 3 {
			varName := match[1]
			method := strings.ToUpper(match[2])
			path := match[3]

			// Skip if it's the app variable (already handled above)
			if varName == "app" {
				continue
			}

			// Skip common non-route variable names (test code, internal objects)
			skipVars := map[string]bool{
				"this":     true, // PHPUnit test methods like $this->get()
				"response": true,
				"request":  true,
				"client":   true,
				"http":     true,
				"browser":  true,
			}
			if skipVars[varName] {
				continue
			}

			// Use the group prefix if we can find it
			prefix := p.findPrefixForVariable(varName, groupPrefixes)

			if method == "ANY" {
				for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
					route := p.createRoute(m, path, prefix, file.Path, lineNum)
					routes = append(routes, route)
				}
			} else {
				route := p.createRoute(method, path, prefix, file.Path, lineNum)
				routes = append(routes, route)
			}
		}

		// Check for ->map(['GET', 'POST'], '/path', ...)
		if match := slimMapRegex.FindStringSubmatch(line); len(match) > 2 {
			methodsStr := match[1]
			path := match[2]

			methods := parseMethodsList(methodsStr)
			for _, method := range methods {
				route := p.createRoute(method, path, currentPrefix, file.Path, lineNum)
				routes = append(routes, route)
			}
		}
	}

	return routes
}

// groupPrefixInfo tracks group prefix information.
type groupPrefixInfo struct {
	varName string
	prefix  string
}

// findGroupPrefixes finds all group prefix definitions in the content.
func (p *Plugin) findGroupPrefixes(content string) []groupPrefixInfo {
	var prefixes []groupPrefixInfo

	// Match patterns like: ->group('/api', function ($group) { ... })
	re := regexp.MustCompile(`->\s*group\s*\(\s*['"]([^'"]+)['"]\s*,\s*function\s*\(\s*\$(\w+)`)
	matches := re.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			prefixes = append(prefixes, groupPrefixInfo{
				varName: match[2],
				prefix:  match[1],
			})
		}
	}

	return prefixes
}

// findPrefixForVariable finds the prefix associated with a group variable.
func (p *Plugin) findPrefixForVariable(varName string, prefixes []groupPrefixInfo) string {
	for _, pf := range prefixes {
		if pf.varName == varName {
			return pf.prefix
		}
	}
	return ""
}

// createRoute creates a route from the extracted information.
func (p *Plugin) createRoute(method, path, prefix, filePath string, lineNum int) types.Route {
	fullPath := combinePaths(prefix, path)
	fullPath = convertSlimPathParams(fullPath)
	params := extractPathParams(fullPath)
	operationID := generateOperationID(method, fullPath, "")
	tags := inferTags(fullPath)

	return types.Route{
		Method:      method,
		Path:        fullPath,
		Handler:     "",
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceFile:  filePath,
		SourceLine:  lineNum,
	}
}

// parseMethodsList parses methods from a Slim map declaration.
func parseMethodsList(s string) []string {
	var methods []string

	re := regexp.MustCompile(`['"]([A-Z]+)['"]`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) > 1 {
			methods = append(methods, match[1])
		}
	}

	return methods
}

// slimParamRegex matches Slim path parameters like {param} or {param:pattern}.
var slimParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]+)?\}`)

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}:]+)\}`)

// convertSlimPathParams converts Slim path params to OpenAPI format.
// Slim uses {param} or {param:pattern} format.
func convertSlimPathParams(path string) string {
	// Remove pattern constraints: {id:\d+} -> {id}
	return slimParamRegex.ReplaceAllString(path, "{$1}")
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

// combinePaths combines a prefix and path.
func combinePaths(prefix, path string) string {
	if prefix == "" {
		if !strings.HasPrefix(path, "/") {
			return "/" + path
		}
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

// ExtractSchemas extracts schema definitions (Slim doesn't have standard schemas).
func (p *Plugin) ExtractSchemas(_ []scanner.SourceFile) ([]types.Schema, error) {
	// Slim doesn't have a standard schema definition pattern
	return []types.Schema{}, nil
}

// Register registers the Slim plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
