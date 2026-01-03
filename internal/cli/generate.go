// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/openapi"
	"github.com/api2spec/api2spec/internal/plugins"
	_ "github.com/api2spec/api2spec/internal/plugins/actix"   // Register actix plugin
	_ "github.com/api2spec/api2spec/internal/plugins/aspnet"  // Register aspnet plugin
	_ "github.com/api2spec/api2spec/internal/plugins/axum"    // Register axum plugin
	_ "github.com/api2spec/api2spec/internal/plugins/chi"     // Register chi plugin
	_ "github.com/api2spec/api2spec/internal/plugins/echo"    // Register echo plugin
	_ "github.com/api2spec/api2spec/internal/plugins/elysia"  // Register elysia plugin
	_ "github.com/api2spec/api2spec/internal/plugins/express" // Register express plugin
	_ "github.com/api2spec/api2spec/internal/plugins/fastapi" // Register fastapi plugin
	_ "github.com/api2spec/api2spec/internal/plugins/fastify" // Register fastify plugin
	_ "github.com/api2spec/api2spec/internal/plugins/fiber"   // Register fiber plugin
	_ "github.com/api2spec/api2spec/internal/plugins/flask"   // Register flask plugin
	_ "github.com/api2spec/api2spec/internal/plugins/gin"     // Register gin plugin
	_ "github.com/api2spec/api2spec/internal/plugins/gleam"   // Register gleam plugin
	_ "github.com/api2spec/api2spec/internal/plugins/hono"    // Register hono plugin
	_ "github.com/api2spec/api2spec/internal/plugins/koa"     // Register koa plugin
	_ "github.com/api2spec/api2spec/internal/plugins/ktor"    // Register ktor plugin
	_ "github.com/api2spec/api2spec/internal/plugins/laravel" // Register laravel plugin
	_ "github.com/api2spec/api2spec/internal/plugins/nestjs"  // Register nestjs plugin
	_ "github.com/api2spec/api2spec/internal/plugins/phoenix" // Register phoenix plugin
	_ "github.com/api2spec/api2spec/internal/plugins/rails"   // Register rails plugin
	_ "github.com/api2spec/api2spec/internal/plugins/rocket"  // Register rocket plugin
	_ "github.com/api2spec/api2spec/internal/plugins/sinatra" // Register sinatra plugin
	_ "github.com/api2spec/api2spec/internal/plugins/spring"  // Register spring plugin
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
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

Frameworks:
  chi          go-chi/chi router (auto-detected)
  auto         Auto-detect framework (default)

