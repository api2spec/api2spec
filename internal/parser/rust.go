// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

// RustParser provides Rust AST parsing capabilities using tree-sitter.
type RustParser struct {
	parser *sitter.Parser
}

// NewRustParser creates a new Rust parser.
func NewRustParser() *RustParser {
	parser := sitter.NewParser()
	parser.SetLanguage(rust.GetLanguage())
	return &RustParser{
		parser: parser,
	}
}

// ParsedRustFile represents a parsed Rust source file.
type ParsedRustFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content []byte

	// Tree is the tree-sitter parse tree
	Tree *sitter.Tree

	// RootNode is the root node of the AST
	RootNode *sitter.Node

	// Functions contains extracted function definitions
	Functions []RustFunction

	// Structs contains extracted struct definitions
	Structs []RustStruct

	// ImplBlocks contains extracted impl blocks
	ImplBlocks []RustImplBlock

	// MacroInvocations contains extracted macro invocations
	MacroInvocations []RustMacroInvocation

	// Uses contains use statements
	Uses []RustUse
}

// RustFunction represents a function definition.
type RustFunction struct {
	// Name is the function name
	Name string

	// Attributes are the function attributes
	Attributes []RustAttribute

	// Parameters are the function parameters
	Parameters []RustParameter

	// ReturnType is the return type if present
	ReturnType string

	// IsAsync indicates if the function is async
	IsAsync bool

	// IsPublic indicates if the function is public
	IsPublic bool

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// RustAttribute represents an attribute on a function, struct, or impl.
type RustAttribute struct {
	// Name is the attribute name (e.g., "get", "route", "derive")
	Name string

	// Arguments are the attribute arguments
	Arguments []string

	// Raw is the raw attribute text
	Raw string

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// RustParameter represents a function parameter.
type RustParameter struct {
	// Name is the parameter name
	Name string

	// Type is the type annotation
	Type string

	// IsMutable indicates if the parameter is mutable
	IsMutable bool

	// IsSelf indicates if this is a self parameter
	IsSelf bool
}

// RustStruct represents a struct definition.
type RustStruct struct {
	// Name is the struct name
	Name string

	// Attributes are the struct attributes
	Attributes []RustAttribute

	// Fields are the struct fields
	Fields []RustField

	// IsPublic indicates if the struct is public
	IsPublic bool

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// RustField represents a field in a struct.
type RustField struct {
	// Name is the field name
	Name string

	// Type is the type annotation
	Type string

	// Attributes are field attributes (like #[serde(rename = "...")])
	Attributes []RustAttribute

	// IsPublic indicates if the field is public
	IsPublic bool
}

// RustImplBlock represents an impl block.
type RustImplBlock struct {
	// TypeName is the type being implemented
	TypeName string

	// TraitName is the trait being implemented (empty if inherent impl)
	TraitName string

	// Methods are the methods in this impl block
	Methods []RustFunction

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// RustMacroInvocation represents a macro invocation.
type RustMacroInvocation struct {
	// Name is the macro name
	Name string

	// Arguments is the raw macro arguments
	Arguments string

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// RustUse represents a use statement.
type RustUse struct {
	// Path is the full use path
	Path string

	// Alias is the alias if present
	Alias string

	// Line is the source line number
	Line int
}

// ParseSource parses Rust source code from a string.
func (p *RustParser) ParseSource(filename string, source string) (*ParsedRustFile, error) {
	return p.Parse(filename, []byte(source))
}

// Parse parses Rust source code from bytes.
func (p *RustParser) Parse(filename string, content []byte) (*ParsedRustFile, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Rust: %w", err)
	}

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("failed to get root node")
	}

	pf := &ParsedRustFile{
		Path:             filename,
		Content:          content,
		Tree:             tree,
		RootNode:         rootNode,
		Functions:        []RustFunction{},
		Structs:          []RustStruct{},
		ImplBlocks:       []RustImplBlock{},
		MacroInvocations: []RustMacroInvocation{},
		Uses:             []RustUse{},
	}

	// Extract definitions
	pf.Uses = p.ExtractUses(rootNode, content)
	pf.Functions = p.ExtractFunctions(rootNode, content)
	pf.Structs = p.ExtractStructs(rootNode, content)
	pf.ImplBlocks = p.ExtractImplBlocks(rootNode, content)
	pf.MacroInvocations = p.ExtractMacroInvocations(rootNode, content)

	return pf, nil
}

// ParseFile parses a Rust source file from disk.
func (p *RustParser) ParseFile(path string) (*ParsedRustFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return p.Parse(path, content)
}

// ExtractUses extracts all use statements from the AST.
func (p *RustParser) ExtractUses(rootNode *sitter.Node, content []byte) []RustUse {
	var uses []RustUse

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "use_declaration" {
			use := p.parseUseDeclaration(node, content)
			if use != nil {
				uses = append(uses, *use)
			}
		}
		return true
	})

	return uses
}

// parseUseDeclaration parses a use declaration.
func (p *RustParser) parseUseDeclaration(node *sitter.Node, content []byte) *RustUse {
	use := &RustUse{
		Line: int(node.StartPoint().Row) + 1,
	}

	// Get the full use path
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "use_wildcard", "use_list", "scoped_identifier", "identifier", "scoped_use_list":
			use.Path = child.Content(content)
		case "use_as_clause":
			// Handle use X as Y
			for j := 0; j < int(child.ChildCount()); j++ {
				subChild := child.Child(j)
				if subChild.Type() == "identifier" {
					if use.Path == "" {
						use.Path = subChild.Content(content)
					} else {
						use.Alias = subChild.Content(content)
					}
				}
			}
		}
	}

	return use
}

