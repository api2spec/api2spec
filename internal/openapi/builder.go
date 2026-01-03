// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package openapi provides OpenAPI specification building and manipulation.
package openapi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/api2spec/api2spec/internal/config"
	"github.com/api2spec/api2spec/pkg/types"
)

// Builder constructs OpenAPI specifications from routes and schemas.
type Builder struct {
	config *config.Config
}

// NewBuilder creates a new OpenAPI builder with the given configuration.
func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{
		config: cfg,
	}
}

// Build creates an OpenAPI document from routes and schemas.
func (b *Builder) Build(routes []types.Route, schemas []types.Schema) (*types.OpenAPI, error) {
	doc := &types.OpenAPI{
		OpenAPI: b.config.OpenAPI.Version,
		Info:    b.buildInfo(),
		Servers: b.buildServers(),
		Paths:   make(map[string]types.PathItem),
		Tags:    b.buildTags(),
	}

	// Build paths from routes
	if err := b.buildPaths(doc, routes); err != nil {
		return nil, fmt.Errorf("failed to build paths: %w", err)
	}

	// Build components from schemas
	if len(schemas) > 0 {
		doc.Components = b.buildComponents(schemas)
	}

	// Add security if configured
	if len(b.config.OpenAPI.Security.Schemes) > 0 {
		doc.Security = b.buildSecurity()
		if doc.Components == nil {
			doc.Components = &types.Components{}
		}
		doc.Components.SecuritySchemes = b.buildSecuritySchemes()
	}

	return doc, nil
}

// buildInfo constructs the Info object from configuration.
func (b *Builder) buildInfo() types.Info {
	info := types.Info{
		Title:          b.config.OpenAPI.Info.Title,
		Description:    b.config.OpenAPI.Info.Description,
		TermsOfService: b.config.OpenAPI.Info.TermsOfService,
		Version:        b.config.OpenAPI.Info.Version,
	}

	if b.config.OpenAPI.Info.Contact.Name != "" ||
		b.config.OpenAPI.Info.Contact.Email != "" ||
		b.config.OpenAPI.Info.Contact.URL != "" {
		info.Contact = &types.Contact{
			Name:  b.config.OpenAPI.Info.Contact.Name,
			URL:   b.config.OpenAPI.Info.Contact.URL,
			Email: b.config.OpenAPI.Info.Contact.Email,
		}
	}

	if b.config.OpenAPI.Info.License.Name != "" {
		info.License = &types.License{
			Name: b.config.OpenAPI.Info.License.Name,
			URL:  b.config.OpenAPI.Info.License.URL,
		}
	}

	return info
}

// buildServers constructs the servers list from configuration.
func (b *Builder) buildServers() []types.Server {
	servers := make([]types.Server, 0, len(b.config.OpenAPI.Servers))
	for _, s := range b.config.OpenAPI.Servers {
		servers = append(servers, types.Server{
			URL:         s.URL,
			Description: s.Description,
		})
	}
	return servers
}

