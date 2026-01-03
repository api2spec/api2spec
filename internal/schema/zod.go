// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package schema provides schema extraction and conversion utilities.
package schema

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/pkg/types"
)

// ZodParser parses Zod schema definitions and converts them to OpenAPI schemas.
type ZodParser struct {
	tsParser *parser.TypeScriptParser
	registry *Registry
}

// NewZodParser creates a new Zod parser.
func NewZodParser(tsParser *parser.TypeScriptParser) *ZodParser {
	return &ZodParser{
		tsParser: tsParser,
		registry: NewRegistry(),
	}
}

// ParseZodSchema converts a Zod schema call_expression node to an OpenAPI schema.
func (p *ZodParser) ParseZodSchema(node *sitter.Node, content []byte) (*types.Schema, error) {
	if node == nil {
		return &types.Schema{}, nil
	}

	return p.parseZodExpression(node, content), nil
}

// parseZodExpression parses a Zod expression (call_expression or member_expression chain).
func (p *ZodParser) parseZodExpression(node *sitter.Node, content []byte) *types.Schema {
	if node == nil {
		return &types.Schema{}
	}

	// Handle call expressions like z.string(), z.object({...}), etc.
	if node.Type() == "call_expression" {
		return p.parseZodCall(node, content)
	}

	// Handle member expressions for chained calls
	if node.Type() == "member_expression" {
		return p.parseZodMemberChain(node, content)
	}

	return &types.Schema{}
}

// parseZodCall parses a Zod call expression.
func (p *ZodParser) parseZodCall(node *sitter.Node, content []byte) *types.Schema {
	// Get the callee
	callee := node.Child(0)
	if callee == nil {
		return &types.Schema{}
	}

	calleeText := callee.Content(content)

	// Get base schema from the Zod method
	schema := p.getBaseZodSchema(calleeText, node, content)

	// Apply any chained modifiers if this is a chained call
	if callee.Type() == "member_expression" {
		// Check if object is a call_expression (method chain)
		objNode := callee.Child(0)
		if objNode != nil && objNode.Type() == "call_expression" {
			// Parse the base call first
			baseSchema := p.parseZodCall(objNode, content)
			// Then apply the modifier
			propNode := callee.Child(2)
			if propNode != nil {
				schema = p.applyZodModifier(baseSchema, propNode.Content(content), node, content)
			}
		}
	}

	return schema
}

// getBaseZodSchema returns the base schema for a Zod method.
func (p *ZodParser) getBaseZodSchema(calleeText string, node *sitter.Node, content []byte) *types.Schema {
	// Extract the Zod method name
	method := extractZodMethod(calleeText)

	switch method {
	case "string":
		return &types.Schema{Type: "string"}
	case "number":
		return &types.Schema{Type: "number"}
	case "boolean":
		return &types.Schema{Type: "boolean"}
	case "bigint":
		return &types.Schema{Type: "integer", Format: "int64"}
	case "date":
		return &types.Schema{Type: "string", Format: "date-time"}
	case "undefined", "void":
		return &types.Schema{}
	case "null":
		return &types.Schema{Type: "null"}
	case "any", "unknown":
		return &types.Schema{}
	case "never":
		return &types.Schema{Not: &types.Schema{}}
	case "object":
		return p.parseZodObject(node, content)
	case "array":
		return p.parseZodArray(node, content)
	case "enum":
		return p.parseZodEnum(node, content)
	case "nativeEnum":
		return p.parseZodEnum(node, content)
	case "literal":
		return p.parseZodLiteral(node, content)
	case "union":
		return p.parseZodUnion(node, content)
	case "intersection":
		return p.parseZodIntersection(node, content)
	case "tuple":
		return p.parseZodTuple(node, content)
	case "record":
		return p.parseZodRecord(node, content)
	case "optional":
		// z.optional(schema) wraps another schema - optional is tracked at property level
		return p.parseZodWrapped(node, content, false)
	case "nullable":
		return p.parseZodWrapped(node, content, true)
	case "lazy":
		// Lazy schemas reference themselves, return empty for now
		return &types.Schema{}
	default:
		// Unknown method, return empty schema
		return &types.Schema{}
	}
}

