// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package config provides configuration loading and validation for api2spec.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the api2spec configuration.
type Config struct {
	// Framework is the web framework to use (chi, gin, echo, fiber, gorilla, stdlib)
	Framework string `mapstructure:"framework" yaml:"framework" json:"framework"`

	// Output is the output file path for the generated OpenAPI spec
	Output string `mapstructure:"output" yaml:"output" json:"output"`

	// Format is the output format (yaml, json)
	Format string `mapstructure:"format" yaml:"format" json:"format"`

	// OpenAPI contains OpenAPI-specific configuration
	OpenAPI OpenAPIConfig `mapstructure:"openapi" yaml:"openapi" json:"openapi"`

	// Source contains source code scanning configuration
	Source SourceConfig `mapstructure:"source" yaml:"source" json:"source"`

	// Generation contains generation behavior configuration
	Generation GenerationConfig `mapstructure:"generation" yaml:"generation" json:"generation"`

	// Watch contains file watching configuration
	Watch WatchConfig `mapstructure:"watch" yaml:"watch" json:"watch"`
}

// OpenAPIConfig contains OpenAPI specification configuration.
type OpenAPIConfig struct {
	// Version is the OpenAPI version to generate (3.0.3, 3.1.0)
	Version string `mapstructure:"version" yaml:"version" json:"version"`

	// Info contains API metadata
	Info InfoConfig `mapstructure:"info" yaml:"info" json:"info"`

	// Servers is a list of server configurations
	Servers []ServerConfig `mapstructure:"servers" yaml:"servers" json:"servers"`

	// Tags is a list of tag configurations
	Tags []TagConfig `mapstructure:"tags" yaml:"tags" json:"tags"`

	// Security contains security scheme configurations
	Security SecurityConfig `mapstructure:"security" yaml:"security" json:"security"`
}

// InfoConfig contains API metadata.
type InfoConfig struct {
	// Title is the API title
	Title string `mapstructure:"title" yaml:"title" json:"title"`

	// Description is the API description
	Description string `mapstructure:"description" yaml:"description" json:"description"`

	// Version is the API version
	Version string `mapstructure:"version" yaml:"version" json:"version"`

	// TermsOfService is the URL to terms of service
	TermsOfService string `mapstructure:"termsOfService" yaml:"termsOfService" json:"termsOfService"`

	// Contact contains contact information
	Contact ContactConfig `mapstructure:"contact" yaml:"contact" json:"contact"`

	// License contains license information
	License LicenseConfig `mapstructure:"license" yaml:"license" json:"license"`
}

// ContactConfig contains contact information.
type ContactConfig struct {
	// Name is the contact name
	Name string `mapstructure:"name" yaml:"name" json:"name"`

	// URL is the contact URL
	URL string `mapstructure:"url" yaml:"url" json:"url"`

	// Email is the contact email
	Email string `mapstructure:"email" yaml:"email" json:"email"`
}

// LicenseConfig contains license information.
type LicenseConfig struct {
	// Name is the license name
	Name string `mapstructure:"name" yaml:"name" json:"name"`

	// URL is the license URL
	URL string `mapstructure:"url" yaml:"url" json:"url"`
}

// ServerConfig contains server configuration.
type ServerConfig struct {
	// URL is the server URL
	URL string `mapstructure:"url" yaml:"url" json:"url"`

	// Description is the server description
	Description string `mapstructure:"description" yaml:"description" json:"description"`
}

// TagConfig contains tag configuration.
type TagConfig struct {
	// Name is the tag name
	Name string `mapstructure:"name" yaml:"name" json:"name"`

	// Description is the tag description
	Description string `mapstructure:"description" yaml:"description" json:"description"`
}

// SecurityConfig contains security configuration.
type SecurityConfig struct {
	// Schemes is a map of security scheme configurations
	Schemes map[string]SecuritySchemeConfig `mapstructure:"schemes" yaml:"schemes" json:"schemes"`

	// Default is a list of default security requirements
	Default []string `mapstructure:"default" yaml:"default" json:"default"`
}

// SecuritySchemeConfig contains security scheme configuration.
type SecuritySchemeConfig struct {
	// Type is the security scheme type (apiKey, http, oauth2, openIdConnect)
	Type string `mapstructure:"type" yaml:"type" json:"type"`

	// Name is the name of the header, query, or cookie parameter
	Name string `mapstructure:"name" yaml:"name" json:"name"`

	// In is the location (header, query, cookie)
	In string `mapstructure:"in" yaml:"in" json:"in"`

	// Scheme is the HTTP authorization scheme (bearer, basic)
	Scheme string `mapstructure:"scheme" yaml:"scheme" json:"scheme"`

	// BearerFormat is the format of the bearer token
	BearerFormat string `mapstructure:"bearerFormat" yaml:"bearerFormat" json:"bearerFormat"`

	// Description is a description of the security scheme
	Description string `mapstructure:"description" yaml:"description" json:"description"`
}

