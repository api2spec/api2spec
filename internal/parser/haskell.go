// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// HaskellParser provides Haskell parsing capabilities using regex patterns.
type HaskellParser struct{}

// NewHaskellParser creates a new Haskell parser.
func NewHaskellParser() *HaskellParser {
	return &HaskellParser{}
}

// HaskellDataType represents a Haskell data type definition.
type HaskellDataType struct {
	// Name is the data type name
	Name string

	// Fields are the record fields (if it's a record type)
	Fields []HaskellField

	// Line is the source line number
	Line int
}

// HaskellField represents a record field.
type HaskellField struct {
	// Name is the field name
	Name string

	// Type is the field type
	Type string

	// IsOptional indicates if the field is Maybe T
	IsOptional bool

	// Line is the source line number
	Line int
}

// ServantEndpoint represents a Servant API endpoint definition.
type ServantEndpoint struct {
	// Method is the HTTP method
	Method string

	// Path is the route path
	Path string

	// QueryParams are query parameters
	QueryParams []HaskellQueryParam

	// CaptureParams are path capture parameters
	CaptureParams []HaskellCaptureParam

	// RequestBody is the request body type if any
	RequestBody string

	// ResponseType is the response type
	ResponseType string

	// ContentType is the content type (e.g., JSON, PlainText)
	ContentType string

	// Line is the source line number
	Line int
}

// HaskellQueryParam represents a Servant QueryParam.
type HaskellQueryParam struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string

	// Required indicates if the param is required
	Required bool
}

// HaskellCaptureParam represents a Servant Capture parameter.
type HaskellCaptureParam struct {
	// Name is the parameter name
	Name string

	// Type is the parameter type
	Type string
}

// ParsedHaskellFile represents a parsed Haskell source file.
type ParsedHaskellFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// ModuleName is the module name
	ModuleName string

	// Imports are the import statements
	Imports []string

	// DataTypes are the extracted data type definitions
	DataTypes []HaskellDataType

	// ServantEndpoints are the extracted Servant API endpoints
	ServantEndpoints []ServantEndpoint

	// TypeAliases are type alias definitions (for combined APIs)
	TypeAliases []HaskellTypeAlias
}

// HaskellTypeAlias represents a type alias.
type HaskellTypeAlias struct {
	// Name is the type alias name
	Name string

	// Definition is the type definition
	Definition string

	// Line is the source line number
	Line int
}

// Regex patterns for Haskell parsing
var (
	// Matches module declaration
	haskellModuleRegex = regexp.MustCompile(`(?m)^module\s+([A-Z][\w.]*)\s+`)

	// Matches import statements
	haskellImportRegex = regexp.MustCompile(`(?m)^import\s+(?:qualified\s+)?([A-Z][\w.]*)`)

	// Matches data type definitions
	haskellDataRegex = regexp.MustCompile(`(?m)^data\s+(\w+)(?:\s+\w+)*\s*=\s*(\w+)\s*\{([^}]+)\}`)

	// Matches newtype definitions
	haskellNewtypeRegex = regexp.MustCompile(`(?m)^newtype\s+(\w+)(?:\s+\w+)*\s*=\s*(\w+)\s*\{([^}]+)\}`)

	// Matches type alias definitions (type API = ...)
	haskellTypeAliasRegex = regexp.MustCompile(`(?m)^type\s+(\w+)\s*=\s*(.+?)(?:\n\s{0,2}\S|\z)`)

	// Matches Servant API type definitions
	// type API = "users" :> Get '[JSON] [User]
	servantAPITypeRegex = regexp.MustCompile(`(?m)^type\s+(\w+)\s*=\s*(.+)`)

	// Matches Servant HTTP method combinators
	servantMethodRegex = regexp.MustCompile(`(Get|Post|Put|Delete|Patch|Head|Options)\s*'\[\s*(\w+)\s*\]`)

	// Matches Servant path segment: "users" :>
	servantPathSegmentRegex = regexp.MustCompile(`"([^"]+)"\s*:>`)

	// Matches Servant Capture: Capture "id" Int :>
	servantCaptureRegex = regexp.MustCompile(`Capture\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant QueryParam: QueryParam "name" Text :>
	servantQueryParamRegex = regexp.MustCompile(`QueryParam\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant QueryParam': QueryParam' '[Required] "name" Text :>
	servantQueryParamModRegex = regexp.MustCompile(`QueryParam'\s*'\[([^\]]+)\]\s+"([^"]+)"\s+(\w+)\s*:>`)

	// Matches Servant ReqBody: ReqBody '[JSON] CreateUser :>
	servantReqBodyRegex = regexp.MustCompile(`ReqBody\s*'\[\s*(\w+)\s*\]\s+(\w+)\s*:>`)

	// Matches Servant combined APIs: UserAPI :<|> ProductAPI
	servantCombinedAPIRegex = regexp.MustCompile(`(\w+)\s*:<\|>\s*(\w+)`)

	// Matches record field: fieldName :: Type
	haskellRecordFieldRegex = regexp.MustCompile(`(\w+)\s*::\s*([^,}]+)`)
)

// Parse parses Haskell source code.
func (p *HaskellParser) Parse(filename string, content []byte) *ParsedHaskellFile {
	src := string(content)
	pf := &ParsedHaskellFile{
		Path:             filename,
		Content:          src,
		Imports:          []string{},
		DataTypes:        []HaskellDataType{},
		ServantEndpoints: []ServantEndpoint{},
		TypeAliases:      []HaskellTypeAlias{},
	}

	// Extract module name
	if match := haskellModuleRegex.FindStringSubmatch(src); len(match) > 1 {
		pf.ModuleName = match[1]
	}

	// Extract imports
	pf.Imports = p.extractImports(src)

	// Extract data types
	pf.DataTypes = p.extractDataTypes(src)

	// Extract type aliases
	pf.TypeAliases = p.extractTypeAliases(src)

	// Extract Servant endpoints
	pf.ServantEndpoints = p.extractServantEndpoints(src)

	return pf
}

// extractImports extracts import statements from Haskell source.
func (p *HaskellParser) extractImports(src string) []string {
	var imports []string
	matches := haskellImportRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, strings.TrimSpace(match[1]))
		}
	}
	return imports
}

