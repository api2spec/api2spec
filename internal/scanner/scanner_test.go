// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temporary directory with test files.
func setupTestDir(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	return tmpDir
}

func TestNew_DefaultConfig(t *testing.T) {
	scanner := New(Config{})

	assert.NotNil(t, scanner)
	assert.Equal(t, ".", scanner.config.BasePath)
	assert.NotEmpty(t, scanner.config.IncludePatterns)
}

func TestNew_CustomConfig(t *testing.T) {
	scanner := New(Config{
		BasePath:        "/custom/path",
		IncludePatterns: []string{"**/*.go"},
		ExcludePatterns: []string{"vendor/**"},
	})

	assert.Equal(t, "/custom/path", scanner.config.BasePath)
	assert.Equal(t, []string{"**/*.go"}, scanner.config.IncludePatterns)
	assert.Equal(t, []string{"vendor/**"}, scanner.config.ExcludePatterns)
}

func TestScanner_Scan_BasicFiles(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":        "package main",
		"handler.go":     "package main",
		"internal/api.go": "package internal",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Verify all files are Go files
	for _, f := range files {
		assert.Equal(t, "go", f.Language)
		assert.NotEmpty(t, f.Content)
		assert.False(t, f.ModTime.IsZero())
	}
}

func TestScanner_Scan_ExcludePatterns(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":            "package main",
		"main_test.go":       "package main",
		"vendor/dep/dep.go":  "package dep",
		"internal/api.go":    "package internal",
		"internal/api_test.go": "package internal",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
		ExcludePatterns: []string{"vendor/**", "**/*_test.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Verify no test files or vendor files
	for _, f := range files {
		assert.NotContains(t, f.Path, "_test.go")
		assert.NotContains(t, f.Path, "vendor")
	}
}

func TestScanner_Scan_MultipleLanguages(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":        "package main",
		"handler.ts":     "export const handler = () => {}",
		"utils.js":       "module.exports = {}",
		"readme.md":      "# README",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go", "**/*.ts", "**/*.js"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 3)

	// Check languages
	languages := make(map[string]int)
	for _, f := range files {
		languages[f.Language]++
	}

	assert.Equal(t, 1, languages["go"])
	assert.Equal(t, 1, languages["typescript"])
	assert.Equal(t, 1, languages["javascript"])
}

func TestScanner_Scan_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestScanner_Scan_NoMatchingFiles(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"readme.md":  "# README",
		"config.yml": "key: value",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestScanner_Scan_NestedDirectories(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"cmd/api/main.go":              "package main",
		"internal/handler/user.go":     "package handler",
		"internal/handler/post.go":     "package handler",
		"internal/model/user.go":       "package model",
		"pkg/utils/strings.go":         "package utils",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 5)
}

func TestScanner_ScanPath_SingleFile(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go": "package main",
	})

	scanner := New(Config{
		BasePath: tmpDir,
	})

	files, err := scanner.ScanPath(filepath.Join(tmpDir, "main.go"))
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "go", files[0].Language)
}

func TestScanner_ScanPath_NonexistentPath(t *testing.T) {
	scanner := New(Config{})

	_, err := scanner.ScanPath("/nonexistent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestScanner_ScanPaths_MultiplePaths(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"cmd/main.go":         "package main",
		"internal/api.go":     "package internal",
		"pkg/utils.go":        "package pkg",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	paths := []string{
		filepath.Join(tmpDir, "cmd"),
		filepath.Join(tmpDir, "internal"),
	}

	files, err := scanner.ScanPaths(paths)
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestScanner_ScanPaths_DeduplicatesFiles(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go": "package main",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
	})

	// Scan the same path twice
	paths := []string{tmpDir, tmpDir}

	files, err := scanner.ScanPaths(paths)
	require.NoError(t, err)
	assert.Len(t, files, 1)
}

func TestScanner_Scan_ExtensionFilter(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":     "package main",
		"handler.ts":  "export {}",
		"utils.js":    "module.exports = {}",
	})

	scanner := New(Config{
		BasePath:   tmpDir,
		Extensions: []string{".go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "go", files[0].Language)
}

func TestScanner_FileCount(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":        "package main",
		"handler.go":     "package main",
		"main_test.go":   "package main",
		"vendor/dep.go":  "package dep",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.go"},
		ExcludePatterns: []string{"vendor/**", "**/*_test.go"},
	})

	count, err := scanner.FileCount()
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestScanner_Scan_SpecificPatterns(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"main.go":             "package main",
		"internal/handler.go": "package internal",
		"cmd/api/main.go":     "package main",
		"scripts/build.go":    "package scripts",
	})

	// Only scan internal directory
	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"internal/**/*.go"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0].Path, "internal")
}

func TestScanner_Scan_TypeScriptFiles(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"index.ts":           "export {}",
		"components/App.tsx": "export const App = () => {}",
		"utils.mjs":          "export default {}",
	})

	scanner := New(Config{
		BasePath:        tmpDir,
		IncludePatterns: []string{"**/*.ts", "**/*.tsx", "**/*.mjs"},
	})

	files, err := scanner.Scan()
	require.NoError(t, err)
	assert.Len(t, files, 3)

	for _, f := range files {
		if filepath.Ext(f.Path) == ".mjs" {
			assert.Equal(t, "javascript", f.Language)
		} else {
			assert.Equal(t, "typescript", f.Language)
		}
	}
}
