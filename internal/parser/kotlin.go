// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// KotlinParser provides Kotlin parsing capabilities using regex patterns.
type KotlinParser struct{}

// NewKotlinParser creates a new Kotlin parser.
func NewKotlinParser() *KotlinParser {
	return &KotlinParser{}
}

// KotlinClass represents a Kotlin class definition.
type KotlinClass struct {
	// Name is the class name
	Name string

	// Package is the package name
	Package string

	// Annotations are the class annotations
	Annotations []KotlinAnnotation

	// SuperClass is the parent class
	SuperClass string

	// Interfaces are the implemented interfaces
	Interfaces []string

	// Functions are the class functions
	Functions []KotlinFunction

	// Line is the source line number
	Line int
}

// KotlinAnnotation represents a Kotlin annotation.
type KotlinAnnotation struct {
	// Name is the annotation name
	Name string

	// Arguments are the annotation arguments
	Arguments []string

	// Line is the source line number
	Line int
}

// KotlinFunction represents a Kotlin function definition.
type KotlinFunction struct {
	// Name is the function name
	Name string

	// Annotations are the function annotations
	Annotations []KotlinAnnotation

	// Parameters are the function parameters
	Parameters []KotlinParameter

	// ReturnType is the return type
	ReturnType string

	// IsSuspend indicates if the function is a suspend function
	IsSuspend bool

	// Line is the source line number
	Line int
}

// KotlinParameter represents a function parameter.
type KotlinParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Default is the default value
	Default string

	// Annotations are the parameter annotations
	Annotations []KotlinAnnotation
}

// KotlinRoute represents a Ktor route definition.
type KotlinRoute struct {
	// Method is the HTTP method (get, post, put, delete, etc.)
	Method string

	// Path is the route path
	Path string

	// Handler is the handler expression
	Handler string

	// Line is the source line number
	Line int
}

// ParsedKotlinFile represents a parsed Kotlin source file.
type ParsedKotlinFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Package is the package name
	Package string

	// Imports are the import statements
	Imports []string

	// Classes are the extracted class definitions
	Classes []KotlinClass

	// TopLevelFunctions are functions defined at package level
	TopLevelFunctions []KotlinFunction

	// Routes are the extracted Ktor route definitions
	Routes []KotlinRoute
}

// Regex patterns for Kotlin parsing
var (
	// Matches package declaration
	kotlinPackageRegex = regexp.MustCompile(`(?m)^package\s+([^\s]+)`)

	// Matches import statements
	kotlinImportRegex = regexp.MustCompile(`(?m)^import\s+([^\s]+)`)

	// Matches annotations
	kotlinAnnotationRegex = regexp.MustCompile(`@(\w+)(?:\s*\(\s*([^)]*)\s*\))?`)

	// Matches class definitions
	kotlinClassRegex = regexp.MustCompile(`(?ms)((?:@\w+(?:\s*\([^)]*\))?\s*)*)\s*(?:data\s+)?(?:open\s+)?(?:abstract\s+)?class\s+(\w+)(?:\s*\([^)]*\))?(?:\s*:\s*([^{]+))?`)

	// Matches function definitions
	kotlinFunctionRegex = regexp.MustCompile(`(?ms)((?:@\w+(?:\s*\([^)]*\))?\s*)*)\s*(?:suspend\s+)?(fun)\s+(\w+)\s*\(([^)]*)\)(?:\s*:\s*([\w<>,\s?]+))?`)

	// Matches Ktor route definitions
	// get("/path") { ... }
	kotlinRouteRegex = regexp.MustCompile(`(?m)(get|post|put|delete|patch|head|options)\s*\(\s*"([^"]+)"\s*\)`)
)

// Parse parses Kotlin source code.
func (p *KotlinParser) Parse(filename string, content []byte) *ParsedKotlinFile {
	src := string(content)
	pf := &ParsedKotlinFile{
		Path:              filename,
		Content:           src,
		Imports:           []string{},
		Classes:           []KotlinClass{},
		TopLevelFunctions: []KotlinFunction{},
		Routes:            []KotlinRoute{},
	}

	// Extract package
	if match := kotlinPackageRegex.FindStringSubmatch(src); len(match) > 1 {
		pf.Package = match[1]
	}

	// Extract imports
	pf.Imports = p.extractImports(src)

	// Extract classes
	pf.Classes = p.extractClasses(src)

	// Extract top-level functions
	pf.TopLevelFunctions = p.extractTopLevelFunctions(src)

	// Extract routes
	pf.Routes = p.extractRoutes(src)

	return pf
}

