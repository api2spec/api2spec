// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package micronaut provides a plugin for extracting routes from Micronaut framework applications.
package micronaut

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

// Plugin implements the FrameworkPlugin interface for Micronaut framework.
type Plugin struct {
	javaParser   *parser.JavaParser
	kotlinParser *parser.KotlinParser
}

// New creates a new Micronaut plugin instance.
func New() *Plugin {
	return &Plugin{
		javaParser:   parser.NewJavaParser(),
		kotlinParser: parser.NewKotlinParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "micronaut"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".java", ".kt"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "micronaut",
		Version:     "1.0.0",
		Description: "Extracts routes from Micronaut framework applications",
		SupportedFrameworks: []string{
			"io.micronaut",
			"Micronaut",
		},
	}
}

// Detect checks if Micronaut is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check build.gradle for io.micronaut
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	if found, _ := p.checkFileForDependency(gradlePath, "io.micronaut"); found {
		return true, nil
	}

	// Check build.gradle.kts
	gradleKtsPath := filepath.Join(projectRoot, "build.gradle.kts")
	if found, _ := p.checkFileForDependency(gradleKtsPath, "io.micronaut"); found {
		return true, nil
	}

	// Check pom.xml
	pomPath := filepath.Join(projectRoot, "pom.xml")
	if found, _ := p.checkFileForDependency(pomPath, "io.micronaut"); found {
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

// Regex patterns for Micronaut route extraction
var (
	// Matches @Controller("/path") annotation
	controllerRegex = regexp.MustCompile(`@Controller\s*\(\s*"([^"]+)"\s*\)`)

	// Matches @Get("/path"), @Post("/path"), etc.
	httpMethodRegex = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Head|Options)\s*(?:\(\s*"([^"]*)"\s*\)|(?:\s|$))`)

	// Matches method declarations in Java
	javaMethodRegex = regexp.MustCompile(`(?m)(public|private|protected)?\s*([\w<>,\s\[\]?]+)\s+(\w+)\s*\(([^)]*)\)`)

	// Matches function declarations in Kotlin
	kotlinFunctionRegex = regexp.MustCompile(`(?m)fun\s+(\w+)\s*\(([^)]*)\)(?:\s*:\s*([\w<>,\s?]+))?`)

	// Matches @PathVariable, @QueryValue annotations for parameters
	pathVariableRegex = regexp.MustCompile(`@PathVariable\s*(?:\(\s*"?(\w+)"?\s*\))?`)
	queryValueRegex   = regexp.MustCompile(`@QueryValue\s*(?:\(\s*"?(\w+)"?\s*\))?`)
)

// ExtractRoutes parses source files and extracts Micronaut route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "java" && file.Language != "kotlin" {
			continue
		}

		fileRoutes := p.extractRoutesFromFile(file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single Java/Kotlin file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) []types.Route {
	var routes []types.Route
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	// Find controller base path
	basePath := ""
	if match := controllerRegex.FindStringSubmatch(content); len(match) > 1 {
		basePath = match[1]
	}

	// Track pending HTTP method for the next method declaration
	pendingMethod := ""
	pendingPath := ""
	pendingLine := 0

	for i, line := range lines {
		lineNum := i + 1

		// Check for HTTP method annotations
		if match := httpMethodRegex.FindStringSubmatch(line); len(match) > 1 {
			pendingMethod = strings.ToUpper(match[1])
			if len(match) > 2 {
				pendingPath = match[2]
			} else {
				pendingPath = ""
			}
			pendingLine = lineNum
			continue
		}

		// If we have a pending HTTP method, look for the method declaration
		if pendingMethod != "" {
			var methodName string
			var methodParams string

			if file.Language == "java" {
				if match := javaMethodRegex.FindStringSubmatch(line); len(match) > 3 {
					methodName = match[3]
					methodParams = match[4]
				}
			} else if file.Language == "kotlin" {
				if match := kotlinFunctionRegex.FindStringSubmatch(line); len(match) > 1 {
					methodName = match[1]
					if len(match) > 2 {
						methodParams = match[2]
					}
				}
			}

			if methodName != "" {
				fullPath := combinePaths(basePath, pendingPath)
				fullPath = convertMicronautPathParams(fullPath)
				params := extractPathParams(fullPath)

				// Extract query parameters from method signature
				queryParams := extractQueryParams(methodParams)
				params = append(params, queryParams...)

				operationID := generateOperationID(pendingMethod, fullPath, methodName)
				tags := inferTags(fullPath)

				routes = append(routes, types.Route{
					Method:      pendingMethod,
					Path:        fullPath,
					Handler:     methodName,
					OperationID: operationID,
					Tags:        tags,
					Parameters:  params,
					SourceFile:  file.Path,
					SourceLine:  pendingLine,
				})

				// Reset pending
				pendingMethod = ""
				pendingPath = ""
				pendingLine = 0
			}
		}
	}

	return routes
}