// ExtractFunctions extracts all function definitions from the AST.
func (p *RustParser) ExtractFunctions(rootNode *sitter.Node, content []byte) []RustFunction {
	var functions []RustFunction

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "function_item" {
			fn := p.parseFunction(node, content)
			if fn != nil {
				functions = append(functions, *fn)
			}
			return false // Don't recurse into function items
		}
		return true
	})

	return functions
}

// parseFunction parses a function definition.
func (p *RustParser) parseFunction(node *sitter.Node, content []byte) *RustFunction {
	fn := &RustFunction{
		Line:       int(node.StartPoint().Row) + 1,
		Attributes: []RustAttribute{},
		Parameters: []RustParameter{},
		Node:       node,
	}

	// Look for attributes on the previous sibling
	if prevSibling := node.PrevSibling(); prevSibling != nil {
		attrs := p.collectAttributes(prevSibling, content)
		fn.Attributes = append(fn.Attributes, attrs...)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "visibility_modifier":
			if child.Content(content) == "pub" {
				fn.IsPublic = true
			}
		case "function_modifiers":
			modText := child.Content(content)
			if strings.Contains(modText, "async") {
				fn.IsAsync = true
			}
		case "identifier":
			if fn.Name == "" {
				fn.Name = child.Content(content)
			}
		case "parameters":
			fn.Parameters = p.parseParameters(child, content)
		case "type_identifier", "generic_type", "reference_type", "scoped_type_identifier":
			fn.ReturnType = child.Content(content)
		}
	}

	if fn.Name == "" {
		return nil
	}

	return fn
}

// collectAttributes collects attribute nodes before a definition.
func (p *RustParser) collectAttributes(node *sitter.Node, content []byte) []RustAttribute {
	var attrs []RustAttribute

	current := node
	for current != nil && current.Type() == "attribute_item" {
		attr := p.parseAttribute(current, content)
		if attr != nil {
			attrs = append([]RustAttribute{*attr}, attrs...)
		}
		current = current.PrevSibling()
	}

	return attrs
}

// parseAttribute parses an attribute node.
func (p *RustParser) parseAttribute(node *sitter.Node, content []byte) *RustAttribute {
	attr := &RustAttribute{
		Line:      int(node.StartPoint().Row) + 1,
		Arguments: []string{},
		Raw:       node.Content(content),
		Node:      node,
	}

	p.walkNodes(node, func(n *sitter.Node) bool {
		switch n.Type() {
		case "attribute":
			p.parseAttributeInner(n, content, attr)
		}
		return true
	})

	return attr
}

// parseAttributeInner parses the inner content of an attribute.
func (p *RustParser) parseAttributeInner(node *sitter.Node, content []byte, attr *RustAttribute) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier", "scoped_identifier":
			if attr.Name == "" {
				attr.Name = child.Content(content)
			}
		case "token_tree":
			// Arguments in parentheses
			args := child.Content(content)
			// Remove outer parentheses
			args = strings.TrimPrefix(args, "(")
			args = strings.TrimSuffix(args, ")")
			if args != "" {
				attr.Arguments = append(attr.Arguments, args)
			}
		case "meta_arguments":
			// Handle #[attr(args)]
			argsText := child.Content(content)
			attr.Arguments = append(attr.Arguments, argsText)
		}
	}
}

// parseParameters parses function parameters.
func (p *RustParser) parseParameters(node *sitter.Node, content []byte) []RustParameter {
	var params []RustParameter

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "parameter":
			param := p.parseParameter(child, content)
			if param != nil {
				params = append(params, *param)
			}
		case "self_parameter":
			params = append(params, RustParameter{
				Name:   "self",
				IsSelf: true,
			})
		}
	}

	return params
}

