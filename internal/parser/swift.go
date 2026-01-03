// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// SwiftParser provides Swift parsing capabilities using regex patterns.
type SwiftParser struct{}

// NewSwiftParser creates a new Swift parser.
func NewSwiftParser() *SwiftParser {
	return &SwiftParser{}
}

// SwiftStruct represents a Swift struct definition.
type SwiftStruct struct {
	// Name is the struct name
	Name string

	// Fields are the struct fields
	Fields []SwiftField

	// ConformsToContent indicates if the struct conforms to Content protocol
	ConformsToContent bool

	// Line is the source line number
	Line int
}

// SwiftField represents a struct field.
type SwiftField struct {
	// Name is the field name
	Name string

	// Type is the field type
	Type string

	// IsOptional indicates if the field is optional (T?)
	IsOptional bool

	// Line is the source line number
	Line int
}

// SwiftRoute represents a route extracted from Swift web framework code.
type SwiftRoute struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Handler is the handler function/method name
	Handler string

	// GroupPrefix is the route group prefix if any
	GroupPrefix string

	// Line is the source line number
	Line int
}

// ParsedSwiftFile represents a parsed Swift source file.
type ParsedSwiftFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Imports are the import statements
	Imports []string

	// Structs are the extracted struct definitions
	Structs []SwiftStruct

	// Routes are the extracted route definitions
	Routes []SwiftRoute

	// RouteGroups are the extracted route group definitions
	RouteGroups []SwiftRouteGroup
}

// SwiftRouteGroup represents a Vapor route group.
type SwiftRouteGroup struct {
	// Name is the variable name for the group
	Name string

	// Prefix is the group prefix path
	Prefix string

	// Line is the source line number
	Line int
}

// Regex patterns for Swift parsing
var (
	// Matches import statements
	swiftImportRegex = regexp.MustCompile(`(?m)^import\s+(\w+)`)

	// Matches struct definitions with optional protocol conformance
	swiftStructRegex = regexp.MustCompile(`(?m)struct\s+(\w+)\s*(?::\s*([^{]+))?\s*\{`)

	// Matches Vapor app.get/post/put/delete/patch routes
	// app.get("users") { req in ... }
	// app.post("users", ":id") { req -> User in ... }
	vaporRouteRegex = regexp.MustCompile(`(?m)(\w+)\.(get|post|put|delete|patch|head|options)\s*\(\s*([^)]*)\s*\)\s*\{`)

	// Matches Vapor grouped routes: app.grouped("api").get("users") { ... }
	vaporGroupedRouteRegex = regexp.MustCompile(`(?m)(\w+)\.grouped\s*\(\s*"([^"]+)"\s*\)\s*\.\s*(get|post|put|delete|patch|head|options)\s*\(\s*([^)]*)\s*\)\s*\{`)

	// Matches route group assignment: let users = app.grouped("users")
	vaporRouteGroupRegex = regexp.MustCompile(`(?m)let\s+(\w+)\s*=\s*(\w+)\.grouped\s*\(\s*"([^"]+)"\s*\)`)

	// Matches controller routes: routes.get("path", use: handler)
	vaporControllerRouteRegex = regexp.MustCompile(`(?m)(\w+)\.(get|post|put|delete|patch|head|options)\s*\(\s*([^,)]*)\s*,\s*use\s*:\s*(\w+)\s*\)`)

	// Matches controller boot function
	vaporBootFunctionRegex = regexp.MustCompile(`(?m)func\s+boot\s*\(\s*routes\s*:\s*RoutesBuilder\s*\)`)

	// Matches Swift property: let/var name: Type
	swiftPropertyRegex = regexp.MustCompile(`(?m)(?:let|var)\s+(\w+)\s*:\s*([^=\n]+)`)

	// Matches path segments in route definitions
	swiftPathSegmentRegex = regexp.MustCompile(`"([^"]+)"`)

	// Matches path parameters like ":id", ":userId"
	swiftPathParamRegex = regexp.MustCompile(`:(\w+)`)
)

// Parse parses Swift source code.
func (p *SwiftParser) Parse(filename string, content []byte) *ParsedSwiftFile {
	src := string(content)
	pf := &ParsedSwiftFile{
		Path:        filename,
		Content:     src,
		Imports:     []string{},
		Structs:     []SwiftStruct{},
		Routes:      []SwiftRoute{},
		RouteGroups: []SwiftRouteGroup{},
	}

	// Extract imports
	pf.Imports = p.extractImports(src)

	// Extract structs
	pf.Structs = p.extractStructs(src)

	// Extract route groups first
	pf.RouteGroups = p.extractRouteGroups(src)

	// Extract routes
	pf.Routes = p.extractRoutes(src, pf.RouteGroups)

	return pf
}

