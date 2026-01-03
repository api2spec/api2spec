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
	checkStrict bool
	checkIgnore []string
	checkCI     bool
)

var checkCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "Check routes and validate OpenAPI specification",
	Long: `Check validates your route definitions and OpenAPI specification.

This command performs the following checks:
  - Route syntax and structure validation
  - Schema completeness and correctness
  - OpenAPI specification compliance
  - Detection of missing or incomplete documentation

Use --strict mode for comprehensive validation suitable for CI pipelines.

Example:
  api2spec check                      # Basic validation
  api2spec check --strict             # Strict validation
  api2spec check --ci                 # CI mode (exit code reflects status)
  api2spec check --ignore E001,E002   # Ignore specific error codes`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkStrict, "strict", false, "enable strict validation mode")
	checkCmd.Flags().StringSliceVar(&checkIgnore, "ignore", nil, "error codes to ignore")
	checkCmd.Flags().BoolVar(&checkCI, "ci", false, "CI mode: non-zero exit on warnings")
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Determine paths to check
	paths := args
	if len(paths) == 0 {
		paths = cfg.Source.Paths
	}

	printVerbose("Check configuration:")
	printVerbose("  Strict mode: %t", checkStrict)
	printVerbose("  CI mode: %t", checkCI)
	if len(checkIgnore) > 0 {
		printVerbose("  Ignored errors: %s", strings.Join(checkIgnore, ", "))
	}
	printVerbose("  Paths: %s", strings.Join(paths, ", "))

	// TODO: Implement actual check logic
	printInfo("Check command not yet implemented")
	printInfo("Would validate routes in paths: %s", strings.Join(paths, ", "))

	return nil
}
