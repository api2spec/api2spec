// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package types provides core data structures for OpenAPI specification generation.
package types

// Route represents an HTTP route extracted from source code.
type Route struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Method string `json:"method" yaml:"method"`

	// Path is the URL path pattern (e.g., "/users/{id}")
	Path string `json:"path" yaml:"path"`

	// Handler is the name of the handler function
	Handler string `json:"handler,omitempty" yaml:"handler,omitempty"`

	// Summary is a brief description of the route
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Description is a detailed description of the route
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Tags are used to group routes in the OpenAPI spec
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// OperationID is a unique identifier for the operation
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`

	// Parameters are the route parameters (path, query, header, cookie)
	Parameters []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// RequestBody describes the request body for POST/PUT/PATCH
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`

	// Responses maps status codes to response definitions
	Responses map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`

	// Security specifies the security requirements for this route
	Security []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`

	// Deprecated indicates if the route is deprecated
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// SourceFile is the file where this route was defined
	SourceFile string `json:"sourceFile,omitempty" yaml:"sourceFile,omitempty"`

	// SourceLine is the line number where this route was defined
	SourceLine int `json:"sourceLine,omitempty" yaml:"sourceLine,omitempty"`
}

// Parameter represents an OpenAPI parameter.
type Parameter struct {
	// Name is the parameter name
	Name string `json:"name" yaml:"name"`

	// In is the location of the parameter (path, query, header, cookie)
	In string `json:"in" yaml:"in"`

	// Description is a brief description of the parameter
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Required indicates if the parameter is required
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	// Deprecated indicates if the parameter is deprecated
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// Schema defines the type of the parameter
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Example is an example value for the parameter
	Example interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody represents an OpenAPI request body.
type RequestBody struct {
	// Description is a brief description of the request body
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Required indicates if the request body is required
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	// Content maps media types to their schemas
	Content map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// Response represents an OpenAPI response.
type Response struct {
	// Description is a brief description of the response
	Description string `json:"description" yaml:"description"`

	// Headers maps header names to header definitions
	Headers map[string]Header `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Content maps media types to their schemas
	Content map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

// Header represents an OpenAPI header.
type Header struct {
	// Description is a brief description of the header
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Required indicates if the header is required
	Required bool `json:"required,omitempty" yaml:"required,omitempty"`

	// Deprecated indicates if the header is deprecated
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// Schema defines the type of the header
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// MediaType represents an OpenAPI media type.
type MediaType struct {
	// Schema defines the structure of the content
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Example is an example of the content
	Example interface{} `json:"example,omitempty" yaml:"example,omitempty"`

	// Examples maps example names to example objects
	Examples map[string]Example `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// Example represents an OpenAPI example.
type Example struct {
	// Summary is a brief summary of the example
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Description is a detailed description of the example
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Value is the example value
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`

	// ExternalValue is a URL pointing to the example
	ExternalValue string `json:"externalValue,omitempty" yaml:"externalValue,omitempty"`
}