// parseZodObject parses z.object({...}).
func (p *ZodParser) parseZodObject(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{
		Type:       "object",
		Properties: make(map[string]*types.Schema),
	}

	// Get arguments
	args := p.getCallArguments(node)
	if len(args) == 0 {
		return schema
	}

	// First argument should be an object literal
	objArg := args[0]
	if objArg.Type() != "object" {
		return schema
	}

	var requiredFields []string

	// Walk the object properties
	p.walkNodes(objArg, func(n *sitter.Node) bool {
		if n.Type() == "pair" || n.Type() == "property" {
			name, propSchema, isOptional := p.parseObjectProperty(n, content)
			if name != "" {
				schema.Properties[name] = propSchema
				if !isOptional {
					requiredFields = append(requiredFields, name)
				}
			}
			return false
		}
		// Handle shorthand properties like { name } instead of { name: name }
		if n.Type() == "shorthand_property_identifier" {
			name := n.Content(content)
			schema.Properties[name] = &types.Schema{}
			return false
		}
		return true
	})

	if len(requiredFields) > 0 {
		schema.Required = requiredFields
	}

	return schema
}

// parseObjectProperty parses a property in a z.object().
func (p *ZodParser) parseObjectProperty(node *sitter.Node, content []byte) (string, *types.Schema, bool) {
	var name string
	var valueNode *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_identifier", "string":
			if name == "" {
				name = child.Content(content)
				// Remove quotes from string keys
				name = strings.Trim(name, `"'`)
			}
		case "call_expression":
			valueNode = child
		case "member_expression":
			// Could be a reference to another schema
			valueNode = child
		}
	}

	if valueNode == nil {
		return name, &types.Schema{}, false
	}

	propSchema := p.parseZodExpression(valueNode, content)
	isOptional := p.hasOptionalModifier(valueNode, content)

	return name, propSchema, isOptional
}

// hasOptionalModifier checks if the Zod expression has .optional() in its chain.
func (p *ZodParser) hasOptionalModifier(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}

	nodeContent := node.Content(content)
	return strings.Contains(nodeContent, ".optional()")
}

// parseZodArray parses z.array(schema).
func (p *ZodParser) parseZodArray(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{
		Type: "array",
	}

	args := p.getCallArguments(node)
	if len(args) > 0 {
		itemSchema := p.parseZodExpression(args[0], content)
		schema.Items = itemSchema
	}

	return schema
}

// parseZodEnum parses z.enum([...]).
func (p *ZodParser) parseZodEnum(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{
		Type: "string",
	}

	args := p.getCallArguments(node)
	if len(args) == 0 {
		return schema
	}

	// First argument should be an array
	arrArg := args[0]
	if arrArg.Type() != "array" {
		return schema
	}

	var enumValues []any
	p.walkNodes(arrArg, func(n *sitter.Node) bool {
		if n.Type() == "string" {
			val := n.Content(content)
			// Remove quotes
			val = strings.Trim(val, `"'`)
			enumValues = append(enumValues, val)
		}
		return true
	})

	if len(enumValues) > 0 {
		schema.Enum = enumValues
	}

	return schema
}

// parseZodLiteral parses z.literal(value).
func (p *ZodParser) parseZodLiteral(node *sitter.Node, content []byte) *types.Schema {
	args := p.getCallArguments(node)
	if len(args) == 0 {
		return &types.Schema{}
	}

	arg := args[0]
	argText := arg.Content(content)

	switch arg.Type() {
	case "string":
		val := strings.Trim(argText, `"'`)
		return &types.Schema{
			Type: "string",
			Enum: []any{val},
		}
	case "number":
		if v, err := strconv.ParseFloat(argText, 64); err == nil {
			if strings.Contains(argText, ".") {
				return &types.Schema{
					Type: "number",
					Enum: []any{v},
				}
			}
			return &types.Schema{
				Type: "integer",
				Enum: []any{int(v)},
			}
		}
	case "true":
		return &types.Schema{
			Type: "boolean",
			Enum: []any{true},
		}
	case "false":
		return &types.Schema{
			Type: "boolean",
			Enum: []any{false},
		}
	}

	return &types.Schema{}
}

