// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// CSharpParser provides C# parsing capabilities using regex patterns.
// Tree-sitter-c-sharp setup is complex, so we use regex for now.
type CSharpParser struct{}

// NewCSharpParser creates a new C# parser.
func NewCSharpParser() *CSharpParser {
	return &CSharpParser{}
}

// CSharpClass represents a C# class definition.
type CSharpClass struct {
	// Name is the class name
	Name string

	// Attributes are the class attributes (e.g., [ApiController])
	Attributes []CSharpAttribute

	// BaseClasses are the base classes/interfaces
	BaseClasses []string

	// Methods are the class methods
	Methods []CSharpMethod

	// Line is the source line number
	Line int
}

// CSharpAttribute represents a C# attribute.
type CSharpAttribute struct {
	// Name is the attribute name (e.g., "Route", "HttpGet")
	Name string

	// Arguments are the attribute arguments
	Arguments []string

	// Line is the source line number
	Line int
}

// CSharpMethod represents a C# method definition.
type CSharpMethod struct {
	// Name is the method name
	Name string

	// Attributes are the method attributes
	Attributes []CSharpAttribute

	// Parameters are the method parameters
	Parameters []CSharpParameter

	// ReturnType is the return type
	ReturnType string

	// IsAsync indicates if the method is async
	IsAsync bool

	// IsPublic indicates if the method is public
	IsPublic bool

	// Line is the source line number
	Line int
}

// CSharpParameter represents a method parameter.
type CSharpParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Attributes are the parameter attributes (e.g., [FromBody])
	Attributes []CSharpAttribute
}

// ParsedCSharpFile represents a parsed C# source file.
type ParsedCSharpFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Classes are the extracted class definitions
	Classes []CSharpClass

	// Usings are the using statements
	Usings []string

	// MinimalAPIRoutes are routes defined using minimal API syntax
	MinimalAPIRoutes []CSharpMinimalRoute
}

// CSharpMinimalRoute represents a minimal API route definition.
type CSharpMinimalRoute struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string

	// Path is the route path
	Path string

	// Handler is the handler expression
	Handler string

	// Line is the source line number
	Line int
}

// Regex patterns for C# parsing
var (
	// Matches using statements
	csharpUsingRegex = regexp.MustCompile(`(?m)^using\s+([^;]+);`)

	// Matches class definitions with optional attributes
	csharpClassRegex = regexp.MustCompile(`(?ms)((?:\[[^\]]+\]\s*)*)\s*(public|internal|private|protected)?\s*(abstract|sealed|static|partial)?\s*class\s+(\w+)(?:\s*:\s*([^{]+))?`)

	// Matches attributes
	csharpAttributeRegex = regexp.MustCompile(`\[(\w+)(?:\s*\(([^)]*)\))?\]`)

	// Matches method definitions with optional attributes
	csharpMethodRegex = regexp.MustCompile(`(?ms)((?:\[[^\]]+\]\s*)*)\s*(public|private|protected|internal)?\s*(async)?\s*(static)?\s*([\w<>,\s\[\]?]+)\s+(\w+)\s*\(([^)]*)\)`)

	// Matches minimal API route definitions
	csharpMinimalAPIRegex = regexp.MustCompile(`(?m)(?:app|builder\.Services)\s*\.\s*Map(Get|Post|Put|Delete|Patch)\s*\(\s*"([^"]+)"`)
)

// Parse parses C# source code.
func (p *CSharpParser) Parse(filename string, content []byte) *ParsedCSharpFile {
	src := string(content)
	pf := &ParsedCSharpFile{
		Path:             filename,
		Content:          src,
		Classes:          []CSharpClass{},
		Usings:           []string{},
		MinimalAPIRoutes: []CSharpMinimalRoute{},
	}

	// Extract using statements
	pf.Usings = p.extractUsings(src)

	// Extract classes
	pf.Classes = p.extractClasses(src)

	// Extract minimal API routes
	pf.MinimalAPIRoutes = p.extractMinimalAPIRoutes(src)

	return pf
}

// extractUsings extracts using statements from C# source.
func (p *CSharpParser) extractUsings(src string) []string {
	var usings []string
	matches := csharpUsingRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			usings = append(usings, strings.TrimSpace(match[1]))
		}
	}
	return usings
}

