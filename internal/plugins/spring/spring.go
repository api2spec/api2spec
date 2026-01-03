// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package spring provides a plugin for extracting routes from Spring Boot applications.
package spring

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

// httpMethods maps Spring annotations to HTTP methods.
var httpMethods = map[string]string{
	"GetMapping":     "GET",
	"PostMapping":    "POST",
	"PutMapping":     "PUT",
	"DeleteMapping":  "DELETE",
	"PatchMapping":   "PATCH",
	"RequestMapping": "", // Method determined by method attribute
}

// Plugin implements the FrameworkPlugin interface for Spring Boot.
type Plugin struct {
	javaParser *parser.JavaParser
}

// New creates a new Spring Boot plugin instance.
func New() *Plugin {
	return &Plugin{
		javaParser: parser.NewJavaParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "spring"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".java"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "spring",
		Version:     "1.0.0",
		Description: "Extracts routes from Spring Boot applications",
		SupportedFrameworks: []string{
			"spring-boot",
			"Spring Boot",
			"Spring MVC",
		},
	}
}

// Detect checks if Spring Boot is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check pom.xml for Spring Boot
	pomPath := filepath.Join(projectRoot, "pom.xml")
	if found, _ := p.checkFileForDependency(pomPath, "spring-boot"); found {
		return true, nil
	}

	// Check build.gradle for Spring Boot
	gradlePath := filepath.Join(projectRoot, "build.gradle")
	if found, _ := p.checkFileForDependency(gradlePath, "spring-boot"); found {
		return true, nil
	}

	// Check build.gradle.kts for Spring Boot
	gradleKtsPath := filepath.Join(projectRoot, "build.gradle.kts")
	if found, _ := p.checkFileForDependency(gradleKtsPath, "spring-boot"); found {
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

// ExtractRoutes parses source files and extracts Spring Boot route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "java" {
			continue
		}

		pf := p.javaParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			if !p.isController(class) {
				continue
			}

			classRoutes := p.extractRoutesFromController(class, file.Path)
			routes = append(routes, classRoutes...)
		}
	}

	return routes, nil
}

// isController checks if a class is a Spring controller.
func (p *Plugin) isController(class parser.JavaClass) bool {
	return class.HasAnnotation("RestController") ||
		class.HasAnnotation("Controller")
}

// extractRoutesFromController extracts routes from a Spring controller.
func (p *Plugin) extractRoutesFromController(class parser.JavaClass, filePath string) []types.Route {
	var routes []types.Route

	// Get base path from @RequestMapping on class
	basePath := ""
	if reqMapping := class.GetAnnotation("RequestMapping"); reqMapping != nil {
		basePath = reqMapping.Value
		if basePath == "" {
			if v, ok := reqMapping.Attributes["value"]; ok {
				basePath = v
			} else if path, ok := reqMapping.Attributes["path"]; ok {
				basePath = path
			}
		}
	}

	controllerName := strings.TrimSuffix(class.Name, "Controller")

	// Extract routes from methods
	for _, method := range class.Methods {
		methodRoutes := p.extractRoutesFromMethod(method, basePath, controllerName, filePath)
		routes = append(routes, methodRoutes...)
	}

	return routes
}