// parseParameter parses a single function parameter.
func (p *RustParser) parseParameter(node *sitter.Node, content []byte) *RustParameter {
	param := &RustParameter{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if param.Name == "" {
				param.Name = child.Content(content)
			}
		case "mutable_specifier":
			param.IsMutable = true
		default:
			// Type nodes
			if strings.Contains(child.Type(), "type") || child.Type() == "generic_type" {
				param.Type = child.Content(content)
			}
		}
	}

	if param.Name == "" {
		return nil
	}

	return param
}

// ExtractStructs extracts all struct definitions from the AST.
func (p *RustParser) ExtractStructs(rootNode *sitter.Node, content []byte) []RustStruct {
	var structs []RustStruct

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "struct_item" {
			s := p.parseStruct(node, content)
			if s != nil {
				structs = append(structs, *s)
			}
			return false
		}
		return true
	})

	return structs
}

// parseStruct parses a struct definition.
func (p *RustParser) parseStruct(node *sitter.Node, content []byte) *RustStruct {
	s := &RustStruct{
		Line:       int(node.StartPoint().Row) + 1,
		Attributes: []RustAttribute{},
		Fields:     []RustField{},
		Node:       node,
	}

	// Look for attributes on the previous sibling
	if prevSibling := node.PrevSibling(); prevSibling != nil {
		attrs := p.collectAttributes(prevSibling, content)
		s.Attributes = append(s.Attributes, attrs...)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "visibility_modifier":
			if child.Content(content) == "pub" {
				s.IsPublic = true
			}
		case "type_identifier":
			if s.Name == "" {
				s.Name = child.Content(content)
			}
		case "field_declaration_list":
			s.Fields = p.parseFields(child, content)
		}
	}

	if s.Name == "" {
		return nil
	}

	return s
}

// parseFields parses struct fields.
func (p *RustParser) parseFields(node *sitter.Node, content []byte) []RustField {
	var fields []RustField

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "field_declaration" {
			field := p.parseField(child, content)
			if field != nil {
				fields = append(fields, *field)
			}
		}
	}

	return fields
}

// parseField parses a single struct field.
func (p *RustParser) parseField(node *sitter.Node, content []byte) *RustField {
	field := &RustField{
		Attributes: []RustAttribute{},
	}

	// Look for attributes on previous sibling
	if prevSibling := node.PrevSibling(); prevSibling != nil {
		attrs := p.collectAttributes(prevSibling, content)
		field.Attributes = append(field.Attributes, attrs...)
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "visibility_modifier":
			if child.Content(content) == "pub" {
				field.IsPublic = true
			}
		case "field_identifier":
			field.Name = child.Content(content)
		default:
			// Type nodes
			if strings.Contains(child.Type(), "type") || child.Type() == "generic_type" {
				field.Type = child.Content(content)
			}
		}
	}

	if field.Name == "" {
		return nil
	}

	return field
}

// ExtractImplBlocks extracts all impl blocks from the AST.
func (p *RustParser) ExtractImplBlocks(rootNode *sitter.Node, content []byte) []RustImplBlock {
	var implBlocks []RustImplBlock

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "impl_item" {
			impl := p.parseImplBlock(node, content)
			if impl != nil {
				implBlocks = append(implBlocks, *impl)
			}
			return false
		}
		return true
	})

	return implBlocks
}

// parseImplBlock parses an impl block.
func (p *RustParser) parseImplBlock(node *sitter.Node, content []byte) *RustImplBlock {
	impl := &RustImplBlock{
		Line:    int(node.StartPoint().Row) + 1,
		Methods: []RustFunction{},
		Node:    node,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "type_identifier", "generic_type", "scoped_type_identifier":
			if impl.TypeName == "" {
				impl.TypeName = child.Content(content)
			}
		case "trait_bounds":
			// impl Trait for Type
			impl.TraitName = child.Content(content)
		case "declaration_list":
			// Extract methods from impl body
			impl.Methods = p.extractImplMethods(child, content)
		}
	}

	return impl
}

// extractImplMethods extracts methods from an impl block body.
func (p *RustParser) extractImplMethods(node *sitter.Node, content []byte) []RustFunction {
	var methods []RustFunction

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "function_item" {
			fn := p.parseFunction(child, content)
			if fn != nil {
				methods = append(methods, *fn)
			}
		}
	}

	return methods
}

