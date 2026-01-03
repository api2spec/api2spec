// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/pkg/types"
)

func TestRegistry_AddAndGet(t *testing.T) {
	reg := NewRegistry()

	schema := &types.Schema{
		Type:  "object",
		Title: "User",
	}

	reg.Add("User", schema)

	got, ok := reg.Get("User")
	require.True(t, ok)
	assert.Equal(t, "User", got.Title)
}

func TestRegistry_GetNonExistent(t *testing.T) {
	reg := NewRegistry()

	got, ok := reg.Get("NonExistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestRegistry_Has(t *testing.T) {
	reg := NewRegistry()

	reg.Add("User", &types.Schema{Title: "User"})

	assert.True(t, reg.Has("User"))
	assert.False(t, reg.Has("Order"))
}

func TestRegistry_All(t *testing.T) {
	reg := NewRegistry()

	reg.Add("User", &types.Schema{Title: "User"})
	reg.Add("Order", &types.Schema{Title: "Order"})

	all := reg.All()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "User")
	assert.Contains(t, all, "Order")
}

func TestRegistry_Names(t *testing.T) {
	reg := NewRegistry()

	reg.Add("Zebra", &types.Schema{Title: "Zebra"})
	reg.Add("Alpha", &types.Schema{Title: "Alpha"})
	reg.Add("Beta", &types.Schema{Title: "Beta"})

	names := reg.Names()
	assert.Equal(t, []string{"Alpha", "Beta", "Zebra"}, names)
}

func TestRegistry_Count(t *testing.T) {
	reg := NewRegistry()

	assert.Equal(t, 0, reg.Count())

	reg.Add("One", &types.Schema{})
	assert.Equal(t, 1, reg.Count())

	reg.Add("Two", &types.Schema{})
	assert.Equal(t, 2, reg.Count())
}

func TestRegistry_Clear(t *testing.T) {
	reg := NewRegistry()

	reg.Add("User", &types.Schema{})
	reg.Add("Order", &types.Schema{})

	assert.Equal(t, 2, reg.Count())

	reg.Clear()

	assert.Equal(t, 0, reg.Count())
	assert.False(t, reg.Has("User"))
}

func TestRegistry_Remove(t *testing.T) {
	reg := NewRegistry()

	reg.Add("User", &types.Schema{})

	assert.True(t, reg.Has("User"))

	removed := reg.Remove("User")
	assert.True(t, removed)
	assert.False(t, reg.Has("User"))

	// Remove non-existent
	removed = reg.Remove("NonExistent")
	assert.False(t, removed)
}

func TestRegistry_Merge(t *testing.T) {
	reg1 := NewRegistry()
	reg2 := NewRegistry()

	reg1.Add("User", &types.Schema{Title: "User"})
	reg2.Add("Order", &types.Schema{Title: "Order"})
	reg2.Add("Product", &types.Schema{Title: "Product"})

	reg1.Merge(reg2)

	assert.Equal(t, 3, reg1.Count())
	assert.True(t, reg1.Has("User"))
	assert.True(t, reg1.Has("Order"))
	assert.True(t, reg1.Has("Product"))
}

func TestRegistry_MergeNil(t *testing.T) {
	reg := NewRegistry()
	reg.Add("User", &types.Schema{Title: "User"})

	// Should not panic
	reg.Merge(nil)

	assert.Equal(t, 1, reg.Count())
}

func TestRegistry_ToSlice(t *testing.T) {
	reg := NewRegistry()

	reg.Add("User", &types.Schema{Title: "User"})
	reg.Add("Order", &types.Schema{Title: "Order"})

	slice := reg.ToSlice()
	assert.Len(t, slice, 2)

	// Verify schemas are present (order not guaranteed)
	titles := make(map[string]bool)
	for _, s := range slice {
		titles[s.Title] = true
	}
	assert.True(t, titles["User"])
	assert.True(t, titles["Order"])
}

func TestRegistry_AllReturnsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Add("User", &types.Schema{Title: "User"})

	all := reg.All()

	// Modifying the returned map should not affect the registry
	delete(all, "User")

	assert.True(t, reg.Has("User"))
}