// parseZodUnion parses z.union([...]).
func (p *ZodParser) parseZodUnion(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{}

	args := p.getCallArguments(node)
	if len(args) == 0 {
		return schema
	}

	// First argument should be an array of schemas
	arrArg := args[0]
	if arrArg.Type() != "array" {
		return schema
	}

	var oneOf []*types.Schema
	p.walkNodes(arrArg, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			itemSchema := p.parseZodExpression(n, content)
			oneOf = append(oneOf, itemSchema)
			return false
		}
		return true
	})

	if len(oneOf) > 0 {
		schema.OneOf = oneOf
	}

	return schema
}

// parseZodIntersection parses z.intersection(a, b).
func (p *ZodParser) parseZodIntersection(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{}

	args := p.getCallArguments(node)
	if len(args) < 2 {
		return schema
	}

	var allOf []*types.Schema
	for _, arg := range args {
		if arg.Type() == "call_expression" {
			itemSchema := p.parseZodExpression(arg, content)
			allOf = append(allOf, itemSchema)
		}
	}

	if len(allOf) > 0 {
		schema.AllOf = allOf
	}

	return schema
}

// parseZodTuple parses z.tuple([...]).
func (p *ZodParser) parseZodTuple(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{
		Type: "array",
	}

	// Tuples in OpenAPI are represented as arrays with prefixItems (JSON Schema draft 2020-12)
	// For OpenAPI 3.0, we use items with oneOf as a workaround

	args := p.getCallArguments(node)
	if len(args) == 0 {
		return schema
	}

	arrArg := args[0]
	if arrArg.Type() != "array" {
		return schema
	}

	var items []*types.Schema
	p.walkNodes(arrArg, func(n *sitter.Node) bool {
		if n.Type() == "call_expression" {
			itemSchema := p.parseZodExpression(n, content)
			items = append(items, itemSchema)
			return false
		}
		return true
	})

	if len(items) == 1 {
		schema.Items = items[0]
	} else if len(items) > 1 {
		// Use oneOf for multiple tuple types
		schema.Items = &types.Schema{
			OneOf: items,
		}
		minItems := len(items)
		maxItems := len(items)
		schema.MinItems = &minItems
		schema.MaxItems = &maxItems
	}

	return schema
}

// parseZodRecord parses z.record(keySchema, valueSchema) or z.record(valueSchema).
func (p *ZodParser) parseZodRecord(node *sitter.Node, content []byte) *types.Schema {
	schema := &types.Schema{
		Type: "object",
	}

	args := p.getCallArguments(node)
	if len(args) == 0 {
		return schema
	}

	// z.record(valueSchema) - single argument
	// z.record(keySchema, valueSchema) - two arguments
	var valueArg *sitter.Node
	if len(args) == 1 {
		valueArg = args[0]
	} else if len(args) >= 2 {
		valueArg = args[1]
	}

	if valueArg != nil && valueArg.Type() == "call_expression" {
		valueSchema := p.parseZodExpression(valueArg, content)
		schema.AdditionalProperties = valueSchema
	}

	return schema
}

// parseZodWrapped parses z.optional(schema) or z.nullable(schema).
// Note: isOptional is not needed here since optional is tracked at the property level in OpenAPI.
func (p *ZodParser) parseZodWrapped(node *sitter.Node, content []byte, isNullable bool) *types.Schema {
	args := p.getCallArguments(node)
	if len(args) == 0 {
		return &types.Schema{Nullable: isNullable}
	}

	schema := p.parseZodExpression(args[0], content)
	if isNullable {
		schema.Nullable = true
	}
	return schema
}

