// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package fastapi provides a plugin for extracting routes from FastAPI framework applications.
package fastapi

import (
	"bufio"
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

// Plugin implements the FrameworkPlugin interface for FastAPI framework.
type Plugin struct {
	pyParser *parser.PythonParser
}

// New creates a new FastAPI plugin instance.
func New() *Plugin {
	return &Plugin{
		pyParser: parser.NewPythonParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "fastapi"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".py"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "fastapi",
		Version:     "1.0.0",
		Description: "Extracts routes from FastAPI framework applications",
		SupportedFrameworks: []string{
			"fastapi",
		},
	}
}

// Detect checks if FastAPI is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check requirements.txt
	reqPath := filepath.Join(projectRoot, "requirements.txt")
	if found, _ := p.checkFileForDependency(reqPath, "fastapi"); found {
		return true, nil
	}

	// Check pyproject.toml
	pyprojectPath := filepath.Join(projectRoot, "pyproject.toml")
	if found, _ := p.checkFileForDependency(pyprojectPath, "fastapi"); found {
		return true, nil
	}

	// Check setup.py
	setupPath := filepath.Join(projectRoot, "setup.py")
	if found, _ := p.checkFileForDependency(setupPath, "fastapi"); found {
		return true, nil
	}

	// Check Pipfile
	pipfilePath := filepath.Join(projectRoot, "Pipfile")
	if found, _ := p.checkFileForDependency(pipfilePath, "fastapi"); found {
		return true, nil
	}

	// Check poetry.lock
	poetryLockPath := filepath.Join(projectRoot, "poetry.lock")
	if found, _ := p.checkFileForDependency(poetryLockPath, "fastapi"); found {
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

// ExtractRoutes parses source files and extracts FastAPI route definitions.
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

// routerInfo tracks information about a FastAPI router or app.
type routerInfo struct {
	name   string
	prefix string
}

// extractRoutesFromFile extracts routes from a single Python file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.pyParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports FastAPI
	if !p.hasFastAPIImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Track routers and their prefixes
	routers := p.findRouters(pf.RootNode, file.Content)

	// Extract routes from decorated functions
	for _, fn := range pf.DecoratedFunctions {
		fnRoutes := p.extractRoutesFromFunction(fn, file.Content, routers)
		for i := range fnRoutes {
			fnRoutes[i].SourceFile = file.Path
			routes = append(routes, fnRoutes[i])
		}
	}

	return routes, nil
}

// hasFastAPIImport checks if the file imports FastAPI.
func (p *Plugin) hasFastAPIImport(pf *parser.ParsedPythonFile) bool {
	for _, imp := range pf.Imports {
		if strings.Contains(strings.ToLower(imp.Module), "fastapi") {
			return true
		}
	}
	return false
}

// findRouters finds APIRouter definitions in the file.
func (p *Plugin) findRouters(rootNode *sitter.Node, content []byte) map[string]*routerInfo {
	routers := make(map[string]*routerInfo)

	p.pyParser.WalkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "assignment" {
			router := p.parseRouter(node, content)
			if router != nil {
				routers[router.name] = router
			}
		}
		return true
	})

	return routers
}

// parseRouter parses an APIRouter assignment.
func (p *Plugin) parseRouter(node *sitter.Node, content []byte) *routerInfo {
	var varName string
	var isRouter bool
	var prefix string

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if varName == "" {
				varName = child.Content(content)
			}
		case "call":
			callText := child.Content(content)
			if strings.Contains(callText, "APIRouter") || strings.Contains(callText, "FastAPI") {
				isRouter = true
				// Try to extract prefix
				prefix = p.extractRouterPrefix(child, content)
			}
		}
	}

	if isRouter && varName != "" {
		return &routerInfo{
			name:   varName,
			prefix: prefix,
		}
	}

	return nil
}

