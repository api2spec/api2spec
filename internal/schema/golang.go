// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package schema provides schema extraction and conversion utilities.
package schema

import (
	"strconv"
	"strings"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/pkg/types"
)

// GoSchemaExtractor converts Go struct definitions to JSON Schemas.
type GoSchemaExtractor struct {
	// registry stores discovered schemas for reference resolution
	registry *Registry
}

// NewGoSchemaExtractor creates a new Go schema extractor.
func NewGoSchemaExtractor() *GoSchemaExtractor {
	return &GoSchemaExtractor{
		registry: NewRegistry(),
	}
}

// ExtractFromStruct converts a StructDefinition to a JSON Schema.
func (e *GoSchemaExtractor) ExtractFromStruct(def parser.StructDefinition) *types.Schema {
	schema := &types.Schema{
		Type:        "object",
		Title:       def.Name,
		Description: def.Description,
		Properties:  make(map[string]*types.Schema),
	}

	var requiredFields []string

	for _, field := range def.Fields {
		// Skip fields that are not serialized
		if field.JSONName == "-" {
			continue
		}

		propSchema := e.fieldToSchema(field)
		schema.Properties[field.JSONName] = propSchema

		// Determine if field is required
		if e.isFieldRequired(field) {
			requiredFields = append(requiredFields, field.JSONName)
		}
	}

	if len(requiredFields) > 0 {
		schema.Required = requiredFields
	}

	// Register the schema for reference
	if def.Name != "" {
		e.registry.Add(def.Name, schema)
	}

	return schema
}

// fieldToSchema converts a struct field to a JSON Schema.
func (e *GoSchemaExtractor) fieldToSchema(field parser.StructField) *types.Schema {
	schema := e.typeToSchema(field)

	// Apply description
	if field.Description != "" {
		schema.Description = field.Description
	}

	// Apply validation constraints
	e.applyValidationTags(schema, field)

	return schema
}

// typeToSchema converts a Go type to a JSON Schema.
func (e *GoSchemaExtractor) typeToSchema(field parser.StructField) *types.Schema {
	// Handle pointer types - the underlying type determines the schema,
	// but the field becomes nullable/optional
	if field.IsPointer && field.TypeKind == parser.KindPointer {
		// Extract the underlying type name
		underlyingType := strings.TrimPrefix(field.Type, "*")
		return e.primitiveOrRefSchema(underlyingType, field)
	}

	switch field.TypeKind {
	case parser.KindPrimitive:
		return e.primitiveToSchema(field.Type)

	case parser.KindTime:
		return &types.Schema{
			Type:   "string",
			Format: "date-time",
		}

	case parser.KindSlice:
		itemSchema := e.elementTypeToSchema(field.ElementType)
		return &types.Schema{
			Type:  "array",
			Items: itemSchema,
		}

	case parser.KindMap:
		// Maps become objects with additionalProperties
		valueSchema := e.elementTypeToSchema(field.ElementType)
		return &types.Schema{
			Type:                 "object",
			AdditionalProperties: valueSchema,
		}

	case parser.KindStruct:
		// Check for nested inline struct
		if len(field.NestedStruct) > 0 {
			return e.nestedStructToSchema(field.NestedStruct)
		}
		// Reference to another struct
		return SchemaRef(field.Type)

	case parser.KindInterface:
		// interface{} or any becomes an empty schema
		return &types.Schema{}

	case parser.KindPointer:
		// Pointer to complex type
		underlyingType := strings.TrimPrefix(field.Type, "*")
		// Check if it's time.Time
		if underlyingType == "time.Time" {
			return &types.Schema{
				Type:     "string",
				Format:   "date-time",
				Nullable: true,
			}
		}
		// Check if it's a primitive
		if isPrimitive(underlyingType) {
			s := e.primitiveToSchema(underlyingType)
			s.Nullable = true
			return s
		}
		// It's a reference to a struct
		return SchemaRef(underlyingType)

	default:
		// Unknown type - return empty schema
		return &types.Schema{}
	}
}

// primitiveOrRefSchema handles pointer types.
// TODO: Use field parameter to apply validation constraints from struct tags.
func (e *GoSchemaExtractor) primitiveOrRefSchema(typeName string, _ parser.StructField) *types.Schema {
	// Check for time.Time
	if typeName == "time.Time" {
		return &types.Schema{
			Type:     "string",
			Format:   "date-time",
			Nullable: true,
		}
	}

	// Check if it's a primitive type
	if isPrimitive(typeName) {
		s := e.primitiveToSchema(typeName)
		s.Nullable = true
		return s
	}

	// It's a reference to another struct
	return SchemaRef(typeName)
}

// primitiveToSchema converts a Go primitive type to a JSON Schema.
func (e *GoSchemaExtractor) primitiveToSchema(typeName string) *types.Schema {
	switch typeName {
	case "string":
		return &types.Schema{Type: "string"}

	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return &types.Schema{Type: "integer"}

	case "float32", "float64":
		return &types.Schema{Type: "number"}

	case "bool":
		return &types.Schema{Type: "boolean"}

	case "byte":
		return &types.Schema{Type: "integer"}

	case "rune":
		return &types.Schema{Type: "integer"}

	default:
		// Unknown type - return string as default
		return &types.Schema{Type: "string"}
	}
}

