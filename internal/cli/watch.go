// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/internal/openapi"
	"github.com/api2spec/api2spec/internal/plugins"
	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
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

// Watcher handles file watching and spec regeneration.
type Watcher struct {
	cfg           *config.Config
	watcher       *fsnotify.Watcher
	paths         []string
	debounce      time.Duration
	onChangeCmd   string
	mu            sync.Mutex
	lastRegen     time.Time
	pendingChange bool
	plugin        plugins.FrameworkPlugin
}

// NewWatcher creates a new Watcher instance.
func NewWatcher(cfg *config.Config, paths []string, plugin plugins.FrameworkPlugin) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	debounce := time.Duration(cfg.Watch.Debounce) * time.Millisecond
	if debounce <= 0 {
		debounce = 500 * time.Millisecond
	}

	return &Watcher{
		cfg:         cfg,
		watcher:     fsWatcher,
		paths:       paths,
		debounce:    debounce,
		onChangeCmd: cfg.Watch.OnChange,
		plugin:      plugin,
	}, nil
}

// Close closes the watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// Watch starts watching for file changes.
func (w *Watcher) Watch(ctx context.Context) error {
	// Add all paths to watch
	for _, path := range w.paths {
		if err := w.addPath(path); err != nil {
			return fmt.Errorf("failed to add watch path %s: %w", path, err)
		}
	}

	// Run initial generation
	if err := w.regenerate(); err != nil {
		printError("Initial generation failed: %v", err)
	}

	// Debounce timer
	var debounceTimer *time.Timer
	var debounceTimerMu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}

			// Only handle write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Check if this is a file we care about
			if !w.shouldWatch(event.Name) {
				continue
			}

			printVerbose("File changed: %s", event.Name)

			// Debounce changes
			debounceTimerMu.Lock()
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(w.debounce, func() {
				if err := w.regenerate(); err != nil {
					printError("Regeneration failed: %v", err)
				}
			})
			debounceTimerMu.Unlock()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			printError("Watch error: %v", err)
		}
	}
}

// addPath adds a path and its subdirectories to the watcher.
func (w *Watcher) addPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return w.watcher.Add(absPath)
	}

	// Walk directory and add all subdirectories
	return filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if info.IsDir() {
			// Skip excluded directories
			for _, exclude := range w.cfg.Source.Exclude {
				if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
					return filepath.SkipDir
				}
				// Also check vendor and hidden directories
				base := filepath.Base(path)
				if base == "vendor" || base == "node_modules" || strings.HasPrefix(base, ".") {
					return filepath.SkipDir
				}
			}

			printVerbose("Watching: %s", path)
			return w.watcher.Add(path)
		}
		return nil
	})
}