// extractImports extracts import statements from Kotlin source.
func (p *KotlinParser) extractImports(src string) []string {
	var imports []string
	matches := kotlinImportRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, strings.TrimSpace(match[1]))
		}
	}
	return imports
}

// extractClasses extracts class definitions from Kotlin source.
func (p *KotlinParser) extractClasses(src string) []KotlinClass {
	var classes []KotlinClass

	matches := kotlinClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		class := KotlinClass{
			Line:        line,
			Annotations: []KotlinAnnotation{},
			Interfaces:  []string{},
			Functions:   []KotlinFunction{},
		}

		// Extract annotations (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			annoStr := src[match[2]:match[3]]
			class.Annotations = p.extractAnnotations(annoStr)
		}

		// Extract class name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			class.Name = src[match[4]:match[5]]
		}

		// Extract super class/interfaces (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			superStr := src[match[6]:match[7]]
			parts := strings.Split(superStr, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				// Remove constructor params from parent class
				if idx := strings.Index(part, "("); idx > 0 {
					part = part[:idx]
				}
				if part != "" {
					if class.SuperClass == "" {
						class.SuperClass = part
					} else {
						class.Interfaces = append(class.Interfaces, part)
					}
				}
			}
		}

		// Find the class body and extract functions
		classStart := match[0]
		classBody := p.findClassBody(src[classStart:])
		if classBody != "" {
			class.Functions = p.extractFunctions(classBody, line)
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// findClassBody finds the body of a class (between { and }).
func (p *KotlinParser) findClassBody(src string) string {
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

// extractFunctions extracts function definitions from a class body.
func (p *KotlinParser) extractFunctions(body string, baseLineOffset int) []KotlinFunction {
	var functions []KotlinFunction

	matches := kotlinFunctionRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 12 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		fn := KotlinFunction{
			Line:        line,
			Annotations: []KotlinAnnotation{},
			Parameters:  []KotlinParameter{},
		}

		// Extract annotations (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			annoStr := body[match[2]:match[3]]
			fn.Annotations = p.extractAnnotations(annoStr)
		}

		// Check for suspend (in the matched area before fun keyword)
		funStart := match[4]
		if funStart > 0 {
			preFunc := body[match[0]:funStart]
			fn.IsSuspend = strings.Contains(preFunc, "suspend")
		}

		// Extract function name (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			fn.Name = body[match[6]:match[7]]
		}

		// Extract parameters (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			paramStr := body[match[8]:match[9]]
			fn.Parameters = p.extractParameters(paramStr)
		}

		// Extract return type (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			fn.ReturnType = strings.TrimSpace(body[match[10]:match[11]])
		}

		if fn.Name != "" {
			functions = append(functions, fn)
		}
	}

	return functions
}

// extractTopLevelFunctions extracts top-level functions from the source.
func (p *KotlinParser) extractTopLevelFunctions(src string) []KotlinFunction {
	// For simplicity, extract all functions; in practice,
	// we'd need to filter out those inside classes
	return p.extractFunctions(src, 0)
}

// extractAnnotations extracts annotations from an annotation string.
func (p *KotlinParser) extractAnnotations(src string) []KotlinAnnotation {
	var annotations []KotlinAnnotation

	matches := kotlinAnnotationRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		anno := KotlinAnnotation{
			Name:      match[1],
			Arguments: []string{},
		}

		if len(match) > 2 && match[2] != "" {
			// Parse annotation arguments
			argsStr := strings.TrimSpace(match[2])
			if argsStr != "" {
				args := splitKotlinAnnotationArgs(argsStr)
				anno.Arguments = args
			}
		}

		annotations = append(annotations, anno)
	}

	return annotations
}

