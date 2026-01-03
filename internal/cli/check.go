// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/openapi"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// Exit codes for check command
const (
	ExitCodeMatch       = 0 // Spec matches implementation
	ExitCodeDifference  = 1 // Spec differs from implementation
	ExitCodeCheckError  = 2 // Error during analysis
)

var (
	checkStrict bool
	checkIgnore []string
	checkCI     bool
)

var checkCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "Check if spec matches current implementation",
	Long: `Check validates that your OpenAPI specification matches your current code.

This command generates a spec from your source code and compares it with
the existing spec file. It's useful for CI pipelines to ensure the spec
is always in sync with the implementation.

Exit codes:
  0  Spec matches implementation
  1  Spec differs from implementation
  2  Error during analysis

Example:
  api2spec check                      # Basic validation
  api2spec check --strict             # Fail on any difference (default)
  api2spec check --ci                 # CI mode with appropriate exit codes
  api2spec check --ignore paths       # Ignore path differences`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkStrict, "strict", true, "fail on any difference")
	checkCmd.Flags().StringSliceVar(&checkIgnore, "ignore", nil, "patterns to ignore in comparison (paths, schemas)")
	checkCmd.Flags().BoolVar(&checkCI, "ci", false, "CI mode: use exit codes for status")
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		if checkCI {
			os.Exit(ExitCodeCheckError)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply command-line overrides
	if output != "" {
		cfg.Output = output
	}
	if format != "" {
		cfg.Format = format
	}
	if framework != "" {
		cfg.Framework = framework
	}

	// Determine paths to check
	paths := args
	if len(paths) == 0 {
		paths = cfg.Source.Paths
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		if checkCI {
			os.Exit(ExitCodeCheckError)
		}
		return fmt.Errorf("invalid configuration: %w", err)
	}

	printVerbose("Check configuration:")
	printVerbose("  Strict mode: %t", checkStrict)
	printVerbose("  CI mode: %t", checkCI)
	if len(checkIgnore) > 0 {
		printVerbose("  Ignored patterns: %s", strings.Join(checkIgnore, ", "))
	}
	printVerbose("  Paths: %s", strings.Join(paths, ", "))
	printVerbose("  Spec file: %s", cfg.Output)

	// Check if spec file exists
	if _, err := os.Stat(cfg.Output); os.IsNotExist(err) {
		printError("Spec file not found: %s", cfg.Output)
		printInfo("Run 'api2spec generate' first to create the spec file")
		if checkCI {
			os.Exit(ExitCodeDifference)
		}
		return fmt.Errorf("spec file not found: %s", cfg.Output)
	}

	// Read existing spec
	existingSpec, err := openapi.ReadFile(cfg.Output)
	if err != nil {
		if checkCI {
			os.Exit(ExitCodeCheckError)
		}
		return fmt.Errorf("failed to read existing spec: %w", err)
	}

	// Generate spec from current code
	generatedSpec, err := generateSpecFromCode(cfg, paths)
	if err != nil {
		if checkCI {
			os.Exit(ExitCodeCheckError)
		}
		return fmt.Errorf("failed to generate spec from code: %w", err)
	}

	// Compare specs
	differ := openapi.NewDiffer()
	diffResult, err := differ.Diff(existingSpec, generatedSpec)
	if err != nil {
		if checkCI {
			os.Exit(ExitCodeCheckError)
		}
		return fmt.Errorf("failed to compare specs: %w", err)
	}

	// Apply ignore patterns
	diffResult = applyIgnorePatterns(diffResult, checkIgnore)

	// Report results
	if diffResult.IsEmpty() {
		printInfo("Spec is in sync with implementation")
		if checkCI {
			os.Exit(ExitCodeMatch)
		}
		return nil
	}

	// Print differences
	printInfo("Spec differs from implementation:\n")
	printInfo(diffResult.Summary)
	printInfo("")

	// Print detailed changes
	if len(diffResult.PathChanges) > 0 {
		printInfo("Path changes:")
		for _, change := range diffResult.PathChanges {
			symbol := getChangeSymbol(change.Type)
			printInfo("  %s %s %s", symbol, change.Method, change.Path)
		}
		printInfo("")
	}

	if len(diffResult.SchemaChanges) > 0 {
		printInfo("Schema changes:")
		for _, change := range diffResult.SchemaChanges {
			symbol := getChangeSymbol(change.Type)
			printInfo("  %s %s", symbol, change.Name)
		}
		printInfo("")
	}

	if diffResult.HasBreakingChanges {
		printError("Breaking changes detected!")
	}

	printInfo("Run 'api2spec generate' to update the spec file")

	if checkStrict || (checkCI && !diffResult.IsEmpty()) {
		if checkCI {
			os.Exit(ExitCodeDifference)
		}
		return fmt.Errorf("spec differs from implementation")
	}

	return nil
}

