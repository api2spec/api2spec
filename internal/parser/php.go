// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// PHPParser provides PHP parsing capabilities using regex patterns.
type PHPParser struct{}

// NewPHPParser creates a new PHP parser.
func NewPHPParser() *PHPParser {
	return &PHPParser{}
}

// PHPClass represents a PHP class definition.
type PHPClass struct {
	// Name is the class name
	Name string

	// Namespace is the class namespace
	Namespace string

	// Extends is the parent class
	Extends string

	// Implements are the implemented interfaces
	Implements []string

	// Properties are the class properties
	Properties []PHPProperty

	// Methods are the class methods
	Methods []PHPMethod

	// IsEloquentModel indicates if this class extends Eloquent Model
	IsEloquentModel bool

	// Fillable contains Eloquent $fillable field names
	Fillable []string

	// Casts contains Eloquent $casts field type mappings
	Casts map[string]string

	// Line is the source line number
	Line int
}

// PHPProperty represents a PHP class property.
type PHPProperty struct {
	// Name is the property name
	Name string

	// Type is the property type
	Type string

	// Visibility is the property visibility (public, private, protected)
	Visibility string

	// IsNullable indicates if the property can be null
	IsNullable bool

	// IsReadonly indicates if the property is readonly
	IsReadonly bool

	// Line is the source line number
	Line int
}

// PHPMethod represents a PHP method definition.
type PHPMethod struct {
	// Name is the method name
	Name string

	// Visibility is the method visibility (public, private, protected)
	Visibility string

	// Parameters are the method parameters
	Parameters []PHPParameter

	// ReturnType is the return type
	ReturnType string

	// Line is the source line number
	Line int
}

// PHPParameter represents a method parameter.
type PHPParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Default is the default value
	Default string

	// IsOptional indicates if the parameter is optional
	IsOptional bool

	// IsPromoted indicates if this is a constructor promoted property (PHP 8+)
	IsPromoted bool

	// Visibility is set for promoted properties (public, private, protected)
	Visibility string
}

// PHPRoute represents a Laravel route definition.
type PHPRoute struct {
	// Method is the HTTP method (get, post, put, delete, etc.)
	Method string

	// Path is the route path
	Path string

	// Controller is the controller class
	Controller string

	// Action is the controller method
	Action string

	// Name is the route name
	Name string

	// Line is the source line number
	Line int
}

// PHPResourceRoute represents a Laravel resource route.
type PHPResourceRoute struct {
	// Path is the route path/resource name
	Path string

	// Controller is the controller class
	Controller string

	// Only specifies which actions to include
	Only []string

	// Except specifies which actions to exclude
	Except []string

	// IsAPI indicates if it's an apiResource
	IsAPI bool

	// Line is the source line number
	Line int
}

// ParsedPHPFile represents a parsed PHP source file.
type ParsedPHPFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Namespace is the file namespace
	Namespace string

	// Uses are the use statements
	Uses []string

	// Classes are the extracted class definitions
	Classes []PHPClass

	// Routes are the extracted route definitions
	Routes []PHPRoute

	// ResourceRoutes are the extracted resource route definitions
	ResourceRoutes []PHPResourceRoute

	// RouteGroups tracks route group prefixes
	RouteGroups []PHPRouteGroup
}

// PHPRouteGroup represents a Laravel route group.
type PHPRouteGroup struct {
	// Prefix is the route prefix
	Prefix string

	// Middleware is the group middleware
	Middleware []string

	// Line is the source line number
	Line int
}

