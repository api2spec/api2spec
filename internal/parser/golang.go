// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// GoParser provides Go AST parsing capabilities.
type GoParser struct {
	fset *token.FileSet
}

// NewGoParser creates a new Go parser.
func NewGoParser() *GoParser {
	return &GoParser{
		fset: token.NewFileSet(),
	}
}

// ParsedFile represents a parsed Go source file.
type ParsedFile struct {
	// Path is the file path
	Path string

	// AST is the parsed AST
	AST *ast.File

	// FileSet is the token file set for position information
	FileSet *token.FileSet
}

// StructField represents a field in a Go struct.
type StructField struct {
	// Name is the field name in Go
	Name string

	// JSONName is the json tag name (or field name if no tag)
	JSONName string

	// Type is the Go type as a string
	Type string

	// TypeKind classifies the type (primitive, struct, slice, map, pointer)
	TypeKind TypeKind

	// Omitempty indicates if omitempty is set in json tag
	Omitempty bool

	// IsPointer indicates if the field is a pointer type
	IsPointer bool

	// IsRequired indicates if the field is required (from validate tag)
	IsRequired bool

	// ValidationTags contains parsed validation constraints
	ValidationTags map[string]string

	// ElementType is the element type for slices/maps
	ElementType string

	// KeyType is the key type for maps (usually "string")
	KeyType string

	// Description is from a doc comment
	Description string

	// NestedStruct contains nested struct fields if TypeKind is KindStruct
	NestedStruct []StructField

	// Position is the source location
	Position token.Position
}

// TypeKind classifies Go types for schema conversion.
type TypeKind int

const (
	KindPrimitive TypeKind = iota
	KindStruct
	KindSlice
	KindMap
	KindPointer
	KindInterface
	KindTime
	KindUnknown
)

// StructDefinition represents a parsed Go struct.
type StructDefinition struct {
	// Name is the struct name
	Name string

	// Fields are the struct fields
	Fields []StructField

	// Description is from the struct's doc comment
	Description string

	// Position is the source location
	Position token.Position

	// Embedded contains names of embedded types
	Embedded []string
}

// ParseSource parses Go source code from a string.
func (p *GoParser) ParseSource(filename, source string) (*ParsedFile, error) {
	file, err := parser.ParseFile(p.fset, filename, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go source: %w", err)
	}

	return &ParsedFile{
		Path:    filename,
		AST:     file,
		FileSet: p.fset,
	}, nil
}

// ParseFile parses a Go source file from disk.
func (p *GoParser) ParseFile(path string) (*ParsedFile, error) {
	file, err := parser.ParseFile(p.fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file %s: %w", path, err)
	}

	return &ParsedFile{
		Path:    path,
		AST:     file,
		FileSet: p.fset,
	}, nil
}

// ExtractStructs extracts all struct definitions from a parsed file.
func (p *GoParser) ExtractStructs(pf *ParsedFile) []StructDefinition {
	var structs []StructDefinition

	for _, decl := range pf.AST.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			def := p.parseStructType(typeSpec.Name.Name, structType, pf)

			// Get doc comment - check typeSpec first, then fall back to genDecl
			if typeSpec.Doc != nil {
				def.Description = strings.TrimSpace(typeSpec.Doc.Text())
			} else if genDecl.Doc != nil && len(genDecl.Specs) == 1 {
				// Use GenDecl doc only if there's a single spec (standalone type declaration)
				def.Description = strings.TrimSpace(genDecl.Doc.Text())
			}

			def.Position = p.fset.Position(typeSpec.Pos())
			structs = append(structs, def)
		}
	}

	return structs
}

// parseStructType parses a struct type into a StructDefinition.
func (p *GoParser) parseStructType(name string, st *ast.StructType, pf *ParsedFile) StructDefinition {
	def := StructDefinition{
		Name: name,
	}

	if st.Fields == nil {
		return def
	}

	for _, field := range st.Fields.List {
		// Handle embedded fields
		if len(field.Names) == 0 {
			typeName := p.typeToString(field.Type)
			def.Embedded = append(def.Embedded, typeName)
			continue
		}

		for _, fieldName := range field.Names {
			sf := p.parseField(fieldName.Name, field, pf)
			def.Fields = append(def.Fields, sf)
		}
	}

	return def
}

