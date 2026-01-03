// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Config holds scanner configuration.
type Config struct {
	// BasePath is the base directory for scanning (defaults to current directory)
	BasePath string

	// IncludePatterns are glob patterns for files to include (e.g., "**/*.go")
	IncludePatterns []string

	// ExcludePatterns are glob patterns for files to exclude (e.g., "vendor/**")
	ExcludePatterns []string

	// Extensions filters files by extension (e.g., []string{".go", ".ts"})
	// If empty, all supported extensions are included
	Extensions []string
}

// Scanner discovers source files in a project.
type Scanner struct {
	config Config
}

// New creates a new Scanner with the given configuration.
func New(config Config) *Scanner {
	// Apply defaults
	if config.BasePath == "" {
		config.BasePath = "."
	}
	if len(config.IncludePatterns) == 0 {
		config.IncludePatterns = []string{"**/*.go", "**/*.ts", "**/*.js"}
	}

	return &Scanner{
		config: config,
	}
}

// Scan discovers all source files matching the configuration.
func (s *Scanner) Scan() ([]SourceFile, error) {
	basePath, err := filepath.Abs(s.config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base path: %w", err)
	}
	return s.ScanPath(basePath)
}

// ScanPath scans a specific path for source files.
func (s *Scanner) ScanPath(path string) ([]SourceFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("path does not exist: %s", absPath)
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	// If path is a file, check if it matches and return it
	if !info.IsDir() {
		if s.shouldIncludeFile(absPath, info) {
			content, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %w", err)
			}
			return []SourceFile{
				{
					Path:     absPath,
					Language: DetectLanguage(absPath),
					Content:  content,
					ModTime:  info.ModTime(),
				},
			}, nil
		}
		return nil, nil
	}

	// Walk the directory
	var files []SourceFile
	err = filepath.WalkDir(absPath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip inaccessible paths
			return nil
		}

		// Skip directories (but continue walking)
		if d.IsDir() {
			// Check if directory should be excluded
			relPath, _ := filepath.Rel(absPath, filePath)
			if s.shouldExcludeDir(relPath) {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if s.shouldIncludeFile(filePath, info) {
			content, err := os.ReadFile(filePath)
			if err != nil {
				// Skip files we can't read
				return nil
			}
			files = append(files, SourceFile{
				Path:     filePath,
				Language: DetectLanguage(filePath),
				Content:  content,
				ModTime:  info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// ScanPaths scans multiple paths for source files.
func (s *Scanner) ScanPaths(paths []string) ([]SourceFile, error) {
	var allFiles []SourceFile
	seen := make(map[string]bool)

	for _, path := range paths {
		files, err := s.ScanPath(path)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !seen[f.Path] {
				seen[f.Path] = true
				allFiles = append(allFiles, f)
			}
		}
	}

	return allFiles, nil
}

// shouldIncludeFile checks if a file should be included based on patterns and extensions.
func (s *Scanner) shouldIncludeFile(filePath string, info fs.FileInfo) bool {
	// Skip directories
	if info.IsDir() {
		return false
	}

	// Check extension filter
	if len(s.config.Extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(filePath))
		found := false
		for _, e := range s.config.Extensions {
			if strings.ToLower(e) == ext {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	} else {
		// Use default supported extensions
		if !IsSupportedFile(filePath) {
			return false
		}
	}

	// Get relative path for pattern matching
	basePath, _ := filepath.Abs(s.config.BasePath)
	relPath, err := filepath.Rel(basePath, filePath)
	if err != nil {
		relPath = filepath.Base(filePath)
	}

	// Normalize path separators for pattern matching
	relPath = filepath.ToSlash(relPath)

	// Check exclude patterns first
	if s.matchesPatterns(relPath, s.config.ExcludePatterns) {
		return false
	}

	// Check include patterns
	if len(s.config.IncludePatterns) > 0 {
		return s.matchesPatterns(relPath, s.config.IncludePatterns)
	}

	return true
}

// shouldExcludeDir checks if a directory should be excluded.
func (s *Scanner) shouldExcludeDir(relPath string) bool {
	if relPath == "" || relPath == "." {
		return false
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Check each exclude pattern
	for _, pattern := range s.config.ExcludePatterns {
		// Check if the directory matches the start of an exclude pattern
		// e.g., "vendor" matches "vendor/**"
		dirPattern := strings.TrimSuffix(pattern, "/**")
		dirPattern = strings.TrimSuffix(dirPattern, "/*")

		if relPath == dirPattern {
			return true
		}

		// Also check if the pattern would match any file in this directory
		matched, _ := doublestar.Match(pattern, relPath+"/dummy.go")
		if matched {
			return true
		}
	}

	return false
}

// matchesPatterns checks if a path matches any of the given patterns.
func (s *Scanner) matchesPatterns(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			// Invalid pattern, skip
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// FileCount returns a quick count of matching files without reading content.
func (s *Scanner) FileCount() (int, error) {
	basePath, err := filepath.Abs(s.config.BasePath)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve base path: %w", err)
	}

	count := 0
	err = filepath.WalkDir(basePath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			relPath, _ := filepath.Rel(basePath, filePath)
			if s.shouldExcludeDir(relPath) {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if s.shouldIncludeFile(filePath, info) {
			count++
		}

		return nil
	})

	return count, err
}
