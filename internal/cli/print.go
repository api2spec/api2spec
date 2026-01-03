// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/openapi"
	"github.com/api2spec/api2spec/pkg/types"
)

var printCmd = &cobra.Command{
	Use:   "print [file]",
	Short: "Print the OpenAPI specification to stdout",
	Long: `Print the OpenAPI specification to standard output.

If a file is provided, it will print that file. Otherwise, it will
generate and print the specification from the current source code.

This is useful for piping the output to other tools or for quick inspection.

Example:
  api2spec print                      # Generate and print
  api2spec print openapi.yaml         # Print existing file
  api2spec print -f json              # Print in JSON format
  api2spec print | jq '.paths'        # Pipe to jq for processing`,
	RunE: runPrint,
}

func runPrint(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply command-line overrides
	if format != "" {
		cfg.Format = format
	}
	if framework != "" {
		cfg.Framework = framework
	}

	outputFormat := cfg.Format
	if outputFormat == "" {
		outputFormat = "yaml"
	}

	printVerbose("Print configuration:")
	printVerbose("  Format: %s", outputFormat)

	var spec *types.OpenAPI

	if len(args) > 0 {
		// Print existing file
		filePath := args[0]
		printVerbose("Reading spec from: %s", filePath)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		spec, err = openapi.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
	} else {
		// Generate spec from code
		printVerbose("Generating spec from source code...")

		spec, err = generateSpecFromCode(cfg, cfg.Source.Paths)
		if err != nil {
			return fmt.Errorf("failed to generate spec: %w", err)
		}
	}

	// Write to stdout
	writer := openapi.NewWriter()

	var output string
	switch outputFormat {
	case "json":
		output, err = writer.ToJSON(spec)
	default:
		output, err = writer.ToYAML(spec)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize spec: %w", err)
	}

	// Print to stdout (without using printInfo to avoid newline issues)
	fmt.Print(output)

	return nil
}