// extractRoutesFromMethod extracts routes from a controller method.
func (p *Plugin) extractRoutesFromMethod(method parser.JavaMethod, basePath, controllerName, filePath string) []types.Route {
	var routes []types.Route

	for _, anno := range method.Annotations {
		defaultMethod, ok := httpMethods[anno.Name]
		if !ok {
			continue
		}

		// Determine HTTP method
		httpMethod := defaultMethod
		if httpMethod == "" {
			// For @RequestMapping, check method attribute
			if m, ok := anno.Attributes["method"]; ok {
				httpMethod = parseRequestMethod(m)
			} else {
				httpMethod = "GET" // Default
			}
		}

		// Get path from annotation
		path := anno.Value
		if path == "" {
			if v, ok := anno.Attributes["value"]; ok {
				path = v
			} else if pathAttr, ok := anno.Attributes["path"]; ok {
				path = pathAttr
			}
		}

		// Combine base path and method path
		fullPath := combinePaths(basePath, path)

		// Convert to OpenAPI format
		fullPath = convertSpringPathParams(fullPath)

		// Extract path parameters
		params := extractPathParams(fullPath)

		// Check for request body
		for _, param := range method.Parameters {
			if hasRequestBodyAnnotation(param) {
				// Request body parameter found - type extraction is a future enhancement
				_ = param // Mark as intentionally unused
			}
		}

		// Generate operation ID
		operationID := generateOperationID(httpMethod, fullPath, method.Name)

		// Use controller name as tag
		tags := []string{controllerName}

		routes = append(routes, types.Route{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     controllerName + "." + method.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceFile:  filePath,
			SourceLine:  method.Line,
		})
	}

	return routes
}

// parseRequestMethod parses the HTTP method from a RequestMethod enum value.
func parseRequestMethod(value string) string {
	value = strings.ToUpper(value)
	value = strings.TrimPrefix(value, "REQUESTMETHOD.")
	value = strings.TrimPrefix(value, "REQUEST_METHOD.")
	switch value {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
		return value
	default:
		return "GET"
	}
}

// hasRequestBodyAnnotation checks if a parameter has @RequestBody annotation.
func hasRequestBodyAnnotation(param parser.JavaParameter) bool {
	for _, anno := range param.Annotations {
		if anno.Name == "RequestBody" {
			return true
		}
	}
	return false
}

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertSpringPathParams converts Spring-style path params to OpenAPI format.
// Spring already uses {param} format, but we clean up any regex patterns.
func convertSpringPathParams(path string) string {
	// Remove regex patterns like {id:\\d+}
	regexParamRegex := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*):[^}]+\}`)
	return regexParamRegex.ReplaceAllString(path, "{$1}")
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

// combinePaths combines a base path and a relative path.
func combinePaths(base, relative string) string {
	if base == "" {
		if relative == "" {
			return "/"
		}
		if !strings.HasPrefix(relative, "/") {
			return "/" + relative
		}
		return relative
	}

	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}

	if relative == "" {
		return base
	}

	base = strings.TrimSuffix(base, "/")
	if !strings.HasPrefix(relative, "/") {
		relative = "/" + relative
	}

	return base + relative
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

// ExtractSchemas extracts schema definitions from Java DTOs.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "java" {
			continue
		}

		pf := p.javaParser.Parse(file.Path, file.Content)

		for _, class := range pf.Classes {
			// Skip controllers
			if p.isController(class) {
				continue
			}

			// Check if this looks like a DTO
			if strings.HasSuffix(class.Name, "Dto") ||
				strings.HasSuffix(class.Name, "DTO") ||
				strings.HasSuffix(class.Name, "Request") ||
				strings.HasSuffix(class.Name, "Response") {

				schema := p.classToSchema(class)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// classToSchema converts a Java class to an OpenAPI schema.
func (p *Plugin) classToSchema(class parser.JavaClass) *types.Schema {
	schema := &types.Schema{
		Title:      class.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// In a real implementation, we'd parse fields directly
	// For now, we use getter methods
	for _, method := range class.Methods {
		if strings.HasPrefix(method.Name, "get") && len(method.Parameters) == 0 {
			propName := strings.TrimPrefix(method.Name, "get")
			if len(propName) > 0 {
				propName = strings.ToLower(propName[:1]) + propName[1:]
			}

			openAPIType, format := parser.JavaTypeToOpenAPI(method.ReturnType)
			propSchema := &types.Schema{
				Type:   openAPIType,
				Format: format,
			}

			schema.Properties[propName] = propSchema
		}
	}

	return schema
}

// Register registers the Spring Boot plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
