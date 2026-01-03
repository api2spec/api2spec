// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"strings"

	"github.com/api2spec/api2spec/pkg/types"
)

// MergeStrategy defines how to handle conflicts during merge.
type MergeStrategy string

const (
	// MergeStrategyGeneratedWins overwrites with generated values on conflict.
	MergeStrategyGeneratedWins MergeStrategy = "generated-wins"

	// MergeStrategyExistingWins keeps existing values on conflict.
	MergeStrategyExistingWins MergeStrategy = "existing-wins"

	// MergeStrategyMerge attempts to merge values intelligently.
	MergeStrategyMerge MergeStrategy = "merge"
)

// MergeOptions configures the merge behavior.
type MergeOptions struct {
	// ConflictStrategy defines how to handle conflicts.
	ConflictStrategy MergeStrategy

	// PreserveDescriptions keeps human-written descriptions from existing spec.
	PreserveDescriptions bool

	// PreserveExamples keeps manually added examples from existing spec.
	PreserveExamples bool

	// PreserveTags keeps custom tags from existing spec.
	PreserveTags bool

	// PreserveExtensions keeps x-* extensions from existing spec.
	PreserveExtensions bool

	// MarkRemovedAsDeprecated marks routes removed from source as deprecated
	// instead of deleting them.
	MarkRemovedAsDeprecated bool

	// PreserveInfo preserves info from the existing document.
	PreserveInfo bool

	// PreserveServers preserves servers from the existing document.
	PreserveServers bool

	// PreserveSecurity preserves security from the existing document.
	PreserveSecurity bool
}

// DefaultMergeOptions returns the default merge options.
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		ConflictStrategy:        MergeStrategyMerge,
		PreserveDescriptions:    true,
		PreserveExamples:        true,
		PreserveTags:            true,
		PreserveExtensions:      true,
		MarkRemovedAsDeprecated: false,
		PreserveInfo:            true,
		PreserveServers:         true,
		PreserveSecurity:        true,
	}
}

