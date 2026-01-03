// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// JavaParser provides Java parsing capabilities using regex patterns.
type JavaParser struct{}

// NewJavaParser creates a new Java parser.
func NewJavaParser() *JavaParser {
	return &JavaParser{}
}

// JavaClass represents a Java class definition.
type JavaClass struct {
	// Name is the class name
	Name string

	// Package is the package name
	Package string

	// Annotations are the class annotations
	Annotations []JavaAnnotation

	// Extends is the parent class
	Extends string

	// Implements are the implemented interfaces
	Implements []string

	// Methods are the class methods
	Methods []JavaMethod

	// Line is the source line number
	Line int
}

// JavaAnnotation represents a Java annotation.
type JavaAnnotation struct {
	// Name is the annotation name (e.g., "RestController", "GetMapping")
	Name string

	// Value is the annotation value (if single value)
	Value string

	// Attributes are named annotation attributes
	Attributes map[string]string

	// Line is the source line number
	Line int
}

// JavaMethod represents a Java method definition.
type JavaMethod struct {
	// Name is the method name
	Name string

	// Annotations are the method annotations
	Annotations []JavaAnnotation

	// Parameters are the method parameters
	Parameters []JavaParameter

	// ReturnType is the return type
	ReturnType string

	// Visibility is the method visibility (public, private, protected)
	Visibility string

	// Line is the source line number
	Line int
}

// JavaParameter represents a method parameter.
type JavaParameter struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Annotations are the parameter annotations
	Annotations []JavaAnnotation
}

// ParsedJavaFile represents a parsed Java source file.
type ParsedJavaFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Package is the package name
	Package string

	// Imports are the import statements
	Imports []string

	// Classes are the extracted class definitions
	Classes []JavaClass
}

// Regex patterns for Java parsing
var (
	// Matches package declaration
	javaPackageRegex = regexp.MustCompile(`(?m)^package\s+([^;]+);`)

	// Matches import statements
	javaImportRegex = regexp.MustCompile(`(?m)^import\s+(?:static\s+)?([^;]+);`)

	// Matches annotations
	javaAnnotationRegex = regexp.MustCompile(`@(\w+)(?:\s*\(\s*([^)]*)\s*\))?`)

	// Matches class definitions with annotations
	javaClassRegex = regexp.MustCompile(`(?ms)((?:@\w+(?:\s*\([^)]*\))?\s*)*)\s*(public|private|protected)?\s*(abstract|final)?\s*class\s+(\w+)(?:\s+extends\s+(\w+))?(?:\s+implements\s+([^{]+))?`)

	// Matches method definitions with annotations
	javaMethodRegex = regexp.MustCompile(`(?ms)((?:@\w+(?:\s*\([^)]*\))?\s*)*)\s*(public|private|protected)?\s*(?:static\s+)?(?:final\s+)?([\w<>,\s\[\]?]+)\s+(\w+)\s*\(([^)]*)\)`)
)

// Parse parses Java source code.
func (p *JavaParser) Parse(filename string, content []byte) *ParsedJavaFile {
	src := string(content)
	pf := &ParsedJavaFile{
		Path:    filename,
		Content: src,
		Imports: []string{},
		Classes: []JavaClass{},
	}

	// Extract package
	if match := javaPackageRegex.FindStringSubmatch(src); len(match) > 1 {
		pf.Package = match[1]
	}

	// Extract imports
	pf.Imports = p.extractImports(src)

	// Extract classes
	pf.Classes = p.extractClasses(src)

	return pf
}

// extractImports extracts import statements from Java source.
func (p *JavaParser) extractImports(src string) []string {
	var imports []string
	matches := javaImportRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, strings.TrimSpace(match[1]))
		}
	}
	return imports
}

