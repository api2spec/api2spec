// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package drf provides a plugin for extracting routes from Django REST Framework applications.
package drf

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

// Plugin implements the FrameworkPlugin interface for Django REST Framework.
type Plugin struct {
	pyParser *parser.PythonParser
}

// New creates a new DRF plugin instance.
func New() *Plugin {
	return &Plugin{
		pyParser: parser.NewPythonParser(),
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "drf"
}

// Extensions returns the file extensions this plugin handles.
func (p *Plugin) Extensions() []string {
	return []string{".py"}
}

// Info returns plugin metadata.
func (p *Plugin) Info() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "drf",
		Version:     "1.0.0",
		Description: "Extracts routes from Django REST Framework applications",
		SupportedFrameworks: []string{
			"djangorestframework",
			"Django REST Framework",
			"DRF",
		},
	}
}

// Detect checks if DRF is used in the project.
func (p *Plugin) Detect(projectRoot string) (bool, error) {
	// Check requirements.txt
	reqPath := filepath.Join(projectRoot, "requirements.txt")
	if found, _ := p.checkFileForDependency(reqPath, "djangorestframework"); found {
		return true, nil
	}

	// Check pyproject.toml
	pyprojectPath := filepath.Join(projectRoot, "pyproject.toml")
	if found, _ := p.checkFileForDependency(pyprojectPath, "djangorestframework"); found {
		return true, nil
	}

	// Check setup.py
	setupPath := filepath.Join(projectRoot, "setup.py")
	if found, _ := p.checkFileForDependency(setupPath, "djangorestframework"); found {
		return true, nil
	}

	// Check Pipfile
	pipfilePath := filepath.Join(projectRoot, "Pipfile")
	if found, _ := p.checkFileForDependency(pipfilePath, "djangorestframework"); found {
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

// ExtractRoutes parses source files and extracts DRF route definitions.
func (p *Plugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	var routes []types.Route

	for _, file := range files {
		if file.Language != "python" {
			continue
		}

		fileRoutes, err := p.extractRoutesFromFile(file)
		if err != nil {
			continue
		}

		routes = append(routes, fileRoutes...)
	}

	return routes, nil
}

// extractRoutesFromFile extracts routes from a single Python file.
func (p *Plugin) extractRoutesFromFile(file scanner.SourceFile) ([]types.Route, error) {
	pf, err := p.pyParser.Parse(file.Path, file.Content)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	// Check if this file imports DRF
	if !p.hasDRFImport(pf) {
		return nil, nil
	}

	var routes []types.Route

	// Extract routes from @api_view decorated functions
	for _, fn := range pf.DecoratedFunctions {
		fnRoutes := p.extractRoutesFromAPIView(fn, file.Path)
		routes = append(routes, fnRoutes...)
	}

	// Extract routes from ViewSet classes
	for _, cls := range pf.Classes {
		if p.isViewSet(cls) {
			clsRoutes := p.extractRoutesFromViewSet(cls, file.Path)
			routes = append(routes, clsRoutes...)
		}
	}

	return routes, nil
}

// hasDRFImport checks if the file imports DRF.
func (p *Plugin) hasDRFImport(pf *parser.ParsedPythonFile) bool {
	for _, imp := range pf.Imports {
		if strings.Contains(imp.Module, "rest_framework") {
			return true
		}
	}
	return false
}

// isViewSet checks if a class is a DRF ViewSet.
func (p *Plugin) isViewSet(cls parser.PythonClass) bool {
	for _, base := range cls.Bases {
		if strings.Contains(base, "ViewSet") ||
			strings.Contains(base, "ModelViewSet") ||
			strings.Contains(base, "GenericViewSet") ||
			strings.Contains(base, "ReadOnlyModelViewSet") ||
			strings.Contains(base, "APIView") {
			return true
		}
	}
	return false
}

// extractRoutesFromAPIView extracts routes from an @api_view decorated function.
func (p *Plugin) extractRoutesFromAPIView(fn parser.PythonDecoratedFunction, filePath string) []types.Route {
	var routes []types.Route

	for _, dec := range fn.Decorators {
		if dec.Name != "api_view" {
			continue
		}

		// Parse methods from @api_view(['GET', 'POST'])
		methods := p.parseAPIViewMethods(dec)
		if len(methods) == 0 {
			methods = []string{"GET"} // Default
		}

		// Generate routes for each method
		for _, method := range methods {
			// Path will be determined from urls.py, use function name as placeholder
			path := "/" + strings.ToLower(fn.Name)
			path = strings.ReplaceAll(path, "_", "-")

			params := extractPathParams(path)
			operationID := generateOperationID(method, path, fn.Name)
			tags := inferTags(path)

			routes = append(routes, types.Route{
				Method:      method,
				Path:        path,
				Handler:     fn.Name,
				OperationID: operationID,
				Tags:        tags,
				Parameters:  params,
				SourceFile:  filePath,
				SourceLine:  fn.Line,
			})
		}
	}

	return routes
}

// parseAPIViewMethods parses methods from @api_view decorator.
func (p *Plugin) parseAPIViewMethods(dec parser.PythonDecorator) []string {
	var methods []string

	if len(dec.Arguments) > 0 {
		arg := dec.Arguments[0]
		// Parse ['GET', 'POST'] format
		methods = parseMethodsList(arg)
	}

	return methods
}

// extractRoutesFromViewSet extracts routes from a ViewSet class.
func (p *Plugin) extractRoutesFromViewSet(cls parser.PythonClass, filePath string) []types.Route {
	var routes []types.Route

	// Infer base path from class name
	basePath := "/" + strings.ToLower(strings.TrimSuffix(cls.Name, "ViewSet"))
	basePath = strings.TrimSuffix(basePath, "view")

	// Check for ModelViewSet standard actions
	isModelViewSet := false
	for _, base := range cls.Bases {
		if strings.Contains(base, "ModelViewSet") {
			isModelViewSet = true
			break
		}
	}

	if isModelViewSet {
		// Add standard CRUD routes
		standardActions := []struct {
			method string
			path   string
			action string
		}{
			{"GET", basePath, "list"},
			{"POST", basePath, "create"},
			{"GET", basePath + "/{id}", "retrieve"},
			{"PUT", basePath + "/{id}", "update"},
			{"PATCH", basePath + "/{id}", "partial_update"},
			{"DELETE", basePath + "/{id}", "destroy"},
		}

		for _, action := range standardActions {
			params := extractPathParams(action.path)
			operationID := generateOperationID(action.method, action.path, action.action)
			tags := []string{cls.Name}

			routes = append(routes, types.Route{
				Method:      action.method,
				Path:        action.path,
				Handler:     cls.Name + "." + action.action,
				OperationID: operationID,
				Tags:        tags,
				Parameters:  params,
				SourceFile:  filePath,
				SourceLine:  cls.Line,
			})
		}
	}

	// Extract custom actions from @action decorated methods
	for _, method := range cls.Methods {
		for _, dec := range method.Decorators {
			if dec.Name == "action" {
				actionRoutes := p.extractActionRoutes(method, dec, basePath, cls.Name, filePath)
				routes = append(routes, actionRoutes...)
			}
		}
	}

	// Extract routes from standard method overrides (get, post, put, etc.)
	for _, method := range cls.Methods {
		httpMethod, ok := httpMethods[strings.ToLower(method.Name)]
		if !ok {
			continue
		}

		path := basePath
		params := extractPathParams(path)
		operationID := generateOperationID(httpMethod, path, method.Name)
		tags := []string{cls.Name}

		routes = append(routes, types.Route{
			Method:      httpMethod,
			Path:        path,
			Handler:     cls.Name + "." + method.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceFile:  filePath,
			SourceLine:  method.Line,
		})
	}

	return routes
}

// extractActionRoutes extracts routes from an @action decorated method.
func (p *Plugin) extractActionRoutes(method parser.PythonDecoratedFunction, dec parser.PythonDecorator, basePath, className, filePath string) []types.Route {
	var routes []types.Route

	// Parse methods from @action(methods=['get', 'post'])
	methods := []string{"GET"} // Default
	if methodsStr, ok := dec.KeywordArguments["methods"]; ok {
		methods = parseMethodsList(methodsStr)
	}

	// Check detail=True/False for path
	detail := false
	if detailStr, ok := dec.KeywordArguments["detail"]; ok {
		detail = strings.ToLower(detailStr) == "true"
	}

	// Determine URL name
	urlName := method.Name
	if urlPath, ok := dec.KeywordArguments["url_path"]; ok {
		urlName = strings.Trim(urlPath, `"'`)
	}

	// Build path
	var path string
	if detail {
		path = basePath + "/{id}/" + urlName
	} else {
		path = basePath + "/" + urlName
	}

	for _, httpMethod := range methods {
		params := extractPathParams(path)
		operationID := generateOperationID(httpMethod, path, method.Name)
		tags := []string{className}

		routes = append(routes, types.Route{
			Method:      httpMethod,
			Path:        path,
			Handler:     className + "." + method.Name,
			OperationID: operationID,
			Tags:        tags,
			Parameters:  params,
			SourceFile:  filePath,
			SourceLine:  method.Line,
		})
	}

	return routes
}

// ExtractSchemas extracts schema definitions from DRF serializers.
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

		// Look for Serializer classes
		for _, cls := range pf.Classes {
			if p.isSerializer(cls) {
				schema := p.serializerToSchema(cls)
				if schema != nil {
					schemas = append(schemas, *schema)
				}
			}
		}

		// Also extract Pydantic models if present
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

// isSerializer checks if a class is a DRF Serializer.
func (p *Plugin) isSerializer(cls parser.PythonClass) bool {
	for _, base := range cls.Bases {
		if strings.Contains(base, "Serializer") ||
			strings.Contains(base, "ModelSerializer") {
			return true
		}
	}
	return false
}

// serializerToSchema converts a DRF Serializer to an OpenAPI schema.
func (p *Plugin) serializerToSchema(cls parser.PythonClass) *types.Schema {
	schema := &types.Schema{
		Title:      cls.Name,
		Type:       "object",
		Properties: make(map[string]*types.Schema),
		Required:   []string{},
	}

	// DRF serializers define fields in the class body
	// This is a simplified extraction; full parsing would require more analysis
	return schema
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

		openAPIType, format := parser.PythonTypeToOpenAPI(field.Type)
		propSchema.Type = openAPIType
		if format != "" {
			propSchema.Format = format
		}

		if strings.HasPrefix(field.Type, "List[") || strings.HasPrefix(field.Type, "list[") {
			propSchema.Type = "array"
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

// braceParamRegex matches OpenAPI-style path parameters like {param}.
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

// generateOperationID generates an operation ID from method, path, and handler.
func generateOperationID(method, path, handler string) string {
	if handler != "" && handler != "<anonymous>" {
		return strings.ToLower(method) + toTitleCase(handler)
	}

	cleanPath := braceParamRegex.ReplaceAllString(path, "By${1}")
	cleanPath = strings.ReplaceAll(cleanPath, "/", " ")
	cleanPath = strings.ReplaceAll(cleanPath, "-", " ")
	cleanPath = strings.ReplaceAll(cleanPath, "_", " ")
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

// parseMethodsList parses a methods list from a string like "['GET', 'POST']".
func parseMethodsList(s string) []string {
	var methods []string

	// Extract strings from array-like syntax
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

// Register registers the DRF plugin with the global registry.
func Register() {
	plugins.MustRegister(New())
}

func init() {
	Register()
}
