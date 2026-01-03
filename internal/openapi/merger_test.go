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

	assert.Equal(t, MergeStrategyMerge, opts.ConflictStrategy)
	assert.True(t, opts.PreserveDescriptions)
	assert.True(t, opts.PreserveExamples)
	assert.True(t, opts.PreserveTags)
	assert.True(t, opts.PreserveExtensions)
	assert.False(t, opts.MarkRemovedAsDeprecated)
	assert.True(t, opts.PreserveInfo)
	assert.True(t, opts.PreserveServers)
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

func TestMerger_Merge_NilGenerated(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Info: types.Info{
			Title:   "Existing API",
			Version: "1.0.0",
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.Merge(existing, nil)

	require.NoError(t, err)
	assert.Equal(t, existing, result)
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
	// Tags should be merged: generated + existing not in generated
	require.Len(t, result.Tags, 2)
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
		ConflictStrategy: MergeStrategyGeneratedWins,
		PreserveInfo:     false,
		PreserveServers:  false,
		PreserveTags:     false,
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
	// Smart behavior: if existing info is empty, use generated info
	assert.Equal(t, "Generated API", result.Info.Title)
}

func TestMerger_MergePaths_Added(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {Get: &types.Operation{Summary: "List users"}},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users":  {Get: &types.Operation{Summary: "List users"}},
			"/orders": {Get: &types.Operation{Summary: "List orders"}},
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.MergeWithResult(existing, generated)

	require.NoError(t, err)
	assert.Len(t, result.Document.Paths, 2)
	assert.Contains(t, result.AddedPaths, "/orders")
	assert.Contains(t, result.UpdatedPaths, "/users")
}

func TestMerger_MergePaths_Removed(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users":  {Get: &types.Operation{Summary: "List users"}},
			"/legacy": {Get: &types.Operation{Summary: "Legacy endpoint"}},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {Get: &types.Operation{Summary: "List users"}},
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.MergeWithResult(existing, generated)

	require.NoError(t, err)
	assert.Len(t, result.Document.Paths, 1)
	assert.Contains(t, result.RemovedPaths, "/legacy")
}

func TestMerger_MergePaths_MarkDeprecated(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users":  {Get: &types.Operation{Summary: "List users"}},
			"/legacy": {Get: &types.Operation{Summary: "Legacy endpoint"}},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {Get: &types.Operation{Summary: "List users"}},
		},
	}

	opts := DefaultMergeOptions()
	opts.MarkRemovedAsDeprecated = true
	merger := NewMerger(opts)
	result, err := merger.MergeWithResult(existing, generated)

	require.NoError(t, err)
	assert.Len(t, result.Document.Paths, 2)
	assert.True(t, result.Document.Paths["/legacy"].Get.Deprecated)
}

func TestMerger_MergeOperation_PreserveDescription(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {
				Get: &types.Operation{
					Summary:     "Existing summary",
					Description: "Detailed description of endpoint",
				},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {
				Get: &types.Operation{
					Summary: "Generated summary",
					// No description
				},
			},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveDescriptions = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	op := result.Paths["/users"].Get
	assert.Equal(t, "Generated summary", op.Summary)                 // Takes generated
	assert.Equal(t, "Detailed description of endpoint", op.Description) // Preserves existing
}

func TestMerger_MergeSchemas_Added(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {Type: "object", Title: "User"},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User":  {Type: "object", Title: "User"},
				"Order": {Type: "object", Title: "Order"},
			},
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.MergeWithResult(existing, generated)

	require.NoError(t, err)
	assert.Len(t, result.Document.Components.Schemas, 2)
	assert.Contains(t, result.AddedSchemas, "Order")
	assert.Contains(t, result.UpdatedSchemas, "User")
}

func TestMerger_MergeSchemas_PreserveDescription(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {
					Type:        "object",
					Title:       "User",
					Description: "Detailed user description",
				},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {
					Type:  "object",
					Title: "User",
					// No description
				},
			},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveDescriptions = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	assert.Equal(t, "Detailed user description", result.Components.Schemas["User"].Description)
}

