// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/pkg/types"
)

func TestDefaultMergeOptions(t *testing.T) {
	opts := DefaultMergeOptions()

	assert.Equal(t, MergeStrategyOverwrite, opts.Strategy)
	assert.False(t, opts.PreservePaths)
	assert.False(t, opts.PreserveSchemas)
	assert.True(t, opts.PreserveInfo)
	assert.True(t, opts.PreserveServers)
	assert.True(t, opts.PreserveTags)
	assert.True(t, opts.PreserveSecurity)
}

func TestNewMerger(t *testing.T) {
	opts := DefaultMergeOptions()
	merger := NewMerger(opts)

	assert.NotNil(t, merger)
	assert.Equal(t, opts, merger.options)
}

func TestMerger_Merge_NilExisting(t *testing.T) {
	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.Merge(nil, generated)

	require.NoError(t, err)
	assert.Equal(t, generated, result)
}

func TestMerger_Merge_PreserveInfo(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:       "Original API",
			Description: "Original description",
			Version:     "2.0.0",
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveInfo = true
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	assert.Equal(t, "Original API", result.Info.Title)
	assert.Equal(t, "Original description", result.Info.Description)
	assert.Equal(t, "2.0.0", result.Info.Version)
}

func TestMerger_Merge_PreserveServers(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Servers: []types.Server{
			{URL: "https://existing.example.com", Description: "Existing server"},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Servers: []types.Server{
			{URL: "https://generated.example.com"},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveServers = true
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	require.Len(t, result.Servers, 1)
	assert.Equal(t, "https://existing.example.com", result.Servers[0].URL)
}

func TestMerger_Merge_PreserveTags(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Tags: []types.Tag{
			{Name: "existing-tag", Description: "Existing tag"},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Tags: []types.Tag{
			{Name: "generated-tag"},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveTags = true
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	require.Len(t, result.Tags, 1)
	assert.Equal(t, "existing-tag", result.Tags[0].Name)
}

func TestMerger_Merge_PreserveSecurity(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Security: []map[string][]string{
			{"bearerAuth": {}},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Security: []map[string][]string{
			{"apiKey": {}},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveSecurity = true
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	require.Len(t, result.Security, 1)
	assert.Contains(t, result.Security[0], "bearerAuth")
}

func TestMerger_Merge_DoNotPreserve(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Original API",
			Version: "2.0.0",
		},
		Servers: []types.Server{
			{URL: "https://existing.example.com"},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
		Servers: []types.Server{
			{URL: "https://generated.example.com"},
		},
	}

	opts := MergeOptions{
		Strategy:        MergeStrategyOverwrite,
		PreserveInfo:    false,
		PreserveServers: false,
		PreserveTags:    false,
	}
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	assert.Equal(t, "Generated API", result.Info.Title)
	assert.Equal(t, "https://generated.example.com", result.Servers[0].URL)
}

func TestMergeDefault(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Original API",
			Version: "2.0.0",
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
	}

	result, err := MergeDefault(existing, generated)

	require.NoError(t, err)
	// Default preserves info
	assert.Equal(t, "Original API", result.Info.Title)
}

func TestMerger_Merge_EmptyExistingInfo(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "",
			Version: "",
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Generated API",
			Version: "1.0.0",
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveInfo = true
	merger := NewMerger(opts)

	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	// Current stub behavior: if existing info is empty, use generated info
	// This is smart behavior - don't preserve empty values
	assert.Equal(t, "Generated API", result.Info.Title)
}
