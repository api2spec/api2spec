// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	assert.Equal(t, "chi", cfg.Framework)
	assert.Equal(t, "openapi.yaml", cfg.Output)
	assert.Equal(t, "yaml", cfg.Format)
	assert.Equal(t, "3.0.3", cfg.OpenAPI.Version)
	assert.Equal(t, "API", cfg.OpenAPI.Info.Title)
	assert.Equal(t, "1.0.0", cfg.OpenAPI.Info.Version)
	assert.Equal(t, "full", cfg.Generation.Mode)
	assert.False(t, cfg.Generation.Merge)
	assert.False(t, cfg.Watch.Enabled)
	assert.Equal(t, 500, cfg.Watch.Debounce)
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Create a temp directory with no config file
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)

	// Should return default config
	assert.Equal(t, "chi", cfg.Framework)
	assert.Equal(t, "openapi.yaml", cfg.Output)
}

func TestLoad_YAMLConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	configContent := `
framework: gin
output: api.yaml
format: yaml
openapi:
  version: "3.1.0"
  info:
    title: "My API"
    version: "2.0.0"
    description: "A test API"
generation:
  mode: routes-only
  merge: true
`
	configPath := filepath.Join(tmpDir, "api2spec.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "gin", cfg.Framework)
	assert.Equal(t, "api.yaml", cfg.Output)
	assert.Equal(t, "yaml", cfg.Format)
	assert.Equal(t, "3.1.0", cfg.OpenAPI.Version)
	assert.Equal(t, "My API", cfg.OpenAPI.Info.Title)
	assert.Equal(t, "2.0.0", cfg.OpenAPI.Info.Version)
	assert.Equal(t, "A test API", cfg.OpenAPI.Info.Description)
	assert.Equal(t, "routes-only", cfg.Generation.Mode)
	assert.True(t, cfg.Generation.Merge)
}

func TestLoad_JSONConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	configContent := `{
  "framework": "echo",
  "output": "openapi.json",
  "format": "json",
  "openapi": {
    "version": "3.0.3",
    "info": {
      "title": "Echo API",
      "version": "1.0.0"
    }
  }
}`
	configPath := filepath.Join(tmpDir, "api2spec.json")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "echo", cfg.Framework)
	assert.Equal(t, "openapi.json", cfg.Output)
	assert.Equal(t, "json", cfg.Format)
}

func TestLoad_DotPrefixedConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	configContent := `
framework: fiber
output: spec.yaml
`
	configPath := filepath.Join(tmpDir, ".api2spec.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "fiber", cfg.Framework)
	assert.Equal(t, "spec.yaml", cfg.Output)
}

func TestLoad_ExplicitConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
framework: gorilla
output: custom.yaml
`
	configPath := filepath.Join(tmpDir, "custom-config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, "gorilla", cfg.Framework)
	assert.Equal(t, "custom.yaml", cfg.Output)
}

func TestLoad_ConfigFilePriority(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create both api2spec.yaml and .api2spec.yaml
	// api2spec.yaml should take priority
	err = os.WriteFile(filepath.Join(tmpDir, "api2spec.yaml"), []byte("framework: chi\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, ".api2spec.yaml"), []byte("framework: gin\n"), 0644)
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := Load("")
	require.NoError(t, err)

	assert.Equal(t, "chi", cfg.Framework)
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := Default()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_InvalidFramework(t *testing.T) {
	cfg := Default()
	cfg.Framework = "invalid-framework"

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "framework", valErrs[0].Field)
}

func TestValidate_InvalidFormat(t *testing.T) {
	cfg := Default()
	cfg.Format = "xml"

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "format", valErrs[0].Field)
}

func TestValidate_InvalidMode(t *testing.T) {
	cfg := Default()
	cfg.Generation.Mode = "invalid-mode"

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "generation.mode", valErrs[0].Field)
}

func TestValidate_InvalidOpenAPIVersion(t *testing.T) {
	cfg := Default()
	cfg.OpenAPI.Version = "2.0"

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "openapi.version", valErrs[0].Field)
}

func TestValidate_NegativeDebounce(t *testing.T) {
	cfg := Default()
	cfg.Watch.Debounce = -1

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "watch.debounce", valErrs[0].Field)
}

func TestValidate_MissingTitle(t *testing.T) {
	cfg := Default()
	cfg.OpenAPI.Info.Title = ""

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "openapi.info.title", valErrs[0].Field)
}

func TestValidate_MissingVersion(t *testing.T) {
	cfg := Default()
	cfg.OpenAPI.Info.Version = ""

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 1)
	assert.Equal(t, "openapi.info.version", valErrs[0].Field)
}

func TestValidate_MultipleErrors(t *testing.T) {
	cfg := Default()
	cfg.Framework = "invalid"
	cfg.Format = "xml"
	cfg.Generation.Mode = "bad"

	err := cfg.Validate()
	require.Error(t, err)

	var valErrs ValidationErrors
	require.ErrorAs(t, err, &valErrs)
	assert.Len(t, valErrs, 3)
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Field:   "framework",
		Message: "unsupported framework",
	}
	assert.Contains(t, err.Error(), "framework")
	assert.Contains(t, err.Error(), "unsupported framework")
}

func TestValidationErrors_Error(t *testing.T) {
	errs := ValidationErrors{
		{Field: "field1", Message: "error1"},
		{Field: "field2", Message: "error2"},
	}
	errStr := errs.Error()
	assert.Contains(t, errStr, "field1")
	assert.Contains(t, errStr, "error1")
	assert.Contains(t, errStr, "field2")
	assert.Contains(t, errStr, "error2")
}

func TestValidationErrors_ErrorEmpty(t *testing.T) {
	errs := ValidationErrors{}
	assert.Equal(t, "no validation errors", errs.Error())
}

func TestValidationErrors_ErrorSingle(t *testing.T) {
	errs := ValidationErrors{
		{Field: "field1", Message: "error1"},
	}
	// Single error should use the ValidationError format
	assert.Contains(t, errs.Error(), "config validation error")
}

func TestLoadFromPath(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := `
framework: stdlib
output: stdlib-api.yaml
`
	err := os.WriteFile(filepath.Join(tmpDir, "api2spec.yaml"), []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadFromPath(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "stdlib", cfg.Framework)
	assert.Equal(t, "stdlib-api.yaml", cfg.Output)
}

func TestLoadFromPath_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := LoadFromPath(tmpDir)
	require.NoError(t, err)

	// Should return default config
	assert.Equal(t, "chi", cfg.Framework)
}
