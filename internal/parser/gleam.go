// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// GleamParser provides Gleam parsing capabilities using regex patterns.
type GleamParser struct{}

// NewGleamParser creates a new Gleam parser.
func NewGleamParser() *GleamParser {
	return &GleamParser{}
}

// GleamModule represents a Gleam module.
type GleamModule struct {
	// Name is the module name (derived from filename)
	Name string

	// Imports are the imported modules
	Imports []GleamImport

	// Functions are the module functions
	Functions []GleamFunction

	// Types are the type definitions
	Types []GleamType
}

// GleamImport represents an import statement.
type GleamImport struct {
	// Module is the imported module name
	Module string

	// Alias is the import alias
	Alias string

	// Exposed are the exposed items
	Exposed []string

	// Line is the source line number
	Line int
}

// GleamFunction represents a Gleam function definition.
type GleamFunction struct {
	// Name is the function name
	Name string

	// Parameters are the function parameters
	Parameters []GleamParameter

	// ReturnType is the return type
	ReturnType string

	// IsPublic indicates if the function is public
	IsPublic bool

	// Line is the source line number
	Line int
}

// GleamParameter represents a function parameter.
type GleamParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Label is the external parameter label
	Label string
}

// GleamType represents a Gleam type definition.
type GleamType struct {
	// Name is the type name
	Name string

	// IsPublic indicates if the type is public
	IsPublic bool

	// Variants are the type variants (for custom types)
	Variants []GleamVariant

	// Fields are the fields (for record types)
	Fields []GleamField

	// Line is the source line number
	Line int
}

// GleamVariant represents a variant of a custom type.
type GleamVariant struct {
	// Name is the variant name
	Name string

	// Fields are the variant fields
	Fields []GleamField
}

// GleamField represents a field in a type or variant.
type GleamField struct {
	// Name is the field name
	Name string

	// Type is the field type
	Type string
}

// GleamRoute represents a Wisp/Gleam HTTP route.
type GleamRoute struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Handler is the handler function
	Handler string

	// Line is the source line number
	Line int
}

// ParsedGleamFile represents a parsed Gleam source file.
type ParsedGleamFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Module is the module definition
	Module GleamModule

	// Routes are the extracted route definitions
	Routes []GleamRoute
}

// Regex patterns for Gleam parsing
var (
	// Matches import statements
	gleamImportRegex = regexp.MustCompile(`(?m)^import\s+([\w/]+)(?:\s+as\s+(\w+))?(?:\s*\.\{([^}]+)\})?`)

	// Matches public function definitions
	gleamPubFnRegex = regexp.MustCompile(`(?m)^pub\s+fn\s+(\w+)\s*\(([^)]*)\)(?:\s*->\s*([\w\(\),\s<>]+))?`)

	// Matches private function definitions
	gleamFnRegex = regexp.MustCompile(`(?m)^fn\s+(\w+)\s*\(([^)]*)\)(?:\s*->\s*([\w\(\),\s<>]+))?`)

	// Matches type definitions
	gleamTypeRegex = regexp.MustCompile(`(?m)^(pub\s+)?type\s+(\w+)(?:\s*\{([^}]+)\})?`)

	// Matches router.get/post/etc style
	gleamRouterRegex = regexp.MustCompile(`(?m)router\.(get|post|put|delete|patch)\s*\(\s*"([^"]+)"\s*,\s*(\w+)`)
)

// Parse parses Gleam source code.
func (p *GleamParser) Parse(filename string, content []byte) *ParsedGleamFile {
	src := string(content)
	pf := &ParsedGleamFile{
		Path:    filename,
		Content: src,
		Module: GleamModule{
			Imports:   []GleamImport{},
			Functions: []GleamFunction{},
			Types:     []GleamType{},
		},
		Routes: []GleamRoute{},
	}

	// Derive module name from filename
	pf.Module.Name = deriveGleamModuleName(filename)

	// Extract imports
	pf.Module.Imports = p.extractImports(src)

	// Extract functions
	pf.Module.Functions = p.extractFunctions(src)

	// Extract types
	pf.Module.Types = p.extractTypes(src)

	// Extract routes
	pf.Routes = p.extractRoutes(src)

	return pf
}

// deriveGleamModuleName derives the module name from the filename.
func deriveGleamModuleName(filename string) string {
	// Remove path and extension
	name := filename
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[:idx]
	}
	return name
}