// extractClasses extracts class definitions from C# source.
func (p *CSharpParser) extractClasses(src string) []CSharpClass {
	var classes []CSharpClass

	matches := csharpClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		class := CSharpClass{
			Line:        line,
			Attributes:  []CSharpAttribute{},
			BaseClasses: []string{},
			Methods:     []CSharpMethod{},
		}

		// Extract attributes (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			attrStr := src[match[2]:match[3]]
			class.Attributes = p.extractAttributes(attrStr)
		}

		// Extract class name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			class.Name = src[match[8]:match[9]]
		}

		// Extract base classes (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			baseStr := src[match[10]:match[11]]
			bases := strings.Split(baseStr, ",")
			for _, base := range bases {
				base = strings.TrimSpace(base)
				if base != "" {
					class.BaseClasses = append(class.BaseClasses, base)
				}
			}
		}

		// Find the class body and extract methods
		classStart := match[0]
		classBody := p.findClassBody(src[classStart:])
		if classBody != "" {
			class.Methods = p.extractMethods(classBody, line)
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// findClassBody finds the body of a class (between { and }).
func (p *CSharpParser) findClassBody(src string) string {
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
func (p *CSharpParser) extractMethods(body string, baseLineOffset int) []CSharpMethod {
	var methods []CSharpMethod

	matches := csharpMethodRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 16 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		method := CSharpMethod{
			Line:       line,
			Attributes: []CSharpAttribute{},
			Parameters: []CSharpParameter{},
		}

		// Extract attributes (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			attrStr := body[match[2]:match[3]]
			method.Attributes = p.extractAttributes(attrStr)
		}

		// Extract visibility (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			visibility := body[match[4]:match[5]]
			method.IsPublic = visibility == "public"
		}

		// Check async (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			method.IsAsync = true
		}

		// Extract return type (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			method.ReturnType = strings.TrimSpace(body[match[10]:match[11]])
		}

		// Extract method name (group 6)
		if match[12] >= 0 && match[13] >= 0 {
			method.Name = body[match[12]:match[13]]
		}

		// Extract parameters (group 7)
		if match[14] >= 0 && match[15] >= 0 {
			paramStr := body[match[14]:match[15]]
			method.Parameters = p.extractParameters(paramStr)
		}

		if method.Name != "" {
			methods = append(methods, method)
		}
	}

	return methods
}

// extractAttributes extracts attributes from an attribute string.
func (p *CSharpParser) extractAttributes(src string) []CSharpAttribute {
	var attrs []CSharpAttribute

	matches := csharpAttributeRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		attr := CSharpAttribute{
			Name:      match[1],
			Arguments: []string{},
		}

		if len(match) > 2 && match[2] != "" {
			// Parse arguments
			args := strings.Split(match[2], ",")
			for _, arg := range args {
				arg = strings.TrimSpace(arg)
				// Remove quotes from string arguments
				arg = strings.Trim(arg, `"`)
				if arg != "" {
					attr.Arguments = append(attr.Arguments, arg)
				}
			}
		}

		attrs = append(attrs, attr)
	}

	return attrs
}