Example:
  api2spec generate                           # Generate from current directory
  api2spec generate ./cmd ./internal          # Generate from specific paths
  api2spec generate --mode routes-only        # Generate routes only
  api2spec generate --merge                   # Merge with existing spec
  api2spec generate --dry-run                 # Preview without writing
  api2spec generate --framework chi           # Use chi plugin explicitly`,
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

	// Determine project root for framework detection
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	// Get or detect framework plugin
	var plugin plugins.FrameworkPlugin
	if cfg.Framework == "" || cfg.Framework == "auto" {
		printVerbose("Auto-detecting framework...")
		plugin, err = plugins.Detect(projectRoot)
		if err != nil {
			printVerbose("Framework detection failed: %v", err)
			printInfo("No framework detected. Available plugins: %s", strings.Join(plugins.List(), ", "))
			printInfo("Use --framework to specify a framework or ensure go.mod contains framework imports")
		} else {
			printInfo("Detected framework: %s", plugin.Name())
		}
	} else {
		plugin = plugins.Get(cfg.Framework)
		if plugin == nil {
			return fmt.Errorf("unknown framework %q. Available: %s", cfg.Framework, strings.Join(plugins.List(), ", "))
		}
		printVerbose("Using framework: %s", plugin.Name())
	}

	// Create scanner with config
	scannerCfg := scanner.Config{
		IncludePatterns: cfg.Source.Include,
		ExcludePatterns: cfg.Source.Exclude,
	}

	// Scan for source files
	var files []scanner.SourceFile
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", path, err)
		}
		scannerCfg.BasePath = absPath
		s := scanner.New(scannerCfg)
		pathFiles, err := s.Scan()
		if err != nil {
			return fmt.Errorf("failed to scan path %s: %w", path, err)
		}
		files = append(files, pathFiles...)
	}

	// Print discovered files in verbose mode
	printVerbose("Discovered %d source files:", len(files))
	for _, f := range files {
		printVerbose("  [%s] %s", f.Language, f.Path)
	}

	if len(files) == 0 {
		printInfo("No source files found matching patterns")
		return nil
	}

	printInfo("Found %d source files to analyze", len(files))

	// Group files by language for display
	byLanguage := make(map[string][]scanner.SourceFile)
	for _, f := range files {
		byLanguage[f.Language] = append(byLanguage[f.Language], f)
	}

	for lang, langFiles := range byLanguage {
		printVerbose("  %s: %d files", lang, len(langFiles))
	}

	// Extract routes and schemas using plugin
	var routes []types.Route
	var schemas []types.Schema

	if plugin != nil {
		printInfo("Extracting routes and schemas using %s plugin...", plugin.Name())

		// Extract routes (if mode allows)
		if cfg.Generation.Mode == "full" || cfg.Generation.Mode == "routes-only" {
			extractedRoutes, err := plugin.ExtractRoutes(files)
			if err != nil {
				return fmt.Errorf("failed to extract routes: %w", err)
			}
			routes = extractedRoutes
			printInfo("Found %d routes", len(routes))

			for _, r := range routes {
				printVerbose("  %s %s -> %s", r.Method, r.Path, r.Handler)
			}
		}

		// Extract schemas (if mode allows)
		if cfg.Generation.Mode == "full" || cfg.Generation.Mode == "schemas-only" {
			extractedSchemas, err := plugin.ExtractSchemas(files)
			if err != nil {
				return fmt.Errorf("failed to extract schemas: %w", err)
			}
			schemas = extractedSchemas
			printInfo("Found %d schemas", len(schemas))

			for _, s := range schemas {
				printVerbose("  %s", s.Title)
			}
		}
	} else {
		printInfo("No plugin available - generating empty specification")
	}

	// Create OpenAPI builder
	builder := openapi.NewBuilder(cfg)

	doc, err := builder.Build(routes, schemas)
	if err != nil {
		return fmt.Errorf("failed to build OpenAPI spec: %w", err)
	}

	// Handle merge if requested
	if cfg.Generation.Merge {
		if _, err := os.Stat(cfg.Output); err == nil {
			printVerbose("Merging with existing spec: %s", cfg.Output)
			existing, err := openapi.ReadFile(cfg.Output)
			if err != nil {
				return fmt.Errorf("failed to read existing spec for merge: %w", err)
			}
			doc, err = openapi.MergeDefault(existing, doc)
			if err != nil {
				return fmt.Errorf("failed to merge specs: %w", err)
			}
		} else {
			printVerbose("No existing spec found at %s, creating new", cfg.Output)
		}
	}

	// Write output
	writer := openapi.NewWriter()

	if generateDryRun {
		// Print to stdout
		var output string
		if cfg.Format == "json" {
			output, err = writer.ToJSON(doc)
		} else {
			output, err = writer.ToYAML(doc)
		}
		if err != nil {
			return fmt.Errorf("failed to serialize spec: %w", err)
		}
		fmt.Print(output)
		return nil
	}

	// Write to file
	if err := writer.WriteFile(doc, cfg.Output, cfg.Format); err != nil {
		return fmt.Errorf("failed to write spec: %w", err)
	}

	printInfo("OpenAPI specification written to: %s", cfg.Output)
	return nil
}