// MergeResult contains the result of a merge operation.
type MergeResult struct {
	// Document is the merged OpenAPI document.
	Document *types.OpenAPI

	// AddedPaths is a list of paths that were added from generated spec.
	AddedPaths []string

	// RemovedPaths is a list of paths that were removed (or deprecated).
	RemovedPaths []string

	// UpdatedPaths is a list of paths that were updated.
	UpdatedPaths []string

	// AddedSchemas is a list of schemas that were added.
	AddedSchemas []string

	// RemovedSchemas is a list of schemas that were removed.
	RemovedSchemas []string

	// UpdatedSchemas is a list of schemas that were updated.
	UpdatedSchemas []string
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
func (m *Merger) Merge(existing, generated *types.OpenAPI) (*types.OpenAPI, error) {
	result, err := m.MergeWithResult(existing, generated)
	if err != nil {
		return nil, err
	}
	return result.Document, nil
}

// MergeWithResult combines documents and returns detailed merge results.
func (m *Merger) MergeWithResult(existing, generated *types.OpenAPI) (*MergeResult, error) {
	result := &MergeResult{
		AddedPaths:     []string{},
		RemovedPaths:   []string{},
		UpdatedPaths:   []string{},
		AddedSchemas:   []string{},
		RemovedSchemas: []string{},
		UpdatedSchemas: []string{},
	}

	if existing == nil {
		result.Document = generated
		if generated != nil && generated.Paths != nil {
			for path := range generated.Paths {
				result.AddedPaths = append(result.AddedPaths, path)
			}
		}
		return result, nil
	}

	if generated == nil {
		result.Document = existing
		return result, nil
	}

	// Start with the generated document as the base
	merged := &types.OpenAPI{
		OpenAPI: generated.OpenAPI,
		Info:    generated.Info,
	}

	// Merge info
	merged.Info = m.mergeInfo(existing.Info, generated.Info)

	// Merge servers
	merged.Servers = m.mergeServers(existing.Servers, generated.Servers)

	// Merge paths
	merged.Paths, result.AddedPaths, result.RemovedPaths, result.UpdatedPaths = m.mergePaths(existing.Paths, generated.Paths)

	// Merge components
	merged.Components, result.AddedSchemas, result.RemovedSchemas, result.UpdatedSchemas = m.mergeComponents(existing.Components, generated.Components)

	// Merge tags
	merged.Tags = m.mergeTags(existing.Tags, generated.Tags)

	// Merge security
	merged.Security = m.mergeSecurity(existing.Security, generated.Security)

	// Preserve external docs from existing if not in generated
	if generated.ExternalDocs == nil && existing.ExternalDocs != nil {
		merged.ExternalDocs = existing.ExternalDocs
	} else {
		merged.ExternalDocs = generated.ExternalDocs
	}

	result.Document = merged
	return result, nil
}

// mergeInfo merges Info objects.
func (m *Merger) mergeInfo(existing, generated types.Info) types.Info {
	if m.options.PreserveInfo {
		// Keep existing info but update version if generated has one
		result := existing

		// Always use generated title if it's not empty and existing is default
		if generated.Title != "" && (existing.Title == "" || existing.Title == "API") {
			result.Title = generated.Title
		}

		// Keep existing description if preserving and it exists
		if !m.options.PreserveDescriptions || existing.Description == "" {
			result.Description = generated.Description
		}

		return result
	}

	return generated
}

// mergeServers merges Server arrays.
func (m *Merger) mergeServers(existing, generated []types.Server) []types.Server {
	if m.options.PreserveServers && len(existing) > 0 {
		return existing
	}
	if len(generated) > 0 {
		return generated
	}
	return existing
}

// mergePaths merges Path objects and tracks changes.
func (m *Merger) mergePaths(existing, generated map[string]types.PathItem) (map[string]types.PathItem, []string, []string, []string) {
	var added, removed, updated []string

	if existing == nil && generated == nil {
		return nil, added, removed, updated
	}

	result := make(map[string]types.PathItem)

	// Track which paths exist in generated
	generatedPaths := make(map[string]bool)
	for path := range generated {
		generatedPaths[path] = true
	}

	// Process generated paths
	for path, genItem := range generated {
		if existItem, exists := existing[path]; exists {
			// Path exists in both - merge
			result[path] = m.mergePathItem(existItem, genItem)
			updated = append(updated, path)
		} else {
			// New path from generated
			result[path] = genItem
			added = append(added, path)
		}
	}

	// Handle paths that exist only in existing
	for path, existItem := range existing {
		if !generatedPaths[path] {
			if m.options.MarkRemovedAsDeprecated {
				// Mark all operations as deprecated instead of removing
				deprecatedItem := m.deprecatePathItem(existItem)
				result[path] = deprecatedItem
				removed = append(removed, path+" (deprecated)")
			} else {
				// Path is removed - don't include it
				removed = append(removed, path)
			}
		}
	}

	return result, added, removed, updated
}

// mergePathItem merges two PathItem objects.
func (m *Merger) mergePathItem(existing, generated types.PathItem) types.PathItem {
	result := generated

	// Preserve path-level description if option is set
	if m.options.PreserveDescriptions && existing.Description != "" && generated.Description == "" {
		result.Description = existing.Description
	}

	// Merge operations
	result.Get = m.mergeOperation(existing.Get, generated.Get)
	result.Post = m.mergeOperation(existing.Post, generated.Post)
	result.Put = m.mergeOperation(existing.Put, generated.Put)
	result.Delete = m.mergeOperation(existing.Delete, generated.Delete)
	result.Patch = m.mergeOperation(existing.Patch, generated.Patch)
	result.Head = m.mergeOperation(existing.Head, generated.Head)
	result.Options = m.mergeOperation(existing.Options, generated.Options)
	result.Trace = m.mergeOperation(existing.Trace, generated.Trace)

	// Merge path-level parameters
	result.Parameters = m.mergeParameters(existing.Parameters, generated.Parameters)

	return result
}

// mergeOperation merges two Operation objects.
func (m *Merger) mergeOperation(existing, generated *types.Operation) *types.Operation {
	if generated == nil {
		if existing != nil && m.options.MarkRemovedAsDeprecated {
			deprecated := *existing
			deprecated.Deprecated = true
			return &deprecated
		}
		return nil
	}

	if existing == nil {
		return generated
	}

	result := *generated

	// Preserve description
	if m.options.PreserveDescriptions {
		if existing.Description != "" && generated.Description == "" {
			result.Description = existing.Description
		}
		if existing.Summary != "" && generated.Summary == "" {
			result.Summary = existing.Summary
		}
	}

	// Preserve tags (merge them)
	if m.options.PreserveTags && len(existing.Tags) > 0 {
		result.Tags = m.mergeStringSlices(existing.Tags, generated.Tags)
	}

	// Merge parameters
	result.Parameters = m.mergeParameters(existing.Parameters, generated.Parameters)

	// Merge responses
	result.Responses = m.mergeResponses(existing.Responses, generated.Responses)

	// Preserve request body examples
	if m.options.PreserveExamples && existing.RequestBody != nil && generated.RequestBody != nil {
		result.RequestBody = m.mergeRequestBody(existing.RequestBody, generated.RequestBody)
	}

	// Preserve security if it was explicitly set
	if m.options.PreserveSecurity && len(existing.Security) > 0 && len(generated.Security) == 0 {
		result.Security = existing.Security
	}

	return &result
}

// mergeParameters merges parameter arrays.
func (m *Merger) mergeParameters(existing, generated []types.Parameter) []types.Parameter {
	if len(generated) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return generated
	}

	// Build a map of generated parameters by name+in
	genParams := make(map[string]types.Parameter)
	for _, p := range generated {
		key := p.Name + ":" + p.In
		genParams[key] = p
	}

	result := make([]types.Parameter, 0, len(generated))

	// Add all generated parameters, merging with existing if present
	for _, genParam := range generated {
		key := genParam.Name + ":" + genParam.In
		merged := genParam

		// Look for matching existing parameter
		for _, existParam := range existing {
			existKey := existParam.Name + ":" + existParam.In
			if existKey == key {
				// Preserve description if option is set
				if m.options.PreserveDescriptions && existParam.Description != "" && genParam.Description == "" {
					merged.Description = existParam.Description
				}
				// Preserve example
				if m.options.PreserveExamples && existParam.Example != nil && genParam.Example == nil {
					merged.Example = existParam.Example
				}
				break
			}
		}

		result = append(result, merged)
	}

	return result
}

