// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/plugins"
)

var (
	initFramework   string
	initForce       bool
	initInteractive bool
	initTitle       string
	initVersion     string
	initDescription string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new api2spec configuration file",
	Long: `Initialize a new api2spec configuration file in the current directory.

This command creates an api2spec.yaml file with sensible defaults
that you can customize for your project.

Features:
  - Auto-detects your web framework from go.mod
  - Infers API title from module name
  - Detects common entry point patterns
  - Sets up appropriate exclude patterns

Example:
  api2spec init                         # Auto-detect framework and create config
  api2spec init --framework gin         # Create config for Gin framework
  api2spec init --force                 # Overwrite existing config
  api2spec init --interactive           # Interactive mode with prompts
  api2spec init --title "My API"        # Set custom API title`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initFramework, "framework", "", "web framework to use. If not specified, auto-detects from project files")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
	initCmd.Flags().BoolVarP(&initInteractive, "interactive", "i", false, "interactive mode with prompts")
	initCmd.Flags().StringVar(&initTitle, "title", "", "API title for OpenAPI info")
	initCmd.Flags().StringVar(&initVersion, "version", "", "API version for OpenAPI info")
	initCmd.Flags().StringVar(&initDescription, "description", "", "API description for OpenAPI info")
}

func runInit(cmd *cobra.Command, args []string) error {
	configFile := "api2spec.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil && !initForce {
		return fmt.Errorf("config file %s already exists, use --force to overwrite", configFile)
	}

	// Determine project root
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	// Create config with sensible defaults
	cfg := config.Default()

	// Detect framework if not specified
	fw := initFramework
	if framework != "" {
		fw = framework
	}

	if fw == "" {
		printVerbose("Auto-detecting framework...")
		detectedPlugin, err := plugins.Detect(projectRoot)
		if err != nil {
			printVerbose("Framework detection failed: %v", err)
			printInfo("No framework auto-detected. Using 'auto' mode.")
			fw = "auto"
		} else {
			fw = detectedPlugin.Name()
			printInfo("Detected framework: %s", fw)
		}
	} else {
		// Validate framework using plugins registry
		if fw != "auto" && plugins.Get(fw) == nil {
			return fmt.Errorf("unsupported framework %q, must be one of: %s, auto", fw, strings.Join(plugins.List(), ", "))
		}
	}
	cfg.Framework = fw

	// Detect project info from go.mod
	projectInfo := detectProjectInfo(projectRoot)

	// Set API info from detection or flags
	if initTitle != "" {
		cfg.OpenAPI.Info.Title = initTitle
	} else if projectInfo.Title != "" {
		cfg.OpenAPI.Info.Title = projectInfo.Title
	}

	if initVersion != "" {
		cfg.OpenAPI.Info.Version = initVersion
	}

	if initDescription != "" {
		cfg.OpenAPI.Info.Description = initDescription
	} else if projectInfo.Description != "" {
		cfg.OpenAPI.Info.Description = projectInfo.Description
	}

	// Detect entry points based on project structure
	entryPoints := detectEntryPoints(projectRoot)
	if len(entryPoints) > 0 {
		cfg.Source.Paths = entryPoints
		printVerbose("Detected entry points: %s", strings.Join(entryPoints, ", "))
	}

	// Interactive mode
	if initInteractive && isTerminal() {
		cfg, err = interactiveInit(cfg)
		if err != nil {
			return fmt.Errorf("interactive init failed: %w", err)
		}
	}

	// Build YAML with comments
	output := buildConfigYAML(cfg)

	// Write config file
	if err := os.WriteFile(configFile, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	printInfo("Created %s", configFile)
	printVerbose("Framework: %s", cfg.Framework)
	printVerbose("Output: %s", cfg.Output)
	printVerbose("Paths: %s", strings.Join(cfg.Source.Paths, ", "))

	return nil
}

// projectInfo holds information detected from the project.
type projectInfo struct {
	Title       string
	Module      string
	Description string
}

// detectProjectInfo detects project information from go.mod.
func detectProjectInfo(projectRoot string) projectInfo {
	info := projectInfo{}

	goModPath := filepath.Join(projectRoot, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "module ") {
			info.Module = strings.TrimPrefix(line, "module ")
			info.Module = strings.TrimSpace(info.Module)

			// Extract a title from the module path
			// e.g., "github.com/user/my-api" -> "my-api"
			parts := strings.Split(info.Module, "/")
			if len(parts) > 0 {
				name := parts[len(parts)-1]
				// Convert kebab-case or snake_case to title case
				name = strings.ReplaceAll(name, "-", " ")
				name = strings.ReplaceAll(name, "_", " ")
				info.Title = strings.Title(name) + " API"
			}
			break
		}
	}

	return info
}

// detectEntryPoints detects common entry point patterns in the project.
func detectEntryPoints(projectRoot string) []string {
	var paths []string

	// Common patterns for Go projects
	patterns := []struct {
		path     string
		priority int
	}{
		{"./cmd", 1},
		{"./internal", 2},
		{"./pkg", 3},
		{"./api", 4},
		{"./handlers", 5},
		{"./routes", 6},
		{"./router", 7},
		{"./server", 8},
	}

	for _, p := range patterns {
		fullPath := filepath.Join(projectRoot, p.path)
		if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
			paths = append(paths, p.path)
		}
	}

	// If no common directories found, use current directory
	if len(paths) == 0 {
		paths = []string{"."}
	}

	return paths
}

// isTerminal checks if stdin is a terminal.
func isTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// interactiveInit prompts the user for configuration options.
func interactiveInit(cfg *config.Config) (*config.Config, error) {
	reader := bufio.NewReader(os.Stdin)

	// API Title
	fmt.Printf("API Title [%s]: ", cfg.OpenAPI.Info.Title)
	title, _ := reader.ReadString('\n')
	title = strings.TrimSpace(title)
	if title != "" {
		cfg.OpenAPI.Info.Title = title
	}

	// API Version
	fmt.Printf("API Version [%s]: ", cfg.OpenAPI.Info.Version)
	version, _ := reader.ReadString('\n')
	version = strings.TrimSpace(version)
	if version != "" {
		cfg.OpenAPI.Info.Version = version
	}

	// API Description
	fmt.Printf("API Description [%s]: ", cfg.OpenAPI.Info.Description)
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)
	if description != "" {
		cfg.OpenAPI.Info.Description = description
	}

	// Framework
	fmt.Printf("Framework [%s]: ", cfg.Framework)
	framework, _ := reader.ReadString('\n')
	framework = strings.TrimSpace(framework)
	if framework != "" {
		cfg.Framework = framework
	}

	// Output file
	fmt.Printf("Output file [%s]: ", cfg.Output)
	output, _ := reader.ReadString('\n')
	output = strings.TrimSpace(output)
	if output != "" {
		cfg.Output = output
	}

	// Output format
	fmt.Printf("Output format (yaml/json) [%s]: ", cfg.Format)
	format, _ := reader.ReadString('\n')
	format = strings.TrimSpace(format)
	if format != "" {
		cfg.Format = format
	}

	return cfg, nil
}

// buildConfigYAML builds a YAML config with helpful comments.
func buildConfigYAML(cfg *config.Config) string {
	// First, marshal to get the base YAML
	data, _ := yaml.Marshal(cfg)

	// Add header comment
	header := `# api2spec configuration file
# https://github.com/api2spec/api2spec

`
	return header + string(data)
}