// parseField parses a struct field.
func (p *GoParser) parseField(name string, field *ast.Field, pf *ParsedFile) StructField {
	sf := StructField{
		Name:           name,
		JSONName:       name,
		ValidationTags: make(map[string]string),
	}

	// Parse type
	sf.Type = p.typeToString(field.Type)
	sf.TypeKind = p.classifyType(field.Type)
	sf.IsPointer = sf.TypeKind == KindPointer

	// Extract element type for slices and maps
	switch t := field.Type.(type) {
	case *ast.ArrayType:
		sf.ElementType = p.typeToString(t.Elt)
	case *ast.MapType:
		sf.KeyType = p.typeToString(t.Key)
		sf.ElementType = p.typeToString(t.Value)
	case *ast.StarExpr:
		// For pointers, get the underlying type info
		switch inner := t.X.(type) {
		case *ast.ArrayType:
			sf.ElementType = p.typeToString(inner.Elt)
		case *ast.MapType:
			sf.KeyType = p.typeToString(inner.Key)
			sf.ElementType = p.typeToString(inner.Value)
		}
	}

	// Parse inline struct
	if structType, ok := field.Type.(*ast.StructType); ok {
		nested := p.parseStructType("", structType, pf)
		sf.NestedStruct = nested.Fields
	}

	// Parse tags
	if field.Tag != nil {
		sf.parseTag(field.Tag.Value)
	}

	// Parse doc comment
	if field.Doc != nil {
		sf.Description = strings.TrimSpace(field.Doc.Text())
	} else if field.Comment != nil {
		sf.Description = strings.TrimSpace(field.Comment.Text())
	}

	sf.Position = p.fset.Position(field.Pos())

	return sf
}

// parseTag parses struct field tags.
func (sf *StructField) parseTag(tagValue string) {
	// Remove backticks
	tagValue = strings.Trim(tagValue, "`")

	// Use reflect.StructTag for proper parsing
	tag := reflect.StructTag(tagValue)

	// Parse json tag
	if jsonTag, ok := tag.Lookup("json"); ok {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			sf.JSONName = parts[0]
		}
		for _, part := range parts[1:] {
			if part == "omitempty" {
				sf.Omitempty = true
			}
		}
		// json:"-" means the field is not serialized
		if len(parts) > 0 && parts[0] == "-" {
			sf.JSONName = "-"
		}
	}

	// Parse validate tag
	if validateTag, ok := tag.Lookup("validate"); ok {
		sf.parseValidateTag(validateTag)
	}
}

// parseValidateTag parses the validate struct tag.
func (sf *StructField) parseValidateTag(tag string) {
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if part == "required" {
			sf.IsRequired = true
			continue
		}

		// Parse key=value format
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			value := part[idx+1:]
			sf.ValidationTags[key] = value
		} else {
			// Boolean tags like "email", "url", etc.
			sf.ValidationTags[part] = "true"
		}
	}
}

// typeToString converts an AST type expression to a string.
func (p *GoParser) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return p.typeToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + p.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + p.typeToString(t.Elt)
		}
		// Fixed-size array
		return fmt.Sprintf("[%s]%s", p.exprToString(t.Len), p.typeToString(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", p.typeToString(t.Key), p.typeToString(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.ChanType:
		return "chan"
	case *ast.FuncType:
		return "func"
	default:
		return "unknown"
	}
}

// exprToString converts an expression to a string (for array lengths).
func (p *GoParser) exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	default:
		return "..."
	}
}

