// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToLowerCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single lowercase", "a", "a"},
		{"single uppercase", "A", "a"},
		{"PascalCase", "UserName", "userName"},
		{"already camelCase", "userName", "userName"},
		{"all uppercase", "ID", "iD"},
		{"acronym prefix", "HTTPRequest", "hTTPRequest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLowerCamelCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractInnerType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple type", "string", "string"},
		{"array type", "User[]", "User"},
		{"generic List", "List<User>", "User"},
		{"generic IEnumerable", "IEnumerable<Product>", "Product"},
		{"nested generic", "Dictionary<string, User>", "string, User"},
		{"generic with spaces", "List< User >", "User"},
		{"no generic markers", "string", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractInnerType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
