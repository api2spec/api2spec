// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package plugins provides framework plugin infrastructure for route and schema extraction.
package plugins

import (
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// FrameworkPlugin defines the interface for framework-specific route and schema extraction.
type FrameworkPlugin interface {
	// Name returns the plugin identifier (e.g., "chi", "gin", "echo").
	Name() string

	// Extensions returns the file extensions this plugin handles (e.g., []string{".go"}).
	Extensions() []string

	// Detect checks if this framework is used in the project.
	// It typically looks for framework imports in go.mod or source files.
	Detect(projectRoot string) (bool, error)

	// ExtractRoutes parses source files and extracts route definitions.
	// The returned routes include path, method, handler info, and parameters.
	ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error)

	// ExtractSchemas parses source files and extracts schema definitions.
	// The returned schemas are derived from struct definitions with json tags.
	ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error)
}

// PluginInfo provides metadata about a plugin.
type PluginInfo struct {
	// Name is the plugin identifier
	Name string

	// Version is the plugin version
	Version string

	// Description describes the plugin's purpose
	Description string

	// SupportedFrameworks lists framework versions supported by this plugin
	SupportedFrameworks []string
}

// InfoProvider is an optional interface plugins can implement to provide metadata.
type InfoProvider interface {
	// Info returns plugin metadata.
	Info() PluginInfo
}

// RouteExtractor is a minimal interface for plugins that only extract routes.
type RouteExtractor interface {
	ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error)
}

// SchemaExtractor is a minimal interface for plugins that only extract schemas.
type SchemaExtractor interface {
	ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error)
}