// splitKotlinAnnotationArgs splits annotation arguments.
func splitKotlinAnnotationArgs(src string) []string {
	var args []string
	var current strings.Builder
	inString := false
	depth := 0

	for _, ch := range src {
		switch ch {
		case '"':
			inString = !inString
			current.WriteRune(ch)
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if !inString && depth == 0 {
				arg := strings.TrimSpace(current.String())
				arg = strings.Trim(arg, `"`)
				if arg != "" {
					args = append(args, arg)
				}
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		arg := strings.TrimSpace(current.String())
		arg = strings.Trim(arg, `"`)
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

// extractParameters extracts parameters from a parameter string.
func (p *KotlinParser) extractParameters(src string) []KotlinParameter {
	var params []KotlinParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma, handling generics
	paramStrs := splitKotlinParameters(src)

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := KotlinParameter{
			Annotations: []KotlinAnnotation{},
		}

		// Check for annotations
		annoMatches := kotlinAnnotationRegex.FindAllStringSubmatch(paramStr, -1)
		for _, match := range annoMatches {
			if len(match) >= 2 {
				anno := KotlinAnnotation{
					Name:      match[1],
					Arguments: []string{},
				}
				if len(match) > 2 && match[2] != "" {
					anno.Arguments = append(anno.Arguments, strings.Trim(match[2], `"`))
				}
				param.Annotations = append(param.Annotations, anno)
			}
		}

		// Remove annotations from param string
		cleanParam := kotlinAnnotationRegex.ReplaceAllString(paramStr, "")
		cleanParam = strings.TrimSpace(cleanParam)

		// Check for default value
		if idx := strings.Index(cleanParam, "="); idx > 0 {
			param.Default = strings.TrimSpace(cleanParam[idx+1:])
			cleanParam = strings.TrimSpace(cleanParam[:idx])
		}

		// Split into name: Type
		if idx := strings.Index(cleanParam, ":"); idx > 0 {
			param.Name = strings.TrimSpace(cleanParam[:idx])
			// Handle val/var prefix
			param.Name = strings.TrimPrefix(param.Name, "val ")
			param.Name = strings.TrimPrefix(param.Name, "var ")
			param.Type = strings.TrimSpace(cleanParam[idx+1:])
		} else {
			param.Name = cleanParam
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// splitKotlinParameters splits a parameter string by comma, handling generics.
func splitKotlinParameters(src string) []string {
	var params []string
	var current strings.Builder
	depth := 0
	inString := false

	for _, ch := range src {
		switch ch {
		case '"':
			inString = !inString
			current.WriteRune(ch)
		case '<', '(':
			depth++
			current.WriteRune(ch)
		case '>', ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 && !inString {
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

// extractRoutes extracts Ktor route definitions.
func (p *KotlinParser) extractRoutes(src string) []KotlinRoute {
	var routes []KotlinRoute

	matches := kotlinRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := KotlinRoute{
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

// HasAnnotation checks if a function has a specific annotation.
func (f *KotlinFunction) HasAnnotation(name string) bool {
	for _, anno := range f.Annotations {
		if anno.Name == name {
			return true
		}
	}
	return false
}

// GetAnnotation returns an annotation by name, or nil if not found.
func (f *KotlinFunction) GetAnnotation(name string) *KotlinAnnotation {
	for i := range f.Annotations {
		if f.Annotations[i].Name == name {
			return &f.Annotations[i]
		}
	}
	return nil
}

// IsSupported returns whether Kotlin parsing is supported.
func (p *KotlinParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *KotlinParser) SupportedExtensions() []string {
	return []string{".kt", ".kts"}
}

// KotlinTypeToOpenAPI converts a Kotlin type to an OpenAPI type.
func KotlinTypeToOpenAPI(ktType string) (openAPIType string, format string) {
	// Trim whitespace and handle nullable types
	ktType = strings.TrimSpace(ktType)
	ktType = strings.TrimSuffix(ktType, "?")

	// Handle List types
	if strings.HasPrefix(ktType, "List<") || strings.HasPrefix(ktType, "MutableList<") ||
		strings.HasPrefix(ktType, "Collection<") || strings.HasPrefix(ktType, "Array<") {
		return "array", ""
	}

	// Handle Map types
	if strings.HasPrefix(ktType, "Map<") || strings.HasPrefix(ktType, "MutableMap<") {
		return "object", ""
	}

	switch ktType {
	case "String":
		return "string", ""
	case "Int", "Long", "Short", "Byte":
		return "integer", ""
	case "UInt", "ULong", "UShort", "UByte":
		return "integer", ""
	case "Float", "Double":
		return "number", ""
	case "Boolean":
		return "boolean", ""
	case "LocalDateTime", "ZonedDateTime", "OffsetDateTime", "Instant":
		return "string", "date-time"
	case "LocalDate":
		return "string", "date"
	case "LocalTime", "OffsetTime":
		return "string", "time"
	case "UUID":
		return "string", "uuid"
	case "ByteArray":
		return "string", "binary"
	case "Unit", "Nothing":
		return "", ""
	default:
		return "object", ""
	}
}