// extractImports extracts import statements from Gleam source.
func (p *GleamParser) extractImports(src string) []GleamImport {
	var imports []GleamImport

	matches := gleamImportRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		imp := GleamImport{
			Line:    line,
			Exposed: []string{},
		}

		// Extract module name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			imp.Module = src[match[2]:match[3]]
		}

		// Extract alias (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			imp.Alias = src[match[4]:match[5]]
		}

		// Extract exposed items (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			exposedStr := src[match[6]:match[7]]
			items := strings.Split(exposedStr, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item != "" {
					imp.Exposed = append(imp.Exposed, item)
				}
			}
		}

		if imp.Module != "" {
			imports = append(imports, imp)
		}
	}

	return imports
}

// extractFunctions extracts function definitions from Gleam source.
func (p *GleamParser) extractFunctions(src string) []GleamFunction {
	var functions []GleamFunction

	// Extract public functions
	pubMatches := gleamPubFnRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range pubMatches {
		fn := p.parseFunctionMatch(src, match, true)
		if fn != nil {
			functions = append(functions, *fn)
		}
	}

	// Extract private functions
	fnMatches := gleamFnRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range fnMatches {
		fn := p.parseFunctionMatch(src, match, false)
		if fn != nil {
			functions = append(functions, *fn)
		}
	}

	return functions
}

// parseFunctionMatch parses a function from a regex match.
func (p *GleamParser) parseFunctionMatch(src string, match []int, isPublic bool) *GleamFunction {
	if len(match) < 8 {
		return nil
	}

	line := countLines(src[:match[0]])

	fn := GleamFunction{
		Line:       line,
		IsPublic:   isPublic,
		Parameters: []GleamParameter{},
	}

	// Extract function name (group 1)
	if match[2] >= 0 && match[3] >= 0 {
		fn.Name = src[match[2]:match[3]]
	}

	// Extract parameters (group 2)
	if match[4] >= 0 && match[5] >= 0 {
		paramStr := src[match[4]:match[5]]
		fn.Parameters = p.extractParameters(paramStr)
	}

	// Extract return type (group 3)
	if match[6] >= 0 && match[7] >= 0 {
		fn.ReturnType = strings.TrimSpace(src[match[6]:match[7]])
	}

	if fn.Name == "" {
		return nil
	}

	return &fn
}

// extractParameters extracts parameters from a parameter string.
func (p *GleamParser) extractParameters(src string) []GleamParameter {
	var params []GleamParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma, handling nested types
	paramStrs := splitGleamParameters(src)

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := GleamParameter{}

		// Check for label: name: Type syntax
		if idx := strings.Index(paramStr, ":"); idx > 0 {
			beforeColon := strings.TrimSpace(paramStr[:idx])
			afterColon := strings.TrimSpace(paramStr[idx+1:])

			// Check if there's a second colon for label: name: Type
			if idx2 := strings.Index(afterColon, ":"); idx2 > 0 {
				param.Label = beforeColon
				param.Name = strings.TrimSpace(afterColon[:idx2])
				param.Type = strings.TrimSpace(afterColon[idx2+1:])
			} else {
				param.Name = beforeColon
				param.Type = afterColon
			}
		} else {
			param.Name = paramStr
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// splitGleamParameters splits a parameter string by comma, handling nested types.
func splitGleamParameters(src string) []string {
	var params []string
	var current strings.Builder
	depth := 0

	for _, ch := range src {
		switch ch {
		case '(', '<', '{':
			depth++
			current.WriteRune(ch)
		case ')', '>', '}':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				params = append(params, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		params = append(params, current.String())
	}

	return params
}

// extractTypes extracts type definitions from Gleam source.
func (p *GleamParser) extractTypes(src string) []GleamType {
	var types []GleamType

	matches := gleamTypeRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		t := GleamType{
			Line:     line,
			Variants: []GleamVariant{},
			Fields:   []GleamField{},
		}

		// Check if public (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			t.IsPublic = true
		}

		// Extract type name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			t.Name = src[match[4]:match[5]]
		}

		// Extract type body (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			bodyStr := src[match[6]:match[7]]
			t.Fields = p.extractTypeFields(bodyStr)
		}

		if t.Name != "" {
			types = append(types, t)
		}
	}

	return types
}

