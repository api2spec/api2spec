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
	watchMode     string
	watchDebounce int
	watchOnChange string
)

var watchCmd = &cobra.Command{
	Use:   "watch [paths...]",
	Short: "Watch for file changes and regenerate specification",
	Long: `Watch for file changes and automatically regenerate the OpenAPI specification.

This command monitors your source files for changes and triggers a regeneration
when files are modified. It's useful during development to keep your API
documentation in sync with your code.

Example:
  api2spec watch                          # Watch current directory
  api2spec watch ./cmd ./internal         # Watch specific paths
  api2spec watch --debounce 1000          # Wait 1s before regenerating
  api2spec watch --on-change "make test"  # Run command after regeneration`,
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().StringVarP(&watchMode, "mode", "m", "full", "generation mode: full, routes-only, schemas-only")
	watchCmd.Flags().IntVar(&watchDebounce, "debounce", 500, "debounce duration in milliseconds")
	watchCmd.Flags().StringVar(&watchOnChange, "on-change", "", "command to run after regeneration")
}

func runWatch(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply command-line overrides
	if watchMode != "" {
		cfg.Generation.Mode = watchMode
	}
	if watchDebounce > 0 {
		cfg.Watch.Debounce = watchDebounce
	}
	if watchOnChange != "" {
		cfg.Watch.OnChange = watchOnChange
	}

	// Determine paths to watch
	paths := args
	if len(paths) == 0 {
		paths = cfg.Source.Paths
	}

	printVerbose("Watch configuration:")
	printVerbose("  Mode: %s", cfg.Generation.Mode)
	printVerbose("  Debounce: %dms", cfg.Watch.Debounce)
	if cfg.Watch.OnChange != "" {
		printVerbose("  On change: %s", cfg.Watch.OnChange)
	}
	printVerbose("  Paths: %s", strings.Join(paths, ", "))

	printInfo("Watching for changes in: %s", strings.Join(paths, ", "))
	printInfo("Press Ctrl+C to stop")

	// TODO: Implement actual watch logic
	printInfo("Watch command not yet implemented")

	return nil
}