// classifyType determines the TypeKind for a type expression.
func (p *GoParser) classifyType(expr ast.Expr) TypeKind {
	switch t := expr.(type) {
	case *ast.Ident:
		return p.classifyTypeName(t.Name)
	case *ast.SelectorExpr:
		// Check for time.Time
		if pkg, ok := t.X.(*ast.Ident); ok {
			if pkg.Name == "time" && t.Sel.Name == "Time" {
				return KindTime
			}
		}
		return KindUnknown
	case *ast.StarExpr:
		return KindPointer
	case *ast.ArrayType:
		return KindSlice
	case *ast.MapType:
		return KindMap
	case *ast.InterfaceType:
		return KindInterface
	case *ast.StructType:
		return KindStruct
	default:
		return KindUnknown
	}
}

// classifyTypeName classifies a type by its name.
func (p *GoParser) classifyTypeName(name string) TypeKind {
	switch name {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune":
		return KindPrimitive
	default:
		// Assume it's a struct type (could be a type alias)
		return KindStruct
	}
}

// FindImports returns all imports in the file.
func (p *GoParser) FindImports(pf *ParsedFile) map[string]string {
	imports := make(map[string]string)

	for _, imp := range pf.AST.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := ""

		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			// Use last component of path as default name
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}

		imports[name] = path
	}

	return imports
}

// HasImport checks if a file imports a specific package.
func (p *GoParser) HasImport(pf *ParsedFile, packagePath string) bool {
	for _, imp := range pf.AST.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if path == packagePath {
			return true
		}
	}
	return false
}

// FunctionCall represents a function call extracted from the AST.
type FunctionCall struct {
	// Receiver is the receiver expression (e.g., "r" in "r.Get(...)")
	Receiver string

	// Method is the method name (e.g., "Get")
	Method string

	// Arguments are the call arguments
	Arguments []string

	// Position is the source location
	Position token.Position
}

// FindMethodCalls finds all method calls matching a pattern.
func (p *GoParser) FindMethodCalls(pf *ParsedFile, receiverPattern, methodPattern string) []FunctionCall {
	var calls []FunctionCall

	recvRe := regexp.MustCompile(receiverPattern)
	methodRe := regexp.MustCompile(methodPattern)

	ast.Inspect(pf.AST, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		receiver := p.exprToIdent(selExpr.X)
		method := selExpr.Sel.Name

		if !recvRe.MatchString(receiver) || !methodRe.MatchString(method) {
			return true
		}

		call := FunctionCall{
			Receiver: receiver,
			Method:   method,
			Position: p.fset.Position(callExpr.Pos()),
		}

		for _, arg := range callExpr.Args {
			call.Arguments = append(call.Arguments, p.argToString(arg))
		}

		calls = append(calls, call)
		return true
	})

	return calls
}

// exprToIdent tries to get an identifier name from an expression.
func (p *GoParser) exprToIdent(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return p.exprToIdent(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}

// argToString converts a call argument to a string.
func (p *GoParser) argToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return strings.Trim(e.Value, `"`)
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return p.exprToIdent(e.X) + "." + e.Sel.Name
	case *ast.FuncLit:
		return "<func>"
	default:
		return "<expr>"
	}
}

// ExtractStringLiteral extracts the string value from a BasicLit if it's a string.
func ExtractStringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	// Unquote the string
	s, err := strconv.Unquote(lit.Value)
	if err != nil {
		return strings.Trim(lit.Value, `"`), true
	}
	return s, true
}

// --- Doc Comment Annotation Parsing ---

// DocAnnotations contains parsed annotations from doc comments.
// Supports common swagger/openapi annotations like @summary, @description, etc.
type DocAnnotations struct {
	// Summary is a brief description of the operation (from @summary)
	Summary string

	// Description is a detailed description (from @description or first line of comment)
	Description string

	// Tags are tags for grouping (from @tags)
	Tags []string

	// Deprecated indicates if the operation is deprecated (from @deprecated)
	Deprecated bool

	// OperationID is a unique identifier for the operation (from @operationId)
	OperationID string

	// Parameters are parameter definitions (from @param)
	Parameters []ParamAnnotation

	// Responses are response definitions (from @success, @failure, @response)
	Responses []ResponseAnnotation

	// Security are security requirements (from @security)
	Security []string

	// Accept is the accepted content type (from @accept)
	Accept string

	// Produce is the produced content type (from @produce)
	Produce string

	// Router is an explicit route definition (from @router)
	Router *RouterAnnotation
}