// extractRouterPrefix extracts the prefix from an APIRouter call.
func (p *Plugin) extractRouterPrefix(node *sitter.Node, content []byte) string {
	args := p.pyParser.GetCallArguments(node, content)
	for _, arg := range args {
		if arg.Type() == "keyword_argument" {
			argText := arg.Content(content)
			if strings.Contains(argText, "prefix") {
				// Extract the value
				re := regexp.MustCompile(`prefix\s*=\s*['"]([^'"]+)['"]`)
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
func (p *Plugin) extractRoutesFromFunction(fn parser.PythonDecoratedFunction, content []byte, routers map[string]*routerInfo) []types.Route {
	var routes []types.Route

	for _, dec := range fn.Decorators {
		route := p.parseRouteDecorator(dec, fn, content, routers)
		if route != nil {
			routes = append(routes, *route)
		}
	}

	return routes
}

// parseRouteDecorator parses a route decorator and extracts route information.
func (p *Plugin) parseRouteDecorator(dec parser.PythonDecorator, fn parser.PythonDecoratedFunction, content []byte, routers map[string]*routerInfo) *types.Route {
	// Check for @app.get, @router.post, etc.
	parts := strings.Split(dec.Name, ".")
	if len(parts) < 2 {
		return nil
	}

	objectName := parts[0]
	methodName := parts[1]

	// Check if it's an HTTP method decorator
	httpMethod, ok := httpMethods[strings.ToLower(methodName)]
	if !ok {
		return nil
	}

	// Determine the prefix from router
	prefix := ""
	if router, ok := routers[objectName]; ok {
		prefix = router.prefix
	}

	// Get the path from decorator arguments
	var path string
	if len(dec.Arguments) > 0 {
		path = dec.Arguments[0]
	}

	if path == "" {
		return nil
	}

	// Combine prefix and path
	fullPath := combinePaths(prefix, path)

	// FastAPI uses {param} format already, but let's ensure consistency
	fullPath = normalizePathParams(fullPath)

	// Extract path parameters
	params := extractPathParams(fullPath)

	// Extract additional parameters from function signature
	queryParams := p.extractQueryParams(fn, content)
	params = append(params, queryParams...)

	// Generate operation ID
	operationID := generateOperationID(httpMethod, fullPath, fn.Name)

	// Infer tags from path
	tags := inferTags(fullPath)

	// Check for response_model in decorator arguments
	var responseSchema *types.Schema
	if responseModel, ok := dec.KeywordArguments["response_model"]; ok {
		responseSchema = &types.Schema{
			Ref: "#/components/schemas/" + responseModel,
		}
	}

	route := &types.Route{
		Method:      httpMethod,
		Path:        fullPath,
		Handler:     fn.Name,
		OperationID: operationID,
		Tags:        tags,
		Parameters:  params,
		SourceLine:  fn.Line,
	}

	// Add response if we have a response_model
	if responseSchema != nil {
		route.Responses = map[string]types.Response{
			"200": {
				Description: "Successful Response",
				Content: map[string]types.MediaType{
					"application/json": {
						Schema: responseSchema,
					},
				},
			},
		}
	}

	// Check for request body from typed parameters
	requestBody := p.extractRequestBody(fn, content)
	if requestBody != nil {
		route.RequestBody = requestBody
	}

	return route
}

// extractQueryParams extracts query parameters from function signature.
func (p *Plugin) extractQueryParams(fn parser.PythonDecoratedFunction, _ []byte) []types.Parameter {
	var params []types.Parameter

	for _, param := range fn.Parameters {
		// Skip common non-query parameters
		if param.Name == "self" || param.Name == "request" || param.Name == "db" ||
			param.Name == "session" || param.Name == "background_tasks" {
			continue
		}

		// Check if it's a path parameter (these are handled separately)
		// Path params are typically typed as Path(...) or have no default
		if strings.Contains(param.Type, "Path") {
			continue
		}

		// Check if it's a query parameter (Query(...) or has default)
		if strings.Contains(param.Type, "Query") || !param.IsRequired {
			openAPIType, format := parser.PythonTypeToOpenAPI(param.Type)

			queryParam := types.Parameter{
				Name:     param.Name,
				In:       "query",
				Required: param.IsRequired,
				Schema: &types.Schema{
					Type:   openAPIType,
					Format: format,
				},
			}

			params = append(params, queryParam)
		}
	}

	return params
}

// extractRequestBody extracts request body from function signature.
func (p *Plugin) extractRequestBody(fn parser.PythonDecoratedFunction, _ []byte) *types.RequestBody {
	for _, param := range fn.Parameters {
		// Look for Pydantic model types (typically the request body)
		// These are usually capitalized and not standard types
		if param.Type == "" || strings.HasPrefix(param.Type, "Optional") {
			continue
		}

		// Skip common non-body parameters
		if param.Name == "self" || param.Name == "request" || param.Name == "db" ||
			param.Name == "session" || param.Name == "background_tasks" ||
			strings.Contains(param.Type, "Query") || strings.Contains(param.Type, "Path") ||
			strings.Contains(param.Type, "Header") || strings.Contains(param.Type, "Cookie") {
			continue
		}

		// Check if type looks like a Pydantic model (capitalized, not a builtin)
		typeName := param.Type
		if strings.Contains(typeName, "[") {
			// Handle Optional[Type], List[Type], etc.
			typeName = extractGenericType(typeName)
		}

		builtinTypes := map[string]bool{
			"str": true, "int": true, "float": true, "bool": true,
			"list": true, "dict": true, "set": true, "tuple": true,
			"bytes": true, "None": true, "Any": true,
		}

		if builtinTypes[strings.ToLower(typeName)] {
			continue
		}

		// This looks like a Pydantic model
		if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' {
			return &types.RequestBody{
				Required: param.IsRequired,
				Content: map[string]types.MediaType{
					"application/json": {
						Schema: &types.Schema{
							Ref: "#/components/schemas/" + typeName,
						},
					},
				},
			}
		}
	}

	return nil
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

		// Handle Optional types
		if strings.HasPrefix(field.Type, "Optional[") {
			propSchema.Nullable = true
			innerType := extractGenericType(field.Type)
			innerOpenAPIType, innerFormat := parser.PythonTypeToOpenAPI(innerType)
			propSchema.Type = innerOpenAPIType
			propSchema.Format = innerFormat
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

// braceParamRegex matches OpenAPI-style path parameters like {param}.
var braceParamRegex = regexp.MustCompile(`\{([^}:]+)(?::[^}]+)?\}`)

// normalizePathParams normalizes path parameters to OpenAPI format.
// FastAPI already uses {param} format, but may include type hints like {item_id:int}.
func normalizePathParams(path string) string {
	// Remove type hints from path parameters: {item_id:int} -> {item_id}
	return braceParamRegex.ReplaceAllString(path, "{$1}")
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
	cleanPath := braceParamRegex.ReplaceAllString(path, "By${1}")
	cleanPath = strings.ReplaceAll(cleanPath, "/", " ")
	cleanPath = strings.TrimSpace(cleanPath)

	words := strings.Fields(cleanPath)
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

// extractGenericType extracts the inner type from a generic like List[str].
func extractGenericType(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// Register registers the FastAPI plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