// buildTags constructs the tags list from configuration.
func (b *Builder) buildTags() []types.Tag {
	tags := make([]types.Tag, 0, len(b.config.OpenAPI.Tags))
	for _, t := range b.config.OpenAPI.Tags {
		tags = append(tags, types.Tag{
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return tags
}

// buildPaths constructs paths from routes.
func (b *Builder) buildPaths(doc *types.OpenAPI, routes []types.Route) error {
	for _, route := range routes {
		pathItem, exists := doc.Paths[route.Path]
		if !exists {
			pathItem = types.PathItem{}
		}

		operation := b.routeToOperation(route)

		switch strings.ToUpper(route.Method) {
		case "GET":
			pathItem.Get = operation
		case "POST":
			pathItem.Post = operation
		case "PUT":
			pathItem.Put = operation
		case "DELETE":
			pathItem.Delete = operation
		case "PATCH":
			pathItem.Patch = operation
		case "OPTIONS":
			pathItem.Options = operation
		case "HEAD":
			pathItem.Head = operation
		case "TRACE":
			pathItem.Trace = operation
		default:
			return fmt.Errorf("unsupported HTTP method: %s", route.Method)
		}

		doc.Paths[route.Path] = pathItem
	}

	return nil
}

// routeToOperation converts a Route to an OpenAPI Operation.
func (b *Builder) routeToOperation(route types.Route) *types.Operation {
	op := &types.Operation{
		Tags:        route.Tags,
		Summary:     route.Summary,
		Description: route.Description,
		OperationID: route.OperationID,
		Deprecated:  route.Deprecated,
	}

	// Copy parameters
	if len(route.Parameters) > 0 {
		op.Parameters = make([]types.Parameter, len(route.Parameters))
		copy(op.Parameters, route.Parameters)
	}

	// Copy request body
	if route.RequestBody != nil {
		op.RequestBody = route.RequestBody
	}

	// Copy responses
	if len(route.Responses) > 0 {
		op.Responses = make(map[string]types.Response)
		for code, resp := range route.Responses {
			op.Responses[code] = resp
		}
	} else {
		// Add default responses based on config
		op.Responses = b.buildDefaultResponses()
	}

	// Copy security
	if len(route.Security) > 0 {
		op.Security = route.Security
	}

	return op
}

// buildDefaultResponses creates default responses based on configuration.
func (b *Builder) buildDefaultResponses() map[string]types.Response {
	responses := make(map[string]types.Response)

	for _, code := range b.config.Generation.DefaultResponses {
		switch code {
		case "200":
			responses["200"] = types.Response{
				Description: "Successful response",
			}
		case "201":
			responses["201"] = types.Response{
				Description: "Created",
			}
		case "204":
			responses["204"] = types.Response{
				Description: "No content",
			}
		case "400":
			responses["400"] = types.Response{
				Description: "Bad request",
			}
		case "401":
			responses["401"] = types.Response{
				Description: "Unauthorized",
			}
		case "403":
			responses["403"] = types.Response{
				Description: "Forbidden",
			}
		case "404":
			responses["404"] = types.Response{
				Description: "Not found",
			}
		case "500":
			responses["500"] = types.Response{
				Description: "Internal server error",
			}
		default:
			responses[code] = types.Response{
				Description: fmt.Sprintf("Response %s", code),
			}
		}
	}

	// Ensure at least one response exists
	if len(responses) == 0 {
		responses["200"] = types.Response{
			Description: "Successful response",
		}
	}

	return responses
}

// buildComponents constructs the Components object from schemas.
func (b *Builder) buildComponents(schemas []types.Schema) *types.Components {
	components := &types.Components{
		Schemas: make(map[string]*types.Schema),
	}

	for i := range schemas {
		schema := schemas[i]
		name := schema.Title
		if name == "" {
			// Generate a name if not provided
			name = fmt.Sprintf("Schema%d", i+1)
		}
		components.Schemas[name] = &schema
	}

	return components
}

// buildSecurity constructs the global security requirements.
func (b *Builder) buildSecurity() []map[string][]string {
	if len(b.config.OpenAPI.Security.Default) == 0 {
		return nil
	}

	security := make([]map[string][]string, 0, len(b.config.OpenAPI.Security.Default))
	for _, name := range b.config.OpenAPI.Security.Default {
		security = append(security, map[string][]string{
			name: {},
		})
	}

	return security
}

// buildSecuritySchemes constructs security scheme definitions.
func (b *Builder) buildSecuritySchemes() map[string]types.SecurityScheme {
	schemes := make(map[string]types.SecurityScheme)

	for name, cfg := range b.config.OpenAPI.Security.Schemes {
		scheme := types.SecurityScheme{
			Type:         cfg.Type,
			Description:  cfg.Description,
			Name:         cfg.Name,
			In:           cfg.In,
			Scheme:       cfg.Scheme,
			BearerFormat: cfg.BearerFormat,
		}
		schemes[name] = scheme
	}

	return schemes
}

// SchemaRef creates a reference to a schema in components.
func SchemaRef(schemaName string) *types.Schema {
	return &types.Schema{
		Ref: "#/components/schemas/" + schemaName,
	}
}

// SortedPaths returns a sorted list of path keys for deterministic output.
func SortedPaths(paths map[string]types.PathItem) []string {
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SortedSchemas returns a sorted list of schema keys for deterministic output.
func SortedSchemas(schemas map[string]*types.Schema) []string {
	keys := make([]string, 0, len(schemas))
	for k := range schemas {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