// ParamAnnotation represents a parsed @param annotation.
type ParamAnnotation struct {
	// Name is the parameter name
	Name string

	// In is the location (path, query, header, body, formData)
	In string

	// Type is the data type
	Type string

	// Required indicates if the parameter is required
	Required bool

	// Description is the parameter description
	Description string
}

// ResponseAnnotation represents a parsed @success/@failure annotation.
type ResponseAnnotation struct {
	// Code is the HTTP status code
	Code string

	// Type is the response data type
	Type string

	// Description is the response description
	Description string
}

// RouterAnnotation represents a parsed @router annotation.
type RouterAnnotation struct {
	// Path is the route path
	Path string

	// Method is the HTTP method
	Method string
}

// ParseDocComment parses a doc comment string and extracts annotations.
func ParseDocComment(comment string) *DocAnnotations {
	if comment == "" {
		return &DocAnnotations{}
	}

	annotations := &DocAnnotations{}
	lines := strings.Split(comment, "\n")

	var descriptionLines []string
	foundAnnotation := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for annotations (starting with @)
		if strings.HasPrefix(line, "@") {
			foundAnnotation = true
			parseAnnotationLine(line, annotations)
		} else if !foundAnnotation {
			// Lines before first annotation become description
			descriptionLines = append(descriptionLines, line)
		}
	}

	// Set description from non-annotation lines if no @description was found
	if annotations.Description == "" && len(descriptionLines) > 0 {
		// Use first line as summary if no @summary
		if annotations.Summary == "" && len(descriptionLines) > 0 {
			annotations.Summary = descriptionLines[0]
		}
		annotations.Description = strings.Join(descriptionLines, " ")
	}

	return annotations
}

// parseAnnotationLine parses a single annotation line.
func parseAnnotationLine(line string, annotations *DocAnnotations) {
	// Remove the @ prefix
	line = strings.TrimPrefix(line, "@")

	// Split into annotation name and value
	parts := strings.SplitN(line, " ", 2)
	name := strings.ToLower(parts[0])
	value := ""
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}

	switch name {
	case "summary":
		annotations.Summary = value

	case "description", "desc":
		annotations.Description = value

	case "tags", "tag":
		// Tags can be comma-separated
		tags := strings.Split(value, ",")
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				annotations.Tags = append(annotations.Tags, tag)
			}
		}

	case "deprecated":
		annotations.Deprecated = true

	case "operationid", "id":
		annotations.OperationID = value

	case "param":
		param := parseParamAnnotation(value)
		if param != nil {
			annotations.Parameters = append(annotations.Parameters, *param)
		}

	case "success", "failure", "response":
		resp := parseResponseAnnotation(value)
		if resp != nil {
			annotations.Responses = append(annotations.Responses, *resp)
		}

	case "security":
		if value != "" {
			annotations.Security = append(annotations.Security, value)
		}

	case "accept":
		annotations.Accept = value

	case "produce":
		annotations.Produce = value

	case "router":
		router := parseRouterAnnotation(value)
		if router != nil {
			annotations.Router = router
		}
	}
}

// parseParamAnnotation parses @param annotation value.
// Format: name location type required "description"
// Example: @param id path string true "User ID"
func parseParamAnnotation(value string) *ParamAnnotation {
	// Match: name location type required "description"
	// or: name location type required description (without quotes)
	parts := splitAnnotationParts(value)
	if len(parts) < 4 {
		return nil
	}

	param := &ParamAnnotation{
		Name: parts[0],
		In:   parts[1],
		Type: parts[2],
	}

	// Parse required (true/false)
	param.Required = strings.ToLower(parts[3]) == "true"

	// Remaining parts are description
	if len(parts) > 4 {
		param.Description = strings.Join(parts[4:], " ")
		param.Description = strings.Trim(param.Description, `"`)
	}

	return param
}