// generateSpecFromCode generates an OpenAPI spec from the source code.
func generateSpecFromCode(cfg *config.Config, paths []string) (*types.OpenAPI, error) {
	// Determine project root for framework detection
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("failed to determine project root: %w", err)
	}

	// Get or detect framework plugin
	var plugin plugins.FrameworkPlugin
	if cfg.Framework == "" || cfg.Framework == "auto" {
		plugin, err = plugins.Detect(projectRoot)
		if err != nil {
			printVerbose("Framework detection failed: %v", err)
		}
	} else {
		plugin = plugins.Get(cfg.Framework)
		if plugin == nil {
			return nil, fmt.Errorf("unknown framework %q", cfg.Framework)
		}
	}

	// Scan for source files
	scannerCfg := scanner.Config{
		IncludePatterns: cfg.Source.Include,
		ExcludePatterns: cfg.Source.Exclude,
	}

	var files []scanner.SourceFile
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
		}
		scannerCfg.BasePath = absPath
		s := scanner.New(scannerCfg)
		pathFiles, err := s.Scan()
		if err != nil {
			return nil, fmt.Errorf("failed to scan path %s: %w", path, err)
		}
		files = append(files, pathFiles...)
	}

	printVerbose("Scanned %d source files", len(files))

	// Extract routes and schemas
	var routes []types.Route
	var schemas []types.Schema

	if plugin != nil {
		if cfg.Generation.Mode == "full" || cfg.Generation.Mode == "routes-only" {
			extractedRoutes, err := plugin.ExtractRoutes(files)
			if err != nil {
				return nil, fmt.Errorf("failed to extract routes: %w", err)
			}
			routes = extractedRoutes
		}

		if cfg.Generation.Mode == "full" || cfg.Generation.Mode == "schemas-only" {
			extractedSchemas, err := plugin.ExtractSchemas(files)
			if err != nil {
				return nil, fmt.Errorf("failed to extract schemas: %w", err)
			}
			schemas = extractedSchemas
		}
	}

	printVerbose("Found %d routes and %d schemas", len(routes), len(schemas))

	// Build OpenAPI spec
	builder := openapi.NewBuilder(cfg)
	doc, err := builder.Build(routes, schemas)
	if err != nil {
		return nil, fmt.Errorf("failed to build OpenAPI spec: %w", err)
	}

	return doc, nil
}

// applyIgnorePatterns filters out changes that match ignore patterns.
func applyIgnorePatterns(result *openapi.DiffResult, patterns []string) *openapi.DiffResult {
	if len(patterns) == 0 {
		return result
	}

	filtered := &openapi.DiffResult{
		PathChanges:   make([]openapi.PathChange, 0),
		SchemaChanges: make([]openapi.SchemaChange, 0),
	}

	// Filter path changes
	for _, change := range result.PathChanges {
		if !matchesAnyPattern(change.Path, patterns) {
			filtered.PathChanges = append(filtered.PathChanges, change)
		}
	}

	// Filter schema changes
	for _, change := range result.SchemaChanges {
		if !matchesAnyPattern(change.Name, patterns) {
			filtered.SchemaChanges = append(filtered.SchemaChanges, change)
		}
	}

	// Recalculate breaking changes
	for _, change := range filtered.PathChanges {
		if change.Type == openapi.DiffTypeRemoved {
			filtered.HasBreakingChanges = true
			break
		}
	}
	if !filtered.HasBreakingChanges {
		for _, change := range filtered.SchemaChanges {
			if change.Type == openapi.DiffTypeRemoved {
				filtered.HasBreakingChanges = true
				break
			}
		}
	}

	// Regenerate summary
	filtered.Summary = generateFilteredSummary(filtered)

	return filtered
}

// matchesAnyPattern checks if a string matches any of the given patterns.
func matchesAnyPattern(s string, patterns []string) bool {
	for _, pattern := range patterns {
		// Simple prefix/suffix matching
		if strings.HasPrefix(pattern, "*") {
			if strings.HasSuffix(s, pattern[1:]) {
				return true
			}
		} else if strings.HasSuffix(pattern, "*") {
			if strings.HasPrefix(s, pattern[:len(pattern)-1]) {
				return true
			}
		} else if strings.Contains(pattern, "*") {
			// Use filepath.Match for glob patterns
			if matched, _ := filepath.Match(pattern, s); matched {
				return true
			}
		} else {
			// Exact match
			if s == pattern {
				return true
			}
		}
	}
	return false
}

// generateFilteredSummary generates a summary for filtered results.
func generateFilteredSummary(result *openapi.DiffResult) string {
	if result.IsEmpty() {
		return "No changes detected (after applying filters)"
	}

	var parts []string
	pathAdded, pathRemoved, pathModified := 0, 0, 0
	for _, c := range result.PathChanges {
		switch c.Type {
		case openapi.DiffTypeAdded:
			pathAdded++
		case openapi.DiffTypeRemoved:
			pathRemoved++
		case openapi.DiffTypeModified:
			pathModified++
		}
	}

	schemaAdded, schemaRemoved, schemaModified := 0, 0, 0
	for _, c := range result.SchemaChanges {
		switch c.Type {
		case openapi.DiffTypeAdded:
			schemaAdded++
		case openapi.DiffTypeRemoved:
			schemaRemoved++
		case openapi.DiffTypeModified:
			schemaModified++
		}
	}

	if pathAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) added", pathAdded))
	}
	if pathRemoved > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) removed", pathRemoved))
	}
	if pathModified > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) modified", pathModified))
	}
	if schemaAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) added", schemaAdded))
	}
	if schemaRemoved > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) removed", schemaRemoved))
	}
	if schemaModified > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) modified", schemaModified))
	}

	summary := strings.Join(parts, ", ")
	if result.HasBreakingChanges {
		summary += " [BREAKING CHANGES DETECTED]"
	}

	return summary
}

// getChangeSymbol returns a symbol for the change type.
func getChangeSymbol(t openapi.DiffType) string {
	switch t {
	case openapi.DiffTypeAdded:
		return "+"
	case openapi.DiffTypeRemoved:
		return "-"
	case openapi.DiffTypeModified:
		return "~"
	default:
		return " "
	}
}
