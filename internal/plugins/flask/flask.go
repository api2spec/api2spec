// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package flask provides a plugin for extracting routes from Flask framework applications.
package flask

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// httpMethods maps HTTP method names to their uppercase forms.
var httpMethods = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"delete":  "DELETE",
	"patch":   "PATCH",
	"head":    "HEAD",
	"options": "OPTIONS",
}

// Plugin implements the FrameworkPlugin interface for Flask framework.
type Plugin struct {
	pyParser *parser.PythonParser
}

// New creates a new Flask plugin instance.
func New() *Plugin {
	return &Plugin{
		pyParser: parser.NewPythonParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "flask"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".py"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "flask",
		Version:     "1.0.0",
		Description: "Extracts routes from Flask framework applications",
		SupportedFrameworks: []string{
			"flask",
			"flask-restful",
			"flask-restx",
		},
	}
}

// Detect checks if Flask is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check requirements.txt
	reqPath := filepath.Join(projectRoot, "requirements.txt")
	if found, _ := p.checkFileForDependency(reqPath, "flask"); found {
		return true, nil
	}

	// Check pyproject.toml
	pyprojectPath := filepath.Join(projectRoot, "pyproject.toml")
	if found, _ := p.checkFileForDependency(pyprojectPath, "flask"); found {
		return true, nil
	}

	// Check setup.py
	setupPath := filepath.Join(projectRoot, "setup.py")
	if found, _ := p.checkFileForDependency(setupPath, "flask"); found {
		return true, nil
	}

	// Check Pipfile
	pipfilePath := filepath.Join(projectRoot, "Pipfile")
	if found, _ := p.checkFileForDependency(pipfilePath, "flask"); found {
		return true, nil
	}

	// Check poetry.lock or pyproject.toml for poetry projects
	poetryLockPath := filepath.Join(projectRoot, "poetry.lock")
	if found, _ := p.checkFileForDependency(poetryLockPath, "flask"); found {
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
	defer file.Close()

	scanner := bufio.NewScanner(file)
	depLower := strings.ToLower(dep)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, depLower) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractRoutes parses source files and extracts Flask route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "python" {
			continue
		}

		fileRoutes, err := p.extractRoutesFromFile(file)
		if err != nil {
			// Log error but continue with other files
			continue
		}

		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// blueprintInfo tracks information about a Flask Blueprint.
type blueprintInfo struct {
	name      string
	urlPrefix string
}

// extractRoutesFromFile extracts routes from a single Python file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.pyParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports Flask
	if !p.hasFlaskImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Track blueprints and their prefixes
	blueprints := p.findBlueprints(pf.RootNode, file.Content)

	// Extract routes from decorated functions
	for _, fn := range pf.DecoratedFunctions {
		fnRoutes := p.extractRoutesFromFunction(fn, file.Content, blueprints)
		for i := range fnRoutes {
			fnRoutes[i].SourceFile = file.Path
			routes = append(routes, fnRoutes[i])
		}
	}

	// Extract routes from MethodView classes
	for _, cls := range pf.Classes {
		clsRoutes := p.extractRoutesFromClass(cls, file.Content, blueprints)
		for i := range clsRoutes {
			clsRoutes[i].SourceFile = file.Path
			routes = append(routes, clsRoutes[i])
		}
	}

	return routes, nil
}

// hasFlaskImport checks if the file imports Flask.
func (p *Plugin) hasFlaskImport(pf *parser.ParsedPythonFile) bool {
	for _, imp := range pf.Imports {
		if strings.Contains(strings.ToLower(imp.Module), "flask") {
			return true
		}
	}
	return false
}

// findBlueprints finds Blueprint definitions in the file.
func (p *Plugin) findBlueprints(rootNode *sitter.Node, content []byte) map[string]*blueprintInfo {
	blueprints := make(map[string]*blueprintInfo)

	p.pyParser.WalkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "assignment" {
			bp := p.parseBlueprint(node, content)
			if bp != nil {
				blueprints[bp.name] = bp
			}
		}
		return true
	})

	return blueprints
}

// parseBlueprint parses a Blueprint assignment.
func (p *Plugin) parseBlueprint(node *sitter.Node, content []byte) *blueprintInfo {
	var varName string
	var isBlueprint bool
	var urlPrefix string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if varName == "" {
				varName = child.Content(content)
			}
		case "call":
			callText := child.Content(content)
			if strings.Contains(callText, "Blueprint") {
				isBlueprint = true
				// Try to extract url_prefix
				urlPrefix = p.extractBlueprintUrlPrefix(child, content)
			}
		}
	}

	if isBlueprint && varName != "" {
		return &blueprintInfo{
			name:      varName,
			urlPrefix: urlPrefix,
		}
	}

	return nil
}

