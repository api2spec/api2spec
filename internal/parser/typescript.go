// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"fmt"
)

// TypeScriptParser provides TypeScript/JavaScript AST parsing capabilities.
// This is a stub implementation - full tree-sitter integration will be added in Phase 3.
type TypeScriptParser struct {
	// TODO: Add tree-sitter parser instance when implementing full support
}

// NewTypeScriptParser creates a new TypeScript parser.
func NewTypeScriptParser() *TypeScriptParser {
	return &TypeScriptParser{}
}

// ParsedTSFile represents a parsed TypeScript source file.
type ParsedTSFile struct {
	// Path is the file path
	Path string

	// Interfaces contains extracted interface definitions
	Interfaces []TSInterface

	// TypeAliases contains extracted type alias definitions
	TypeAliases []TSTypeAlias

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

// ParseSource parses TypeScript source code from a string.
// This is a stub implementation that returns an empty result.
func (p *TypeScriptParser) ParseSource(filename, source string) (*ParsedTSFile, error) {
	// TODO: Implement tree-sitter parsing in Phase 3
	return &ParsedTSFile{
		Path:        filename,
		Interfaces:  []TSInterface{},
		TypeAliases: []TSTypeAlias{},
		Exports:     []string{},
	}, nil
}

// ParseFile parses a TypeScript source file from disk.
// This is a stub implementation that returns an empty result.
func (p *TypeScriptParser) ParseFile(path string) (*ParsedTSFile, error) {
	// TODO: Implement tree-sitter parsing in Phase 3
	return &ParsedTSFile{
		Path:        path,
		Interfaces:  []TSInterface{},
		TypeAliases: []TSTypeAlias{},
		Exports:     []string{},
	}, nil
}

// ExtractInterfaces extracts all interface definitions from a parsed file.
// This is a stub implementation that returns an empty slice.
func (p *TypeScriptParser) ExtractInterfaces(pf *ParsedTSFile) []TSInterface {
	// TODO: Implement in Phase 3
	return pf.Interfaces
}

// ExtractTypeAliases extracts all type alias definitions from a parsed file.
// This is a stub implementation that returns an empty slice.
func (p *TypeScriptParser) ExtractTypeAliases(pf *ParsedTSFile) []TSTypeAlias {
	// TODO: Implement in Phase 3
	return pf.TypeAliases
}

// IsSupported returns whether TypeScript parsing is fully implemented.
// Returns false until tree-sitter integration is complete.
func (p *TypeScriptParser) IsSupported() bool {
	return false
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *TypeScriptParser) SupportedExtensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx"}
}

// TypeScriptTypeToOpenAPI converts a TypeScript type to an OpenAPI type.
// This is a basic implementation that handles common types.
func TypeScriptTypeToOpenAPI(tsType string) (openAPIType string, format string) {
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
		if len(tsType) > 2 && tsType[len(tsType)-2:] == "[]" {
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
	if len(tsType) > 2 && tsType[len(tsType)-2:] == "[]" {
		return TSKindArray
	}

	// Check for union (contains |)
	for _, ch := range tsType {
		if ch == '|' {
			return TSKindUnion
		}
	}

	// Check for intersection (contains &)
	for _, ch := range tsType {
		if ch == '&' {
			return TSKindIntersection
		}
	}

	// Check for generic (contains <)
	for _, ch := range tsType {
		if ch == '<' {
			return TSKindGeneric
		}
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
// Returns nil if validation passes, or an error describing the issue.
func ValidateTypeScriptFile(source string) error {
	// Basic validation - check for syntax issues
	// This is a placeholder for actual validation logic

	if len(source) == 0 {
		return fmt.Errorf("empty source file")
	}

	return nil
}
