// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/openapi"
	"github.com/api2spec/api2spec/pkg/types"
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
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
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

	printVerbose("Diff configuration:")
	printVerbose("  Color: %t", diffColor)
	printVerbose("  Unified context: %d", diffUnified)
	printVerbose("  Side-by-side: %t", diffSideBySide)

	var specA, specB *types.OpenAPI
	var labelA, labelB string

	switch len(args) {
	case 0:
		// Compare existing spec against generated
		printVerbose("Comparing existing spec against generated...")

		// Check if spec file exists
		if _, err := os.Stat(cfg.Output); os.IsNotExist(err) {
			return fmt.Errorf("spec file not found: %s. Run 'api2spec generate' first", cfg.Output)
		}

		specA, err = openapi.ReadFile(cfg.Output)
		if err != nil {
			return fmt.Errorf("failed to read existing spec: %w", err)
		}
		labelA = cfg.Output

		specB, err = generateSpecFromCode(cfg, cfg.Source.Paths)
		if err != nil {
			return fmt.Errorf("failed to generate spec from code: %w", err)
		}
		labelB = "<generated>"

	case 1:
		// Compare provided file against generated
		printVerbose("Comparing %s against generated...", args[0])

		specA, err = openapi.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read spec file %s: %w", args[0], err)
		}
		labelA = args[0]

		specB, err = generateSpecFromCode(cfg, cfg.Source.Paths)
		if err != nil {
			return fmt.Errorf("failed to generate spec from code: %w", err)
		}
		labelB = "<generated>"

	case 2:
		// Compare two files
		printVerbose("Comparing %s against %s...", args[0], args[1])

		specA, err = openapi.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read spec file %s: %w", args[0], err)
		}
		labelA = args[0]

		specB, err = openapi.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("failed to read spec file %s: %w", args[1], err)
		}
		labelB = args[1]

	default:
		return fmt.Errorf("too many arguments: expected at most 2 files")
	}

	// Perform diff
	differ := openapi.NewDiffer()
	result, err := differ.Diff(specA, specB)
	if err != nil {
		return fmt.Errorf("failed to compare specs: %w", err)
	}

	// Print results
	if result.IsEmpty() {
		printInfo("No differences found between %s and %s", labelA, labelB)
		return nil
	}

	if diffSideBySide {
		printSideBySideDiff(result, labelA, labelB, diffColor)
	} else {
		printUnifiedDiff(result, labelA, labelB, diffColor)
	}

	return nil
}

// printUnifiedDiff prints the diff in unified format.
func printUnifiedDiff(result *openapi.DiffResult, labelA, labelB string, color bool) {
	// Header
	fmt.Printf("--- %s\n", labelA)
	fmt.Printf("+++ %s\n", labelB)
	fmt.Println()

	// Summary
	fmt.Println(result.Summary)
	fmt.Println()

	// Path changes
	if len(result.PathChanges) > 0 {
		fmt.Println("=== Paths ===")

		// Sort for deterministic output
		changes := make([]openapi.PathChange, len(result.PathChanges))
		copy(changes, result.PathChanges)
		sort.Slice(changes, func(i, j int) bool {
			if changes[i].Path != changes[j].Path {
				return changes[i].Path < changes[j].Path
			}
			return changes[i].Method < changes[j].Method
		})

		for _, change := range changes {
			line := fmt.Sprintf("%s %s", change.Method, change.Path)
			printDiffLine(change.Type, line, color)
		}
		fmt.Println()
	}

	// Schema changes
	if len(result.SchemaChanges) > 0 {
		fmt.Println("=== Schemas ===")

		// Sort for deterministic output
		changes := make([]openapi.SchemaChange, len(result.SchemaChanges))
		copy(changes, result.SchemaChanges)
		sort.Slice(changes, func(i, j int) bool {
			return changes[i].Name < changes[j].Name
		})

		for _, change := range changes {
			printDiffLine(change.Type, change.Name, color)
		}
		fmt.Println()
	}

	// Breaking changes warning
	if result.HasBreakingChanges {
		if color {
			fmt.Println("\033[1;31mWARNING: Breaking changes detected!\033[0m")
		} else {
			fmt.Println("WARNING: Breaking changes detected!")
		}
	}
}