// parseZodMemberChain parses method chains like z.string().email().min(1).
// TODO: Implement full member chain parsing for edge cases.
func (p *ZodParser) parseZodMemberChain(_ *sitter.Node, _ []byte) *types.Schema {
	// This is called when we have something like z.string().email
	// The node is the member_expression, not the call
	// Most chains are handled via call_expression, this is a fallback
	return &types.Schema{}
}

// applyZodModifier applies a Zod modifier method to a schema.
func (p *ZodParser) applyZodModifier(schema *types.Schema, method string, callNode *sitter.Node, content []byte) *types.Schema {
	args := p.getCallArguments(callNode)

	switch method {
	case "optional":
		// Optional doesn't change the schema itself, just removes from required
		return schema
	case "nullable":
		schema.Nullable = true
	case "min":
		if len(args) > 0 {
			if v := p.extractNumber(args[0], content); v != nil {
				val := *v
				switch schema.Type {
				case "string":
					intVal := int(val)
					schema.MinLength = &intVal
				case "number", "integer":
					schema.Minimum = &val
				case "array":
					intVal := int(val)
					schema.MinItems = &intVal
				}
			}
		}
	case "max":
		if len(args) > 0 {
			if v := p.extractNumber(args[0], content); v != nil {
				val := *v
				switch schema.Type {
				case "string":
					intVal := int(val)
					schema.MaxLength = &intVal
				case "number", "integer":
					schema.Maximum = &val
				case "array":
					intVal := int(val)
					schema.MaxItems = &intVal
				}
			}
		}
	case "length":
		if len(args) > 0 {
			if v := p.extractNumber(args[0], content); v != nil {
				intVal := int(*v)
				switch schema.Type {
				case "string":
					schema.MinLength = &intVal
					schema.MaxLength = &intVal
				case "array":
					schema.MinItems = &intVal
					schema.MaxItems = &intVal
				}
			}
		}
	case "email":
		schema.Format = "email"
	case "url":
		schema.Format = "uri"
	case "uri":
		schema.Format = "uri"
	case "uuid":
		schema.Format = "uuid"
	case "cuid":
		schema.Format = "cuid"
	case "cuid2":
		schema.Format = "cuid2"
	case "ulid":
		schema.Format = "ulid"
	case "datetime":
		schema.Format = "date-time"
	case "date":
		schema.Format = "date"
	case "time":
		schema.Format = "time"
	case "duration":
		schema.Format = "duration"
	case "ip":
		schema.Format = "ip"
	case "ipv4":
		schema.Format = "ipv4"
	case "ipv6":
		schema.Format = "ipv6"
	case "emoji":
		// No standard format, skip
	case "regex":
		if len(args) > 0 {
			// First arg is a regex literal
			regexText := args[0].Content(content)
			// Remove /.../ delimiters if present
			regexText = strings.TrimPrefix(regexText, "/")
			if idx := strings.LastIndex(regexText, "/"); idx > 0 {
				regexText = regexText[:idx]
			}
			schema.Pattern = regexText
		}
	case "startsWith", "endsWith", "includes":
		// Could create a pattern, but for now skip
	case "int":
		schema.Type = "integer"
	case "positive":
		val := 1.0
		schema.Minimum = &val
		schema.ExclusiveMinimum = false
	case "negative":
		val := -1.0
		schema.Maximum = &val
		schema.ExclusiveMaximum = false
	case "nonnegative":
		val := 0.0
		schema.Minimum = &val
	case "nonpositive":
		val := 0.0
		schema.Maximum = &val
	case "multipleOf":
		if len(args) > 0 {
			if v := p.extractNumber(args[0], content); v != nil {
				schema.MultipleOf = v
			}
		}
	case "finite":
		// No direct OpenAPI equivalent
	case "safe":
		// No direct OpenAPI equivalent
	case "nonempty":
		val := 1
		switch schema.Type {
		case "string":
			schema.MinLength = &val
		case "array":
			schema.MinItems = &val
		}
	case "describe":
		if len(args) > 0 {
			desc := args[0].Content(content)
			desc = strings.Trim(desc, `"'`)
			schema.Description = desc
		}
	case "default":
		if len(args) > 0 {
			schema.Default = p.extractLiteralValue(args[0], content)
		}
	case "trim", "toLowerCase", "toUpperCase":
		// These are transformations, no schema impact
	case "catch":
		// Error handling, no schema impact
	case "brand":
		// Type branding, no schema impact
	case "readonly":
		schema.ReadOnly = true
	case "pipe", "transform":
		// Transformations, no schema impact
	case "refine", "superRefine":
		// Custom validation, no schema impact
	case "passthrough", "strict", "strip":
		// Object mode modifiers, no direct OpenAPI equivalent
	case "partial":
		// Make all properties optional - handled at object level
	case "required":
		// Make all properties required - handled at object level
	case "deepPartial":
		// Recursive partial - complex, skip for now
	case "pick", "omit":
		// Object property selection - would need type analysis
	case "extend", "merge":
		// Object extension - would need to merge schemas
	}

	return schema
}

