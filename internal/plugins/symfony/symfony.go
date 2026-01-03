// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package symfony provides a plugin for extracting routes from Symfony framework applications.
package symfony

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

// Plugin implements the FrameworkPlugin interface for Symfony framework.
type Plugin struct{}

// New creates a new Symfony plugin instance.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "symfony"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".php"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "symfony",
		Version:     "1.0.0",
		Description: "Extracts routes from Symfony framework applications",
		SupportedFrameworks: []string{
			"symfony/framework-bundle",
			"Symfony",
		},
	}
}

// Detect checks if Symfony is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check composer.json for symfony/framework-bundle
	composerPath := filepath.Join(projectRoot, "composer.json")
	if found, _ := p.checkFileForDependency(composerPath, "symfony/framework-bundle"); found {
		return true, nil
	}

	// Check for bin/console (Symfony signature)
	consolePath := filepath.Join(projectRoot, "bin", "console")
	if _, err := os.Stat(consolePath); err == nil {
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

// Regex patterns for Symfony route extraction
var (
	// Matches PHP 8 attribute: #[Route('/path', methods: ['GET'])]
	php8RouteAttrRegex = regexp.MustCompile(`#\[Route\s*\(\s*['"]([^'"]+)['"](?:\s*,\s*(?:name:\s*['"][^'"]+['"]|methods:\s*\[([^\]]+)\]|[^)]+))*\s*\)\]`)

	// Matches annotation style: @Route("/path", methods={"GET"})
	annotationRouteRegex = regexp.MustCompile(`@Route\s*\(\s*['"]([^'"]+)['"](?:\s*,\s*(?:name\s*=\s*['"][^'"]+['"]|methods\s*=\s*\{([^}]+)\}|[^)]+))*\s*\)`)

	// Matches class-level route prefix
	classRoutePrefixRegex = regexp.MustCompile(`(?:#\[Route\s*\(\s*['"]([^'"]+)['"]|@Route\s*\(\s*['"]([^'"]+)['"])`)

	// Matches public function declarations
	publicFunctionRegex = regexp.MustCompile(`(?m)public\s+function\s+(\w+)\s*\(`)

	// Matches class declarations
	classRegex = regexp.MustCompile(`(?m)class\s+(\w+)(?:\s+extends\s+\w+)?`)
)

// ExtractRoutes parses source files and extracts Symfony route definitions.
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

	// Find class name
	className := ""
	if match := classRegex.FindStringSubmatch(content); len(match) > 1 {
		className = match[1]
	}

	// Find class-level route prefix
	classPrefix := ""
	classMatch := classRoutePrefixRegex.FindStringSubmatch(content)
	if len(classMatch) > 2 {
		if classMatch[1] != "" {
			classPrefix = classMatch[1]
		} else if classMatch[2] != "" {
			classPrefix = classMatch[2]
		}
	}

	// Extract routes from PHP 8 attributes and annotations
	for i, line := range lines {
		lineNum := i + 1

		// Check for PHP 8 Route attribute
		if match := php8RouteAttrRegex.FindStringSubmatch(line); len(match) > 1 {
			path := match[1]
			methods := []string{"GET"}
			if len(match) > 2 && match[2] != "" {
				methods = parseSymfonyMethods(match[2])
			}

			// Find the associated function
			funcName := p.findNextFunction(lines, i)
			if funcName == "" {
				continue
			}

			for _, method := range methods {
				fullPath := combinePaths(classPrefix, path)
				fullPath = convertSymfonyPathParams(fullPath)
				params := extractPathParams(fullPath)
				operationID := generateOperationID(method, fullPath, funcName)
				tags := inferTags(fullPath, className)

				routes = append(routes, types.Route{
					Method:      method,
					Path:        fullPath,
					Handler:     className + "::" + funcName,
					OperationID: operationID,
					Tags:        tags,
					Parameters:  params,
					SourceFile:  file.Path,
					SourceLine:  lineNum,
				})
			}
		}

		// Check for annotation style @Route
		if match := annotationRouteRegex.FindStringSubmatch(line); len(match) > 1 {
			path := match[1]
			methods := []string{"GET"}
			if len(match) > 2 && match[2] != "" {
				methods = parseSymfonyMethods(match[2])
			}

			// Find the associated function
			funcName := p.findNextFunction(lines, i)
			if funcName == "" {
				continue
			}

			for _, method := range methods {
				fullPath := combinePaths(classPrefix, path)
				fullPath = convertSymfonyPathParams(fullPath)
				params := extractPathParams(fullPath)
				operationID := generateOperationID(method, fullPath, funcName)
				tags := inferTags(fullPath, className)

				routes = append(routes, types.Route{
					Method:      method,
					Path:        fullPath,
					Handler:     className + "::" + funcName,
					OperationID: operationID,
					Tags:        tags,
					Parameters:  params,
					SourceFile:  file.Path,
					SourceLine:  lineNum,
				})
			}
		}
	}

	return routes
}

// findNextFunction finds the next public function after the given line.
func (p *Plugin) findNextFunction(lines []string, startLine int) string {
	for i := startLine + 1; i < len(lines) && i < startLine+10; i++ {
		if match := publicFunctionRegex.FindStringSubmatch(lines[i]); len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

// parseSymfonyMethods parses methods from Symfony route declaration.
func parseSymfonyMethods(s string) []string {
	var methods []string

	// Match 'GET', 'POST', etc.
	re := regexp.MustCompile(`['"]([A-Z]+)['"]`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) > 1 {
			methods = append(methods, match[1])
		}
	}

	if len(methods) == 0 {
		return []string{"GET"}
	}

	return methods
}

// symfonyParamRegex matches Symfony path parameters like {param}.
var symfonyParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// convertSymfonyPathParams converts Symfony path params to OpenAPI format.
// Symfony already uses {param} format.
func convertSymfonyPathParams(path string) string {
	// Symfony uses {param} format which is already OpenAPI compatible
	return path
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

// inferTags infers tags from the route path and class name.
func inferTags(path, className string) []string {
	// Use controller name as tag if available
	if className != "" {
		// Remove Controller suffix
		tag := strings.TrimSuffix(className, "Controller")
		if tag != "" {
			return []string{tag}
		}
	}

	// Fall back to path-based tag
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

// ExtractSchemas extracts schema definitions from Symfony request classes.
func (p *Plugin) ExtractSchemas(_ []scanner.SourceFile) ([]types.Schema, error) {
	// Symfony doesn't have a standard schema definition pattern
	// Request classes and DTOs could be parsed, but that's complex
	return []types.Schema{}, nil
}

// Register registers the Symfony plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
