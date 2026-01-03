// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/pkg/types"
)

func createTestDoc() *types.OpenAPI {
	return &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:       "Test API",
			Description: "A test API",
			Version:     "1.0.0",
		},
		Servers: []types.Server{
			{URL: "https://api.example.com", Description: "Production"},
		},
		Paths: map[string]types.PathItem{
			"/users": {
				Get: &types.Operation{
					Summary: "List users",
					Responses: map[string]types.Response{
						"200": {Description: "Success"},
					},
				},
			},
		},
	}
}

func TestNewWriter(t *testing.T) {
	writer := NewWriter()
	assert.NotNil(t, writer)
	assert.Equal(t, 2, writer.Indent)
}

func TestWriter_WriteYAML(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	var buf bytes.Buffer
	err := writer.WriteYAML(doc, &buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "openapi: 3.0.3")
	assert.Contains(t, output, "title: Test API")
	assert.Contains(t, output, "version: 1.0.0")
	assert.Contains(t, output, "/users:")
}

func TestWriter_WriteJSON(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	var buf bytes.Buffer
	err := writer.WriteJSON(doc, &buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"openapi": "3.0.3"`)
	assert.Contains(t, output, `"title": "Test API"`)
	assert.Contains(t, output, `"version": "1.0.0"`)
	assert.Contains(t, output, `"/users":`)
}

func TestWriter_WriteFile_YAML(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "openapi.yaml")

	err := writer.WriteFile(doc, path, "yaml")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(content), "openapi:")
	assert.Contains(t, string(content), "title: Test API")
}

func TestWriter_WriteFile_JSON(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "openapi.json")

	err := writer.WriteFile(doc, path, "json")
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Contains(t, string(content), `"openapi":`)
	assert.Contains(t, string(content), `"title": "Test API"`)
}

func TestWriter_WriteFile_InferFormat(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()
	tmpDir := t.TempDir()

	tests := []struct {
		filename string
		contains string
	}{
		{"spec.yaml", "openapi:"},
		{"spec.yml", "openapi:"},
		{"spec.json", `"openapi":`},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			err := writer.WriteFile(doc, path, "")
			require.NoError(t, err)

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Contains(t, string(content), tt.contains)
		})
	}
}

func TestWriter_WriteFile_CreatesDirectory(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "dir", "openapi.yaml")

	err := writer.WriteFile(doc, path, "yaml")
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestWriter_WriteFile_UnsupportedFormat(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "openapi.txt")

	err := writer.WriteFile(doc, path, "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestWriter_ToYAML(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	output, err := writer.ToYAML(doc)
	require.NoError(t, err)

	assert.Contains(t, output, "openapi:")
	assert.Contains(t, output, "title: Test API")
}

func TestWriter_ToJSON(t *testing.T) {
	writer := NewWriter()
	doc := createTestDoc()

	output, err := writer.ToJSON(doc)
	require.NoError(t, err)

	assert.Contains(t, output, `"openapi":`)
	assert.Contains(t, output, `"title": "Test API"`)
}

func TestReadFile_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.yaml")

	content := `openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0
paths: {}`

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	doc, err := ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "Test API", doc.Info.Title)
}

func TestReadFile_JSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.json")

	content := `{
  "openapi": "3.0.3",
  "info": {
    "title": "Test API",
    "version": "1.0.0"
  },
  "paths": {}
}`

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	doc, err := ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "Test API", doc.Info.Title)
}

func TestReadFile_NonExistent(t *testing.T) {
	_, err := ReadFile("/nonexistent/path/spec.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestReadFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.yaml")

	err := os.WriteFile(path, []byte("invalid: yaml: content: {"), 0o644)
	require.NoError(t, err)

	_, err = ReadFile(path)
	assert.Error(t, err)
}

func TestReadFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.json")

	err := os.WriteFile(path, []byte("{invalid json}"), 0o644)
	require.NoError(t, err)

	_, err = ReadFile(path)
	assert.Error(t, err)
}

func TestReadFile_UnknownExtension(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.txt")

	// Should try YAML first
	content := `openapi: "3.0.3"
info:
  title: Test API
  version: 1.0.0`

	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)

	doc, err := ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "Test API", doc.Info.Title)
}

func TestWriter_WriteYAML_WithSchemas(t *testing.T) {
	writer := NewWriter()
	doc := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {
					Type: "object",
					Properties: map[string]*types.Schema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
					Required: []string{"id"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := writer.WriteYAML(doc, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "components:")
	assert.Contains(t, output, "schemas:")
	assert.Contains(t, output, "User:")
}

func TestWriter_CustomIndent(t *testing.T) {
	writer := &Writer{Indent: 4}
	doc := createTestDoc()

	var buf bytes.Buffer
	err := writer.WriteJSON(doc, &buf)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Find a line that should be indented
	for _, line := range lines {
		if strings.Contains(line, `"title"`) {
			// Should have 4-space indentation
			assert.True(t, strings.HasPrefix(line, "    "))
			break
		}
	}
}

func TestRoundTrip_YAML(t *testing.T) {
	writer := NewWriter()
	original := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.yaml")

	err := writer.WriteFile(original, path, "yaml")
	require.NoError(t, err)

	loaded, err := ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, original.OpenAPI, loaded.OpenAPI)
	assert.Equal(t, original.Info.Title, loaded.Info.Title)
	assert.Equal(t, original.Info.Version, loaded.Info.Version)
	assert.Equal(t, len(original.Servers), len(loaded.Servers))
	assert.Equal(t, len(original.Paths), len(loaded.Paths))
}

func TestRoundTrip_JSON(t *testing.T) {
	writer := NewWriter()
	original := createTestDoc()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "spec.json")

	err := writer.WriteFile(original, path, "json")
	require.NoError(t, err)

	loaded, err := ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, original.OpenAPI, loaded.OpenAPI)
	assert.Equal(t, original.Info.Title, loaded.Info.Title)
	assert.Equal(t, original.Info.Version, loaded.Info.Version)
}
