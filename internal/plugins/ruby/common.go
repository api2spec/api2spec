// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package ruby provides shared utilities for Ruby framework plugins.
package ruby

import (
	"regexp"
	"strings"

	"github.com/api2spec/api2spec/pkg/types"
)

// BraceParamRegex matches OpenAPI-style path parameters like {param}.
var BraceParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// symbolKeyRegex matches modern Ruby hash syntax: key: value.
var symbolKeyRegex = regexp.MustCompile(`(\w+):\s*([^,}\n]+)`)

// rocketKeyRegex matches old-style Ruby hash syntax: :key => value.
var rocketKeyRegex = regexp.MustCompile(`:(\w+)\s*=>\s*([^,}\n]+)`)

// integerRegex matches integer values.
var integerRegex = regexp.MustCompile(`^\d+$`)

// floatRegex matches floating-point values.
var floatRegex = regexp.MustCompile(`^\d+\.\d+$`)

// ExtractHashProperties extracts properties from a Ruby hash literal.
// It handles both modern (key: value) and rocket (:key => value) syntax.
// skipKeys allows callers to specify keys to ignore (e.g., "status" in certain contexts).
func ExtractHashProperties(hashContent string, skipKeys map[string]bool) map[string]*types.Schema {
	props := make(map[string]*types.Schema)

	// Extract symbol key pairs (modern Ruby syntax)
	symbolMatches := symbolKeyRegex.FindAllStringSubmatch(hashContent, -1)
	for _, match := range symbolMatches {
		if len(match) < 3 {
			continue
		}
		key := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])

		// Check skip list
		if skipKeys != nil && skipKeys[key] {
			continue
		}

		propSchema := InferPropertyType(key, value)
		props[key] = propSchema
	}

	// Extract rocket-style pairs (:key => value)
	rocketMatches := rocketKeyRegex.FindAllStringSubmatch(hashContent, -1)
	for _, match := range rocketMatches {
		if len(match) < 3 {
			continue
		}
		key := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])

		// Check skip list
		if skipKeys != nil && skipKeys[key] {
			continue
		}

		propSchema := InferPropertyType(key, value)
		props[key] = propSchema
	}

	return props
}

// InferPropertyType infers the JSON Schema type from a Ruby value or property name.
// It uses both the value content and naming conventions to determine the type.
func InferPropertyType(key, value string) *types.Schema {
	schema := &types.Schema{Type: "string"}

	// Infer from value
	value = strings.TrimSpace(value)
	if value != "" {
		// Check for numeric values
		if integerRegex.MatchString(value) {
			schema.Type = "integer"
			return schema
		}
		if floatRegex.MatchString(value) {
			schema.Type = "number"
			return schema
		}
		// Check for boolean values
		if value == "true" || value == "false" {
			schema.Type = "boolean"
			return schema
		}
		// Check for nil
		if value == "nil" {
			schema.Nullable = true
			return schema
		}
	}

	// Infer from common key naming conventions
	keyLower := strings.ToLower(key)
	switch {
	case keyLower == "id" || strings.HasSuffix(keyLower, "_id") || strings.HasSuffix(keyLower, "id"):
		schema.Type = "integer"
	case keyLower == "email":
		schema.Type = "string"
		schema.Format = "email"
	case keyLower == "url" || strings.HasSuffix(keyLower, "_url"):
		schema.Type = "string"
		schema.Format = "uri"
	case keyLower == "uuid" || strings.HasSuffix(keyLower, "_uuid"):
		schema.Type = "string"
		schema.Format = "uuid"
	case strings.Contains(keyLower, "date") || strings.HasSuffix(keyLower, "_at"):
		schema.Type = "string"
		schema.Format = "date-time"
	case strings.HasPrefix(keyLower, "is_") || strings.HasPrefix(keyLower, "has_") || strings.HasPrefix(keyLower, "can_"):
		schema.Type = "boolean"
	case keyLower == "count" || keyLower == "total" || strings.HasSuffix(keyLower, "_count"):
		schema.Type = "integer"
	case keyLower == "price" || keyLower == "amount" || strings.HasSuffix(keyLower, "_price") || strings.HasSuffix(keyLower, "_amount"):
		schema.Type = "number"
	}

	return schema
}
