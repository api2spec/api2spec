// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// ScalaParser provides Scala parsing capabilities using regex patterns.
type ScalaParser struct{}

// NewScalaParser creates a new Scala parser.
func NewScalaParser() *ScalaParser {
	return &ScalaParser{}
}

// ScalaCaseClass represents a Scala case class definition.
type ScalaCaseClass struct {
	// Name is the case class name
	Name string

	// Fields are the case class fields
	Fields []ScalaField

	// Line is the source line number
	Line int
}

// ScalaField represents a case class field.
type ScalaField struct {
	// Name is the field name
	Name string

	// Type is the field type
	Type string

	// IsOptional indicates if the field is Option[T]
	IsOptional bool

	// Line is the source line number
	Line int
}

// ScalaRoute represents a route extracted from Scala web framework code.
type ScalaRoute struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Handler is the handler function/method name
	Handler string

	// Controller is the controller class name
	Controller string

	// Line is the source line number
	Line int
}

// PlayRoute represents a route from Play Framework's conf/routes file.
type PlayRoute struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Controller is the controller class
	Controller string

	// Action is the action method name
	Action string

	// Parameters are the action parameters
	Parameters []string

	// Line is the source line number
	Line int
}

// TapirEndpoint represents a Tapir endpoint definition.
type TapirEndpoint struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// Handler is the handler name
	Handler string

	// InputType is the input type
	InputType string

	// OutputType is the output type
	OutputType string

	// Line is the source line number
	Line int
}

// ParsedScalaFile represents a parsed Scala source file.
type ParsedScalaFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Package is the package name
	Package string

	// Imports are the import statements
	Imports []string

	// CaseClasses are the extracted case class definitions
	CaseClasses []ScalaCaseClass

	// Routes are the extracted route definitions
	Routes []ScalaRoute

	// PlayRoutes are routes from Play's conf/routes file
	PlayRoutes []PlayRoute

	// TapirEndpoints are Tapir endpoint definitions
	TapirEndpoints []TapirEndpoint
}

// Regex patterns for Scala parsing
var (
	// Matches package declaration
	scalaPackageRegex = regexp.MustCompile(`(?m)^package\s+([^\s;]+)`)

	// Matches import statements
	scalaImportRegex = regexp.MustCompile(`(?m)^import\s+([^\s;]+)`)

	// Matches case class definitions
	scalaCaseClassRegex = regexp.MustCompile(`(?m)case\s+class\s+(\w+)\s*\(([^)]*)\)`)

	// Matches Play routes file format: METHOD /path controller.action
	playRouteRegex = regexp.MustCompile(`(?m)^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+(/[^\s]*)\s+([^\s(]+)(?:\(([^)]*)\))?`)

	// Matches Play sub-routes: -> /prefix routes.file
	playSubRouteRegex = regexp.MustCompile(`(?m)^->\s+(/[^\s]*)\s+(\S+)`)

	// Matches Tapir endpoint definitions
	tapirEndpointRegex = regexp.MustCompile(`(?m)endpoint\s*\.\s*(get|post|put|delete|patch|head|options)`)

	// Matches Tapir .in("path") segments
	tapirInRegex = regexp.MustCompile(`\.in\s*\(\s*"([^"]+)"`)

	// Matches Tapir path[Type]("name") segments
	tapirPathParamRegex = regexp.MustCompile(`\.in\s*\(\s*path\s*\[\s*(\w+)\s*\]\s*\(\s*"([^"]+)"\s*\)`)

	// Matches Tapir jsonBody[Type]
	tapirJsonBodyRegex = regexp.MustCompile(`jsonBody\s*\[\s*([^\]]+)\s*\]`)

	// Matches Tapir .serverLogic or .serverLogicSuccess
	tapirServerLogicRegex = regexp.MustCompile(`\.serverLogic(?:Success)?\s*\(`)

	// Matches Scala def definitions (for Play controllers)
	scalaDefRegex = regexp.MustCompile(`(?m)def\s+(\w+)(?:\s*\(([^)]*)\))?\s*=\s*Action`)
)

// Parse parses Scala source code.
func (p *ScalaParser) Parse(filename string, content []byte) *ParsedScalaFile {
	src := string(content)
	pf := &ParsedScalaFile{
		Path:           filename,
		Content:        src,
		Imports:        []string{},
		CaseClasses:    []ScalaCaseClass{},
		Routes:         []ScalaRoute{},
		PlayRoutes:     []PlayRoute{},
		TapirEndpoints: []TapirEndpoint{},
	}

	// Extract package
	if match := scalaPackageRegex.FindStringSubmatch(src); len(match) > 1 {
		pf.Package = match[1]
	}

	// Extract imports
	pf.Imports = p.extractImports(src)

	// Extract case classes
	pf.CaseClasses = p.extractCaseClasses(src)

	// Extract Tapir endpoints
	pf.TapirEndpoints = p.extractTapirEndpoints(src)

	return pf
}