// Regex patterns for PHP parsing
var (
	// Matches namespace declaration
	phpNamespaceRegex = regexp.MustCompile(`(?m)^namespace\s+([^;]+);`)

	// Matches use statements
	phpUseRegex = regexp.MustCompile(`(?m)^use\s+([^;]+);`)

	// Matches class definitions
	phpClassRegex = regexp.MustCompile(`(?ms)(?:abstract\s+)?class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([^{]+))?`)

	// Matches method definitions
	phpMethodRegex = regexp.MustCompile(`(?m)(public|private|protected)?\s*(?:static\s+)?function\s+(\w+)\s*\(([^)]*)\)(?:\s*:\s*\??([\w\\|]+))?`)

	// Matches traditional property declarations (must end with ; to distinguish from constructor params)
	// public string $name; or public ?string $name = 'default';
	phpPropertyRegex = regexp.MustCompile(`(?m)(public|private|protected)\s+(?:(readonly)\s+)?(\??\w+(?:\s*\|\s*\w+)*)\s+\$(\w+)\s*(?:=[^;]+)?;`)

	// Matches Eloquent $fillable array
	// protected $fillable = ['name', 'email', ...];
	phpFillableRegex = regexp.MustCompile(`(?ms)\$fillable\s*=\s*\[(.*?)\]`)

	// Matches Eloquent $casts array
	// protected $casts = ['email_verified_at' => 'datetime', ...];
	phpCastsRegex = regexp.MustCompile(`(?ms)\$casts\s*=\s*\[(.*?)\]`)

	// Matches Laravel route definitions
	// Route::get('/path', [Controller::class, 'method'])
	phpRouteRegex = regexp.MustCompile(`(?m)Route::(get|post|put|patch|delete|options|any)\s*\(\s*['"]([^'"]+)['"]\s*,\s*(?:\[\s*([^:]+)::class\s*,\s*['"](\w+)['"]\s*\]|['"]([^'"]+)['"])`)

	// Matches Laravel resource routes
	// Route::resource('users', UserController::class)
	phpResourceRegex = regexp.MustCompile(`(?m)Route::(resource|apiResource)\s*\(\s*['"]([^'"]+)['"]\s*,\s*([^:]+)::class`)

	// Matches route group with prefix
	// Route::prefix('api')->group(...)
	phpRoutePrefixRegex = regexp.MustCompile(`(?m)Route::prefix\s*\(\s*['"]([^'"]+)['"]`)

	// Matches route group definitions
	phpRouteGroupRegex = regexp.MustCompile(`(?m)Route::group\s*\(\s*\[\s*['"]prefix['"]\s*=>\s*['"]([^'"]+)['"]`)
)

// Parse parses PHP source code.
func (p *PHPParser) Parse(filename string, content []byte) *ParsedPHPFile {
	src := string(content)
	pf := &ParsedPHPFile{
		Path:           filename,
		Content:        src,
		Uses:           []string{},
		Classes:        []PHPClass{},
		Routes:         []PHPRoute{},
		ResourceRoutes: []PHPResourceRoute{},
		RouteGroups:    []PHPRouteGroup{},
	}

	// Extract namespace
	if match := phpNamespaceRegex.FindStringSubmatch(src); len(match) > 1 {
		pf.Namespace = match[1]
	}

	// Extract use statements
	pf.Uses = p.extractUses(src)

	// Extract classes
	pf.Classes = p.extractClasses(src)

	// Extract routes (for route files)
	pf.Routes = p.extractRoutes(src)
	pf.ResourceRoutes = p.extractResourceRoutes(src)
	pf.RouteGroups = p.extractRouteGroups(src)

	return pf
}

// extractUses extracts use statements from PHP source.
func (p *PHPParser) extractUses(src string) []string {
	var uses []string
	matches := phpUseRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			uses = append(uses, strings.TrimSpace(match[1]))
		}
	}
	return uses
}