// extractClasses extracts class definitions from Java source.
func (p *JavaParser) extractClasses(src string) []JavaClass {
	var classes []JavaClass

	matches := javaClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 14 {
			continue
		}

		line := countLines(src[:match[0]])

		class := JavaClass{
			Line:        line,
			Annotations: []JavaAnnotation{},
			Implements:  []string{},
			Methods:     []JavaMethod{},
		}

		// Extract annotations (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			annoStr := src[match[2]:match[3]]
			class.Annotations = p.extractAnnotations(annoStr)
		}

		// Extract class name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			class.Name = src[match[8]:match[9]]
		}

		// Extract parent class (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			class.Extends = src[match[10]:match[11]]
		}

		// Extract implemented interfaces (group 6)
		if match[12] >= 0 && match[13] >= 0 {
			implStr := src[match[12]:match[13]]
			impls := strings.Split(implStr, ",")
			for _, impl := range impls {
				impl = strings.TrimSpace(impl)
				if impl != "" {
					class.Implements = append(class.Implements, impl)
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
func (p *JavaParser) findClassBody(src string) string {
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
func (p *JavaParser) extractMethods(body string, baseLineOffset int) []JavaMethod {
	var methods []JavaMethod

	matches := javaMethodRegex.FindAllStringSubmatchIndex(body, -1)
	for _, match := range matches {
		if len(match) < 12 {
			continue
		}

		line := baseLineOffset + countLines(body[:match[0]])

		method := JavaMethod{
			Line:        line,
			Annotations: []JavaAnnotation{},
			Parameters:  []JavaParameter{},
		}

		// Extract annotations (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			annoStr := body[match[2]:match[3]]
			method.Annotations = p.extractAnnotations(annoStr)
		}

		// Extract visibility (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			method.Visibility = body[match[4]:match[5]]
		} else {
			method.Visibility = "package" // Default package-private
		}

		// Extract return type (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			method.ReturnType = strings.TrimSpace(body[match[6]:match[7]])
		}

		// Extract method name (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			method.Name = body[match[8]:match[9]]
		}

		// Extract parameters (group 5)
		if match[10] >= 0 && match[11] >= 0 {
			paramStr := body[match[10]:match[11]]
			method.Parameters = p.extractParameters(paramStr)
		}

		// Skip constructors (method name equals class name or no return type)
		if method.Name != "" && method.ReturnType != "" {
			methods = append(methods, method)
		}
	}

	return methods
}

// extractAnnotations extracts annotations from an annotation string.
func (p *JavaParser) extractAnnotations(src string) []JavaAnnotation {
	var annotations []JavaAnnotation

	matches := javaAnnotationRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		anno := JavaAnnotation{
			Name:       match[1],
			Attributes: make(map[string]string),
		}

		if len(match) > 2 && match[2] != "" {
			// Parse annotation value/attributes
			valueStr := strings.TrimSpace(match[2])
			if valueStr != "" {
				// Check if it's a single value or named attributes
				if strings.Contains(valueStr, "=") {
					// Named attributes
					attrs := splitJavaAnnotationAttrs(valueStr)
					for _, attr := range attrs {
						parts := strings.SplitN(attr, "=", 2)
						if len(parts) == 2 {
							key := strings.TrimSpace(parts[0])
							value := strings.TrimSpace(parts[1])
							value = strings.Trim(value, `"`)
							anno.Attributes[key] = value
						}
					}
				} else {
					// Single value
					anno.Value = strings.Trim(valueStr, `"`)
				}
			}
		}

		annotations = append(annotations, anno)
	}

	return annotations
}