// parseResponseAnnotation parses @success/@failure annotation value.
// Format: code {type} description
// Example: @success 200 {object} User "Successful response"
func parseResponseAnnotation(value string) *ResponseAnnotation {
	parts := splitAnnotationParts(value)
	if len(parts) < 1 {
		return nil
	}

	resp := &ResponseAnnotation{
		Code: parts[0],
	}

	// Parse type if present (in braces)
	if len(parts) > 1 {
		typePart := parts[1]
		if strings.HasPrefix(typePart, "{") && strings.HasSuffix(typePart, "}") {
			resp.Type = strings.Trim(typePart, "{}")
			// Description is everything after
			if len(parts) > 2 {
				resp.Description = strings.Join(parts[2:], " ")
				resp.Description = strings.Trim(resp.Description, `"`)
			}
		} else {
			// No type, rest is description
			resp.Description = strings.Join(parts[1:], " ")
			resp.Description = strings.Trim(resp.Description, `"`)
		}
	}

	return resp
}

// parseRouterAnnotation parses @router annotation value.
// Format: path [method]
// Example: @router /users/{id} [get]
func parseRouterAnnotation(value string) *RouterAnnotation {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return nil
	}

	router := &RouterAnnotation{
		Path: parts[0],
	}

	// Extract method from brackets
	method := parts[1]
	if strings.HasPrefix(method, "[") && strings.HasSuffix(method, "]") {
		router.Method = strings.ToUpper(strings.Trim(method, "[]"))
	} else {
		router.Method = strings.ToUpper(method)
	}

	return router
}

// splitAnnotationParts splits an annotation value respecting quoted strings.
func splitAnnotationParts(value string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false

	for _, r := range value {
		switch {
		case r == '"':
			inQuotes = !inQuotes
			current.WriteRune(r)
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// FindHandlerDoc finds the doc comment for a handler function by name.
func (p *GoParser) FindHandlerDoc(pf *ParsedFile, handlerName string) *DocAnnotations {
	// Handle package.FuncName format
	parts := strings.Split(handlerName, ".")
	funcName := parts[len(parts)-1]

	for _, decl := range pf.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name == funcName {
			if funcDecl.Doc != nil {
				return ParseDocComment(funcDecl.Doc.Text())
			}
			return &DocAnnotations{}
		}
	}

	return nil
}

// FindMethodDoc finds the doc comment for a method by receiver type and method name.
func (p *GoParser) FindMethodDoc(pf *ParsedFile, receiverType, methodName string) *DocAnnotations {
	for _, decl := range pf.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Name.Name != methodName {
			continue
		}

		// Check receiver
		if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
			continue
		}

		recv := funcDecl.Recv.List[0]
		recvTypeName := p.typeToString(recv.Type)

		// Handle pointer receivers
		recvTypeName = strings.TrimPrefix(recvTypeName, "*")

		if recvTypeName == receiverType {
			if funcDecl.Doc != nil {
				return ParseDocComment(funcDecl.Doc.Text())
			}
			return &DocAnnotations{}
		}
	}

	return nil
}

// ExtractAllHandlerDocs extracts doc annotations for all functions in a file.
func (p *GoParser) ExtractAllHandlerDocs(pf *ParsedFile) map[string]*DocAnnotations {
	result := make(map[string]*DocAnnotations)

	for _, decl := range pf.AST.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if funcDecl.Doc == nil {
			continue
		}

		annotations := ParseDocComment(funcDecl.Doc.Text())

		// Build function name (with receiver if method)
		name := funcDecl.Name.Name
		if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
			recv := funcDecl.Recv.List[0]
			recvType := p.typeToString(recv.Type)
			recvType = strings.TrimPrefix(recvType, "*")
			name = recvType + "." + name
		}

		result[name] = annotations
	}

	return result
}