// extractQueryParams extracts query parameters from method signature.
func extractQueryParams(params string) []types.Parameter {
	var queryParams []types.Parameter

	// Look for @QueryValue annotations
	matches := queryValueRegex.FindAllStringSubmatch(params, -1)
	for _, match := range matches {
		paramName := ""
		if len(match) > 1 && match[1] != "" {
			paramName = match[1]
		}

		if paramName == "" {
			// Try to extract parameter name from the following identifier
			// This is a simplification; real extraction would need full parsing
			continue
		}

		queryParams = append(queryParams, types.Parameter{
			Name:     paramName,
			In:       "query",
			Required: false,
			Schema: &types.Schema{
				Type: "string",
			},
		})
	}

	return queryParams
}

// micronautParamRegex matches Micronaut path parameters like {param}.
var micronautParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertMicronautPathParams converts Micronaut path params to OpenAPI format.
// Micronaut uses {param} format which is already OpenAPI compatible.
func convertMicronautPathParams(path string) string {
	// Micronaut uses {param} format which is already OpenAPI compatible
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

// combinePaths combines a base path and path.
func combinePaths(basePath, path string) string {
	// Handle empty paths
	if basePath == "" && path == "" {
		return "/"
	}

	if basePath == "" {
		if !strings.HasPrefix(path, "/") {
			return "/" + path
		}
		return path
	}

	if path == "" {
		if !strings.HasPrefix(basePath, "/") {
			return "/" + basePath
		}
		return basePath
	}

	basePath = strings.TrimSuffix(basePath, "/")
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return basePath + path
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

// ExtractSchemas extracts schema definitions from Java/Kotlin DTOs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language == "java" {
			pf := p.javaParser.Parse(file.Path, file.Content)
			for _, cls := range pf.Classes {
				if p.isSchemaClass(cls.Name) {
					schema := p.javaClassToSchema(cls)
					if schema != nil {
						schemas = append(schemas, *schema)
					}
				}
			}
		} else if file.Language == "kotlin" {
			pf := p.kotlinParser.Parse(file.Path, file.Content)
			for _, cls := range pf.Classes {
				if p.isSchemaClass(cls.Name) {
					schema := p.kotlinClassToSchema(cls)
					if schema != nil {
						schemas = append(schemas, *schema)
					}
				}
			}
		}
	}

	return schemas, nil
}

// isSchemaClass checks if a class is likely a schema/DTO class.
func (p *Plugin) isSchemaClass(name string) bool {
	if strings.HasSuffix(name, "Dto") ||
		strings.HasSuffix(name, "DTO") ||
		strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "Model") {
		return true
	}
	return false
}

// javaClassToSchema converts a Java class to an OpenAPI schema.
func (p *Plugin) javaClassToSchema(cls parser.JavaClass) *types.Schema {
	schema := &types.Schema{
		Title:      cls.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Java parser extracts methods; property extraction would need enhancement
	return schema
}

// kotlinClassToSchema converts a Kotlin class to an OpenAPI schema.
func (p *Plugin) kotlinClassToSchema(cls parser.KotlinClass) *types.Schema {
	schema := &types.Schema{
		Title:      cls.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Kotlin parser extracts functions; property extraction would need enhancement
	return schema
}

// Register registers the Micronaut plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
