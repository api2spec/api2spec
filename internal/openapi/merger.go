// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"github.com/api2spec/api2spec/pkg/types"
)

// MergeStrategy defines how to handle conflicts during merge.
type MergeStrategy string

const (
	// MergeStrategyKeepExisting keeps the existing value on conflict.
	MergeStrategyKeepExisting MergeStrategy = "keep-existing"

	// MergeStrategyOverwrite overwrites with the generated value on conflict.
	MergeStrategyOverwrite MergeStrategy = "overwrite"

	// MergeStrategyAppend appends new items (for arrays/maps).
	MergeStrategyAppend MergeStrategy = "append"
)

// MergeOptions configures the merge behavior.
type MergeOptions struct {
	// Strategy defines the default merge strategy.
	Strategy MergeStrategy

	// PreservePaths preserves paths from the existing document.
	PreservePaths bool

	// PreserveSchemas preserves schemas from the existing document.
	PreserveSchemas bool

	// PreserveInfo preserves info from the existing document.
	PreserveInfo bool

	// PreserveServers preserves servers from the existing document.
	PreserveServers bool

	// PreserveTags preserves tags from the existing document.
	PreserveTags bool

	// PreserveSecurity preserves security from the existing document.
	PreserveSecurity bool
}

// DefaultMergeOptions returns the default merge options.
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		Strategy:         MergeStrategyOverwrite,
		PreservePaths:    false,
		PreserveSchemas:  false,
		PreserveInfo:     true,
		PreserveServers:  true,
		PreserveTags:     true,
		PreserveSecurity: true,
	}
}

// Merger handles merging OpenAPI documents.
type Merger struct {
	options MergeOptions
}

// NewMerger creates a new Merger with the given options.
func NewMerger(options MergeOptions) *Merger {
	return &Merger{
		options: options,
	}
}

// Merge combines an existing OpenAPI document with a generated one.
// For now, this is a stub that returns the generated document.
// Future implementation will intelligently merge the documents.
func (m *Merger) Merge(existing, generated *types.OpenAPI) (*types.OpenAPI, error) {
	// Stub implementation: just return the generated document
	// TODO: Implement intelligent merging
	//
	// Future implementation should:
	// 1. Preserve manually-added descriptions and examples from existing
	// 2. Merge paths - keep existing paths not in generated, update common ones
	// 3. Merge schemas - keep existing schemas not in generated, update common ones
	// 4. Respect merge options for info, servers, tags, security
	// 5. Track which parts are generated vs manually maintained

	if existing == nil {
		return generated, nil
	}

	// For now, preserve some basic info from existing if configured
	result := generated

	if m.options.PreserveInfo && existing.Info.Title != "" {
		result.Info = existing.Info
	}

	if m.options.PreserveServers && len(existing.Servers) > 0 {
		result.Servers = existing.Servers
	}

	if m.options.PreserveTags && len(existing.Tags) > 0 {
		result.Tags = existing.Tags
	}

	if m.options.PreserveSecurity && len(existing.Security) > 0 {
		result.Security = existing.Security
	}

	return result, nil
}

// MergeDefault merges two documents using default options.
func MergeDefault(existing, generated *types.OpenAPI) (*types.OpenAPI, error) {
	merger := NewMerger(DefaultMergeOptions())
	return merger.Merge(existing, generated)
}
