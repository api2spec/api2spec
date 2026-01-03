// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/api2spec/api2spec/pkg/types"
)

// Writer handles writing OpenAPI documents to various outputs.
type Writer struct {
	// Indent specifies the indentation for JSON output (default: 2 spaces)
	Indent int
}

// NewWriter creates a new Writer with default settings.
func NewWriter() *Writer {
	return &Writer{
		Indent: 2,
	}
}

// WriteYAML writes an OpenAPI document as YAML to the given writer.
func (w *Writer) WriteYAML(doc *types.OpenAPI, out io.Writer) error {
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	defer encoder.Close()

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// WriteJSON writes an OpenAPI document as JSON to the given writer.
func (w *Writer) WriteJSON(doc *types.OpenAPI, out io.Writer) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", strings.Repeat(" ", w.Indent))

	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// WriteFile writes an OpenAPI document to a file.
// The format is determined by the format parameter ("yaml" or "json").
// If format is empty, it is inferred from the file extension.
func (w *Writer) WriteFile(doc *types.OpenAPI, path string, format string) error {
	// Infer format from extension if not specified
	if format == "" {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yaml", ".yml":
			format = "yaml"
		case ".json":
			format = "json"
		default:
			format = "yaml" // Default to YAML
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write content
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return w.WriteYAML(doc, file)
	case "json":
		return w.WriteJSON(doc, file)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ToYAML returns the YAML representation of an OpenAPI document as a string.
func (w *Writer) ToYAML(doc *types.OpenAPI) (string, error) {
	var buf strings.Builder
	if err := w.WriteYAML(doc, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ToJSON returns the JSON representation of an OpenAPI document as a string.
func (w *Writer) ToJSON(doc *types.OpenAPI) (string, error) {
	var buf strings.Builder
	if err := w.WriteJSON(doc, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ReadFile reads an OpenAPI document from a file.
// The format is inferred from the file extension.
func ReadFile(path string) (*types.OpenAPI, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	var doc types.OpenAPI
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &doc); err != nil {
			if err := json.Unmarshal(data, &doc); err != nil {
				return nil, fmt.Errorf("failed to parse file as YAML or JSON")
			}
		}
	}

	return &doc, nil
}