// SourceConfig contains source code scanning configuration.
type SourceConfig struct {
	// Paths is a list of paths to scan
	Paths []string `mapstructure:"paths" yaml:"paths" json:"paths"`

	// Include is a list of glob patterns to include
	Include []string `mapstructure:"include" yaml:"include" json:"include"`

	// Exclude is a list of glob patterns to exclude
	Exclude []string `mapstructure:"exclude" yaml:"exclude" json:"exclude"`
}

// GenerationConfig contains generation behavior configuration.
type GenerationConfig struct {
	// Mode is the generation mode (full, routes-only, schemas-only)
	Mode string `mapstructure:"mode" yaml:"mode" json:"mode"`

	// Merge determines whether to merge with existing spec
	Merge bool `mapstructure:"merge" yaml:"merge" json:"merge"`

	// StrictMode enables strict validation during generation
	StrictMode bool `mapstructure:"strictMode" yaml:"strictMode" json:"strictMode"`

	// DefaultResponses is a list of default response codes to include
	DefaultResponses []string `mapstructure:"defaultResponses" yaml:"defaultResponses" json:"defaultResponses"`
}

// WatchConfig contains file watching configuration.
type WatchConfig struct {
	// Enabled determines whether to enable file watching
	Enabled bool `mapstructure:"enabled" yaml:"enabled" json:"enabled"`

	// Debounce is the debounce duration in milliseconds
	Debounce int `mapstructure:"debounce" yaml:"debounce" json:"debounce"`

	// OnChange is the command to run on change
	OnChange string `mapstructure:"onChange" yaml:"onChange" json:"onChange"`
}

// configFileNames is the list of config file names to search for (in order).
var configFileNames = []string{
	"api2spec.yaml",
	"api2spec.json",
	".api2spec.yaml",
	".api2spec.json",
}

// supportedFrameworks is the list of supported frameworks.
var supportedFrameworks = []string{
	// Special
	"auto",
	// Go
	"chi",
	"gin",
	"echo",
	"fiber",
	"gorilla",
	"stdlib",
	// JavaScript/TypeScript
	"express",
	"fastify",
	"koa",
	"hono",
	"elysia",
	"nestjs",
	// Python
	"flask",
	"fastapi",
	// Rust
	"axum",
	"actix",
	"rocket",
	// JVM
	"spring",
	"ktor",
	// Ruby
	"rails",
	"sinatra",
	// PHP
	"laravel",
	// Elixir
	"phoenix",
	// .NET
	"aspnet",
	// Gleam
	"gleam",
}

// supportedFormats is the list of supported output formats.
var supportedFormats = []string{
	"yaml",
	"json",
}

// supportedModes is the list of supported generation modes.
var supportedModes = []string{
	"full",
	"routes-only",
	"schemas-only",
}

// ErrConfigNotFound is returned when no config file is found.
var ErrConfigNotFound = errors.New("config file not found")

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation error: %s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors.
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	var sb strings.Builder
	sb.WriteString("config validation errors:\n")
	for _, err := range e {
		sb.WriteString("  - ")
		sb.WriteString(err.Field)
		sb.WriteString(": ")
		sb.WriteString(err.Message)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Framework: "auto",
		Output:    "openapi.yaml",
		Format:    "yaml",
		OpenAPI: OpenAPIConfig{
			Version: "3.0.3",
			Info: InfoConfig{
				Title:   "API",
				Version: "1.0.0",
			},
		},
		Source: SourceConfig{
			Paths:   []string{"."},
			Include: []string{"**/*.go", "**/*.ts", "**/*.js", "**/*.py", "**/*.rs", "**/*.java", "**/*.kt", "**/*.rb", "**/*.php", "**/*.ex", "**/*.exs", "**/*.cs", "**/*.gleam"},
			Exclude: []string{
				"vendor/**",
				"**/*_test.go",
				"**/testdata/**",
				"node_modules/**",
				".git/**",
				"dist/**",
				"build/**",
				"target/**",
				"**/*.pb.go",
				"**/mock*.go",
				"**/mocks/**",
			},
		},
		Generation: GenerationConfig{
			Mode:             "full",
			Merge:            false,
			StrictMode:       false,
			DefaultResponses: []string{"200", "400", "500"},
		},
		Watch: WatchConfig{
			Enabled:  false,
			Debounce: 500,
		},
	}
}