// printSideBySideDiff prints the diff in side-by-side format.
func printSideBySideDiff(result *openapi.DiffResult, labelA, labelB string, color bool) {
	// Calculate column width
	const columnWidth = 40
	separator := strings.Repeat("-", columnWidth)

	// Header
	fmt.Printf("%-*s | %-*s\n", columnWidth, labelA, columnWidth, labelB)
	fmt.Printf("%s-+-%s\n", separator, separator)

	// Path changes
	if len(result.PathChanges) > 0 {
		fmt.Println()
		fmt.Printf("%-*s | %-*s\n", columnWidth, "=== Paths (old)", columnWidth, "=== Paths (new)")
		fmt.Printf("%s-+-%s\n", separator, separator)

		// Sort for deterministic output
		changes := make([]openapi.PathChange, len(result.PathChanges))
		copy(changes, result.PathChanges)
		sort.Slice(changes, func(i, j int) bool {
			if changes[i].Path != changes[j].Path {
				return changes[i].Path < changes[j].Path
			}
			return changes[i].Method < changes[j].Method
		})

		for _, change := range changes {
			path := fmt.Sprintf("%s %s", change.Method, change.Path)
			left := ""
			right := ""

			switch change.Type {
			case openapi.DiffTypeAdded:
				right = path
			case openapi.DiffTypeRemoved:
				left = path
			case openapi.DiffTypeModified:
				left = path
				right = path + " (modified)"
			}

			// Truncate if necessary
			if len(left) > columnWidth {
				left = left[:columnWidth-3] + "..."
			}
			if len(right) > columnWidth {
				right = right[:columnWidth-3] + "..."
			}

			printSideBySideLine(left, right, change.Type, columnWidth, color)
		}
	}

	// Schema changes
	if len(result.SchemaChanges) > 0 {
		fmt.Println()
		fmt.Printf("%-*s | %-*s\n", columnWidth, "=== Schemas (old)", columnWidth, "=== Schemas (new)")
		fmt.Printf("%s-+-%s\n", separator, separator)

		// Sort for deterministic output
		changes := make([]openapi.SchemaChange, len(result.SchemaChanges))
		copy(changes, result.SchemaChanges)
		sort.Slice(changes, func(i, j int) bool {
			return changes[i].Name < changes[j].Name
		})

		for _, change := range changes {
			left := ""
			right := ""

			switch change.Type {
			case openapi.DiffTypeAdded:
				right = change.Name
			case openapi.DiffTypeRemoved:
				left = change.Name
			case openapi.DiffTypeModified:
				left = change.Name
				right = change.Name + " (modified)"
			}

			// Truncate if necessary
			if len(left) > columnWidth {
				left = left[:columnWidth-3] + "..."
			}
			if len(right) > columnWidth {
				right = right[:columnWidth-3] + "..."
			}

			printSideBySideLine(left, right, change.Type, columnWidth, color)
		}
	}

	fmt.Println()

	// Breaking changes warning
	if result.HasBreakingChanges {
		if color {
			fmt.Println("\033[1;31mWARNING: Breaking changes detected!\033[0m")
		} else {
			fmt.Println("WARNING: Breaking changes detected!")
		}
	}
}

// printDiffLine prints a single diff line with appropriate formatting.
func printDiffLine(diffType openapi.DiffType, content string, color bool) {
	var prefix string
	var colorCode string
	var resetCode string

	switch diffType {
	case openapi.DiffTypeAdded:
		prefix = "+"
		colorCode = "\033[32m" // Green
	case openapi.DiffTypeRemoved:
		prefix = "-"
		colorCode = "\033[31m" // Red
	case openapi.DiffTypeModified:
		prefix = "~"
		colorCode = "\033[33m" // Yellow
	default:
		prefix = " "
		colorCode = ""
	}

	if color && colorCode != "" {
		resetCode = "\033[0m"
	} else {
		colorCode = ""
		resetCode = ""
	}

	fmt.Printf("%s%s %s%s\n", colorCode, prefix, content, resetCode)
}

// printSideBySideLine prints a side-by-side comparison line.
func printSideBySideLine(left, right string, diffType openapi.DiffType, width int, color bool) {
	var colorCode, resetCode string

	if color {
		switch diffType {
		case openapi.DiffTypeAdded:
			colorCode = "\033[32m" // Green
		case openapi.DiffTypeRemoved:
			colorCode = "\033[31m" // Red
		case openapi.DiffTypeModified:
			colorCode = "\033[33m" // Yellow
		}
		if colorCode != "" {
			resetCode = "\033[0m"
		}
	}

	fmt.Printf("%s%-*s | %-*s%s\n", colorCode, width, left, width, right, resetCode)
}