func TestMerger_MergeSchemas_PreserveExample(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {
					Type:    "object",
					Example: map[string]interface{}{"id": "123", "name": "John"},
				},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			Schemas: map[string]*types.Schema{
				"User": {
					Type: "object",
					// No example
				},
			},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveExamples = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	assert.NotNil(t, result.Components.Schemas["User"].Example)
}

func TestMerger_MergeParameters_PreserveDescription(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users/{id}": {
				Get: &types.Operation{
					Parameters: []types.Parameter{
						{Name: "id", In: "path", Description: "The unique user identifier"},
					},
				},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users/{id}": {
				Get: &types.Operation{
					Parameters: []types.Parameter{
						{Name: "id", In: "path"},
					},
				},
			},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveDescriptions = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	params := result.Paths["/users/{id}"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "The unique user identifier", params[0].Description)
}

func TestMerger_MergeResponses_PreserveDescription(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {
				Get: &types.Operation{
					Responses: map[string]types.Response{
						"200": {Description: "Returns a list of users"},
						"500": {Description: "Internal server error"},
					},
				},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Paths: map[string]types.PathItem{
			"/users": {
				Get: &types.Operation{
					Responses: map[string]types.Response{
						"200": {Description: ""},
					},
				},
			},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveDescriptions = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	responses := result.Paths["/users"].Get.Responses
	assert.Equal(t, "Returns a list of users", responses["200"].Description)
	assert.Equal(t, "Internal server error", responses["500"].Description)
}

func TestMerger_MergeTags_Combine(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Tags: []types.Tag{
			{Name: "users", Description: "User operations"},
			{Name: "admin", Description: "Admin operations"},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Tags: []types.Tag{
			{Name: "users"}, // Same tag, no description
			{Name: "orders", Description: "Order operations"},
		},
	}

	opts := DefaultMergeOptions()
	opts.PreserveTags = true
	opts.PreserveDescriptions = true
	merger := NewMerger(opts)
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	assert.Len(t, result.Tags, 3) // users, orders, admin

	// Find users tag
	var usersTag *types.Tag
	for _, tag := range result.Tags {
		if tag.Name == "users" {
			usersTag = &tag
			break
		}
	}
	require.NotNil(t, usersTag)
	assert.Equal(t, "User operations", usersTag.Description) // Preserved from existing
}

func TestMerger_MergeSecuritySchemes(t *testing.T) {
	existing := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			SecuritySchemes: map[string]types.SecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer"},
				"apiKey":     {Type: "apiKey", Name: "X-API-Key", In: "header"},
			},
		},
	}

	generated := &types.OpenAPI{
		OpenAPI: "3.0.3",
		Components: &types.Components{
			SecuritySchemes: map[string]types.SecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", Description: "New desc"},
			},
		},
	}

	merger := NewMerger(DefaultMergeOptions())
	result, err := merger.Merge(existing, generated)

	require.NoError(t, err)
	schemes := result.Components.SecuritySchemes
	assert.Len(t, schemes, 2)
	// Existing schemes are preserved
	assert.Equal(t, "bearer", schemes["bearerAuth"].Scheme)
	assert.Equal(t, "header", schemes["apiKey"].In)
}

func TestMerger_MergeStringSlices(t *testing.T) {
	m := NewMerger(DefaultMergeOptions())

	existing := []string{"users", "admin"}
	generated := []string{"users", "orders"}

	result := m.mergeStringSlices(existing, generated)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "users")
	assert.Contains(t, result, "orders")
	assert.Contains(t, result, "admin")
}

func TestMergeStrategy_Constants(t *testing.T) {
	assert.Equal(t, MergeStrategy("generated-wins"), MergeStrategyGeneratedWins)
	assert.Equal(t, MergeStrategy("existing-wins"), MergeStrategyExistingWins)
	assert.Equal(t, MergeStrategy("merge"), MergeStrategyMerge)
}