// Load loads the configuration from a file.
// It searches for config files in the following order:
// 1. api2spec.yaml
// 2. api2spec.json
// 3. .api2spec.yaml
// 4. .api2spec.json
//
// If configPath is provided, it will use that path instead.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	if configPath != "" {
		// Use the provided config path
		v.SetConfigFile(configPath)
	} else {
		// Search for config files in order
		found := false
		for _, name := range configFileNames {
			if _, err := os.Stat(name); err == nil {
				v.SetConfigFile(name)
				found = true
				break
			}
		}
		if !found {
			// Return default config if no file found
			return Default(), nil
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// LoadFromPath loads the configuration from a specific directory.
func LoadFromPath(dir string) (*Config, error) {
	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}
	return Default(), nil
}

// setDefaults sets the default values for viper.
func setDefaults(v *viper.Viper) {
	v.SetDefault("framework", "auto")
	v.SetDefault("output", "openapi.yaml")
	v.SetDefault("format", "yaml")
	v.SetDefault("openapi.version", "3.0.3")
	v.SetDefault("openapi.info.title", "API")
	v.SetDefault("openapi.info.version", "1.0.0")
	v.SetDefault("source.paths", []string{"."})
	v.SetDefault("source.include", []string{"**/*.go", "**/*.ts", "**/*.js", "**/*.py", "**/*.rs", "**/*.java", "**/*.kt", "**/*.rb", "**/*.php", "**/*.ex", "**/*.exs", "**/*.cs", "**/*.gleam"})
	v.SetDefault("source.exclude", []string{
		"vendor/**",
		"**/*_test.go",
		"**/testdata/**",
		"node_modules/**",
		".git/**",
		"dist/**",
		"build/**",
		"target/**",
		"**/*.pb.go",
		"**/mock*.go",
		"**/mocks/**",
	})
	v.SetDefault("generation.mode", "full")
	v.SetDefault("generation.merge", false)
	v.SetDefault("generation.strictMode", false)
	v.SetDefault("generation.defaultResponses", []string{"200", "400", "500"})
	v.SetDefault("watch.enabled", false)
	v.SetDefault("watch.debounce", 500)
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	var errs ValidationErrors

	// Validate framework
	if c.Framework != "" && !contains(supportedFrameworks, c.Framework) {
		errs = append(errs, ValidationError{
			Field:   "framework",
			Message: fmt.Sprintf("unsupported framework %q, must be one of: %s", c.Framework, strings.Join(supportedFrameworks, ", ")),
		})
	}

	// Validate format
	if c.Format != "" && !contains(supportedFormats, c.Format) {
		errs = append(errs, ValidationError{
			Field:   "format",
			Message: fmt.Sprintf("unsupported format %q, must be one of: %s", c.Format, strings.Join(supportedFormats, ", ")),
		})
	}

	// Validate generation mode
	if c.Generation.Mode != "" && !contains(supportedModes, c.Generation.Mode) {
		errs = append(errs, ValidationError{
			Field:   "generation.mode",
			Message: fmt.Sprintf("unsupported mode %q, must be one of: %s", c.Generation.Mode, strings.Join(supportedModes, ", ")),
		})
	}

	// Validate OpenAPI version
	if c.OpenAPI.Version != "" {
		if c.OpenAPI.Version != "3.0.3" && c.OpenAPI.Version != "3.1.0" {
			errs = append(errs, ValidationError{
				Field:   "openapi.version",
				Message: fmt.Sprintf("unsupported OpenAPI version %q, must be 3.0.3 or 3.1.0", c.OpenAPI.Version),
			})
		}
	}

	// Validate watch debounce
	if c.Watch.Debounce < 0 {
		errs = append(errs, ValidationError{
			Field:   "watch.debounce",
			Message: "debounce must be non-negative",
		})
	}

	// Validate required fields
	if c.OpenAPI.Info.Title == "" {
		errs = append(errs, ValidationError{
			Field:   "openapi.info.title",
			Message: "title is required",
		})
	}

	if c.OpenAPI.Info.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "openapi.info.version",
			Message: "version is required",
		})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// ConfigFilePath returns the path of the loaded config file, if any.
func ConfigFilePath() string {
	for _, name := range configFileNames {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