// extractTypeFields extracts fields from a type body.
func (p *GleamParser) extractTypeFields(src string) []GleamField {
	var fields []GleamField

	// Simple field extraction: name: Type
	fieldRegex := regexp.MustCompile(`(\w+)\s*:\s*([\w<>(),\s]+)`)
	matches := fieldRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			fields = append(fields, GleamField{
				Name: match[1],
				Type: strings.TrimSpace(match[2]),
			})
		}
	}

	return fields
}

// extractRoutes extracts Wisp/Gleam HTTP route definitions.
func (p *GleamParser) extractRoutes(src string) []GleamRoute {
	var routes []GleamRoute

	// Extract router.get/post/etc style routes
	routerMatches := gleamRouterRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range routerMatches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := GleamRoute{
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

		// Extract handler (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Handler = src[match[6]:match[7]]
		}

		if route.Method != "" && route.Path != "" {
			routes = append(routes, route)
		}
	}

	// Extract path_segments style routes
	pathRoutes := p.extractPathSegmentsRoutes(src)
	routes = append(routes, pathRoutes...)

	return routes
}

// extractPathSegmentsRoutes extracts routes from wisp.path_segments pattern matching.
func (p *GleamParser) extractPathSegmentsRoutes(src string) []GleamRoute {
	var routes []GleamRoute

	// Look for case expressions on path_segments
	caseRegex := regexp.MustCompile(`(?ms)case\s+wisp\.path_segments\(request\)\s*\{([^}]+)\}`)
	caseMatches := caseRegex.FindAllStringSubmatch(src, -1)

	for _, caseMatch := range caseMatches {
		if len(caseMatch) < 2 {
			continue
		}

		caseBody := caseMatch[1]
		line := countLines(src[:strings.Index(src, caseBody)])

		// Extract individual patterns
		patternRegex := regexp.MustCompile(`\[([^\]]+)\]\s*->\s*(\w+)`)
		patternMatches := patternRegex.FindAllStringSubmatch(caseBody, -1)

		for _, patternMatch := range patternMatches {
			if len(patternMatch) < 3 {
				continue
			}

			path := p.pathSegmentsToPath(patternMatch[1])
			handler := patternMatch[2]

			// Default to GET for now
			routes = append(routes, GleamRoute{
				Method:  "GET",
				Path:    path,
				Handler: handler,
				Line:    line,
			})
		}
	}

	return routes
}

// pathSegmentsToPath converts a path segments list to a path string.
func (p *GleamParser) pathSegmentsToPath(segments string) string {
	var parts []string

	// Parse the segments list
	segmentRegex := regexp.MustCompile(`"([^"]+)"`)
	matches := segmentRegex.FindAllStringSubmatch(segments, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			parts = append(parts, match[1])
		}
	}

	// Check for variable segments
	varRegex := regexp.MustCompile(`(\w+)`)
	varMatches := varRegex.FindAllStringSubmatch(segments, -1)
	for _, match := range varMatches {
		if len(match) >= 2 {
			name := match[1]
			// Skip string literals already captured
			if !strings.Contains(segments, `"`+name+`"`) && name != "" {
				parts = append(parts, "{"+name+"}")
			}
		}
	}

	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}

// IsSupported returns whether Gleam parsing is supported.
func (p *GleamParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *GleamParser) SupportedExtensions() []string {
	return []string{".gleam"}
}

// GleamTypeToOpenAPI converts a Gleam type to an OpenAPI type.
func GleamTypeToOpenAPI(gleamType string) (openAPIType string, format string) {
	// Trim whitespace
	gleamType = strings.TrimSpace(gleamType)

	// Handle Option(T)
	if strings.HasPrefix(gleamType, "Option(") {
		innerType := extractGleamGenericType(gleamType)
		return GleamTypeToOpenAPI(innerType)
	}

	// Handle List(T)
	if strings.HasPrefix(gleamType, "List(") {
		return "array", ""
	}

	// Handle Dict(K, V)
	if strings.HasPrefix(gleamType, "Dict(") {
		return "object", ""
	}

	switch gleamType {
	case "String":
		return "string", ""
	case "Int":
		return "integer", ""
	case "Float":
		return "number", ""
	case "Bool":
		return "boolean", ""
	case "Nil":
		return "", ""
	case "BitArray":
		return "string", "binary"
	default:
		return "object", ""
	}
}

// extractGleamGenericType extracts the inner type from a generic like Option(String).
func extractGleamGenericType(s string) string {
	start := strings.Index(s, "(")
	end := strings.LastIndex(s, ")")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}
