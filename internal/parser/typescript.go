// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// TypeScriptParser provides TypeScript/JavaScript AST parsing capabilities using tree-sitter.
type TypeScriptParser struct {
	parser *sitter.Parser
}

// NewTypeScriptParser creates a new TypeScript parser.
func NewTypeScriptParser() *TypeScriptParser {
	parser := sitter.NewParser()
	parser.SetLanguage(typescript.GetLanguage())
	return &TypeScriptParser{
		parser: parser,
	}
}

// ParsedTSFile represents a parsed TypeScript source file.
type ParsedTSFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content []byte

	// Tree is the tree-sitter parse tree
	Tree *sitter.Tree

	// RootNode is the root node of the AST
	RootNode *sitter.Node

	// Interfaces contains extracted interface definitions
	Interfaces []TSInterface

	// TypeAliases contains extracted type alias definitions
	TypeAliases []TSTypeAlias

	// ZodSchemas contains extracted Zod schema definitions
	ZodSchemas []ZodSchema

	// Exports contains exported identifiers
	Exports []string
}

// TSInterface represents a TypeScript interface definition.
type TSInterface struct {
	// Name is the interface name
	Name string

	// Properties are the interface properties
	Properties []TSProperty

	// Description is from JSDoc comment
	Description string

	// Extends lists extended interfaces
	Extends []string

	// IsExported indicates if the interface is exported
	IsExported bool

	// Line is the source line number
	Line int
}

// TSTypeAlias represents a TypeScript type alias.
type TSTypeAlias struct {
	// Name is the type alias name
	Name string

	// Type is the underlying type definition
	Type string

	// Description is from JSDoc comment
	Description string

	// IsExported indicates if the type is exported
	IsExported bool

	// Line is the source line number
	Line int
}

// TSProperty represents a property in a TypeScript interface or type.
type TSProperty struct {
	// Name is the property name
	Name string

	// Type is the TypeScript type
	Type string

	// IsOptional indicates if the property is optional (has ?)
	IsOptional bool

	// IsReadonly indicates if the property is readonly
	IsReadonly bool

	// Description is from JSDoc comment
	Description string
}

// ZodSchema represents a Zod schema variable declaration.
type ZodSchema struct {
	// Name is the schema variable name
	Name string

	// Node is the tree-sitter node for the z.object() call or similar
	Node *sitter.Node

	// IsExported indicates if the schema is exported
	IsExported bool

	// Line is the source line number
	Line int
}

// ParseSource parses TypeScript source code from a string.
func (p *TypeScriptParser) ParseSource(filename string, source string) (*ParsedTSFile, error) {
	return p.Parse(filename, []byte(source))
}

// Parse parses TypeScript source code from bytes.
func (p *TypeScriptParser) Parse(filename string, content []byte) (*ParsedTSFile, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TypeScript: %w", err)
	}

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("failed to get root node")
	}

	pf := &ParsedTSFile{
		Path:        filename,
		Content:     content,
		Tree:        tree,
		RootNode:    rootNode,
		Interfaces:  []TSInterface{},
		TypeAliases: []TSTypeAlias{},
		ZodSchemas:  []ZodSchema{},
		Exports:     []string{},
	}

	// Extract definitions
	pf.Interfaces = p.ExtractInterfaces(rootNode, content)
	pf.TypeAliases = p.ExtractTypeAliases(rootNode, content)
	pf.ZodSchemas = p.ExtractZodSchemas(rootNode, content)
	pf.Exports = p.ExtractExports(rootNode, content)

	return pf, nil
}

// ParseFile parses a TypeScript source file from disk.
func (p *TypeScriptParser) ParseFile(path string) (*ParsedTSFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return p.Parse(path, content)
}

// ExtractInterfaces extracts all interface definitions from the AST.
func (p *TypeScriptParser) ExtractInterfaces(rootNode *sitter.Node, content []byte) []TSInterface {
	var interfaces []TSInterface
	// Track which interfaces we've seen by line number to avoid duplicates
	seen := make(map[int]bool)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check export_statement that wraps interface_declaration
		if node.Type() == "export_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "interface_declaration" {
					iface := p.parseInterfaceDecl(child, content)
					if iface != nil {
						iface.IsExported = true
						if !seen[iface.Line] {
							seen[iface.Line] = true
							interfaces = append(interfaces, *iface)
						}
					}
					return false // Don't recurse into this export_statement
				}
			}
		}
		if node.Type() == "interface_declaration" {
			iface := p.parseInterfaceDecl(node, content)
			if iface != nil {
				if !seen[iface.Line] {
					seen[iface.Line] = true
					interfaces = append(interfaces, *iface)
				}
			}
		}
		return true
	})

	return interfaces
}