// splitJavaAnnotationAttrs splits annotation attributes by comma, handling strings.
func splitJavaAnnotationAttrs(src string) []string {
	var attrs []string
	var current strings.Builder
	inString := false
	depth := 0

	for _, ch := range src {
		switch ch {
		case '"':
			inString = !inString
			current.WriteRune(ch)
		case '{':
			depth++
			current.WriteRune(ch)
		case '}':
			depth--
			current.WriteRune(ch)
		case ',':
			if !inString && depth == 0 {
				attrs = append(attrs, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		attrs = append(attrs, current.String())
	}

	return attrs
}

// extractParameters extracts parameters from a parameter string.
func (p *JavaParser) extractParameters(src string) []JavaParameter {
	var params []JavaParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma, handling generics
	paramStrs := splitJavaParameters(src)

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := JavaParameter{
			Annotations: []JavaAnnotation{},
		}

		// Check for annotations
		annoMatches := javaAnnotationRegex.FindAllStringSubmatch(paramStr, -1)
		for _, match := range annoMatches {
			if len(match) >= 2 {
				anno := JavaAnnotation{
					Name:       match[1],
					Attributes: make(map[string]string),
				}
				if len(match) > 2 && match[2] != "" {
					anno.Value = strings.Trim(match[2], `"`)
				}
				param.Annotations = append(param.Annotations, anno)
			}
		}

		// Remove annotations from param string
		cleanParam := javaAnnotationRegex.ReplaceAllString(paramStr, "")
		cleanParam = strings.TrimSpace(cleanParam)

		// Split into type and name
		parts := strings.Fields(cleanParam)
		if len(parts) >= 2 {
			// Handle final keyword
			if parts[0] == "final" {
				parts = parts[1:]
			}
			if len(parts) >= 2 {
				param.Type = strings.Join(parts[:len(parts)-1], " ")
				param.Name = parts[len(parts)-1]
			}
		} else if len(parts) == 1 {
			param.Name = parts[0]
		}

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// splitJavaParameters splits a parameter string by comma, handling generics.
func splitJavaParameters(src string) []string {
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

// HasAnnotation checks if a method has a specific annotation.
func (m *JavaMethod) HasAnnotation(name string) bool {
	for _, anno := range m.Annotations {
		if anno.Name == name {
			return true
		}
	}
	return false
}

// GetAnnotation returns an annotation by name, or nil if not found.
func (m *JavaMethod) GetAnnotation(name string) *JavaAnnotation {
	for i := range m.Annotations {
		if m.Annotations[i].Name == name {
			return &m.Annotations[i]
		}
	}
	return nil
}

// HasAnnotation checks if a class has a specific annotation.
func (c *JavaClass) HasAnnotation(name string) bool {
	for _, anno := range c.Annotations {
		if anno.Name == name {
			return true
		}
	}
	return false
}

// GetAnnotation returns an annotation by name, or nil if not found.
func (c *JavaClass) GetAnnotation(name string) *JavaAnnotation {
	for i := range c.Annotations {
		if c.Annotations[i].Name == name {
			return &c.Annotations[i]
		}
	}
	return nil
}

// IsSupported returns whether Java parsing is supported.
func (p *JavaParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *JavaParser) SupportedExtensions() []string {
	return []string{".java"}
}

// JavaTypeToOpenAPI converts a Java type to an OpenAPI type.
func JavaTypeToOpenAPI(javaType string) (openAPIType string, format string) {
	// Trim whitespace
	javaType = strings.TrimSpace(javaType)

	// Handle common wrapper types
	if strings.HasPrefix(javaType, "ResponseEntity<") {
		javaType = extractJavaGenericType(javaType)
	}
	if strings.HasPrefix(javaType, "Optional<") {
		javaType = extractJavaGenericType(javaType)
	}
	if strings.HasPrefix(javaType, "CompletableFuture<") {
		javaType = extractJavaGenericType(javaType)
	}

	// Handle List/Collection types
	if strings.HasPrefix(javaType, "List<") || strings.HasPrefix(javaType, "Collection<") ||
		strings.HasPrefix(javaType, "Set<") || strings.HasSuffix(javaType, "[]") {
		return "array", ""
	}

	// Handle Map types
	if strings.HasPrefix(javaType, "Map<") || strings.HasPrefix(javaType, "HashMap<") {
		return "object", ""
	}

	switch javaType {
	case "String":
		return "string", ""
	case "int", "Integer", "long", "Long", "short", "Short", "byte", "Byte":
		return "integer", ""
	case "float", "Float", "double", "Double", "BigDecimal":
		return "number", ""
	case "boolean", "Boolean":
		return "boolean", ""
	case "LocalDateTime", "ZonedDateTime", "OffsetDateTime", "Instant", "Date":
		return "string", "date-time"
	case "LocalDate":
		return "string", "date"
	case "LocalTime", "OffsetTime":
		return "string", "time"
	case "UUID":
		return "string", "uuid"
	case "byte[]":
		return "string", "binary"
	case "void", "Void":
		return "", ""
	default:
		return "object", ""
	}
}

// extractJavaGenericType extracts the inner type from a generic like List<String>.
func extractJavaGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}
