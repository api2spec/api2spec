// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package cli provides the command-line interface for api2spec.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Global flags
var (
	cfgFile   string
	output    string
	format    string
	framework string
	verbose   bool
	quiet     bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "api2spec",
	Short: "Code-first OpenAPI specification generator",
	Long: `api2spec is a code-first OpenAPI specification generator that extracts
route definitions and schema information from your Go source code.

It supports multiple web frameworks including Chi, Gin, Echo, Fiber,
Gorilla Mux, and the standard library net/http.

Example:
  api2spec generate                    # Generate OpenAPI spec from current directory
  api2spec init --framework chi        # Initialize a new config file
  api2spec check --strict              # Validate routes and spec
  api2spec watch                       # Watch for changes and regenerate`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: api2spec.yaml)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "output file path (default: openapi.yaml)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "", "output format: yaml, json (default: yaml)")
	rootCmd.PersistentFlags().StringVar(&framework, "framework", "", "web framework: chi, gin, echo, fiber, gorilla, stdlib")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(printCmd)
}

// GetConfigFile returns the config file path from the flag.
func GetConfigFile() string {
	return cfgFile
}

// GetOutput returns the output file path from the flag.
func GetOutput() string {
	return output
}

// GetFormat returns the output format from the flag.
func GetFormat() string {
	return format
}

// GetFramework returns the framework from the flag.
func GetFramework() string {
	return framework
}

// IsVerbose returns whether verbose output is enabled.
func IsVerbose() bool {
	return verbose
}

// IsQuiet returns whether quiet mode is enabled.
func IsQuiet() bool {
	return quiet
}

// printInfo prints a message if not in quiet mode.
func printInfo(format string, args ...interface{}) {
	if !quiet {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// printVerbose prints a message if verbose mode is enabled.
func printVerbose(format string, args ...interface{}) {
	if verbose && !quiet {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// printError prints an error message.
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