// extractClasses extracts class definitions from PHP source.
func (p *PHPParser) extractClasses(src string) []PHPClass {
	var classes []PHPClass

	matches := phpClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		class := PHPClass{
			Line:       line,
			Implements: []string{},
			Properties: []PHPProperty{},
			Methods:    []PHPMethod{},
			Fillable:   []string{},
			Casts:      make(map[string]string),
		}

		// Extract class name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			class.Name = src[match[2]:match[3]]
		}

		// Extract parent class (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			class.Extends = src[match[4]:match[5]]
			// Check if this is an Eloquent model
			class.IsEloquentModel = isEloquentModel(class.Extends)
		}

		// Extract implemented interfaces (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			implStr := src[match[6]:match[7]]
			impls := strings.Split(implStr, ",")
			for _, impl := range impls {
				impl = strings.TrimSpace(impl)
				if impl != "" {
					class.Implements = append(class.Implements, impl)
				}
			}
		}

		// Find the class body and extract methods/properties
		classStart := match[0]
		classBody := p.findClassBody(src[classStart:])
		if classBody != "" {
			class.Methods = p.extractMethods(classBody, line)
			class.Properties = p.extractProperties(classBody, line)

			// Also extract constructor promoted properties
			for _, method := range class.Methods {
				if method.Name == "__construct" {
					promotedProps := p.extractPromotedProperties(method, line)
					class.Properties = append(class.Properties, promotedProps...)
				}
			}

			// Extract Eloquent $fillable and $casts if this is a model
			if class.IsEloquentModel {
				class.Fillable = p.extractFillable(classBody)
				class.Casts = p.extractCasts(classBody)
			}
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// findClassBody finds the body of a class (between { and }).
func (p *PHPParser) findClassBody(src string) string {
	openBrace := strings.Index(src, "{")
	if openBrace == -1 {
		return ""
	}

	depth := 0
	for i := openBrace; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[openBrace+1 : i]
			}
		}
	}
	return ""
}

// extractMethods extracts method definitions from a class body.
func (p *PHPParser) extractMethods(body string, baseLineOffset int) []PHPMethod {
	var methods []PHPMethod

	matches := phpMethodRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		method := PHPMethod{
			Line:       line,
			Parameters: []PHPParameter{},
		}

		// Extract visibility (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			method.Visibility = body[match[2]:match[3]]
		} else {
			method.Visibility = "public" // Default in PHP
		}

		// Extract method name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			method.Name = body[match[4]:match[5]]
		}

		// Extract parameters (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			paramStr := body[match[6]:match[7]]
			method.Parameters = p.extractParameters(paramStr)
		}

		// Extract return type (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			method.ReturnType = body[match[8]:match[9]]
		}

		if method.Name != "" {
			methods = append(methods, method)
		}
	}

	return methods
}

// extractParameters extracts parameters from a parameter string.
func (p *PHPParser) extractParameters(src string) []PHPParameter {
	var params []PHPParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma
	paramStrs := strings.Split(src, ",")

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := PHPParameter{}

		// Check for default value
		if idx := strings.Index(paramStr, "="); idx != -1 {
			param.Default = strings.TrimSpace(paramStr[idx+1:])
			param.IsOptional = true
			paramStr = strings.TrimSpace(paramStr[:idx])
		}

		// Split into parts
		parts := strings.Fields(paramStr)

		// Handle different formats:
		// - $name (just name)
		// - Type $name (type + name)
		// - public Type $name (visibility + type + name) - constructor promotion
		// - public readonly Type $name (visibility + readonly + type + name)

		nameIdx := -1
		for i, part := range parts {
			if strings.HasPrefix(part, "$") {
				nameIdx = i
				break
			}
		}

		if nameIdx == -1 && len(parts) > 0 {
			// Last part is probably the name
			nameIdx = len(parts) - 1
		}

		if nameIdx >= 0 {
			param.Name = strings.TrimPrefix(parts[nameIdx], "$")

			// Check for visibility modifiers (indicates constructor promotion)
			for i := 0; i < nameIdx; i++ {
				if isVisibilityModifier(parts[i]) {
					param.IsPromoted = true
					param.Visibility = parts[i]
					break
				}
			}

			// Find the type (usually the part just before the name)
			// But skip visibility and readonly modifiers
			for i := nameIdx - 1; i >= 0; i-- {
				part := parts[i]
				if isVisibilityModifier(part) || part == "readonly" {
					continue
				}
				// This should be the type
				typeStr := part
				if strings.HasPrefix(typeStr, "?") {
					param.IsOptional = true
					typeStr = strings.TrimPrefix(typeStr, "?")
				}
				param.Type = typeStr
				break
			}
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// extractProperties extracts traditional property declarations from a class body.
func (p *PHPParser) extractProperties(body string, baseLineOffset int) []PHPProperty {
	var props []PHPProperty

	matches := phpPropertyRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		prop := PHPProperty{
			Line: line,
		}

		// Extract visibility (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			prop.Visibility = body[match[2]:match[3]]
		}

		// Extract readonly modifier (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			prop.IsReadonly = true
		}

		// Extract type (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			typeStr := body[match[6]:match[7]]
			if strings.HasPrefix(typeStr, "?") {
				prop.IsNullable = true
				typeStr = strings.TrimPrefix(typeStr, "?")
			}
			prop.Type = typeStr
		}

		// Extract property name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			prop.Name = body[match[8]:match[9]]
		}

		if prop.Name != "" {
			props = append(props, prop)
		}
	}

	return props
}