// parseInterfaceDecl parses an interface_declaration node.
func (p *TypeScriptParser) parseInterfaceDecl(node *sitter.Node, content []byte) *TSInterface {
	iface := &TSInterface{
		Line: int(node.StartPoint().Row) + 1,
	}

	// Find interface name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "type_identifier":
			iface.Name = child.Content(content)
		case "extends_type_clause":
			// Extract extended interfaces
			p.walkNodes(child, func(n *sitter.Node) bool {
				if n.Type() == "type_identifier" {
					iface.Extends = append(iface.Extends, n.Content(content))
				}
				return true
			})
		case "object_type", "interface_body":
			// Extract properties
			iface.Properties = p.extractObjectProperties(child, content)
		}
	}

	return iface
}

// extractObjectProperties extracts properties from an object_type or interface_body node.
func (p *TypeScriptParser) extractObjectProperties(node *sitter.Node, content []byte) []TSProperty {
	var properties []TSProperty

	p.walkNodes(node, func(n *sitter.Node) bool {
		switch n.Type() {
		case "property_signature":
			prop := p.parsePropertySignature(n, content)
			if prop != nil {
				properties = append(properties, *prop)
			}
			return false // Don't recurse into property_signature
		}
		return true
	})

	return properties
}

// parsePropertySignature parses a property_signature node.
func (p *TypeScriptParser) parsePropertySignature(node *sitter.Node, content []byte) *TSProperty {
	prop := &TSProperty{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_identifier":
			prop.Name = child.Content(content)
		case "?":
			prop.IsOptional = true
		case "readonly":
			prop.IsReadonly = true
		case "type_annotation":
			// Get the type from the type_annotation
			if child.ChildCount() > 1 {
				typeNode := child.Child(1) // Skip the ':'
				prop.Type = typeNode.Content(content)
			}
		}
	}

	return prop
}

// ExtractTypeAliases extracts all type alias definitions from the AST.
func (p *TypeScriptParser) ExtractTypeAliases(rootNode *sitter.Node, content []byte) []TSTypeAlias {
	var typeAliases []TSTypeAlias
	// Track which type aliases we've seen by line number to avoid duplicates
	seen := make(map[int]bool)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check export_statement that wraps type_alias_declaration
		if node.Type() == "export_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "type_alias_declaration" {
					alias := p.parseTypeAliasDecl(child, content)
					if alias != nil {
						alias.IsExported = true
						if !seen[alias.Line] {
							seen[alias.Line] = true
							typeAliases = append(typeAliases, *alias)
						}
					}
					return false // Don't recurse into this export_statement
				}
			}
		}
		if node.Type() == "type_alias_declaration" {
			alias := p.parseTypeAliasDecl(node, content)
			if alias != nil {
				if !seen[alias.Line] {
					seen[alias.Line] = true
					typeAliases = append(typeAliases, *alias)
				}
			}
		}
		return true
	})

	return typeAliases
}

// parseTypeAliasDecl parses a type_alias_declaration node.
func (p *TypeScriptParser) parseTypeAliasDecl(node *sitter.Node, content []byte) *TSTypeAlias {
	alias := &TSTypeAlias{
		Line: int(node.StartPoint().Row) + 1,
	}

	foundEquals := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		switch childType {
		case "type_identifier":
			if alias.Name == "" {
				alias.Name = child.Content(content)
			}
		case "=":
			foundEquals = true
		case "type", ";":
			// Skip these
		default:
			// The type value comes after the '='
			if foundEquals && alias.Type == "" {
				alias.Type = child.Content(content)
			}
		}
	}

	return alias
}

// ExtractZodSchemas extracts Zod schema definitions from the AST.
func (p *TypeScriptParser) ExtractZodSchemas(rootNode *sitter.Node, content []byte) []ZodSchema {
	var schemas []ZodSchema
	// Track which schemas we've seen by line number to avoid duplicates
	seen := make(map[int]bool)

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		// Check export_statement first
		if node.Type() == "export_statement" {
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "lexical_declaration" || child.Type() == "variable_declaration" {
					zodSchemas := p.extractZodFromDeclaration(child, content)
					for j := range zodSchemas {
						zodSchemas[j].IsExported = true
						if !seen[zodSchemas[j].Line] {
							seen[zodSchemas[j].Line] = true
							schemas = append(schemas, zodSchemas[j])
						}
					}
					return false // Don't recurse into this export_statement
				}
			}
		}
		// Look for variable declarations that contain z.object(), z.string(), etc.
		if node.Type() == "lexical_declaration" || node.Type() == "variable_declaration" {
			zodSchemas := p.extractZodFromDeclaration(node, content)
			for _, zs := range zodSchemas {
				if !seen[zs.Line] {
					seen[zs.Line] = true
					schemas = append(schemas, zs)
				}
			}
		}
		return true
	})

	return schemas
}