// extractImports extracts import statements from Swift source.
func (p *SwiftParser) extractImports(src string) []string {
	var imports []string
	matches := swiftImportRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, strings.TrimSpace(match[1]))
		}
	}
	return imports
}

// extractStructs extracts struct definitions from Swift source.
func (p *SwiftParser) extractStructs(src string) []SwiftStruct {
	var structs []SwiftStruct

	matches := swiftStructRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		s := SwiftStruct{
			Line:   line,
			Fields: []SwiftField{},
		}

		// Extract struct name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			s.Name = src[match[2]:match[3]]
		}

		// Check protocol conformance (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			protocols := src[match[4]:match[5]]
			s.ConformsToContent = strings.Contains(protocols, "Content")
		}

		// Find struct body and extract fields
		structStart := match[0]
		structBody := p.findStructBody(src[structStart:])
		if structBody != "" {
			s.Fields = p.extractStructFields(structBody, line)
		}

		if s.Name != "" {
			structs = append(structs, s)
		}
	}

	return structs
}

// findStructBody finds the body of a struct (between { and }).
func (p *SwiftParser) findStructBody(src string) string {
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

// extractStructFields extracts fields from a struct body.
func (p *SwiftParser) extractStructFields(body string, baseLineOffset int) []SwiftField {
	var fields []SwiftField

	matches := swiftPropertyRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		field := SwiftField{
			Line: line,
		}

		// Extract name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			field.Name = body[match[2]:match[3]]
		}

		// Extract type (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			typeStr := strings.TrimSpace(body[match[4]:match[5]])
			field.Type = typeStr

			// Check if optional
			if strings.HasSuffix(typeStr, "?") {
				field.IsOptional = true
				field.Type = strings.TrimSuffix(typeStr, "?")
			}
		}

		if field.Name != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// extractRouteGroups extracts route group definitions.
func (p *SwiftParser) extractRouteGroups(src string) []SwiftRouteGroup {
	var groups []SwiftRouteGroup

	matches := vaporRouteGroupRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		group := SwiftRouteGroup{
			Line: line,
		}

		// Extract variable name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			group.Name = src[match[2]:match[3]]
		}

		// Extract prefix (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			group.Prefix = src[match[6]:match[7]]
		}

		if group.Name != "" {
			groups = append(groups, group)
		}
	}

	return groups
}

// extractRoutes extracts route definitions from Swift source.
func (p *SwiftParser) extractRoutes(src string, groups []SwiftRouteGroup) []SwiftRoute {
	var routes []SwiftRoute

	// Create a map for quick group lookup
	groupMap := make(map[string]string)
	for _, g := range groups {
		groupMap[g.Name] = g.Prefix
	}

	// Extract grouped routes (app.grouped("api").get("users"))
	routes = append(routes, p.extractGroupedRoutes(src)...)

	// Extract controller routes (routes.get("path", use: handler))
	routes = append(routes, p.extractControllerRoutes(src, groupMap)...)

	// Extract simple routes (app.get("users"))
	routes = append(routes, p.extractSimpleRoutes(src, groupMap)...)

	return routes
}