// elementTypeToSchema converts an element type (for slices/maps) to a schema.
func (e *GoSchemaExtractor) elementTypeToSchema(elementType string) *types.Schema {
	// Handle pointer elements
	if strings.HasPrefix(elementType, "*") {
		underlyingType := strings.TrimPrefix(elementType, "*")
		if isPrimitive(underlyingType) {
			s := e.primitiveToSchema(underlyingType)
			s.Nullable = true
			return s
		}
		return SchemaRef(underlyingType)
	}

	// Handle slice of slices
	if strings.HasPrefix(elementType, "[]") {
		innerType := strings.TrimPrefix(elementType, "[]")
		return &types.Schema{
			Type:  "array",
			Items: e.elementTypeToSchema(innerType),
		}
	}

	// Check if it's a primitive
	if isPrimitive(elementType) {
		return e.primitiveToSchema(elementType)
	}

	// Check for time.Time
	if elementType == "time.Time" {
		return &types.Schema{
			Type:   "string",
			Format: "date-time",
		}
	}

	// It's a reference to another struct
	return SchemaRef(elementType)
}

// nestedStructToSchema converts an inline struct to a schema.
func (e *GoSchemaExtractor) nestedStructToSchema(fields []parser.StructField) *types.Schema {
	schema := &types.Schema{
		Type:       "object",
		Properties: make(map[string]*types.Schema),
	}

	var requiredFields []string

	for _, field := range fields {
		if field.JSONName == "-" {
			continue
		}

		propSchema := e.fieldToSchema(field)
		schema.Properties[field.JSONName] = propSchema

		if e.isFieldRequired(field) {
			requiredFields = append(requiredFields, field.JSONName)
		}
	}

	if len(requiredFields) > 0 {
		schema.Required = requiredFields
	}

	return schema
}

// isFieldRequired determines if a field should be marked as required.
func (e *GoSchemaExtractor) isFieldRequired(field parser.StructField) bool {
	// Explicitly required via validate tag
	if field.IsRequired {
		return true
	}

	// Pointer types are optional
	if field.IsPointer {
		return false
	}

	// Fields with omitempty are optional
	if field.Omitempty {
		return false
	}

	// Slices and maps are optional by default (can be empty)
	if field.TypeKind == parser.KindSlice || field.TypeKind == parser.KindMap {
		return false
	}

	// Other fields are required by default
	// This is a conservative approach - fields without explicit markers
	// are considered required. This matches Go's zero-value behavior
	// where fields always have a value.
	return false
}

// applyValidationTags applies validation constraints to a schema.
func (e *GoSchemaExtractor) applyValidationTags(schema *types.Schema, field parser.StructField) {
	for key, value := range field.ValidationTags {
		switch key {
		case "min":
			e.applyMin(schema, value)
		case "max":
			e.applyMax(schema, value)
		case "len":
			e.applyLen(schema, value)
		case "email":
			schema.Format = "email"
		case "url", "uri":
			schema.Format = "uri"
		case "uuid", "uuid4":
			schema.Format = "uuid"
		case "datetime":
			schema.Format = "date-time"
		case "ip":
			schema.Format = "ipv4"
		case "ipv4":
			schema.Format = "ipv4"
		case "ipv6":
			schema.Format = "ipv6"
		case "hostname":
			schema.Format = "hostname"
		case "alphanum":
			schema.Pattern = "^[a-zA-Z0-9]+$"
		case "alpha":
			schema.Pattern = "^[a-zA-Z]+$"
		case "numeric":
			schema.Pattern = "^[0-9]+$"
		case "oneof":
			e.applyOneOf(schema, value)
		}
	}
}

// applyMin applies minimum constraint based on type.
func (e *GoSchemaExtractor) applyMin(schema *types.Schema, value string) {
	switch schema.Type {
	case "string":
		if v, err := strconv.Atoi(value); err == nil {
			schema.MinLength = &v
		}
	case "integer", "number":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			schema.Minimum = &v
		}
	case "array":
		if v, err := strconv.Atoi(value); err == nil {
			schema.MinItems = &v
		}
	}
}

// applyMax applies maximum constraint based on type.
func (e *GoSchemaExtractor) applyMax(schema *types.Schema, value string) {
	switch schema.Type {
	case "string":
		if v, err := strconv.Atoi(value); err == nil {
			schema.MaxLength = &v
		}
	case "integer", "number":
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			schema.Maximum = &v
		}
	case "array":
		if v, err := strconv.Atoi(value); err == nil {
			schema.MaxItems = &v
		}
	}
}

// applyLen applies exact length constraint.
func (e *GoSchemaExtractor) applyLen(schema *types.Schema, value string) {
	if v, err := strconv.Atoi(value); err == nil {
		switch schema.Type {
		case "string":
			schema.MinLength = &v
			schema.MaxLength = &v
		case "array":
			schema.MinItems = &v
			schema.MaxItems = &v
		}
	}
}

// applyOneOf applies enum constraint.
func (e *GoSchemaExtractor) applyOneOf(schema *types.Schema, value string) {
	// oneof=value1 value2 value3
	parts := strings.Fields(value)
	if len(parts) > 0 {
		schema.Enum = make([]interface{}, len(parts))
		for i, p := range parts {
			schema.Enum[i] = p
		}
	}
}

// Registry returns the schema registry.
func (e *GoSchemaExtractor) Registry() *Registry {
	return e.registry
}

// SchemaRef creates a reference to a schema in components.
func SchemaRef(schemaName string) *types.Schema {
	return &types.Schema{
		Ref: "#/components/schemas/" + schemaName,
	}
}

// isPrimitive checks if a type name is a Go primitive.
func isPrimitive(typeName string) bool {
	switch typeName {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "bool", "byte", "rune":
		return true
	default:
		return false
	}
}
