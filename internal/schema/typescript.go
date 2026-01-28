// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package schema provides schema extraction and conversion utilities.
package schema

import (
	"strings"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/pkg/types"
)

// TypeScriptSchemaExtractor converts TypeScript interfaces to JSON Schemas.
type TypeScriptSchemaExtractor struct {
	registry *Registry
}

// NewTypeScriptSchemaExtractor creates a new TypeScript schema extractor.
func NewTypeScriptSchemaExtractor() *TypeScriptSchemaExtractor {
	return &TypeScriptSchemaExtractor{
		registry: NewRegistry(),
	}
}

// ExtractFromInterface converts a TSInterface to a JSON Schema.
func (e *TypeScriptSchemaExtractor) ExtractFromInterface(iface parser.TSInterface) *types.Schema {
	schema := &types.Schema{
		Type:        "object",
		Title:       iface.Name,
		Description: iface.Description,
		Properties:  make(map[string]*types.Schema),
	}

	var requiredFields []string

	for _, prop := range iface.Properties {
		propSchema := e.propertyToSchema(prop)
		schema.Properties[prop.Name] = propSchema

		// Non-optional properties are required
		if !prop.IsOptional {
			requiredFields = append(requiredFields, prop.Name)
		}
	}

	if len(requiredFields) > 0 {
		schema.Required = requiredFields
	}

	// Register the schema for reference
	if iface.Name != "" {
		e.registry.Add(iface.Name, schema)
	}

	return schema
}

// propertyToSchema converts a TypeScript property to a JSON Schema.
func (e *TypeScriptSchemaExtractor) propertyToSchema(prop parser.TSProperty) *types.Schema {
	schema := e.typeToSchema(prop.Type)

	if prop.Description != "" {
		schema.Description = prop.Description
	}

	if prop.IsReadonly {
		schema.ReadOnly = true
	}

	return schema
}

// typeToSchema converts a TypeScript type string to a JSON Schema.
func (e *TypeScriptSchemaExtractor) typeToSchema(tsType string) *types.Schema {
	tsType = strings.TrimSpace(tsType)

	// Handle union types (e.g., "string | number")
	if strings.Contains(tsType, " | ") {
		return e.unionTypeToSchema(tsType)
	}

	// Handle array types
	if strings.HasSuffix(tsType, "[]") {
		elementType := strings.TrimSuffix(tsType, "[]")
		return &types.Schema{
			Type:  "array",
			Items: e.typeToSchema(elementType),
		}
	}

	// Handle Array<T> generic
	if strings.HasPrefix(tsType, "Array<") && strings.HasSuffix(tsType, ">") {
		elementType := tsType[6 : len(tsType)-1]
		return &types.Schema{
			Type:  "array",
			Items: e.typeToSchema(elementType),
		}
	}

	// Handle primitive types
	switch tsType {
	case "string":
		return &types.Schema{Type: "string"}
	case "number":
		return &types.Schema{Type: "number"}
	case "boolean":
		return &types.Schema{Type: "boolean"}
	case "Date":
		return &types.Schema{Type: "string", Format: "date-time"}
	case "null":
		return &types.Schema{Type: "null"}
	case "undefined", "void":
		return &types.Schema{}
	case "any", "unknown":
		return &types.Schema{}
	default:
		// Check for string literal types (e.g., "'active'")
		if (strings.HasPrefix(tsType, "'") && strings.HasSuffix(tsType, "'")) ||
			(strings.HasPrefix(tsType, "\"") && strings.HasSuffix(tsType, "\"")) {
			val := tsType[1 : len(tsType)-1]
			return &types.Schema{
				Type: "string",
				Enum: []any{val},
			}
		}
		// Assume it's a reference to another type
		return SchemaRef(tsType)
	}
}

// unionTypeToSchema converts a TypeScript union type to a JSON Schema.
func (e *TypeScriptSchemaExtractor) unionTypeToSchema(tsType string) *types.Schema {
	parts := strings.Split(tsType, " | ")

	var oneOf []*types.Schema
	for _, part := range parts {
		part = strings.TrimSpace(part)
		oneOf = append(oneOf, e.typeToSchema(part))
	}

	// If it's a nullable type (e.g., "string | null"), simplify
	if len(oneOf) == 2 {
		for i, schema := range oneOf {
			if schema.Type == "null" {
				other := oneOf[1-i]
				other.Nullable = true
				return other
			}
		}
	}

	return &types.Schema{OneOf: oneOf}
}

// Registry returns the schema registry.
func (e *TypeScriptSchemaExtractor) Registry() *Registry {
	return e.registry
}

// ExtractAndRegister parses a TypeScript interface and registers it.
func (e *TypeScriptSchemaExtractor) ExtractAndRegister(iface parser.TSInterface) *types.Schema {
	return e.ExtractFromInterface(iface)
}