// extractDataTypes extracts data type definitions from Haskell source.
func (p *HaskellParser) extractDataTypes(src string) []HaskellDataType {
	var types []HaskellDataType

	// Extract data types
	matches := haskellDataRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		dt := HaskellDataType{
			Line:   line,
			Fields: []HaskellField{},
		}

		// Extract type name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			dt.Name = src[match[2]:match[3]]
		}

		// Extract record fields (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			fieldsStr := src[match[6]:match[7]]
			dt.Fields = p.extractRecordFields(fieldsStr, line)
		}

		if dt.Name != "" {
			types = append(types, dt)
		}
	}

	// Extract newtypes with records
	newtypeMatches := haskellNewtypeRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range newtypeMatches {
		if len(match) < 8 {
			continue
		}

		line := countLines(src[:match[0]])

		dt := HaskellDataType{
			Line:   line,
			Fields: []HaskellField{},
		}

		// Extract type name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			dt.Name = src[match[2]:match[3]]
		}

		// Extract record fields (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			fieldsStr := src[match[6]:match[7]]
			dt.Fields = p.extractRecordFields(fieldsStr, line)
		}

		if dt.Name != "" {
			types = append(types, dt)
		}
	}

	return types
}

// extractRecordFields extracts fields from a Haskell record definition.
func (p *HaskellParser) extractRecordFields(fieldsStr string, baseLine int) []HaskellField {
	var fields []HaskellField

	matches := haskellRecordFieldRegex.FindAllStringSubmatch(fieldsStr, -1)
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		field := HaskellField{
			Line: baseLine,
		}

		field.Name = strings.TrimSpace(match[1])
		typeStr := strings.TrimSpace(match[2])
		field.Type = typeStr

		// Check if optional (Maybe T)
		if strings.HasPrefix(typeStr, "Maybe ") {
			field.IsOptional = true
			field.Type = strings.TrimPrefix(typeStr, "Maybe ")
		}

		if field.Name != "" {
			fields = append(fields, field)
		}
	}

	return fields
}

// extractTypeAliases extracts type alias definitions.
func (p *HaskellParser) extractTypeAliases(src string) []HaskellTypeAlias {
	var aliases []HaskellTypeAlias

	matches := servantAPITypeRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		alias := HaskellTypeAlias{
			Line: line,
		}

		// Extract alias name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			alias.Name = src[match[2]:match[3]]
		}

		// Extract definition (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			// Get the rest of the line and any continuation lines
			def := src[match[4]:match[5]]
			// Look for continuation lines (indented lines)
			endIdx := match[5]
			for endIdx < len(src) {
				nextNewline := strings.Index(src[endIdx:], "\n")
				if nextNewline == -1 {
					break
				}
				nextLineStart := endIdx + nextNewline + 1
				if nextLineStart >= len(src) {
					break
				}
				// Check if next line is a continuation (starts with whitespace)
				if src[nextLineStart] == ' ' || src[nextLineStart] == '\t' {
					// Find end of next line
					nextLineEnd := strings.Index(src[nextLineStart:], "\n")
					if nextLineEnd == -1 {
						def += " " + strings.TrimSpace(src[nextLineStart:])
						break
					}
					def += " " + strings.TrimSpace(src[nextLineStart:nextLineStart+nextLineEnd])
					endIdx = nextLineStart + nextLineEnd
				} else {
					break
				}
			}
			alias.Definition = strings.TrimSpace(def)
		}

		if alias.Name != "" {
			aliases = append(aliases, alias)
		}
	}

	return aliases
}

