// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// CppParser provides C++ parsing capabilities using regex patterns.
type CppParser struct{}

// NewCppParser creates a new C++ parser.
func NewCppParser() *CppParser {
	return &CppParser{}
}

// CppClass represents a C++ class/struct definition.
type CppClass struct {
	// Name is the class/struct name
	Name string

	// IsStruct indicates if this is a struct (vs class)
	IsStruct bool

	// Fields are the struct/class fields
	Fields []CppField

	// Methods are the class methods
	Methods []CppMethod

	// Line is the source line number
	Line int
}

// CppField represents a struct/class field.
type CppField struct {
	// Name is the field name
	Name string

	// Type is the field type
	Type string

	// Line is the source line number
	Line int
}

// CppMethod represents a C++ method definition.
type CppMethod struct {
	// Name is the method name
	Name string

	// ReturnType is the return type
	ReturnType string

	// Parameters are the method parameters
	Parameters []CppParameter

	// Line is the source line number
	Line int
}

// CppParameter represents a method parameter.
type CppParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string
}

// CppRoute represents a route extracted from C++ web framework code.
type CppRoute struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Handler is the handler function/method name
	Handler string

	// ControllerClass is the controller class name (for frameworks like Drogon)
	ControllerClass string

	// Line is the source line number
	Line int
}

// ParsedCppFile represents a parsed C++ source file.
type ParsedCppFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Includes are the #include statements
	Includes []string

	// Classes are the extracted class/struct definitions
	Classes []CppClass

	// Routes are the extracted route definitions
	Routes []CppRoute
}

// Regex patterns for C++ parsing
var (
	// Matches #include statements
	cppIncludeRegex = regexp.MustCompile(`(?m)^#include\s*[<"]([^>"]+)[>"]`)

	// Matches class/struct definitions
	cppClassRegex = regexp.MustCompile(`(?m)(class|struct)\s+(\w+)(?:\s*:\s*(?:public|private|protected)\s+(\w+))?`)

	// Matches Drogon ADD_METHOD_TO macro
	drogonAddMethodRegex = regexp.MustCompile(`(?m)ADD_METHOD_TO\s*\(\s*(\w+)\s*,\s*(\w+)\s*,\s*"([^"]+)"\s*(?:,\s*(Get|Post|Put|Delete|Patch|Head|Options))*`)

	// Matches Drogon METHOD_ADD macro
	drogonMethodAddRegex = regexp.MustCompile(`(?m)METHOD_ADD\s*\(\s*(\w+)\s*::\s*(\w+)\s*,\s*"([^"]+)"\s*(?:,\s*(Get|Post|Put|Delete|Patch|Head|Options))*`)

	// Matches Drogon app().registerHandler lambda style (path only, no explicit method)
	// Used as fallback when no explicit method is specified
	drogonRegisterHandlerRegex = regexp.MustCompile(`(?m)(?:app\(\)|drogon::app\(\))\s*\.\s*registerHandler\s*\(\s*"([^"]+)"`)

	// Matches Drogon app().registerHandler with explicit HTTP method like {Get}, {Post}, etc.
	// Captures: (1) path, (2) HTTP method
	// Uses (?s) for dotall mode and .*? for non-greedy matching across the lambda body
	drogonRegisterHandlerMethodRegex = regexp.MustCompile(`(?s)(?:app\(\)|drogon::app\(\))\s*\.\s*registerHandler\s*\(\s*"([^"]+)".*?,\s*\{(Get|Post|Put|Delete|Patch|Head|Options)\}`)

	// Matches Oat++ ENDPOINT macro
	oatppEndpointRegex = regexp.MustCompile(`(?m)ENDPOINT\s*\(\s*"(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)"\s*,\s*"([^"]+)"\s*,\s*(\w+)`)

	// Matches Oat++ ENDPOINT with PATH/BODY parameters
	oatppEndpointParamsRegex = regexp.MustCompile(`(?m)ENDPOINT\s*\(\s*"(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)"\s*,\s*"([^"]+)"\s*,\s*(\w+)\s*(?:,\s*(?:PATH|BODY_DTO|QUERY)\s*\([^)]+\))*\s*\)`)

	// Matches Crow CROW_ROUTE macro
	crowRouteRegex = regexp.MustCompile(`(?m)CROW_ROUTE\s*\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)(?:\s*\.methods\s*\(([^)]+)\))?`)

	// Matches Crow CROW_BP_ROUTE macro (blueprint routes)
	crowBPRouteRegex = regexp.MustCompile(`(?m)CROW_BP_ROUTE\s*\(\s*(\w+)\s*,\s*"([^"]+)"\s*\)(?:\s*\.methods\s*\(([^)]+)\))?`)

	// Matches C++ struct field
	cppFieldRegex = regexp.MustCompile(`(?m)^\s*([\w:<>,\s]+)\s+(\w+)\s*;`)

	// Matches Crow method like "GET"_method or crow::HTTPMethod::GET
	crowMethodRegex = regexp.MustCompile(`"(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)"_method|crow::HTTPMethod::(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)`)
)