// extractBlueprintUrlPrefix extracts the url_prefix from a Blueprint call.
func (p *Plugin) extractBlueprintUrlPrefix(node *sitter.Node, content []byte) string {
	args := p.pyParser.GetCallArguments(node, content)
	for _, arg := range args {
		if arg.Type() == "keyword_argument" {
			argText := arg.Content(content)
			if strings.Contains(argText, "url_prefix") {
				// Extract the value
				re := regexp.MustCompile(`url_prefix\s*=\s*['"]([^'"]+)['"]`)
				matches := re.FindStringSubmatch(argText)
				if len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}
	return ""
}

// extractRoutesFromFunction extracts routes from a decorated function.
func (p *Plugin) extractRoutesFromFunction(fn parser.PythonDecoratedFunction, content []byte, blueprints map[string]*blueprintInfo) []types.Route {
	var routes []types.Route

	for _, dec := range fn.Decorators {
		route := p.parseRouteDecorator(dec, fn, content, blueprints)
		if route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

// parseRouteDecorator parses a route decorator and extracts route information.
func (p *Plugin) parseRouteDecorator(dec parser.PythonDecorator, fn parser.PythonDecoratedFunction, _ []byte, blueprints map[string]*blueprintInfo) *types.Route {
	// Check for @app.route, @bp.route, @app.get, @bp.get, etc.
	parts := strings.Split(dec.Name, ".")
	if len(parts) < 2 {
		return nil
	}

	objectName := parts[0]
	methodName := parts[1]

	// Determine the prefix from blueprint
	prefix := ""
	if bp, ok := blueprints[objectName]; ok {
		prefix = bp.urlPrefix
	}

	var path string
	var methods []string

	if methodName == "route" {
		// @app.route('/path', methods=['GET', 'POST'])
		if len(dec.Arguments) > 0 {
			path = dec.Arguments[0]
		}
		// Check for methods keyword argument
		if methodsStr, ok := dec.KeywordArguments["methods"]; ok {
			methods = parseMethodsList(methodsStr)
		} else {
			// Default to GET
			methods = []string{"GET"}
		}
	} else if httpMethod, ok := httpMethods[strings.ToLower(methodName)]; ok {
		// @app.get('/path'), @app.post('/path'), etc.
		if len(dec.Arguments) > 0 {
			path = dec.Arguments[0]
		}
		methods = []string{httpMethod}
	} else {
		return nil
	}

	if path == "" {
		return nil
	}

	// Create routes for each method
	if len(methods) == 0 {
		methods = []string{"GET"}
	}

	// For simplicity, return the first method (we could return multiple routes)
	method := methods[0]

	// Combine prefix and path
	fullPath := combinePaths(prefix, path)

	// Extract path parameters (before converting, to get type hints)
	params := extractPathParamsWithTypes(fullPath)

	// Convert Flask path parameters to OpenAPI format
	fullPath = convertPathParams(fullPath)

	// Generate operation ID
	operationID := generateOperationID(method, fullPath, fn.Name)

	// Infer tags from path
	tags := inferTags(fullPath)

	return &types.Route{
		Method:      method,
		Path:        fullPath,
		Handler:     fn.Name,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceLine:  fn.Line,
	}
}

// extractRoutesFromClass extracts routes from a MethodView class.
func (p *Plugin) extractRoutesFromClass(cls parser.PythonClass, content []byte, blueprints map[string]*blueprintInfo) []types.Route {
	var routes []types.Route

	// Check if this is a MethodView class
	isMethodView := false
	for _, base := range cls.Bases {
		if base == "MethodView" || strings.HasSuffix(base, ".MethodView") {
			isMethodView = true
			break
		}
	}

	if !isMethodView {
		return routes
	}

	// Find the path from class decorators or registration
	path := p.findClassRoutePath(cls, content, blueprints)
	if path == "" {
		// Use class name as default path
		path = "/" + strings.ToLower(strings.TrimSuffix(cls.Name, "API"))
	}

	// Convert path parameters
	fullPath := convertPathParams(path)
	params := extractPathParams(fullPath)
	tags := inferTags(fullPath)

	// Extract methods (get, post, put, delete, etc.)
	for _, method := range cls.Methods {
		httpMethod, ok := httpMethods[strings.ToLower(method.Name)]
		if !ok {
			continue
		}

		operationID := generateOperationID(httpMethod, fullPath, method.Name)

		routes = append(routes, types.Route{
			Method:      httpMethod,
			Path:        fullPath,
			Handler:     cls.Name + "." + method.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceLine:  method.Line,
		})
	}

	return routes
}

// findClassRoutePath finds the route path for a MethodView class.
func (p *Plugin) findClassRoutePath(cls parser.PythonClass, _ []byte, _ map[string]*blueprintInfo) string {
	// TODO: Parse app.add_url_rule() calls to find path mappings
	// For now, infer from class name
	name := strings.TrimSuffix(cls.Name, "View")
	name = strings.TrimSuffix(name, "API")
	name = strings.TrimSuffix(name, "Resource")
	return "/" + strings.ToLower(name)
}

// ExtractSchemas extracts schema definitions from Pydantic models.
func (p *Plugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	var schemas []types.Schema

	for _, file := range files {
		if file.Language != "python" {
			continue
		}

		pf, err := p.pyParser.Parse(file.Path, file.Content)
		if err != nil {
			continue
		}

		for _, model := range pf.PydanticModels {
			schema := p.pydanticModelToSchema(model)
			if schema != nil {
				schemas = append(schemas, *schema)
			}
		}

		pf.Close()
	}

	return schemas, nil
}

// pydanticModelToSchema converts a Pydantic model to an OpenAPI schema.
func (p *Plugin) pydanticModelToSchema(model parser.PydanticModel) *types.Schema {
	schema := &types.Schema{
		Title:      model.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	for _, field := range model.Fields {
		propSchema := &types.Schema{}

		// Convert Python type to OpenAPI type
		openAPIType, format := parser.PythonTypeToOpenAPI(field.Type)
		propSchema.Type = openAPIType
		if format != "" {
			propSchema.Format = format
		}

		// Handle array types
		if strings.HasPrefix(field.Type, "List[") || strings.HasPrefix(field.Type, "list[") {
			propSchema.Type = "array"
			// Extract inner type
			innerType := extractGenericType(field.Type)
			innerOpenAPIType, innerFormat := parser.PythonTypeToOpenAPI(innerType)
			propSchema.Items = &types.Schema{
				Type:   innerOpenAPIType,
				Format: innerFormat,
			}
		}

		if field.Description != "" {
			propSchema.Description = field.Description
		}

		schema.Properties[field.Name] = propSchema

		if !field.IsOptional && field.Default == "" {
			schema.Required = append(schema.Required, field.Name)
		}
	}

	return schema
}

// --- Helper Functions ---

// flaskParamRegex matches Flask path parameters like <param>, <int:param>, <string:param>.
var flaskParamRegex = regexp.MustCompile(`<(?:([a-z]+):)?([a-zA-Z_][a-zA-Z0-9_]*)>`)

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// convertPathParams converts Flask-style path params (<id>, <int:id>) to OpenAPI format ({id}).
func convertPathParams(path string) string {
	return flaskParamRegex.ReplaceAllString(path, "{$2}")
}

// extractPathParams extracts path parameters from a route path (OpenAPI format).
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

// extractPathParamsWithTypes extracts path parameters from Flask-style path with type hints.
func extractPathParamsWithTypes(path string) []types.Parameter {
	var params []types.Parameter

	// Extract from Flask-style path: <type:name> or <name>
	matches := flaskParamRegex.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		typeHint := match[1]
		paramName := match[2]

		paramType := "string"
		if typeHint != "" {
			paramType = flaskTypeToOpenAPI(typeHint)
		}

		params = append(params, types.Parameter{
			Name:     paramName,
			In:       "path",
			Required: true,
			Schema: &types.Schema{
				Type: paramType,
			},
		})
	}

	return params
}

// flaskTypeToOpenAPI converts Flask type converters to OpenAPI types.
func flaskTypeToOpenAPI(flaskType string) string {
	switch flaskType {
	case "int":
		return "integer"
	case "float":
		return "number"
	case "path", "string", "any":
		return "string"
	case "uuid":
		return "string" // format: uuid
	default:
		return "string"
	}
}

// combinePaths combines a prefix and path, handling slashes correctly.
func combinePaths(prefix, path string) string {
	if prefix == "" {
		return path
	}

	// Remove trailing slash from prefix
	prefix = strings.TrimSuffix(prefix, "/")

	// Ensure path starts with slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return prefix + path
}

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	// If we have a handler name, use it
	if handler != "" && handler != "<anonymous>" {
		return strings.ToLower(method) + toTitleCase(handler)
	}

	// Generate from path
	// Remove parameter syntax and convert to camelCase
	path = braceParamRegex.ReplaceAllString(path, "By${1}")
	path = strings.ReplaceAll(path, "/", " ")
	path = strings.TrimSpace(path)

	words := strings.Fields(path)
	if len(words) == 0 {
		return strings.ToLower(method)
	}

	// Build camelCase operation ID
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
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		return nil
	}

	// Skip common prefixes like "api", "v1", etc.
	skipPrefixes := map[string]bool{
		"api": true,
		"v1":  true,
		"v2":  true,
		"v3":  true,
	}

	// Find the first meaningful segment
	var tagPart string
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Skip version/api prefixes
		if skipPrefixes[part] {
			continue
		}
		// Skip if it's a parameter
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

// parseMethodsList parses a methods list from a string like "['GET', 'POST']".
func parseMethodsList(s string) []string {
	var methods []string

	// Try to parse as JSON array
	s = strings.ReplaceAll(s, "'", `"`)
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		for _, m := range arr {
			methods = append(methods, strings.ToUpper(m))
		}
		return methods
	}

	// Fallback: extract strings manually
	re := regexp.MustCompile(`['"]([A-Za-z]+)['"]`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) > 1 {
			methods = append(methods, strings.ToUpper(match[1]))
		}
	}

	return methods
}

// extractGenericType extracts the inner type from a generic like List[str].
func extractGenericType(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// Register registers the Flask plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