// extractPromotedProperties extracts constructor promoted properties from a method.
// PHP 8+ allows: public function __construct(public string $name, private int $age)
func (p *PHPParser) extractPromotedProperties(method PHPMethod, classLine int) []PHPProperty {
	var props []PHPProperty

	for _, param := range method.Parameters {
		if !param.IsPromoted {
			continue
		}

		prop := PHPProperty{
			Name:       param.Name,
			Type:       param.Type,
			Visibility: param.Visibility,
			IsNullable: param.IsOptional,
			Line:       method.Line,
		}

		props = append(props, prop)
	}

	return props
}

// isVisibilityModifier checks if a string is a PHP visibility modifier.
func isVisibilityModifier(s string) bool {
	switch s {
	case "public", "private", "protected":
		return true
	}
	return false
}

// isEloquentModel checks if a parent class indicates an Eloquent model.
func isEloquentModel(extends string) bool {
	eloquentParents := []string{
		"Model",
		"Eloquent",
		"Authenticatable",
		"Pivot",
		"MorphPivot",
	}
	for _, parent := range eloquentParents {
		if extends == parent || strings.HasSuffix(extends, "\\"+parent) {
			return true
		}
	}
	return false
}

// extractFillable extracts field names from Eloquent $fillable array.
func (p *PHPParser) extractFillable(body string) []string {
	var fields []string

	match := phpFillableRegex.FindStringSubmatch(body)
	if len(match) < 2 {
		return fields
	}

	// Parse the array contents - match quoted strings
	contents := match[1]
	stringRegex := regexp.MustCompile(`['"]([^'"]+)['"]`)
	stringMatches := stringRegex.FindAllStringSubmatch(contents, -1)

	for _, m := range stringMatches {
		if len(m) >= 2 {
			fields = append(fields, m[1])
		}
	}

	return fields
}

// extractCasts extracts type mappings from Eloquent $casts array.
func (p *PHPParser) extractCasts(body string) map[string]string {
	casts := make(map[string]string)

	match := phpCastsRegex.FindStringSubmatch(body)
	if len(match) < 2 {
		return casts
	}

	// Parse the array contents - match 'key' => 'value' patterns
	contents := match[1]
	// Handle both 'key' => 'value' and 'key' => Type::class
	pairRegex := regexp.MustCompile(`['"]([^'"]+)['"]\s*=>\s*(?:['"]([^'"]+)['"]|(\w+)::class)`)
	pairMatches := pairRegex.FindAllStringSubmatch(contents, -1)

	for _, m := range pairMatches {
		if len(m) >= 2 {
			key := m[1]
			value := m[2]
			if value == "" && len(m) >= 4 {
				value = m[3] // Class reference like Type::class
			}
			if value != "" {
				casts[key] = value
			}
		}
	}

	return casts
}

// extractRoutes extracts Laravel route definitions.
func (p *PHPParser) extractRoutes(src string) []PHPRoute {
	var routes []PHPRoute

	matches := phpRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 12 {
			continue
		}

		line := countLines(src[:match[0]])

		route := PHPRoute{
			Line: line,
		}

		// Extract HTTP method (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.Method = strings.ToUpper(src[match[2]:match[3]])
		}

		// Extract path (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Path = src[match[4]:match[5]]
		}

		// Extract controller class (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Controller = src[match[6]:match[7]]
		}

		// Extract controller action (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			route.Action = src[match[8]:match[9]]
		}

		// Extract inline handler (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			// Inline handler like 'UserController@index'
			handler := src[match[10]:match[11]]
			if parts := strings.Split(handler, "@"); len(parts) == 2 {
				route.Controller = parts[0]
				route.Action = parts[1]
			}
		}

		if route.Method != "" && route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractResourceRoutes extracts Laravel resource route definitions.