// extractZodFromDeclaration extracts Zod schemas from a variable declaration.
func (p *TypeScriptParser) extractZodFromDeclaration(node *sitter.Node, content []byte) []ZodSchema {
	var schemas []ZodSchema

	p.walkNodes(node, func(n *sitter.Node) bool {
		if n.Type() == "variable_declarator" {
			name := ""
			var valueNode *sitter.Node

			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(i)
				switch child.Type() {
				case "identifier":
					name = child.Content(content)
				case "call_expression":
					valueNode = child
				}
			}

			if name != "" && valueNode != nil && p.isZodCall(valueNode, content) {
				schemas = append(schemas, ZodSchema{
					Name: name,
					Node: valueNode,
					Line: int(n.StartPoint().Row) + 1,
				})
			}
		}
		return true
	})

	return schemas
}

// isZodCall checks if a call_expression is a Zod method call.
func (p *TypeScriptParser) isZodCall(node *sitter.Node, content []byte) bool {
	if node.Type() != "call_expression" {
		return false
	}

	// Get the callee
	callee := node.ChildByFieldName("function")
	if callee == nil && node.ChildCount() > 0 {
		callee = node.Child(0)
	}
	if callee == nil {
		return false
	}

	calleeText := callee.Content(content)

	// Check for z.object, z.string, z.number, etc.
	if strings.HasPrefix(calleeText, "z.") {
		return true
	}

	// Check for method chain on z call (e.g., z.string().email())
	if callee.Type() == "member_expression" {
		objectNode := callee.ChildByFieldName("object")
		if objectNode == nil && callee.ChildCount() > 0 {
			objectNode = callee.Child(0)
		}
		if objectNode != nil {
			return p.isZodCall(objectNode, content)
		}
	}

	return false
}

// ExtractExports extracts exported identifiers.
func (p *TypeScriptParser) ExtractExports(rootNode *sitter.Node, content []byte) []string {
	var exports []string

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "export_statement" {
			// Check for default export
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "identifier" {
					exports = append(exports, child.Content(content))
				}
			}
		}
		return true
	})

	return exports
}

// walkNodes walks all nodes in the tree, calling fn for each node.
// If fn returns false, it stops recursing into that node's children.
func (p *TypeScriptParser) walkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
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

// IsSupported returns whether TypeScript parsing is fully implemented.
func (p *TypeScriptParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *TypeScriptParser) SupportedExtensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".mjs"}
}

