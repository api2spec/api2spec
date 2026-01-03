// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	outputFormat := format
	if outputFormat == "" {
		outputFormat = "yaml"
	}

	printVerbose("Print configuration:")
	printVerbose("  Format: %s", outputFormat)

	if len(args) > 0 {
		// Print existing file
		filePath := args[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}
		fmt.Print(string(data))
		return nil
	}

	// TODO: Implement generation and print logic
	printInfo("Print command not yet fully implemented")
	printInfo("Would generate and print OpenAPI spec in %s format", outputFormat)

	return nil
}
