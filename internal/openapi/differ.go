// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package openapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/api2spec/api2spec/pkg/types"
)

// DiffType represents the type of change detected.
type DiffType string

const (
	// DiffTypeAdded indicates a new item was added.
	DiffTypeAdded DiffType = "added"

	// DiffTypeRemoved indicates an item was removed.
	DiffTypeRemoved DiffType = "removed"

	// DiffTypeModified indicates an item was modified.
	DiffTypeModified DiffType = "modified"
)

// PathChange represents a change to a path/operation.
type PathChange struct {
	Type        DiffType
	Path        string
	Method      string
	Description string
}

// SchemaChange represents a change to a schema.
type SchemaChange struct {
	Type        DiffType
	Name        string
	Description string
}

// DiffResult contains the differences between two OpenAPI documents.
type DiffResult struct {
	// PathChanges contains all path/operation changes.
	PathChanges []PathChange

	// SchemaChanges contains all schema changes.
	SchemaChanges []SchemaChange

	// HasBreakingChanges indicates if any breaking changes were detected.
	HasBreakingChanges bool

	// Summary provides a human-readable summary of changes.
	Summary string
}

// IsEmpty returns true if there are no differences.
func (d *DiffResult) IsEmpty() bool {
	return len(d.PathChanges) == 0 && len(d.SchemaChanges) == 0
}

// Differ compares two OpenAPI documents.
type Differ struct{}

// NewDiffer creates a new Differ.
func NewDiffer() *Differ {
	return &Differ{}
}

// Diff compares two OpenAPI documents and returns the differences.
func (d *Differ) Diff(a, b *types.OpenAPI) (*DiffResult, error) {
	result := &DiffResult{
		PathChanges:   []PathChange{},
		SchemaChanges: []SchemaChange{},
	}

	// Compare paths
	d.diffPaths(a, b, result)

	// Compare schemas
	d.diffSchemas(a, b, result)

	// Check for breaking changes
	result.HasBreakingChanges = d.detectBreakingChanges(result)

	// Generate summary
	result.Summary = d.generateSummary(result)

	return result, nil
}

// diffPaths compares the paths between two documents.
func (d *Differ) diffPaths(a, b *types.OpenAPI, result *DiffResult) {
	aPaths := make(map[string]types.PathItem)
	bPaths := make(map[string]types.PathItem)

	if a != nil && a.Paths != nil {
		aPaths = a.Paths
	}
	if b != nil && b.Paths != nil {
		bPaths = b.Paths
	}

	// Find removed and modified paths
	for path, aItem := range aPaths {
		bItem, exists := bPaths[path]
		if !exists {
			// Path was removed
			methods := d.getPathMethods(aItem)
			for _, method := range methods {
				result.PathChanges = append(result.PathChanges, PathChange{
					Type:        DiffTypeRemoved,
					Path:        path,
					Method:      method,
					Description: fmt.Sprintf("Removed %s %s", method, path),
				})
			}
		} else {
			// Check for method changes
			d.diffPathItem(path, aItem, bItem, result)
		}
	}

	// Find added paths
	for path, bItem := range bPaths {
		if _, exists := aPaths[path]; !exists {
			methods := d.getPathMethods(bItem)
			for _, method := range methods {
				result.PathChanges = append(result.PathChanges, PathChange{
					Type:        DiffTypeAdded,
					Path:        path,
					Method:      method,
					Description: fmt.Sprintf("Added %s %s", method, path),
				})
			}
		}
	}
}

