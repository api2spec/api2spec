// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
)

var (
	generateMode    string
	generateMerge   bool
	generateDryRun  bool
	generateInclude []string
	generateExclude []string
)

var generateCmd = &cobra.Command{
	Use:   "generate [paths...]",
	Short: "Generate OpenAPI specification from source code",
	Long: `Generate an OpenAPI specification by analyzing your Go source code.

The generate command scans your source files, extracts route definitions,
and produces an OpenAPI 3.0/3.1 specification document.

Modes:
  full         Generate complete spec with routes and schemas (default)
  routes-only  Generate only route definitions
  schemas-only Generate only schema definitions

Example:
  api2spec generate                           # Generate from current directory
  api2spec generate ./cmd ./internal          # Generate from specific paths
  api2spec generate --mode routes-only        # Generate routes only
  api2spec generate --merge                   # Merge with existing spec
  api2spec generate --dry-run                 # Preview without writing`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&generateMode, "mode", "m", "full", "generation mode: full, routes-only, schemas-only")
	generateCmd.Flags().BoolVar(&generateMerge, "merge", false, "merge with existing spec file")
	generateCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "preview output without writing to file")
	generateCmd.Flags().StringSliceVarP(&generateInclude, "include", "i", nil, "glob patterns to include")
	generateCmd.Flags().StringSliceVarP(&generateExclude, "exclude", "e", nil, "glob patterns to exclude")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply command-line overrides
	if generateMode != "" {
		cfg.Generation.Mode = generateMode
	}
	if generateMerge {
		cfg.Generation.Merge = true
	}
	if len(generateInclude) > 0 {
		cfg.Source.Include = generateInclude
	}
	if len(generateExclude) > 0 {
		cfg.Source.Exclude = generateExclude
	}
	if output != "" {
		cfg.Output = output
	}
	if format != "" {
		cfg.Format = format
	}
	if framework != "" {
		cfg.Framework = framework
	}

	// Determine paths to scan
	paths := args
	if len(paths) == 0 {
		paths = cfg.Source.Paths
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	printVerbose("Configuration:")
	printVerbose("  Framework: %s", cfg.Framework)
	printVerbose("  Mode: %s", cfg.Generation.Mode)
	printVerbose("  Output: %s", cfg.Output)
	printVerbose("  Format: %s", cfg.Format)
	printVerbose("  Paths: %s", strings.Join(paths, ", "))

	if generateDryRun {
		printInfo("Dry run mode - no files will be written")
	}

	// TODO: Implement actual generation logic
	printInfo("Generate command not yet implemented")
	printInfo("Would generate OpenAPI spec from paths: %s", strings.Join(paths, ", "))

	return nil
}
