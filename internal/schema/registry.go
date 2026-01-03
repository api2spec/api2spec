// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package schema

import (
	"sort"
	"sync"

	"github.com/api2spec/api2spec/pkg/types"
)

// Registry stores discovered schemas by name for reference resolution.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]*types.Schema
}

// NewRegistry creates a new schema registry.
func NewRegistry() *Registry {
	return &Registry{
		schemas: make(map[string]*types.Schema),
	}
}

// Add adds a schema to the registry.
func (r *Registry) Add(name string, schema *types.Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schemas[name] = schema
}

// Get returns a schema by name.
func (r *Registry) Get(name string) (*types.Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schema, ok := r.schemas[name]
	return schema, ok
}

// Has checks if a schema exists in the registry.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.schemas[name]
	return ok
}

// All returns all schemas in the registry.
func (r *Registry) All() map[string]*types.Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*types.Schema, len(r.schemas))
	for k, v := range r.schemas {
		result[k] = v
	}
	return result
}

// Names returns all schema names in sorted order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.schemas))
	for name := range r.schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of schemas in the registry.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.schemas)
}

// Clear removes all schemas from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schemas = make(map[string]*types.Schema)
}

// Remove removes a schema from the registry.
func (r *Registry) Remove(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.schemas[name]; ok {
		delete(r.schemas, name)
		return true
	}
	return false
}

// Merge adds all schemas from another registry.
func (r *Registry) Merge(other *Registry) {
	if other == nil {
		return
	}

	other.mu.RLock()
	defer other.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, schema := range other.schemas {
		r.schemas[name] = schema
	}
}

// ToSlice returns all schemas as a slice of types.Schema.
func (r *Registry) ToSlice() []types.Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.Schema, 0, len(r.schemas))
	for _, schema := range r.schemas {
		if schema != nil {
			result = append(result, *schema)
		}
	}
	return result
}