// Parse parses C++ source code.
func (p *CppParser) Parse(filename string, content []byte) *ParsedCppFile {
	src := string(content)
	pf := &ParsedCppFile{
		Path:     filename,
		Content:  src,
		Includes: []string{},
		Classes:  []CppClass{},
		Routes:   []CppRoute{},
	}

	// Extract includes
	pf.Includes = p.extractIncludes(src)

	// Extract classes/structs
	pf.Classes = p.extractClasses(src)

	// Extract routes from various C++ web frameworks
	pf.Routes = p.extractRoutes(src)

	return pf
}

// extractIncludes extracts #include statements from C++ source.
func (p *CppParser) extractIncludes(src string) []string {
	var includes []string
	matches := cppIncludeRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			includes = append(includes, strings.TrimSpace(match[1]))
		}
	}
	return includes
}

// extractClasses extracts class/struct definitions from C++ source.
func (p *CppParser) extractClasses(src string) []CppClass {
	var classes []CppClass

	matches := cppClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		class := CppClass{
			Line:     line,
			Fields:   []CppField{},
			Methods:  []CppMethod{},
			IsStruct: src[match[2]:match[3]] == "struct",
		}

		// Extract class name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			class.Name = src[match[4]:match[5]]
		}

		// Find the class body and extract fields
		classStart := match[0]
		classBody := p.findClassBody(src[classStart:])
		if classBody != "" {
			class.Fields = p.extractFields(classBody, line)
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// findClassBody finds the body of a class (between { and }).
func (p *CppParser) findClassBody(src string) string {
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

// extractFields extracts fields from a class body.
func (p *CppParser) extractFields(body string, baseLineOffset int) []CppField {
	var fields []CppField

	matches := cppFieldRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		field := CppField{
			Line: line,
		}

		// Extract type (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			field.Type = strings.TrimSpace(body[match[2]:match[3]])
		}

		// Extract name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			field.Name = body[match[4]:match[5]]
		}

		// Skip common non-field patterns
		if field.Type == "public" || field.Type == "private" || field.Type == "protected" {
			continue
		}

		if field.Name != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// extractRoutes extracts routes from C++ web framework code.
func (p *CppParser) extractRoutes(src string) []CppRoute {
	var routes []CppRoute

	// Extract Drogon ADD_METHOD_TO routes
	routes = append(routes, p.extractDrogonAddMethodRoutes(src)...)

	// Extract Drogon METHOD_ADD routes
	routes = append(routes, p.extractDrogonMethodAddRoutes(src)...)

	// Extract Drogon registerHandler routes
	routes = append(routes, p.extractDrogonRegisterHandlerRoutes(src)...)

	// Extract Oat++ ENDPOINT routes
	routes = append(routes, p.extractOatppEndpointRoutes(src)...)

	// Extract Crow CROW_ROUTE routes
	routes = append(routes, p.extractCrowRoutes(src)...)

	// Extract Crow CROW_BP_ROUTE routes
	routes = append(routes, p.extractCrowBPRoutes(src)...)

	return routes
}

// extractDrogonAddMethodRoutes extracts routes from Drogon ADD_METHOD_TO macros.
func (p *CppParser) extractDrogonAddMethodRoutes(src string) []CppRoute {
	var routes []CppRoute

	matches := drogonAddMethodRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:   line,
			Method: "GET", // Default
		}

		// Extract controller class (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.ControllerClass = src[match[2]:match[3]]
		}

		// Extract method name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Handler = src[match[4]:match[5]]
		}

		// Extract path (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Path = src[match[6]:match[7]]
		}

		// Extract HTTP method if present (group 4)
		if len(match) >= 10 && match[8] >= 0 && match[9] >= 0 {
			route.Method = strings.ToUpper(src[match[8]:match[9]])
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractDrogonMethodAddRoutes extracts routes from Drogon METHOD_ADD macros.
func (p *CppParser) extractDrogonMethodAddRoutes(src string) []CppRoute {
	var routes []CppRoute

	matches := drogonMethodAddRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:   line,
			Method: "GET", // Default
		}

		// Extract controller class (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.ControllerClass = src[match[2]:match[3]]
		}

		// Extract method name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Handler = src[match[4]:match[5]]
		}

		// Extract path (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Path = src[match[6]:match[7]]
		}

		// Extract HTTP method if present (group 4)
		if len(match) >= 10 && match[8] >= 0 && match[9] >= 0 {
			route.Method = strings.ToUpper(src[match[8]:match[9]])
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractDrogonRegisterHandlerRoutes extracts routes from Drogon app().registerHandler calls.
func (p *CppParser) extractDrogonRegisterHandlerRoutes(src string) []CppRoute {
	var routes []CppRoute
	pathsWithMethod := make(map[string]bool)

	// First, extract routes with explicit HTTP methods using the more specific regex
	// This captures patterns like: app().registerHandler("/path", [...], {Get})
	methodMatches := drogonRegisterHandlerMethodRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range methodMatches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:    line,
			Method:  "GET", // Default
			Handler: "lambda",
		}

		// Extract path (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.Path = src[match[2]:match[3]]
		}

		// Extract HTTP method (group 2) - e.g., Get, Post, Put, Delete
		if match[4] >= 0 && match[5] >= 0 {
			route.Method = strings.ToUpper(src[match[4]:match[5]])
		}

		if route.Path != "" {
			routes = append(routes, route)
			// Track this path so we don't double-count with the fallback regex
			pathsWithMethod[route.Path] = true
		}
	}

	// Then, extract routes without explicit methods (fallback to GET)
	// This captures patterns like: app().registerHandler("/path", [...])
	matches := drogonRegisterHandlerRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:    line,
			Method:  "GET", // Default when no explicit method
			Handler: "lambda",
		}

		// Extract path (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.Path = src[match[2]:match[3]]
		}

		// Skip if we already found this path with an explicit method
		if route.Path != "" && !pathsWithMethod[route.Path] {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractOatppEndpointRoutes extracts routes from Oat++ ENDPOINT macros.
func (p *CppParser) extractOatppEndpointRoutes(src string) []CppRoute {
	var routes []CppRoute

	matches := oatppEndpointRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
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

		// Extract handler name (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Handler = src[match[6]:match[7]]
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractCrowRoutes extracts routes from Crow CROW_ROUTE macros.
func (p *CppParser) extractCrowRoutes(src string) []CppRoute {
	var routes []CppRoute

	matches := crowRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:    line,
			Method:  "GET", // Default
			Handler: "lambda",
		}

		// Extract path (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Path = src[match[4]:match[5]]
		}

		// Extract methods if present (group 3)
		if len(match) >= 8 && match[6] >= 0 && match[7] >= 0 {
			methodsStr := src[match[6]:match[7]]
			methods := p.parseCrowMethods(methodsStr)
			if len(methods) > 0 {
				// For multiple methods, create separate routes
				for _, method := range methods {
					r := route
					r.Method = method
					routes = append(routes, r)
				}
				continue
			}
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractCrowBPRoutes extracts routes from Crow CROW_BP_ROUTE macros.
func (p *CppParser) extractCrowBPRoutes(src string) []CppRoute {
	var routes []CppRoute

	matches := crowBPRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CppRoute{
			Line:    line,
			Method:  "GET", // Default
			Handler: "lambda",
		}

		// Extract path (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Path = src[match[4]:match[5]]
		}

		// Extract methods if present (group 3)
		if len(match) >= 8 && match[6] >= 0 && match[7] >= 0 {
			methodsStr := src[match[6]:match[7]]
			methods := p.parseCrowMethods(methodsStr)
			if len(methods) > 0 {
				for _, method := range methods {
					r := route
					r.Method = method
					routes = append(routes, r)
				}
				continue
			}
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// parseCrowMethods parses Crow method specifications.
func (p *CppParser) parseCrowMethods(methodsStr string) []string {
	var methods []string

	matches := crowMethodRegex.FindAllStringSubmatch(methodsStr, -1)
	for _, match := range matches {
		if len(match) >= 2 && match[1] != "" {
			methods = append(methods, strings.ToUpper(match[1]))
		} else if len(match) >= 3 && match[2] != "" {
			methods = append(methods, strings.ToUpper(match[2]))
		}
	}

	return methods
}

// IsSupported returns whether C++ parsing is supported.
func (p *CppParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *CppParser) SupportedExtensions() []string {
	return []string{".cpp", ".hpp", ".h", ".cc", ".cxx"}
}

// CppTypeToOpenAPI converts a C++ type to an OpenAPI type.
func CppTypeToOpenAPI(cppType string) (openAPIType string, format string) {
	// Trim whitespace and handle common modifiers
	cppType = strings.TrimSpace(cppType)
	cppType = strings.TrimPrefix(cppType, "const ")
	cppType = strings.TrimSuffix(cppType, "&")
	cppType = strings.TrimSuffix(cppType, "*")
	cppType = strings.TrimSpace(cppType)

	// Handle common wrapper types
	if strings.HasPrefix(cppType, "std::optional<") {
		cppType = extractCppGenericType(cppType)
	}
	if strings.HasPrefix(cppType, "std::shared_ptr<") || strings.HasPrefix(cppType, "std::unique_ptr<") {
		cppType = extractCppGenericType(cppType)
	}

	// Handle vector/array types
	if strings.HasPrefix(cppType, "std::vector<") || strings.HasPrefix(cppType, "std::list<") ||
		strings.HasPrefix(cppType, "std::array<") || strings.HasPrefix(cppType, "std::deque<") {
		return "array", ""
	}

	// Handle map types
	if strings.HasPrefix(cppType, "std::map<") || strings.HasPrefix(cppType, "std::unordered_map<") {
		return "object", ""
	}

	switch cppType {
	case "std::string", "string", "char*", "const char*":
		return "string", ""
	case "int", "int32_t", "int32", "long", "long long", "int64_t", "int64", "short", "int16_t":
		return "integer", ""
	case "unsigned int", "uint32_t", "uint32", "unsigned long", "uint64_t", "uint64", "size_t":
		return "integer", ""
	case "float", "double", "long double":
		return "number", ""
	case "bool":
		return "boolean", ""
	case "void":
		return "", ""
	default:
		return "object", ""
	}
}

// extractCppGenericType extracts the inner type from a template like std::vector<string>.
func extractCppGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// ConvertCppPathParams converts C++ path parameters to OpenAPI format.
// Handles both {param} and :param styles.
func ConvertCppPathParams(path string) string {
	// Convert :param to {param}
	colonParamRegex := regexp.MustCompile(`:(\w+)`)
	return colonParamRegex.ReplaceAllString(path, "{$1}")
}

// ExtractCppPathParams extracts parameter names from a path.
func ExtractCppPathParams(path string) []string {
	var params []string

	// Match both {param} and :param styles
	braceRegex := regexp.MustCompile(`\{(\w+)\}`)
	colonRegex := regexp.MustCompile(`:(\w+)`)

	for _, match := range braceRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	for _, match := range colonRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// ConvertCrowPathParams converts Crow path parameters to OpenAPI format.
// Crow uses <type> format, e.g., /users/<int> or /users/<string>
func ConvertCrowPathParams(path string) string {
	// Convert <int>, <uint>, <double>, <string> to {param}
	crowParamRegex := regexp.MustCompile(`<(int|uint|double|string)>`)

	paramCount := 0
	result := crowParamRegex.ReplaceAllStringFunc(path, func(match string) string {
		paramCount++
		return "{param" + string(rune('0'+paramCount)) + "}"
	})

	return result
}