// shouldWatch checks if a file should trigger regeneration.
func (w *Watcher) shouldWatch(path string) bool {
	// Check file extension
	ext := filepath.Ext(path)
	supportedExts := []string{".go", ".ts", ".js"}
	found := false
	for _, e := range supportedExts {
		if ext == e {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// Check exclude patterns
	for _, pattern := range w.cfg.Source.Exclude {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
		// Check for test files
		if strings.HasSuffix(path, "_test.go") && pattern == "**/*_test.go" {
			return false
		}
	}

	return true
}

// regenerate runs the spec generation.
func (w *Watcher) regenerate() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	printInfo("Regenerating specification...")
	start := time.Now()

	// Scan for source files
	scannerCfg := scanner.Config{
		IncludePatterns: w.cfg.Source.Include,
		ExcludePatterns: w.cfg.Source.Exclude,
	}

	var files []scanner.SourceFile
	for _, path := range w.paths {
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

	if len(files) == 0 {
		printInfo("No source files found")
		return nil
	}

	// Extract routes and schemas
	var routes []types.Route
	var schemas []types.Schema

	if w.plugin != nil {
		if w.cfg.Generation.Mode == "full" || w.cfg.Generation.Mode == "routes-only" {
			extractedRoutes, err := w.plugin.ExtractRoutes(files)
			if err != nil {
				return fmt.Errorf("failed to extract routes: %w", err)
			}
			routes = extractedRoutes
		}

		if w.cfg.Generation.Mode == "full" || w.cfg.Generation.Mode == "schemas-only" {
			extractedSchemas, err := w.plugin.ExtractSchemas(files)
			if err != nil {
				return fmt.Errorf("failed to extract schemas: %w", err)
			}
			schemas = extractedSchemas
		}
	}

	// Build OpenAPI spec
	builder := openapi.NewBuilder(w.cfg)
	doc, err := builder.Build(routes, schemas)
	if err != nil {
		return fmt.Errorf("failed to build OpenAPI spec: %w", err)
	}

	// Handle merge if requested
	if w.cfg.Generation.Merge {
		if _, err := os.Stat(w.cfg.Output); err == nil {
			existing, err := openapi.ReadFile(w.cfg.Output)
			if err != nil {
				return fmt.Errorf("failed to read existing spec for merge: %w", err)
			}
			doc, err = openapi.MergeDefault(existing, doc)
			if err != nil {
				return fmt.Errorf("failed to merge specs: %w", err)
			}
		}
	}

	// Write output
	writer := openapi.NewWriter()
	if err := writer.WriteFile(doc, w.cfg.Output, w.cfg.Format); err != nil {
		return fmt.Errorf("failed to write spec: %w", err)
	}

	elapsed := time.Since(start)
	printInfo("Specification regenerated in %v: %s (%d routes, %d schemas)",
		elapsed.Round(time.Millisecond), w.cfg.Output, len(routes), len(schemas))

	w.lastRegen = time.Now()

	// Run on-change command if configured
	if w.onChangeCmd != "" {
		if err := w.runOnChangeCmd(); err != nil {
			printError("On-change command failed: %v", err)
		}
	}

	return nil
}

// runOnChangeCmd executes the on-change command.
func (w *Watcher) runOnChangeCmd() error {
	printVerbose("Running on-change command: %s", w.onChangeCmd)

	// Split command into shell arguments
	cmd := exec.Command("sh", "-c", w.onChangeCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
	if output != "" {
		cfg.Output = output
	}
	if format != "" {
		cfg.Format = format
	}
	if framework != "" {
		cfg.Framework = framework
	}

	// Determine paths to watch
	paths := args
	if len(paths) == 0 {
		paths = cfg.Source.Paths
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	printVerbose("Watch configuration:")
	printVerbose("  Mode: %s", cfg.Generation.Mode)
	printVerbose("  Debounce: %dms", cfg.Watch.Debounce)
	if cfg.Watch.OnChange != "" {
		printVerbose("  On change: %s", cfg.Watch.OnChange)
	}
	printVerbose("  Paths: %s", strings.Join(paths, ", "))

	// Determine project root for framework detection
	projectRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	// Get or detect framework plugin
	var plugin plugins.FrameworkPlugin
	if cfg.Framework == "" || cfg.Framework == "auto" {
		plugin, err = plugins.Detect(projectRoot)
		if err != nil {
			printVerbose("Framework detection failed: %v", err)
			printInfo("No framework detected, watching without plugin")
		} else {
			printInfo("Detected framework: %s", plugin.Name())
		}
	} else {
		plugin = plugins.Get(cfg.Framework)
		if plugin == nil {
			return fmt.Errorf("unknown framework %q", cfg.Framework)
		}
		printVerbose("Using framework: %s", plugin.Name())
	}

	// Create watcher
	watcher, err := NewWatcher(cfg, paths, plugin)
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		printInfo("\nShutting down watcher...")
		cancel()
	}()

	printInfo("Watching for changes in: %s", strings.Join(paths, ", "))
	printInfo("Press Ctrl+C to stop")

	return watcher.Watch(ctx)
}