// extractParameters extracts parameters from a parameter string.
func (p *CSharpParser) extractParameters(src string) []CSharpParameter {
	var params []CSharpParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma, but be careful with generics
	paramStrs := splitParameters(src)

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := CSharpParameter{
			Attributes: []CSharpAttribute{},
		}

		// Check for attributes
		attrMatches := csharpAttributeRegex.FindAllStringSubmatch(paramStr, -1)
		for _, match := range attrMatches {
			if len(match) >= 2 {
				attr := CSharpAttribute{
					Name:      match[1],
					Arguments: []string{},
				}
				if len(match) > 2 && match[2] != "" {
					attr.Arguments = append(attr.Arguments, strings.Trim(match[2], `"`))
				}
				param.Attributes = append(param.Attributes, attr)
			}
		}

		// Remove attributes from param string
		cleanParam := csharpAttributeRegex.ReplaceAllString(paramStr, "")
		cleanParam = strings.TrimSpace(cleanParam)

		// Split into type and name
		parts := strings.Fields(cleanParam)
		if len(parts) >= 2 {
			param.Type = strings.Join(parts[:len(parts)-1], " ")
			param.Name = parts[len(parts)-1]
		} else if len(parts) == 1 {
			param.Name = parts[0]
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// extractMinimalAPIRoutes extracts minimal API route definitions.
func (p *CSharpParser) extractMinimalAPIRoutes(src string) []CSharpMinimalRoute {
	var routes []CSharpMinimalRoute

	matches := csharpMinimalAPIRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := CSharpMinimalRoute{
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

		if route.Method != "" && route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// splitParameters splits a parameter string by comma, handling generics.
func splitParameters(src string) []string {
	var params []string
	var current strings.Builder
	depth := 0

	for _, ch := range src {
		switch ch {
		case '<':
			depth++
			current.WriteRune(ch)
		case '>':
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

// countLines counts the number of lines in a string.
func countLines(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

// HasAttribute checks if a method has a specific attribute.
func (m *CSharpMethod) HasAttribute(name string) bool {
	for _, attr := range m.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}

// GetAttribute returns an attribute by name, or nil if not found.
func (m *CSharpMethod) GetAttribute(name string) *CSharpAttribute {
	for i := range m.Attributes {
		if m.Attributes[i].Name == name {
			return &m.Attributes[i]
		}
	}
	return nil
}

// HasAttribute checks if a class has a specific attribute.
func (c *CSharpClass) HasAttribute(name string) bool {
	for _, attr := range c.Attributes {
		if attr.Name == name {
			return true
		}
	}
	return false
}

// GetAttribute returns an attribute by name, or nil if not found.
func (c *CSharpClass) GetAttribute(name string) *CSharpAttribute {
	for i := range c.Attributes {
		if c.Attributes[i].Name == name {
			return &c.Attributes[i]
		}
	}
	return nil
}

// IsControllerBase checks if the class inherits from ControllerBase.
func (c *CSharpClass) IsControllerBase() bool {
	for _, base := range c.BaseClasses {
		base = strings.TrimSpace(base)
		if base == "ControllerBase" || base == "Controller" ||
			strings.HasSuffix(base, "ControllerBase") ||
			strings.HasSuffix(base, "Controller") {
			return true
		}
	}
	return false
}

// IsSupported returns whether C# parsing is supported.
func (p *CSharpParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *CSharpParser) SupportedExtensions() []string {
	return []string{".cs"}
}

// CSharpTypeToOpenAPI converts a C# type to an OpenAPI type.
func CSharpTypeToOpenAPI(csType string) (openAPIType string, format string) {
	// Trim whitespace and handle nullable types
	csType = strings.TrimSpace(csType)
	csType = strings.TrimSuffix(csType, "?")

	// Handle common wrapper types
	if strings.HasPrefix(csType, "ActionResult<") {
		csType = extractCSharpGenericType(csType)
	}
	if strings.HasPrefix(csType, "Task<") {
		csType = extractCSharpGenericType(csType)
	}
	if strings.HasPrefix(csType, "IActionResult") {
		return "object", ""
	}

	// Handle List/Array types
	if strings.HasPrefix(csType, "List<") || strings.HasPrefix(csType, "IList<") ||
		strings.HasPrefix(csType, "IEnumerable<") || strings.HasSuffix(csType, "[]") {
		return "array", ""
	}

	// Handle Dictionary types
	if strings.HasPrefix(csType, "Dictionary<") || strings.HasPrefix(csType, "IDictionary<") {
		return "object", ""
	}

	switch csType {
	case "string", "String":
		return "string", ""
	case "int", "Int32", "long", "Int64", "short", "Int16":
		return "integer", ""
	case "uint", "UInt32", "ulong", "UInt64", "ushort", "UInt16":
		return "integer", ""
	case "float", "Single", "double", "Double", "decimal", "Decimal":
		return "number", ""
	case "bool", "Boolean":
		return "boolean", ""
	case "DateTime", "DateTimeOffset":
		return "string", "date-time"
	case "DateOnly":
		return "string", "date"
	case "TimeOnly", "TimeSpan":
		return "string", "time"
	case "Guid":
		return "string", "uuid"
	case "byte[]":
		return "string", "binary"
	case "void":
		return "", ""
	default:
		return "object", ""
	}
}

// extractCSharpGenericType extracts the inner type from a generic like List<string>.
func extractCSharpGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}