// diffPathItem compares operations within a path item.
func (d *Differ) diffPathItem(path string, a, b types.PathItem, result *DiffResult) {
	methods := []struct {
		name string
		aOp  *types.Operation
		bOp  *types.Operation
	}{
		{"GET", a.Get, b.Get},
		{"POST", a.Post, b.Post},
		{"PUT", a.Put, b.Put},
		{"DELETE", a.Delete, b.Delete},
		{"PATCH", a.Patch, b.Patch},
		{"OPTIONS", a.Options, b.Options},
		{"HEAD", a.Head, b.Head},
		{"TRACE", a.Trace, b.Trace},
	}

	for _, m := range methods {
		if m.aOp == nil && m.bOp != nil {
			result.PathChanges = append(result.PathChanges, PathChange{
				Type:        DiffTypeAdded,
				Path:        path,
				Method:      m.name,
				Description: fmt.Sprintf("Added %s %s", m.name, path),
			})
		} else if m.aOp != nil && m.bOp == nil {
			result.PathChanges = append(result.PathChanges, PathChange{
				Type:        DiffTypeRemoved,
				Path:        path,
				Method:      m.name,
				Description: fmt.Sprintf("Removed %s %s", m.name, path),
			})
		} else if m.aOp != nil && m.bOp != nil {
			// Check if operation was modified
			if d.operationModified(m.aOp, m.bOp) {
				result.PathChanges = append(result.PathChanges, PathChange{
					Type:        DiffTypeModified,
					Path:        path,
					Method:      m.name,
					Description: fmt.Sprintf("Modified %s %s", m.name, path),
				})
			}
		}
	}
}

// operationModified checks if an operation was modified.
func (d *Differ) operationModified(a, b *types.Operation) bool {
	// Check for basic differences
	if a.Summary != b.Summary ||
		a.Description != b.Description ||
		a.OperationID != b.OperationID ||
		a.Deprecated != b.Deprecated {
		return true
	}

	// Check parameter count
	if len(a.Parameters) != len(b.Parameters) {
		return true
	}

	// Check response count
	if len(a.Responses) != len(b.Responses) {
		return true
	}

	// Check tags
	if len(a.Tags) != len(b.Tags) {
		return true
	}

	return false
}

// getPathMethods returns the HTTP methods defined for a path item.
func (d *Differ) getPathMethods(item types.PathItem) []string {
	var methods []string
	if item.Get != nil {
		methods = append(methods, "GET")
	}
	if item.Post != nil {
		methods = append(methods, "POST")
	}
	if item.Put != nil {
		methods = append(methods, "PUT")
	}
	if item.Delete != nil {
		methods = append(methods, "DELETE")
	}
	if item.Patch != nil {
		methods = append(methods, "PATCH")
	}
	if item.Options != nil {
		methods = append(methods, "OPTIONS")
	}
	if item.Head != nil {
		methods = append(methods, "HEAD")
	}
	if item.Trace != nil {
		methods = append(methods, "TRACE")
	}
	return methods
}

// diffSchemas compares the schemas between two documents.
func (d *Differ) diffSchemas(a, b *types.OpenAPI, result *DiffResult) {
	aSchemas := make(map[string]*types.Schema)
	bSchemas := make(map[string]*types.Schema)

	if a != nil && a.Components != nil && a.Components.Schemas != nil {
		aSchemas = a.Components.Schemas
	}
	if b != nil && b.Components != nil && b.Components.Schemas != nil {
		bSchemas = b.Components.Schemas
	}

	// Find removed and modified schemas
	for name, aSchema := range aSchemas {
		bSchema, exists := bSchemas[name]
		if !exists {
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Type:        DiffTypeRemoved,
				Name:        name,
				Description: fmt.Sprintf("Removed schema: %s", name),
			})
		} else if d.schemaModified(aSchema, bSchema) {
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Type:        DiffTypeModified,
				Name:        name,
				Description: fmt.Sprintf("Modified schema: %s", name),
			})
		}
	}

	// Find added schemas
	for name := range bSchemas {
		if _, exists := aSchemas[name]; !exists {
			result.SchemaChanges = append(result.SchemaChanges, SchemaChange{
				Type:        DiffTypeAdded,
				Name:        name,
				Description: fmt.Sprintf("Added schema: %s", name),
			})
		}
	}
}

// schemaModified checks if a schema was modified.
func (d *Differ) schemaModified(a, b *types.Schema) bool {
	if a == nil || b == nil {
		return a != b
	}

	// Basic property comparison
	if a.Type != b.Type ||
		a.Format != b.Format ||
		a.Title != b.Title ||
		a.Description != b.Description ||
		a.Nullable != b.Nullable ||
		a.Deprecated != b.Deprecated {
		return true
	}

	// Check properties count
	if len(a.Properties) != len(b.Properties) {
		return true
	}

	// Check required fields
	if len(a.Required) != len(b.Required) {
		return true
	}

	return false
}