// extractGroupedRoutes extracts routes with inline grouping.
func (p *SwiftParser) extractGroupedRoutes(src string) []SwiftRoute {
	var routes []SwiftRoute

	matches := vaporGroupedRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := SwiftRoute{
			Line: line,
		}

		// Extract group prefix (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.GroupPrefix = src[match[4]:match[5]]
		}

		// Extract HTTP method (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Method = strings.ToUpper(src[match[6]:match[7]])
		}

		// Extract path segments (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			pathStr := src[match[8]:match[9]]
			route.Path = p.buildPath(route.GroupPrefix, pathStr)
		} else {
			route.Path = "/" + route.GroupPrefix
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractControllerRoutes extracts routes from controller boot functions.
func (p *SwiftParser) extractControllerRoutes(src string, groupMap map[string]string) []SwiftRoute {
	var routes []SwiftRoute

	matches := vaporControllerRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := SwiftRoute{
			Line: line,
		}

		// Extract router variable name (group 1)
		var routerName string
		if match[2] >= 0 && match[3] >= 0 {
			routerName = src[match[2]:match[3]]
		}

		// Extract HTTP method (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Method = strings.ToUpper(src[match[4]:match[5]])
		}

		// Extract path segments (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			pathStr := src[match[6]:match[7]]
			prefix := groupMap[routerName]
			route.Path = p.buildPath(prefix, pathStr)
		}

		// Extract handler name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			route.Handler = src[match[8]:match[9]]
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractSimpleRoutes extracts simple route definitions.
func (p *SwiftParser) extractSimpleRoutes(src string, groupMap map[string]string) []SwiftRoute {
	var routes []SwiftRoute

	matches := vaporRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		route := SwiftRoute{
			Line: line,
		}

		// Extract router variable name (group 1)
		var routerName string
		if match[2] >= 0 && match[3] >= 0 {
			routerName = src[match[2]:match[3]]
		}

		// Skip if this looks like a grouped route (already handled)
		if strings.Contains(src[max(0, match[0]-50):match[0]], ".grouped") {
			continue
		}

		// Extract HTTP method (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Method = strings.ToUpper(src[match[4]:match[5]])
		}

		// Extract path segments (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			pathStr := src[match[6]:match[7]]
			prefix := groupMap[routerName]
			route.Path = p.buildPath(prefix, pathStr)
		}

		if route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// buildPath builds a complete path from a prefix and path segments.
func (p *SwiftParser) buildPath(prefix, pathStr string) string {
	var segments []string

	// Add prefix if present
	if prefix != "" {
		segments = append(segments, prefix)
	}

	// Extract path segments from the string
	segmentMatches := swiftPathSegmentRegex.FindAllStringSubmatch(pathStr, -1)
	for _, segMatch := range segmentMatches {
		if len(segMatch) > 1 {
			segment := segMatch[1]
			// Convert :param to {param}
			segment = swiftPathParamRegex.ReplaceAllString(segment, "{$1}")
			if segment != "" {
				segments = append(segments, segment)
			}
		}
	}

	if len(segments) == 0 {
		return "/"
	}

	path := "/" + strings.Join(segments, "/")
	// Convert any remaining :param to {param}
	path = swiftPathParamRegex.ReplaceAllString(path, "{$1}")
	return path
}

// IsSupported returns whether Swift parsing is supported.
func (p *SwiftParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *SwiftParser) SupportedExtensions() []string {
	return []string{".swift"}
}

// SwiftTypeToOpenAPI converts a Swift type to an OpenAPI type.
func SwiftTypeToOpenAPI(swiftType string) (openAPIType string, format string) {
	// Trim whitespace
	swiftType = strings.TrimSpace(swiftType)

	// Handle optional types
	if strings.HasSuffix(swiftType, "?") {
		swiftType = strings.TrimSuffix(swiftType, "?")
	}

	// Handle array types
	if strings.HasPrefix(swiftType, "[") && strings.HasSuffix(swiftType, "]") {
		return "array", ""
	}
	if strings.HasPrefix(swiftType, "Array<") {
		return "array", ""
	}

	// Handle dictionary types
	if strings.HasPrefix(swiftType, "Dictionary<") || strings.HasPrefix(swiftType, "[") && strings.Contains(swiftType, ":") {
		return "object", ""
	}

	switch swiftType {
	case "String":
		return "string", ""
	case "Int", "Int8", "Int16", "Int32", "Int64":
		return "integer", ""
	case "UInt", "UInt8", "UInt16", "UInt32", "UInt64":
		return "integer", ""
	case "Float":
		return "number", "float"
	case "Double":
		return "number", "double"
	case "Bool":
		return "boolean", ""
	case "Date":
		return "string", "date-time"
	case "UUID":
		return "string", "uuid"
	case "Data":
		return "string", "binary"
	case "URL":
		return "string", "uri"
	default:
		return "object", ""
	}
}

// ConvertVaporPathParams converts Vapor path parameters to OpenAPI format.
// Vapor uses :param format.
func ConvertVaporPathParams(path string) string {
	// Convert :param to {param}
	colonParamRegex := regexp.MustCompile(`:(\w+)`)
	return colonParamRegex.ReplaceAllString(path, "{$1}")
}

// ExtractVaporPathParams extracts parameter names from a Vapor path.
func ExtractVaporPathParams(path string) []string {
	var params []string

	// Match :param style
	colonRegex := regexp.MustCompile(`:(\w+)`)
	for _, match := range colonRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	// Match {param} style (already OpenAPI format)
	braceRegex := regexp.MustCompile(`\{(\w+)\}`)
	for _, match := range braceRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}