// ParsePlayRoutes parses a Play Framework routes file.
func (p *ScalaParser) ParsePlayRoutes(filename string, content []byte) *ParsedScalaFile {
	src := string(content)
	pf := &ParsedScalaFile{
		Path:       filename,
		Content:    src,
		PlayRoutes: []PlayRoute{},
	}

	pf.PlayRoutes = p.extractPlayRoutes(src)

	return pf
}

// extractImports extracts import statements from Scala source.
func (p *ScalaParser) extractImports(src string) []string {
	var imports []string
	matches := scalaImportRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, strings.TrimSpace(match[1]))
		}
	}
	return imports
}

// extractCaseClasses extracts case class definitions from Scala source.
func (p *ScalaParser) extractCaseClasses(src string) []ScalaCaseClass {
	var classes []ScalaCaseClass

	matches := scalaCaseClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		class := ScalaCaseClass{
			Line:   line,
			Fields: []ScalaField{},
		}

		// Extract class name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			class.Name = src[match[2]:match[3]]
		}

		// Extract fields (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			fieldsStr := src[match[4]:match[5]]
			class.Fields = p.extractCaseClassFields(fieldsStr, line)
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// extractCaseClassFields extracts fields from a case class parameter list.
func (p *ScalaParser) extractCaseClassFields(fieldsStr string, baseLine int) []ScalaField {
	var fields []ScalaField

	// Split by comma, handling generics
	fieldStrs := splitScalaParameters(fieldsStr)

	for _, fieldStr := range fieldStrs {
		fieldStr = strings.TrimSpace(fieldStr)
		if fieldStr == "" {
			continue
		}

		field := ScalaField{
			Line: baseLine,
		}

		// Parse field: name: Type
		parts := strings.SplitN(fieldStr, ":", 2)
		if len(parts) < 2 {
			continue
		}

		// Extract name (remove val/var if present)
		name := strings.TrimSpace(parts[0])
		name = strings.TrimPrefix(name, "val ")
		name = strings.TrimPrefix(name, "var ")
		field.Name = strings.TrimSpace(name)

		// Extract type
		typeStr := strings.TrimSpace(parts[1])
		// Remove default value if present
		if eqIdx := strings.Index(typeStr, "="); eqIdx != -1 {
			typeStr = strings.TrimSpace(typeStr[:eqIdx])
		}
		field.Type = typeStr

		// Check if optional
		if strings.HasPrefix(typeStr, "Option[") {
			field.IsOptional = true
		}

		if field.Name != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// splitScalaParameters splits a parameter string by comma, handling generics.
func splitScalaParameters(src string) []string {
	var params []string
	var current strings.Builder
	depth := 0

	for _, ch := range src {
		switch ch {
		case '[', '(':
			depth++
			current.WriteRune(ch)
		case ']', ')':
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

// extractPlayRoutes extracts routes from a Play Framework routes file.
func (p *ScalaParser) extractPlayRoutes(src string) []PlayRoute {
	var routes []PlayRoute

	lines := strings.Split(src, "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to match route definition
		match := playRouteRegex.FindStringSubmatch(line)
		if len(match) >= 4 {
			route := PlayRoute{
				Method: match[1],
				Path:   match[2],
				Line:   lineNum + 1,
			}

			// Parse controller.action
			controllerAction := match[3]
			if dotIdx := strings.LastIndex(controllerAction, "."); dotIdx != -1 {
				route.Controller = controllerAction[:dotIdx]
				route.Action = controllerAction[dotIdx+1:]
			} else {
				route.Action = controllerAction
			}

			// Parse parameters if present
			if len(match) > 4 && match[4] != "" {
				route.Parameters = parsePlayParameters(match[4])
			}

			routes = append(routes, route)
		}
	}

	return routes
}

// parsePlayParameters parses Play route parameters.
func parsePlayParameters(paramsStr string) []string {
	var params []string
	parts := strings.Split(paramsStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Extract parameter name (before colon)
		if colonIdx := strings.Index(part, ":"); colonIdx != -1 {
			params = append(params, strings.TrimSpace(part[:colonIdx]))
		} else {
			params = append(params, part)
		}
	}
	return params
}

// extractTapirEndpoints extracts Tapir endpoint definitions.
func (p *ScalaParser) extractTapirEndpoints(src string) []TapirEndpoint {
	var endpoints []TapirEndpoint

	// Find all endpoint definitions
	methodMatches := tapirEndpointRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range methodMatches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		endpoint := TapirEndpoint{
			Line: line,
		}

		// Extract HTTP method (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			endpoint.Method = strings.ToUpper(src[match[2]:match[3]])
		}

		// Find the rest of the endpoint chain
		chainStart := match[1]
		chainEnd := p.findEndpointChainEnd(src[chainStart:])
		if chainEnd == -1 {
			chainEnd = len(src) - chainStart
		}
		chain := src[chainStart : chainStart+chainEnd]

		// Extract path segments
		endpoint.Path = p.extractTapirPath(chain)

		// Extract output type from jsonBody
		if outputMatch := tapirJsonBodyRegex.FindStringSubmatch(chain); len(outputMatch) > 1 {
			endpoint.OutputType = strings.TrimSpace(outputMatch[1])
		}

		if endpoint.Path != "" {
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// findEndpointChainEnd finds the end of a Tapir endpoint chain.
func (p *ScalaParser) findEndpointChainEnd(src string) int {
	// Look for .serverLogic or end of statement
	if match := tapirServerLogicRegex.FindStringIndex(src); match != nil {
		return match[1]
	}

	// Look for common chain terminators
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			// Check if next line continues the chain
			if i+1 < len(src) && src[i+1] != '.' && src[i+1] != ' ' && src[i+1] != '\t' {
				return i
			}
		}
	}

	return len(src)
}

// extractTapirPath extracts the path from a Tapir endpoint chain.
func (p *ScalaParser) extractTapirPath(chain string) string {
	var pathParts []string

	// Extract static path segments
	inMatches := tapirInRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range inMatches {
		if len(match) > 1 {
			segment := match[1]
			// Handle path with multiple segments like "users" / "list"
			parts := strings.Split(segment, "/")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					pathParts = append(pathParts, part)
				}
			}
		}
	}

	// Extract path parameters
	paramMatches := tapirPathParamRegex.FindAllStringSubmatch(chain, -1)
	for _, match := range paramMatches {
		if len(match) > 2 {
			paramName := match[2]
			pathParts = append(pathParts, "{"+paramName+"}")
		}
	}

	if len(pathParts) == 0 {
		return "/"
	}

	return "/" + strings.Join(pathParts, "/")
}

// IsSupported returns whether Scala parsing is supported.
func (p *ScalaParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *ScalaParser) SupportedExtensions() []string {
	return []string{".scala", ".sc"}
}

// ScalaTypeToOpenAPI converts a Scala type to an OpenAPI type.
func ScalaTypeToOpenAPI(scalaType string) (openAPIType string, format string) {
	// Trim whitespace
	scalaType = strings.TrimSpace(scalaType)

	// Handle Option types
	if strings.HasPrefix(scalaType, "Option[") {
		scalaType = extractScalaGenericType(scalaType)
	}

	// Handle Future types
	if strings.HasPrefix(scalaType, "Future[") {
		scalaType = extractScalaGenericType(scalaType)
	}

	// Handle collection types
	if strings.HasPrefix(scalaType, "List[") || strings.HasPrefix(scalaType, "Seq[") ||
		strings.HasPrefix(scalaType, "Vector[") || strings.HasPrefix(scalaType, "Array[") ||
		strings.HasPrefix(scalaType, "Set[") {
		return "array", ""
	}

	// Handle Map types
	if strings.HasPrefix(scalaType, "Map[") {
		return "object", ""
	}

	switch scalaType {
	case "String":
		return "string", ""
	case "Int", "Integer", "Long", "Short", "Byte":
		return "integer", ""
	case "Float", "Double", "BigDecimal":
		return "number", ""
	case "Boolean":
		return "boolean", ""
	case "java.time.LocalDateTime", "java.time.ZonedDateTime", "java.time.Instant":
		return "string", "date-time"
	case "java.time.LocalDate":
		return "string", "date"
	case "java.time.LocalTime":
		return "string", "time"
	case "java.util.UUID", "UUID":
		return "string", "uuid"
	case "Unit":
		return "", ""
	default:
		return "object", ""
	}
}

// extractScalaGenericType extracts the inner type from a generic like List[String].
func extractScalaGenericType(s string) string {
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}

// ConvertPlayPathParams converts Play path parameters to OpenAPI format.
// Play uses :param and $param<regex> formats.
func ConvertPlayPathParams(path string) string {
	// Convert :param to {param}
	colonParamRegex := regexp.MustCompile(`:(\w+)`)
	path = colonParamRegex.ReplaceAllString(path, "{$1}")

	// Convert $param<regex> to {param}
	dollarParamRegex := regexp.MustCompile(`\$(\w+)<[^>]+>`)
	path = dollarParamRegex.ReplaceAllString(path, "{$1}")

	// Convert $param to {param}
	dollarSimpleRegex := regexp.MustCompile(`\$(\w+)`)
	path = dollarSimpleRegex.ReplaceAllString(path, "{$1}")

	return path
}

// ExtractPlayPathParams extracts parameter names from a Play path.
func ExtractPlayPathParams(path string) []string {
	var params []string

	// Match :param style
	colonRegex := regexp.MustCompile(`:(\w+)`)
	for _, match := range colonRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	// Match $param<regex> and $param styles
	dollarRegex := regexp.MustCompile(`\$(\w+)`)
	for _, match := range dollarRegex.FindAllStringSubmatch(path, -1) {
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