func (p *PHPParser) extractResourceRoutes(src string) []PHPResourceRoute {
	var routes []PHPResourceRoute

	matches := phpResourceRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := PHPResourceRoute{
			Line:   line,
			Only:   []string{},
			Except: []string{},
		}

		// Check resource type (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			routeType := src[match[2]:match[3]]
			route.IsAPI = routeType == "apiResource"
		}

		// Extract path/resource name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Path = src[match[4]:match[5]]
		}

		// Extract controller class (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Controller = src[match[6]:match[7]]
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractRouteGroups extracts Laravel route group definitions.
func (p *PHPParser) extractRouteGroups(src string) []PHPRouteGroup {
	var groups []PHPRouteGroup

	// Check for prefix-style groups
	prefixMatches := phpRoutePrefixRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range prefixMatches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		group := PHPRouteGroup{
			Line:       line,
			Middleware: []string{},
		}

		if match[2] >= 0 && match[3] >= 0 {
			group.Prefix = src[match[2]:match[3]]
		}

		if group.Prefix != "" {
			groups = append(groups, group)
		}
	}

	// Check for group-style with array config
	groupMatches := phpRouteGroupRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range groupMatches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		group := PHPRouteGroup{
			Line:       line,
			Middleware: []string{},
		}

		if match[2] >= 0 && match[3] >= 0 {
			group.Prefix = src[match[2]:match[3]]
		}

		if group.Prefix != "" {
			groups = append(groups, group)
		}
	}

	return groups
}

// IsSupported returns whether PHP parsing is supported.
func (p *PHPParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *PHPParser) SupportedExtensions() []string {
	return []string{".php"}
}

// PHPTypeToOpenAPI converts a PHP type to an OpenAPI type.
func PHPTypeToOpenAPI(phpType string) (openAPIType string, format string) {
	// Trim whitespace and handle nullable types
	phpType = strings.TrimSpace(phpType)
	phpType = strings.TrimPrefix(phpType, "?")

	// Handle array types
	if strings.HasPrefix(phpType, "array") || phpType == "iterable" {
		return "array", ""
	}

	switch phpType {
	case "string":
		return "string", ""
	case "int", "integer":
		return "integer", ""
	case "float", "double":
		return "number", ""
	case "bool", "boolean":
		return "boolean", ""
	case "DateTime", "DateTimeInterface", "Carbon":
		return "string", "date-time"
	case "DateTimeImmutable":
		return "string", "date-time"
	case "mixed", "object":
		return "object", ""
	case "void", "null":
		return "", ""
	default:
		// Could be a class reference
		return "object", ""
	}
}

// ExpandResourceRoutes expands a resource route into individual routes.
func ExpandResourceRoutes(resource PHPResourceRoute) []PHPRoute {
	var routes []PHPRoute

	// Standard resource actions
	actions := []struct {
		method string
		path   string
		action string
	}{
		{"GET", "", "index"},
		{"POST", "", "store"},
		{"GET", "/{id}", "show"},
		{"PUT", "/{id}", "update"},
		{"PATCH", "/{id}", "update"},
		{"DELETE", "/{id}", "destroy"},
	}

	// API resources don't include create/edit (form views)
	if !resource.IsAPI {
		actions = append(actions,
			struct{ method, path, action string }{"GET", "/create", "create"},
			struct{ method, path, action string }{"GET", "/{id}/edit", "edit"},
		)
	}

	basePath := "/" + resource.Path

	for _, a := range actions {
		// Check only/except filters
		if len(resource.Only) > 0 {
			found := false
			for _, o := range resource.Only {
				if o == a.action {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if len(resource.Except) > 0 {
			excluded := false
			for _, e := range resource.Except {
				if e == a.action {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		routes = append(routes, PHPRoute{
			Method:     a.method,
			Path:       basePath + a.path,
			Controller: resource.Controller,
			Action:     a.action,
			Line:       resource.Line,
		})
	}

	return routes
}