// extractServantEndpoints extracts Servant API endpoint definitions.
func (p *HaskellParser) extractServantEndpoints(src string) []ServantEndpoint {
	var endpoints []ServantEndpoint

	// Find all type aliases that look like Servant APIs
	for _, alias := range p.extractTypeAliases(src) {
		def := alias.Definition

		// Check if this is a combined API (using :<|>)
		if strings.Contains(def, ":<|>") {
			// Skip combined APIs, they reference other APIs
			continue
		}

		// Check if this looks like a Servant endpoint
		if !servantMethodRegex.MatchString(def) {
			continue
		}

		endpoint := ServantEndpoint{
			Line: alias.Line,
		}

		// Extract path segments
		var pathParts []string
		segmentMatches := servantPathSegmentRegex.FindAllStringSubmatch(def, -1)
		for _, segMatch := range segmentMatches {
			if len(segMatch) > 1 {
				pathParts = append(pathParts, segMatch[1])
			}
		}

		// Extract capture parameters (path params)
		captureMatches := servantCaptureRegex.FindAllStringSubmatch(def, -1)
		for _, capMatch := range captureMatches {
			if len(capMatch) > 2 {
				endpoint.CaptureParams = append(endpoint.CaptureParams, HaskellCaptureParam{
					Name: capMatch[1],
					Type: capMatch[2],
				})
				// Add to path
				pathParts = append(pathParts, "{"+capMatch[1]+"}")
			}
		}

		// Extract query parameters
		queryMatches := servantQueryParamRegex.FindAllStringSubmatch(def, -1)
		for _, qMatch := range queryMatches {
			if len(qMatch) > 2 {
				endpoint.QueryParams = append(endpoint.QueryParams, HaskellQueryParam{
					Name:     qMatch[1],
					Type:     qMatch[2],
					Required: false, // QueryParam is optional by default
				})
			}
		}

		// Extract required query parameters (QueryParam')
		queryModMatches := servantQueryParamModRegex.FindAllStringSubmatch(def, -1)
		for _, qMatch := range queryModMatches {
			if len(qMatch) > 3 {
				required := strings.Contains(qMatch[1], "Required")
				endpoint.QueryParams = append(endpoint.QueryParams, HaskellQueryParam{
					Name:     qMatch[2],
					Type:     qMatch[3],
					Required: required,
				})
			}
		}

		// Extract request body
		reqBodyMatch := servantReqBodyRegex.FindStringSubmatch(def)
		if len(reqBodyMatch) > 2 {
			endpoint.RequestBody = reqBodyMatch[2]
		}

		// Extract HTTP method and response type
		methodMatch := servantMethodRegex.FindStringSubmatch(def)
		if len(methodMatch) > 2 {
			endpoint.Method = strings.ToUpper(methodMatch[1])
			endpoint.ContentType = methodMatch[2]
		}

		// Extract response type (after the method)
		endpoint.ResponseType = p.extractResponseType(def)

		// Build path
		if len(pathParts) == 0 {
			endpoint.Path = "/"
		} else {
			endpoint.Path = "/" + strings.Join(pathParts, "/")
		}

		if endpoint.Method != "" {
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

// extractResponseType extracts the response type from a Servant endpoint definition.
func (p *HaskellParser) extractResponseType(def string) string {
	// Look for the type after Get/Post/etc '[JSON] Type
	methodRegex := regexp.MustCompile(`(?:Get|Post|Put|Delete|Patch|Head|Options)\s*'\[\s*\w+\s*\]\s+(\[?\w+\]?)`)
	if match := methodRegex.FindStringSubmatch(def); len(match) > 1 {
		return match[1]
	}
	return ""
}

// IsSupported returns whether Haskell parsing is supported.
func (p *HaskellParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *HaskellParser) SupportedExtensions() []string {
	return []string{".hs", ".lhs"}
}

// HaskellTypeToOpenAPI converts a Haskell type to an OpenAPI type.
func HaskellTypeToOpenAPI(haskellType string) (openAPIType string, format string) {
	// Trim whitespace
	haskellType = strings.TrimSpace(haskellType)

	// Handle Maybe types
	if strings.HasPrefix(haskellType, "Maybe ") {
		haskellType = strings.TrimPrefix(haskellType, "Maybe ")
	}

	// Handle list types
	if strings.HasPrefix(haskellType, "[") && strings.HasSuffix(haskellType, "]") {
		return "array", ""
	}

	// Handle Map types
	if strings.HasPrefix(haskellType, "Map ") || strings.HasPrefix(haskellType, "HashMap ") {
		return "object", ""
	}

	switch haskellType {
	case "Text", "String", "ByteString", "LazyText":
		return "string", ""
	case "Int", "Int8", "Int16", "Int32", "Int64", "Integer":
		return "integer", ""
	case "Word", "Word8", "Word16", "Word32", "Word64":
		return "integer", ""
	case "Float":
		return "number", "float"
	case "Double":
		return "number", "double"
	case "Bool":
		return "boolean", ""
	case "UTCTime", "ZonedTime", "LocalTime":
		return "string", "date-time"
	case "Day":
		return "string", "date"
	case "UUID":
		return "string", "uuid"
	default:
		return "object", ""
	}
}

// ExtractServantPathParams extracts parameter names from a Servant path.
func ExtractServantPathParams(path string) []string {
	var params []string

	// Match {param} style
	braceRegex := regexp.MustCompile(`\{(\w+)\}`)
	for _, match := range braceRegex.FindAllStringSubmatch(path, -1) {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}
