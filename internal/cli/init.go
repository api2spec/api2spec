// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/api2spec/api2spec/internal/config"
)

var (
	initFramework string
	initForce     bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new api2spec configuration file",
	Long: `Initialize a new api2spec configuration file in the current directory.

This command creates an api2spec.yaml file with sensible defaults
that you can customize for your project.

Example:
  api2spec init                      # Create default config
  api2spec init --framework gin      # Create config for Gin framework
  api2spec init --force              # Overwrite existing config`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initFramework, "framework", "chi", "web framework to use (chi, gin, echo, fiber, gorilla, stdlib)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
}

func runInit(cmd *cobra.Command, args []string) error {
	configFile := "api2spec.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil && !initForce {
		return fmt.Errorf("config file %s already exists, use --force to overwrite", configFile)
	}

	// Use framework from flag or global flag
	fw := initFramework
	if framework != "" {
		fw = framework
	}

	// Validate framework
	validFrameworks := map[string]bool{
		"chi":    true,
		"gin":    true,
		"echo":   true,
		"fiber":  true,
		"gorilla": true,
		"stdlib": true,
	}
	if !validFrameworks[fw] {
		return fmt.Errorf("unsupported framework %q, must be one of: chi, gin, echo, fiber, gorilla, stdlib", fw)
	}

	// Create default config
	cfg := config.Default()
	cfg.Framework = fw

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	printInfo("Created %s", configFile)
	printVerbose("Framework: %s", fw)
	printVerbose("Output: %s", cfg.Output)

	return nil
}