// extractNumber extracts a number from a node.
func (p *ZodParser) extractNumber(node *sitter.Node, content []byte) *float64 {
	if node == nil {
		return nil
	}

	text := node.Content(content)
	if v, err := strconv.ParseFloat(text, 64); err == nil {
		return &v
	}
	return nil
}

// extractLiteralValue extracts a literal value from a node.
func (p *ZodParser) extractLiteralValue(node *sitter.Node, content []byte) any {
	if node == nil {
		return nil
	}

	text := node.Content(content)
	nodeType := node.Type()

	switch nodeType {
	case "string":
		return strings.Trim(text, `"'`)
	case "number":
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			if strings.Contains(text, ".") {
				return v
			}
			return int(v)
		}
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	case "array":
		// Could recursively extract, but for now return as string
		return text
	case "object":
		// Could recursively extract, but for now return as string
		return text
	}

	return text
}

// getCallArguments returns the arguments from a call_expression node.
func (p *ZodParser) getCallArguments(node *sitter.Node) []*sitter.Node {
	var args []*sitter.Node

	if node.Type() != "call_expression" {
		return args
	}

	// Find arguments node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "arguments" {
			// Collect argument expressions (skip punctuation)
			for j := 0; j < int(child.ChildCount()); j++ {
				arg := child.Child(j)
				if arg.Type() != "," && arg.Type() != "(" && arg.Type() != ")" {
					args = append(args, arg)
				}
			}
			break
		}
	}

	return args
}

// walkNodes walks all nodes in the tree, calling fn for each node.
func (p *ZodParser) walkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
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

// extractZodMethod extracts the Zod method name from a callee string.
func extractZodMethod(callee string) string {
	// Handle chained calls like z.string().email() - check this first
	// Count the dots to detect chains
	parts := strings.Split(callee, ".")
	if len(parts) > 2 {
		// This is a chain, return the last part
		return parts[len(parts)-1]
	}

	// Handle z.string(), z.object(), etc.
	if strings.HasPrefix(callee, "z.") && len(parts) >= 2 {
		return parts[1]
	}

	// Handle simple calls
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}

	return callee
}

// Registry returns the schema registry.
func (p *ZodParser) Registry() *Registry {
	return p.registry
}

// ExtractAndRegister parses a Zod schema and registers it with the given name.
func (p *ZodParser) ExtractAndRegister(name string, node *sitter.Node, content []byte) *types.Schema {
	schema, _ := p.ParseZodSchema(node, content)
	if schema != nil && name != "" {
		schema.Title = name
		p.registry.Add(name, schema)
	}
	return schema
}
