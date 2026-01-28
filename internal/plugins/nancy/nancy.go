// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package nancy provides a plugin for extracting routes from Nancy framework applications.
package nancy

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
	"github.com/api2spec/api2spec/internal/util"
	"github.com/api2spec/api2spec/pkg/types"
)

// Plugin implements the FrameworkPlugin interface for Nancy framework.
type Plugin struct {
	csParser *parser.CSharpParser
}

// New creates a new Nancy plugin instance.
func New() *Plugin {
	return &Plugin{
		csParser: parser.NewCSharpParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "nancy"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".cs"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "nancy",
		Version:     "1.0.0",
		Description: "Extracts routes from Nancy framework applications",
		SupportedFrameworks: []string{
			"Nancy",
			"NancyFx",
		},
	}
}

// Detect checks if Nancy is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check for *.csproj files containing Nancy
	csprojFiles, err := filepath.Glob(filepath.Join(projectRoot, "*.csproj"))
	if err != nil {
		return false, nil
	}

	for _, csprojPath := range csprojFiles {
		if found, _ := p.checkFileForDependency(csprojPath, "Nancy"); found {
			return true, nil
		}
	}

	// Check for packages.config
	packagesPath := filepath.Join(projectRoot, "packages.config")
	if found, _ := p.checkFileForDependency(packagesPath, "Nancy"); found {
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

// Regex patterns for Nancy route extraction
var (
	// Matches Get["/path"] = _ => ...
	nancyRouteRegex = regexp.MustCompile(`(?m)(Get|Post|Put|Delete|Patch|Head|Options)\s*\[\s*"([^"]+)"\s*\]`)

	// Matches Get("/path", ...)
	nancyRouteMethodRegex = regexp.MustCompile(`(?m)(Get|Post|Put|Delete|Patch|Head|Options)\s*\(\s*"([^"]+)"`)

	// Matches class declarations inheriting from NancyModule
	nancyModuleRegex = regexp.MustCompile(`class\s+(\w+)\s*:\s*NancyModule`)

	// Matches the module path in base constructor
	nancyModulePathRegex = regexp.MustCompile(`:\s*base\s*\(\s*"([^"]+)"`)
)

// ExtractRoutes parses source files and extracts Nancy route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "csharp" {
			continue
		}

		fileRoutes := p.extractRoutesFromFile(file)
		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single C# file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) []types.Route {
	var routes []types.Route
	content := string(file.Content)
	lines := strings.Split(content, "\n")

	// Find module class name and base path
	moduleName := ""
	basePath := ""

	if match := nancyModuleRegex.FindStringSubmatch(content); len(match) > 1 {
		moduleName = match[1]
	}

	if match := nancyModulePathRegex.FindStringSubmatch(content); len(match) > 1 {
		basePath = match[1]
	}

	for i, line := range lines {
		lineNum := i + 1

		// Check for indexed property syntax: Get["/path"] = _ => ...
		if match := nancyRouteRegex.FindStringSubmatch(line); len(match) > 2 {
			method := strings.ToUpper(match[1])
			path := match[2]

			fullPath := combinePaths(basePath, path)
			fullPath = convertNancyPathParams(fullPath)
			params := extractPathParams(fullPath)
			operationID := generateOperationID(method, fullPath, "")
			tags := inferTags(fullPath, moduleName)

			routes = append(routes, types.Route{
				Method:      method,
				Path:        fullPath,
				Handler:     moduleName,
				OperationID: operationID,
				Tags:        tags,
				Parameters:  params,
				SourceFile:  file.Path,
				SourceLine:  lineNum,
			})
		}

		// Check for method syntax: Get("/path", ...)
		if match := nancyRouteMethodRegex.FindStringSubmatch(line); len(match) > 2 {
			method := strings.ToUpper(match[1])
			path := match[2]

			fullPath := combinePaths(basePath, path)
			fullPath = convertNancyPathParams(fullPath)
			params := extractPathParams(fullPath)
			operationID := generateOperationID(method, fullPath, "")
			tags := inferTags(fullPath, moduleName)

			routes = append(routes, types.Route{
				Method:      method,
				Path:        fullPath,
				Handler:     moduleName,
				OperationID: operationID,
				Tags:        tags,
				Parameters:  params,
				SourceFile:  file.Path,
				SourceLine:  lineNum,
			})
		}
	}

	return routes
}

