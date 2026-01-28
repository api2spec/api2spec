// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package scanner provides file discovery for source code scanning.
package scanner

import (
	"path/filepath"
	"strings"
	"time"
)

// SourceFile represents a discovered source file.
type SourceFile struct {
	// Path is the absolute path to the file
	Path string

	// Language is the detected programming language ("go", "typescript", "javascript", "python", "rust")
	Language string

	// Content is the file content
	Content []byte

	// ModTime is the last modification time
	ModTime time.Time
}

// languageExtensions maps file extensions to language identifiers.
var languageExtensions = map[string]string{
	".go":    "go",
	".ts":    "typescript",
	".tsx":   "typescript",
	".mts":   "typescript",
	".cts":   "typescript",
	".js":    "javascript",
	".jsx":   "javascript",
	".mjs":   "javascript",
	".cjs":   "javascript",
	".py":    "python",
	".pyw":   "python",
	".rs":    "rust",
	".cs":    "csharp",
	".php":   "php",
	".java":  "java",
	".kt":    "kotlin",
	".kts":   "kotlin",
	".ex":    "elixir",
	".exs":   "elixir",
	".rb":    "ruby",
	".gleam": "gleam",
	".cpp":   "cpp",
	".hpp":   "cpp",
	".h":     "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".scala": "scala",
	".sc":    "scala",
	".swift": "swift",
	".hs":    "haskell",
	".lhs":   "haskell",
	".yaml":  "yaml",
	".yml":   "yaml",
}

// DetectLanguage detects the programming language from a file path.
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := languageExtensions[ext]; ok {
		return lang
	}
	return ""
}

// SupportedExtensions returns a list of supported file extensions.
func SupportedExtensions() []string {
	exts := make([]string, 0, len(languageExtensions))
	for ext := range languageExtensions {
		exts = append(exts, ext)
	}
	return exts
}

// IsSupportedFile checks if a file path has a supported extension.
func IsSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := languageExtensions[ext]
	return ok
}