// mergeResponses merges response maps.
func (m *Merger) mergeResponses(existing, generated map[string]types.Response) map[string]types.Response {
	if len(generated) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return generated
	}

	result := make(map[string]types.Response)

	// Add all generated responses
	for code, genResp := range generated {
		result[code] = genResp
	}

	// Merge with existing
	for code, existResp := range existing {
		if genResp, exists := result[code]; exists {
			// Merge the responses
			merged := genResp

			// Preserve description
			if m.options.PreserveDescriptions && existResp.Description != "" && genResp.Description == "" {
				merged.Description = existResp.Description
			}

			// Preserve examples in content
			if m.options.PreserveExamples && existResp.Content != nil {
				if merged.Content == nil {
					merged.Content = existResp.Content
				} else {
					for mediaType, existContent := range existResp.Content {
						if genContent, ok := merged.Content[mediaType]; ok {
							// Merge content
							if existContent.Example != nil && genContent.Example == nil {
								mergedContent := genContent
								mergedContent.Example = existContent.Example
								merged.Content[mediaType] = mergedContent
							}
						} else {
							// Keep existing content type
							merged.Content[mediaType] = existContent
						}
					}
				}
			}

			result[code] = merged
		} else {
			// Keep existing response codes not in generated
			result[code] = existResp
		}
	}

	return result
}

// mergeRequestBody merges request body objects.
func (m *Merger) mergeRequestBody(existing, generated *types.RequestBody) *types.RequestBody {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}

	result := *generated

	// Preserve description
	if m.options.PreserveDescriptions && existing.Description != "" && generated.Description == "" {
		result.Description = existing.Description
	}

	// Preserve examples in content
	if m.options.PreserveExamples && existing.Content != nil {
		if result.Content == nil {
			result.Content = existing.Content
		} else {
			for mediaType, existContent := range existing.Content {
				if genContent, ok := result.Content[mediaType]; ok {
					if existContent.Example != nil && genContent.Example == nil {
						mergedContent := genContent
						mergedContent.Example = existContent.Example
						result.Content[mediaType] = mergedContent
					}
				}
			}
		}
	}

	return &result
}

