// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/scanner"
	"github.com/api2spec/api2spec/pkg/types"
)

// mockPlugin is a test implementation of FrameworkPlugin.
type mockPlugin struct {
	name       string
	extensions []string
	detected   bool
	detectErr  error
	routes     []types.Route
	routeErr   error
	schemas    []types.Schema
	schemaErr  error
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Extensions() []string {
	return m.extensions
}

func (m *mockPlugin) Detect(projectRoot string) (bool, error) {
	return m.detected, m.detectErr
}

func (m *mockPlugin) ExtractRoutes(files []scanner.SourceFile) ([]types.Route, error) {
	return m.routes, m.routeErr
}

func (m *mockPlugin) ExtractSchemas(files []scanner.SourceFile) ([]types.Schema, error) {
	return m.schemas, m.schemaErr
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		plugin    FrameworkPlugin
		wantErr   bool
		errContains string
	}{
		{
			name: "register valid plugin",
			plugin: &mockPlugin{
				name:       "test-plugin",
				extensions: []string{".go"},
			},
			wantErr: false,
		},
		{
			name:        "register nil plugin",
			plugin:      nil,
			wantErr:     true,
			errContains: "nil plugin",
		},
		{
			name: "register empty name",
			plugin: &mockPlugin{
				name: "",
			},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			err := reg.Register(tt.plugin)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.True(t, reg.Has(tt.plugin.Name()))
			}
		})
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	reg := NewRegistry()

	plugin1 := &mockPlugin{name: "duplicate", extensions: []string{".go"}}
	plugin2 := &mockPlugin{name: "duplicate", extensions: []string{".go"}}

	require.NoError(t, reg.Register(plugin1))

	err := reg.Register(plugin2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestRegistry_MustRegister_Panics(t *testing.T) {
	reg := NewRegistry()

	assert.Panics(t, func() {
		reg.MustRegister(nil)
	})
}

func TestRegistry_Get(t *testing.T) {
	reg := NewRegistry()

	plugin := &mockPlugin{name: "test-plugin", extensions: []string{".go"}}
	require.NoError(t, reg.Register(plugin))

	// Get existing plugin
	got := reg.Get("test-plugin")
	assert.NotNil(t, got)
	assert.Equal(t, "test-plugin", got.Name())

	// Get non-existent plugin
	got = reg.Get("non-existent")
	assert.Nil(t, got)
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	require.NoError(t, reg.Register(&mockPlugin{name: "charlie"}))
	require.NoError(t, reg.Register(&mockPlugin{name: "alpha"}))
	require.NoError(t, reg.Register(&mockPlugin{name: "bravo"}))

	list := reg.List()
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, list)
}

func TestRegistry_Detect(t *testing.T) {
	t.Run("single framework detected", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register(&mockPlugin{name: "framework-a", detected: false})
		reg.Register(&mockPlugin{name: "framework-b", detected: true})
		reg.Register(&mockPlugin{name: "framework-c", detected: false})

		plugin, err := reg.Detect("/project")
		require.NoError(t, err)
		assert.Equal(t, "framework-b", plugin.Name())
	})

	t.Run("no framework detected", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register(&mockPlugin{name: "framework-a", detected: false})
		reg.Register(&mockPlugin{name: "framework-b", detected: false})

		plugin, err := reg.Detect("/project")
		require.Error(t, err)
		assert.Nil(t, plugin)
		assert.Contains(t, err.Error(), "no framework detected")
	})

	t.Run("multiple frameworks detected returns first alphabetically", func(t *testing.T) {
		reg := NewRegistry()

		reg.Register(&mockPlugin{name: "zebra", detected: true})
		reg.Register(&mockPlugin{name: "alpha", detected: true})

		plugin, err := reg.Detect("/project")
		require.NoError(t, err)
		// Should return "alpha" as it comes first alphabetically
		assert.Equal(t, "alpha", plugin.Name())
	})

	t.Run("empty registry", func(t *testing.T) {
		reg := NewRegistry()

		plugin, err := reg.Detect("/project")
		require.Error(t, err)
		assert.Nil(t, plugin)
	})
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()

	assert.Equal(t, 0, reg.Count())

	reg.Register(&mockPlugin{name: "one"})
	assert.Equal(t, 1, reg.Count())

	reg.Register(&mockPlugin{name: "two"})
	assert.Equal(t, 2, reg.Count())
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&mockPlugin{name: "exists"})

	assert.True(t, reg.Has("exists"))
	assert.False(t, reg.Has("does-not-exist"))
}

func TestRegistry_Unregister(t *testing.T) {
	reg := NewRegistry()

	plugin := &mockPlugin{name: "to-remove"}
	reg.Register(plugin)

	assert.True(t, reg.Has("to-remove"))

	err := reg.Unregister("to-remove")
	require.NoError(t, err)
	assert.False(t, reg.Has("to-remove"))

	// Unregister non-existent plugin
	err = reg.Unregister("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_Clear(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&mockPlugin{name: "one"})
	reg.Register(&mockPlugin{name: "two"})

	assert.Equal(t, 2, reg.Count())

	reg.Clear()

	assert.Equal(t, 0, reg.Count())
	assert.Empty(t, reg.List())
}

func TestGlobalRegistryFunctions(t *testing.T) {
	// Clear global registry before test
	Global().Clear()
	defer Global().Clear()

	// Test global functions
	plugin := &mockPlugin{name: "global-test"}

	require.NoError(t, Register(plugin))
	assert.True(t, Has("global-test"))
	assert.NotNil(t, Get("global-test"))
	assert.Contains(t, List(), "global-test")
}