// Close cleans up parser resources.
func (p *TypeScriptParser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// Close cleans up the parsed file resources.
func (pf *ParsedTSFile) Close() {
	if pf.Tree != nil {
		pf.Tree.Close()
	}
}

// TypeScriptTypeToOpenAPI converts a TypeScript type to an OpenAPI type.
func TypeScriptTypeToOpenAPI(tsType string) (openAPIType string, format string) {
	// Trim whitespace
	tsType = strings.TrimSpace(tsType)

	switch tsType {
	case "string":
		return "string", ""
	case "number":
		return "number", ""
	case "boolean":
		return "boolean", ""
	case "Date":
		return "string", "date-time"
	case "any", "unknown":
		return "object", ""
	case "null":
		return "null", ""
	case "undefined":
		return "", ""
	case "void":
		return "", ""
	default:
		// Check for array types
		if strings.HasSuffix(tsType, "[]") {
			return "array", ""
		}
		// Check for Array<T> generic
		if strings.HasPrefix(tsType, "Array<") {
			return "array", ""
		}
		// Assume it's a reference to another type
		return "object", ""
	}
}

// TSTypeKind classifies TypeScript types for schema conversion.
type TSTypeKind int

const (
	TSKindPrimitive TSTypeKind = iota
	TSKindInterface
	TSKindTypeAlias
	TSKindArray
	TSKindUnion
	TSKindIntersection
	TSKindLiteral
	TSKindGeneric
	TSKindUnknown
)

// ClassifyType determines the TSTypeKind for a TypeScript type string.
func ClassifyType(tsType string) TSTypeKind {
	switch tsType {
	case "string", "number", "boolean", "null", "undefined", "void", "any", "unknown":
		return TSKindPrimitive
	}

	// Check for array
	if strings.HasSuffix(tsType, "[]") {
		return TSKindArray
	}
	if strings.HasPrefix(tsType, "Array<") {
		return TSKindArray
	}

	// Check for union (contains |)
	if strings.Contains(tsType, "|") {
		return TSKindUnion
	}

	// Check for intersection (contains &)
	if strings.Contains(tsType, "&") {
		return TSKindIntersection
	}

	// Check for generic (contains <)
	if strings.Contains(tsType, "<") {
		return TSKindGeneric
	}

	// Check for literal types (string or number literals)
	if len(tsType) > 0 && (tsType[0] == '"' || tsType[0] == '\'' || (tsType[0] >= '0' && tsType[0] <= '9')) {
		return TSKindLiteral
	}

	return TSKindUnknown
}

// String returns a string representation of the TSTypeKind.
func (k TSTypeKind) String() string {
	switch k {
	case TSKindPrimitive:
		return "primitive"
	case TSKindInterface:
		return "interface"
	case TSKindTypeAlias:
		return "typeAlias"
	case TSKindArray:
		return "array"
	case TSKindUnion:
		return "union"
	case TSKindIntersection:
		return "intersection"
	case TSKindLiteral:
		return "literal"
	case TSKindGeneric:
		return "generic"
	default:
		return "unknown"
	}
}

// ValidateTypeScriptFile performs basic validation on TypeScript source.
func ValidateTypeScriptFile(source string) error {
	if len(source) == 0 {
		return fmt.Errorf("empty source file")
	}
	return nil
}

// FindCallExpressions finds all call_expression nodes in the AST.
func (p *TypeScriptParser) FindCallExpressions(rootNode *sitter.Node, content []byte) []*sitter.Node {
	var calls []*sitter.Node

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "call_expression" {
			calls = append(calls, node)
		}
		return true
	})

	return calls
}

// GetCalleeText returns the callee text from a call_expression.
func (p *TypeScriptParser) GetCalleeText(node *sitter.Node, content []byte) string {
	if node.Type() != "call_expression" {
		return ""
	}

	// Try to get the function field
	callee := node.ChildByFieldName("function")
	if callee == nil && node.ChildCount() > 0 {
		callee = node.Child(0)
	}
	if callee == nil {
		return ""
	}

	return callee.Content(content)
}

// GetCallArguments returns the arguments from a call_expression.
func (p *TypeScriptParser) GetCallArguments(node *sitter.Node, content []byte) []*sitter.Node {
	var args []*sitter.Node

	if node.Type() != "call_expression" {
		return args
	}

	// Find arguments node
	argNode := node.ChildByFieldName("arguments")
	if argNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "arguments" {
				argNode = child
				break
			}
		}
	}

	if argNode == nil {
		return args
	}

	// Collect argument expressions (skip punctuation)
	for i := 0; i < int(argNode.ChildCount()); i++ {
		child := argNode.Child(i)
		if child.Type() != "," && child.Type() != "(" && child.Type() != ")" {
			args = append(args, child)
		}
	}

	return args
}

// ExtractStringLiteral extracts a string value from a string node.
func (p *TypeScriptParser) ExtractStringLiteral(node *sitter.Node, content []byte) (string, bool) {
	if node == nil {
		return "", false
	}

	nodeType := node.Type()
	if nodeType != "string" && nodeType != "template_string" && nodeType != "string_fragment" {
		return "", false
	}

	text := node.Content(content)

	// Remove quotes
	if len(text) >= 2 {
		if (text[0] == '"' && text[len(text)-1] == '"') ||
			(text[0] == '\'' && text[len(text)-1] == '\'') ||
			(text[0] == '`' && text[len(text)-1] == '`') {
			return text[1 : len(text)-1], true
		}
	}

	return text, true
}

// GetMemberExpressionParts returns the object and property of a member_expression.
func (p *TypeScriptParser) GetMemberExpressionParts(node *sitter.Node, content []byte) (object, property string) {
	if node.Type() != "member_expression" {
		return "", ""
	}

	objNode := node.ChildByFieldName("object")
	if objNode == nil && node.ChildCount() > 0 {
		objNode = node.Child(0)
	}

	propNode := node.ChildByFieldName("property")
	if propNode == nil && node.ChildCount() > 2 {
		propNode = node.Child(2)
	}

	if objNode != nil {
		object = objNode.Content(content)
	}
	if propNode != nil {
		property = propNode.Content(content)
	}

	return object, property
}