// detectBreakingChanges checks if any changes are breaking.
func (d *Differ) detectBreakingChanges(result *DiffResult) bool {
	// Removed paths are breaking
	for _, change := range result.PathChanges {
		if change.Type == DiffTypeRemoved {
			return true
		}
	}

	// Removed schemas are breaking
	for _, change := range result.SchemaChanges {
		if change.Type == DiffTypeRemoved {
			return true
		}
	}

	return false
}

// generateSummary creates a human-readable summary of changes.
func (d *Differ) generateSummary(result *DiffResult) string {
	if result.IsEmpty() {
		return "No changes detected"
	}

	var sb strings.Builder

	// Count changes by type
	pathAdded, pathRemoved, pathModified := 0, 0, 0
	for _, c := range result.PathChanges {
		switch c.Type {
		case DiffTypeAdded:
			pathAdded++
		case DiffTypeRemoved:
			pathRemoved++
		case DiffTypeModified:
			pathModified++
		}
	}

	schemaAdded, schemaRemoved, schemaModified := 0, 0, 0
	for _, c := range result.SchemaChanges {
		switch c.Type {
		case DiffTypeAdded:
			schemaAdded++
		case DiffTypeRemoved:
			schemaRemoved++
		case DiffTypeModified:
			schemaModified++
		}
	}

	// Build summary
	var parts []string

	if pathAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) added", pathAdded))
	}
	if pathRemoved > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) removed", pathRemoved))
	}
	if pathModified > 0 {
		parts = append(parts, fmt.Sprintf("%d path(s) modified", pathModified))
	}
	if schemaAdded > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) added", schemaAdded))
	}
	if schemaRemoved > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) removed", schemaRemoved))
	}
	if schemaModified > 0 {
		parts = append(parts, fmt.Sprintf("%d schema(s) modified", schemaModified))
	}

	sb.WriteString(strings.Join(parts, ", "))

	if result.HasBreakingChanges {
		sb.WriteString(" [BREAKING CHANGES DETECTED]")
	}

	return sb.String()
}

// FormatDiff returns a formatted string representation of the diff.
func FormatDiff(result *DiffResult) string {
	if result.IsEmpty() {
		return "No differences found."
	}

	var sb strings.Builder

	sb.WriteString("=== OpenAPI Diff ===\n\n")
	sb.WriteString(result.Summary)
	sb.WriteString("\n\n")

	if len(result.PathChanges) > 0 {
		sb.WriteString("--- Path Changes ---\n")

		// Sort changes for deterministic output
		changes := make([]PathChange, len(result.PathChanges))
		copy(changes, result.PathChanges)
		sort.Slice(changes, func(i, j int) bool {
			if changes[i].Path != changes[j].Path {
				return changes[i].Path < changes[j].Path
			}
			return changes[i].Method < changes[j].Method
		})

		for _, c := range changes {
			symbol := "  "
			switch c.Type {
			case DiffTypeAdded:
				symbol = "+ "
			case DiffTypeRemoved:
				symbol = "- "
			case DiffTypeModified:
				symbol = "~ "
			}
			sb.WriteString(fmt.Sprintf("%s%s %s\n", symbol, c.Method, c.Path))
		}
		sb.WriteString("\n")
	}

	if len(result.SchemaChanges) > 0 {
		sb.WriteString("--- Schema Changes ---\n")

		// Sort changes for deterministic output
		changes := make([]SchemaChange, len(result.SchemaChanges))
		copy(changes, result.SchemaChanges)
		sort.Slice(changes, func(i, j int) bool {
			return changes[i].Name < changes[j].Name
		})

		for _, c := range changes {
			symbol := "  "
			switch c.Type {
			case DiffTypeAdded:
				symbol = "+ "
			case DiffTypeRemoved:
				symbol = "- "
			case DiffTypeModified:
				symbol = "~ "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", symbol, c.Name))
		}
	}

	return sb.String()
}
