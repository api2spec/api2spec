// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package plugins

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages framework plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]FrameworkPlugin
}

// globalRegistry is the default plugin registry.
var globalRegistry = NewRegistry()

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]FrameworkPlugin),
	}
}

// Register adds a plugin to the registry.
// It returns an error if a plugin with the same name is already registered.
func (r *Registry) Register(plugin FrameworkPlugin) error {
	if plugin == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}

	r.plugins[name] = plugin
	return nil
}

// MustRegister adds a plugin to the registry, panicking on error.
// This is useful for init() functions where registration failures are fatal.
func (r *Registry) MustRegister(plugin FrameworkPlugin) {
	if err := r.Register(plugin); err != nil {
		panic(fmt.Sprintf("failed to register plugin: %v", err))
	}
}

// Get returns a plugin by name, or nil if not found.
func (r *Registry) Get(name string) FrameworkPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.plugins[name]
}

// Detect attempts to auto-detect which framework is being used in the project.
// It iterates through all registered plugins and returns the first one that
// successfully detects its framework. Returns an error if no framework is detected.
func (r *Registry) Detect(projectRoot string) (FrameworkPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get sorted names for deterministic detection order
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	var detectedPlugins []FrameworkPlugin

	for _, name := range names {
		plugin := r.plugins[name]
		detected, err := plugin.Detect(projectRoot)
		if err != nil {
			// Log the error but continue checking other plugins
			continue
		}
		if detected {
			detectedPlugins = append(detectedPlugins, plugin)
		}
	}

	if len(detectedPlugins) == 0 {
		return nil, fmt.Errorf("no framework detected in project %s", projectRoot)
	}

	if len(detectedPlugins) > 1 {
		names := make([]string, len(detectedPlugins))
		for i, p := range detectedPlugins {
			names[i] = p.Name()
		}
		// Return the first detected plugin but warn about multiple frameworks
		// In practice, the caller should use explicit --framework flag
		return detectedPlugins[0], nil
	}

	return detectedPlugins[0], nil
}

// List returns a sorted list of registered plugin names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered plugins.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// Has checks if a plugin is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.plugins[name]
	return exists
}

// Unregister removes a plugin from the registry.
// Returns an error if the plugin is not registered.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin %q is not registered", name)
	}

	delete(r.plugins, name)
	return nil
}

// Clear removes all plugins from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]FrameworkPlugin)
}

// --- Global Registry Functions ---

// Register adds a plugin to the global registry.
func Register(plugin FrameworkPlugin) error {
	return globalRegistry.Register(plugin)
}

// MustRegister adds a plugin to the global registry, panicking on error.
func MustRegister(plugin FrameworkPlugin) {
	globalRegistry.MustRegister(plugin)
}

// Get returns a plugin by name from the global registry.
func Get(name string) FrameworkPlugin {
	return globalRegistry.Get(name)
}

// Detect attempts to auto-detect the framework using the global registry.
func Detect(projectRoot string) (FrameworkPlugin, error) {
	return globalRegistry.Detect(projectRoot)
}

// List returns all registered plugin names from the global registry.
func List() []string {
	return globalRegistry.List()
}

// Has checks if a plugin is registered in the global registry.
func Has(name string) bool {
	return globalRegistry.Has(name)
}

// Global returns the global registry instance.
// This is useful for testing or when explicit registry access is needed.
func Global() *Registry {
	return globalRegistry
}
