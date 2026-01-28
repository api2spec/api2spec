// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package scanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"handler.Go", "go"},
		{"app.ts", "typescript"},
		{"component.tsx", "typescript"},
		{"utils.js", "javascript"},
		{"app.jsx", "javascript"},
		{"module.mjs", "javascript"},
		{"common.cjs", "javascript"},
		{"readme.md", ""},
		{"config.yaml", "yaml"},
		{"routes.yml", "yaml"},
		{"Makefile", ""},
		{"/path/to/file.go", "go"},
		{"/path/to/file.ts", "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DetectLanguage(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := SupportedExtensions()

	assert.NotEmpty(t, exts)
	assert.Contains(t, exts, ".go")
	assert.Contains(t, exts, ".ts")
	assert.Contains(t, exts, ".js")
}

func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"app.ts", true},
		{"component.tsx", true},
		{"utils.js", true},
		{"app.jsx", true},
		{"module.mjs", true},
		{"common.cjs", true},
		{"readme.md", false},
		{"config.yaml", true},
		{"routes.yml", true},
		{"Makefile", false},
		{"image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsSupportedFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