// mergeComponents merges Components objects.
func (m *Merger) mergeComponents(existing, generated *types.Components) (*types.Components, []string, []string, []string) {
	var added, removed, updated []string

	if existing == nil && generated == nil {
		return nil, added, removed, updated
	}

	if generated == nil {
		return existing, added, removed, updated
	}

	if existing == nil {
		if generated.Schemas != nil {
			for name := range generated.Schemas {
				added = append(added, name)
			}
		}
		return generated, added, removed, updated
	}

	result := &types.Components{}

	// Merge schemas
	result.Schemas, added, removed, updated = m.mergeSchemas(existing.Schemas, generated.Schemas)

	// Merge other component types (keep existing if not in generated)
	result.Responses = m.mergeComponentResponses(existing.Responses, generated.Responses)
	result.Parameters = m.mergeComponentParameters(existing.Parameters, generated.Parameters)
	result.Examples = m.mergeComponentExamples(existing.Examples, generated.Examples)
	result.RequestBodies = m.mergeComponentRequestBodies(existing.RequestBodies, generated.RequestBodies)
	result.Headers = m.mergeComponentHeaders(existing.Headers, generated.Headers)
	result.SecuritySchemes = m.mergeSecuritySchemes(existing.SecuritySchemes, generated.SecuritySchemes)
	result.Links = m.mergeComponentLinks(existing.Links, generated.Links)
	result.Callbacks = m.mergeComponentCallbacks(existing.Callbacks, generated.Callbacks)

	return result, added, removed, updated
}

// mergeSchemas merges schema maps.
func (m *Merger) mergeSchemas(existing, generated map[string]*types.Schema) (map[string]*types.Schema, []string, []string, []string) {
	var added, removed, updated []string

	if existing == nil && generated == nil {
		return nil, added, removed, updated
	}

	result := make(map[string]*types.Schema)

	// Track which schemas exist in generated
	generatedSchemas := make(map[string]bool)
	for name := range generated {
		generatedSchemas[name] = true
	}

	// Process generated schemas
	for name, genSchema := range generated {
		if existSchema, exists := existing[name]; exists {
			// Schema exists in both - merge
			result[name] = m.mergeSchema(existSchema, genSchema)
			updated = append(updated, name)
		} else {
			// New schema from generated
			result[name] = genSchema
			added = append(added, name)
		}
	}

	// Handle schemas that exist only in existing
	for name, existSchema := range existing {
		if !generatedSchemas[name] {
			// Keep existing schema (don't remove user-defined schemas)
			result[name] = existSchema
			// Don't mark as removed since we're keeping it
		}
	}

	return result, added, removed, updated
}

// mergeSchema merges two Schema objects.
func (m *Merger) mergeSchema(existing, generated *types.Schema) *types.Schema {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}

	// Start with generated schema
	result := *generated

	// Preserve description
	if m.options.PreserveDescriptions && existing.Description != "" && generated.Description == "" {
		result.Description = existing.Description
	}

	// Preserve example
	if m.options.PreserveExamples && existing.Example != nil && generated.Example == nil {
		result.Example = existing.Example
	}

	// Recursively merge properties
	if existing.Properties != nil && generated.Properties != nil {
		result.Properties = make(map[string]*types.Schema)
		for name, genProp := range generated.Properties {
			if existProp, exists := existing.Properties[name]; exists {
				result.Properties[name] = m.mergeSchema(existProp, genProp)
			} else {
				result.Properties[name] = genProp
			}
		}
		// Keep properties only in existing
		for name, existProp := range existing.Properties {
			if _, exists := generated.Properties[name]; !exists {
				result.Properties[name] = existProp
			}
		}
	}

	// Merge array items
	if existing.Items != nil && generated.Items != nil {
		result.Items = m.mergeSchema(existing.Items, generated.Items)
	}

	return &result
}