// ExtractMacroInvocations extracts macro invocations from the AST.
func (p *RustParser) ExtractMacroInvocations(rootNode *sitter.Node, content []byte) []RustMacroInvocation {
	var macros []RustMacroInvocation

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "macro_invocation" {
			macro := p.parseMacroInvocation(node, content)
			if macro != nil {
				macros = append(macros, *macro)
			}
		}
		return true
	})

	return macros
}

// parseMacroInvocation parses a macro invocation.
func (p *RustParser) parseMacroInvocation(node *sitter.Node, content []byte) *RustMacroInvocation {
	macro := &RustMacroInvocation{
		Line: int(node.StartPoint().Row) + 1,
		Node: node,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier", "scoped_identifier":
			macro.Name = child.Content(content)
		case "token_tree":
			macro.Arguments = child.Content(content)
		}
	}

	return macro
}

// walkNodes walks all nodes in the tree, calling fn for each node.
// If fn returns false, it stops recursing into that node's children.
func (p *RustParser) walkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
	if node == nil {
		return
	}

	if !fn(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNodes(node.Child(i), fn)
	}
}

// WalkNodes is a public method for walking nodes.
func (p *RustParser) WalkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
	p.walkNodes(node, fn)
}

// FindCallExpressions finds all call expression nodes in the AST.
func (p *RustParser) FindCallExpressions(rootNode *sitter.Node, _ []byte) []*sitter.Node {
	var calls []*sitter.Node

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "call_expression" {
			calls = append(calls, node)
		}
		return true
	})

	return calls
}

// IsSupported returns whether Rust parsing is fully implemented.
func (p *RustParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *RustParser) SupportedExtensions() []string {
	return []string{".rs"}
}

// Close cleans up parser resources.
func (p *RustParser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// Close cleans up the parsed file resources.
func (pf *ParsedRustFile) Close() {
	if pf.Tree != nil {
		pf.Tree.Close()
	}
}

// HasUse checks if the file has a specific use statement.
func (pf *ParsedRustFile) HasUse(path string) bool {
	for _, use := range pf.Uses {
		if strings.Contains(use.Path, path) {
			return true
		}
	}
	return false
}

// HasDeriveAttribute checks if a struct has a specific derive attribute.
func (s *RustStruct) HasDeriveAttribute(derive string) bool {
	for _, attr := range s.Attributes {
		if attr.Name == "derive" {
			for _, arg := range attr.Arguments {
				if strings.Contains(arg, derive) {
					return true
				}
			}
		}
	}
	return false
}

// GetSerdeRename gets the serde rename value for a field if present.
func (f *RustField) GetSerdeRename() string {
	for _, attr := range f.Attributes {
		if attr.Name == "serde" {
			for _, arg := range attr.Arguments {
				re := regexp.MustCompile(`rename\s*=\s*"([^"]+)"`)
				matches := re.FindStringSubmatch(arg)
				if len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}
	return ""
}

// RustTypeToOpenAPI converts a Rust type to an OpenAPI type.
func RustTypeToOpenAPI(rustType string) (openAPIType string, format string) {
	// Trim whitespace and handle reference types
	rustType = strings.TrimSpace(rustType)
	rustType = strings.TrimPrefix(rustType, "&")
	rustType = strings.TrimPrefix(rustType, "mut ")

	// Handle Option<T>
	if strings.HasPrefix(rustType, "Option<") {
		innerType := extractRustGenericType(rustType)
		return RustTypeToOpenAPI(innerType)
	}

	// Handle Vec<T>
	if strings.HasPrefix(rustType, "Vec<") {
		return "array", ""
	}

	// Handle HashMap, BTreeMap
	if strings.HasPrefix(rustType, "HashMap<") || strings.HasPrefix(rustType, "BTreeMap<") {
		return "object", ""
	}

	switch rustType {
	case "String", "&str", "str":
		return "string", ""
	case "i8", "i16", "i32", "i64", "isize":
		return "integer", ""
	case "u8", "u16", "u32", "u64", "usize":
		return "integer", ""
	case "f32", "f64":
		return "number", ""
	case "bool":
		return "boolean", ""
	case "Uuid", "uuid::Uuid":
		return "string", "uuid"
	case "DateTime", "chrono::DateTime", "NaiveDateTime":
		return "string", "date-time"
	case "NaiveDate":
		return "string", "date"
	case "()":
		return "", ""
	default:
		// Assume it's a custom type/struct
		return "object", ""
	}
}

// extractRustGenericType extracts the inner type from a generic like Option<String>.
func extractRustGenericType(s string) string {
	start := strings.Index(s, "<")
	end := strings.LastIndex(s, ">")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+1 : end])
}
