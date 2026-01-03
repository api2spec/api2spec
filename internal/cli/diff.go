// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	diffColor      bool
	diffUnified    int
	diffSideBySide bool
)

var diffCmd = &cobra.Command{
	Use:   "diff [file1] [file2]",
	Short: "Compare two OpenAPI specifications",
	Long: `Compare two OpenAPI specifications and show the differences.

If only one file is provided, it will be compared against the generated
specification from the current source code.

If no files are provided, the existing spec file will be compared against
what would be generated from the current source code.

Example:
  api2spec diff                           # Compare current vs generated
  api2spec diff openapi.yaml              # Compare file vs generated
  api2spec diff old.yaml new.yaml         # Compare two files
  api2spec diff --side-by-side            # Side-by-side comparison
  api2spec diff --unified 5               # Show 5 lines of context`,
	RunE: runDiff,
}

func init() {
	diffCmd.Flags().BoolVar(&diffColor, "color", true, "enable colored output")
	diffCmd.Flags().IntVarP(&diffUnified, "unified", "U", 3, "number of context lines in unified diff")
	diffCmd.Flags().BoolVarP(&diffSideBySide, "side-by-side", "y", false, "side-by-side comparison")
}

func runDiff(cmd *cobra.Command, args []string) error {
	printVerbose("Diff configuration:")
	printVerbose("  Color: %t", diffColor)
	printVerbose("  Unified context: %d", diffUnified)
	printVerbose("  Side-by-side: %t", diffSideBySide)

	switch len(args) {
	case 0:
		printInfo("Comparing existing spec against generated...")
	case 1:
		printInfo("Comparing %s against generated...", args[0])
	case 2:
		printInfo("Comparing %s against %s...", args[0], args[1])
	default:
		return fmt.Errorf("too many arguments: expected at most 2 files")
	}

	// TODO: Implement actual diff logic
	printInfo("Diff command not yet implemented")

	return nil
}