// mergeTags merges tag arrays.
func (m *Merger) mergeTags(existing, generated []types.Tag) []types.Tag {
	if !m.options.PreserveTags {
		return generated
	}

	if len(existing) == 0 {
		return generated
	}
	if len(generated) == 0 {
		return existing
	}

	// Build map of existing tags
	existingMap := make(map[string]types.Tag)
	for _, tag := range existing {
		existingMap[tag.Name] = tag
	}

	// Build result with generated tags, merging descriptions from existing
	result := make([]types.Tag, 0, len(generated))
	seenTags := make(map[string]bool)

	for _, genTag := range generated {
		merged := genTag
		if existTag, exists := existingMap[genTag.Name]; exists {
			// Preserve description from existing
			if m.options.PreserveDescriptions && existTag.Description != "" && genTag.Description == "" {
				merged.Description = existTag.Description
			}
			// Preserve external docs
			if existTag.ExternalDocs != nil && genTag.ExternalDocs == nil {
				merged.ExternalDocs = existTag.ExternalDocs
			}
		}
		result = append(result, merged)
		seenTags[genTag.Name] = true
	}

	// Add existing tags not in generated
	for _, existTag := range existing {
		if !seenTags[existTag.Name] {
			result = append(result, existTag)
		}
	}

	return result
}

// mergeSecurity merges security requirement arrays.
func (m *Merger) mergeSecurity(existing, generated []map[string][]string) []map[string][]string {
	if m.options.PreserveSecurity && len(existing) > 0 {
		return existing
	}
	if len(generated) > 0 {
		return generated
	}
	return existing
}

// deprecatePathItem marks all operations in a path item as deprecated.
func (m *Merger) deprecatePathItem(item types.PathItem) types.PathItem {
	result := item

	if result.Get != nil {
		deprecated := *result.Get
		deprecated.Deprecated = true
		result.Get = &deprecated
	}
	if result.Post != nil {
		deprecated := *result.Post
		deprecated.Deprecated = true
		result.Post = &deprecated
	}
	if result.Put != nil {
		deprecated := *result.Put
		deprecated.Deprecated = true
		result.Put = &deprecated
	}
	if result.Delete != nil {
		deprecated := *result.Delete
		deprecated.Deprecated = true
		result.Delete = &deprecated
	}
	if result.Patch != nil {
		deprecated := *result.Patch
		deprecated.Deprecated = true
		result.Patch = &deprecated
	}
	if result.Head != nil {
		deprecated := *result.Head
		deprecated.Deprecated = true
		result.Head = &deprecated
	}
	if result.Options != nil {
		deprecated := *result.Options
		deprecated.Deprecated = true
		result.Options = &deprecated
	}
	if result.Trace != nil {
		deprecated := *result.Trace
		deprecated.Deprecated = true
		result.Trace = &deprecated
	}

	return result
}

// mergeStringSlices merges two string slices, removing duplicates.
func (m *Merger) mergeStringSlices(existing, generated []string) []string {
	seen := make(map[string]bool)
	var result []string

	// Add all from generated first
	for _, s := range generated {
		s = strings.TrimSpace(s)
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}

	// Add from existing if not already present
	for _, s := range existing {
		s = strings.TrimSpace(s)
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}

	return result
}

// Helper functions for merging component types

func (m *Merger) mergeComponentResponses(existing, generated map[string]types.Response) map[string]types.Response {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.Response)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeComponentParameters(existing, generated map[string]types.Parameter) map[string]types.Parameter {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.Parameter)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeComponentExamples(existing, generated map[string]types.Example) map[string]types.Example {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.Example)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeComponentRequestBodies(existing, generated map[string]types.RequestBody) map[string]types.RequestBody {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.RequestBody)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeComponentHeaders(existing, generated map[string]types.Header) map[string]types.Header {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.Header)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeSecuritySchemes(existing, generated map[string]types.SecurityScheme) map[string]types.SecurityScheme {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.SecurityScheme)
	// Keep existing security schemes (don't overwrite)
	for k, v := range existing {
		result[k] = v
	}
	// Add generated ones only if not already present
	for k, v := range generated {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}
	return result
}

func (m *Merger) mergeComponentLinks(existing, generated map[string]types.Link) map[string]types.Link {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]types.Link)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

func (m *Merger) mergeComponentCallbacks(existing, generated map[string]map[string]types.PathItem) map[string]map[string]types.PathItem {
	if generated == nil {
		return existing
	}
	if existing == nil {
		return generated
	}
	result := make(map[string]map[string]types.PathItem)
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}

// MergeDefault merges two documents using default options.
func MergeDefault(existing, generated *types.OpenAPI) (*types.OpenAPI, error) {
	merger := NewMerger(DefaultMergeOptions())
	return merger.Merge(existing, generated)
}