// nancyParamRegex matches Nancy path parameters like {param}.
var nancyParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// braceParamRegex matches OpenAPI-style path parameters.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertNancyPathParams converts Nancy path params to OpenAPI format.
// Nancy uses {param} format which is already OpenAPI compatible.
func convertNancyPathParams(path string) string {
	// Nancy uses {param} format which is already OpenAPI compatible
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
	if basePath == "" {
		if !strings.HasPrefix(path, "/") {
			return "/" + path
		}
		return path
	}

	basePath = strings.TrimSuffix(basePath, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return basePath + path
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	if handler != "" && handler != "<anonymous>" {
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

// inferTags infers tags from the route path and module name.
func inferTags(path, moduleName string) []string {
	// Use module name as tag if available
	if moduleName != "" {
		// Remove Module suffix
		tag := strings.TrimSuffix(moduleName, "Module")
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

// ExtractSchemas extracts schema definitions from C# classes and records.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "csharp" {
			continue
		}

		pf := p.csParser.Parse(file.Path, file.Content)

		// Extract schemas from records (e.g., record User(int Id, string Name))
		for _, record := range pf.Records {
			schema := p.recordToSchema(record)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}

		// Extract schemas from classes with properties
		for _, cls := range pf.Classes {
			// Include classes with properties or common DTO naming patterns
			hasProperties := len(cls.Properties) > 0
			isNamedAsDTO := p.isSchemaClass(cls)

			if hasProperties || isNamedAsDTO {
				schema := p.classToSchema(cls)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}
	}

	return schemas, nil
}

// isSchemaClass checks if a class is likely a schema/DTO class.
func (p *Plugin) isSchemaClass(cls parser.CSharpClass) bool {
	// Check for common naming patterns
	name := cls.Name
	if strings.HasSuffix(name, "Dto") ||
		strings.HasSuffix(name, "DTO") ||
		strings.HasSuffix(name, "Model") ||
		strings.HasSuffix(name, "Request") ||
		strings.HasSuffix(name, "Response") ||
		strings.HasSuffix(name, "ViewModel") {
		return true
	}
	return false
}

// recordToSchema converts a C# record to an OpenAPI schema.
func (p *Plugin) recordToSchema(record parser.CSharpClass) *types.Schema {
	schema := &types.Schema{
		Title:      record.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Convert record properties to schema properties
	for _, prop := range record.Properties {
		propName := util.ToLowerCamelCase(prop.Name)
		openAPIType, format := parser.CSharpTypeToOpenAPI(prop.Type)

		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		// Handle array types
		if openAPIType == "array" {
			innerType := util.ExtractInnerType(prop.Type)
			itemType, itemFormat := parser.CSharpTypeToOpenAPI(innerType)
			propSchema.Items = &types.Schema{
				Type:   itemType,
				Format: itemFormat,
			}
		}

		schema.Properties[propName] = propSchema

		if prop.IsRequired {
			schema.Required = append(schema.Required, propName)
		}
	}

	return schema
}

// classToSchema converts a C# class to an OpenAPI schema.
func (p *Plugin) classToSchema(cls parser.CSharpClass) *types.Schema {
	schema := &types.Schema{
		Title:      cls.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// Extract properties from class properties
	for _, prop := range cls.Properties {
		propName := util.ToLowerCamelCase(prop.Name)
		openAPIType, format := parser.CSharpTypeToOpenAPI(prop.Type)

		propSchema := &types.Schema{
			Type:   openAPIType,
			Format: format,
		}

		// Handle array types
		if openAPIType == "array" {
			innerType := util.ExtractInnerType(prop.Type)
			itemType, itemFormat := parser.CSharpTypeToOpenAPI(innerType)
			propSchema.Items = &types.Schema{
				Type:   itemType,
				Format: itemFormat,
			}
		}

		schema.Properties[propName] = propSchema

		if prop.IsRequired {
			schema.Required = append(schema.Required, propName)
		}
	}

	// Fallback: Extract properties from methods that look like getters
	if len(schema.Properties) == 0 {
		for _, method := range cls.Methods {
			if strings.HasPrefix(method.Name, "get_") {
				propName := util.ToLowerCamelCase(strings.TrimPrefix(method.Name, "get_"))
				openAPIType, format := parser.CSharpTypeToOpenAPI(method.ReturnType)

				propSchema := &types.Schema{
					Type:   openAPIType,
					Format: format,
				}

				schema.Properties[propName] = propSchema
			}
		}
	}

	return schema
}

// Register registers the Nancy plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
