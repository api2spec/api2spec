// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package types

// OpenAPI represents a complete OpenAPI 3.0/3.1 specification document.
type OpenAPI struct {
	// OpenAPI is the OpenAPI specification version (e.g., "3.0.3", "3.1.0")
	OpenAPI string `json:"openapi" yaml:"openapi"`

	// Info provides metadata about the API
	Info Info `json:"info" yaml:"info"`

	// Servers is a list of server objects
	Servers []Server `json:"servers,omitempty" yaml:"servers,omitempty"`

	// Paths holds the available paths and operations
	Paths map[string]PathItem `json:"paths,omitempty" yaml:"paths,omitempty"`

	// Components holds reusable objects
	Components *Components `json:"components,omitempty" yaml:"components,omitempty"`

	// Security is a list of security requirements
	Security []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`

	// Tags is a list of tags used by the specification
	Tags []Tag `json:"tags,omitempty" yaml:"tags,omitempty"`

	// ExternalDocs provides external documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	// Title is the title of the API
	Title string `json:"title" yaml:"title"`

	// Description is a description of the API
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// TermsOfService is a URL to the Terms of Service
	TermsOfService string `json:"termsOfService,omitempty" yaml:"termsOfService,omitempty"`

	// Contact provides contact information
	Contact *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`

	// License provides license information
	License *License `json:"license,omitempty" yaml:"license,omitempty"`

	// Version is the version of the API
	Version string `json:"version" yaml:"version"`
}

// Contact provides contact information.
type Contact struct {
	// Name is the name of the contact
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// URL is the URL of the contact
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Email is the email of the contact
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License provides license information.
type License struct {
	// Name is the name of the license
	Name string `json:"name" yaml:"name"`

	// URL is the URL of the license
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// Identifier is the SPDX license identifier (OpenAPI 3.1+)
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
}

// Server represents an API server.
type Server struct {
	// URL is the URL of the server
	URL string `json:"url" yaml:"url"`

	// Description is a description of the server
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Variables are server variables
	Variables map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable represents a server variable.
type ServerVariable struct {
	// Enum is a list of allowed values
	Enum []string `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Default is the default value
	Default string `json:"default" yaml:"default"`

	// Description is a description of the variable
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem represents an API path.
type PathItem struct {
	// Ref is a reference to another path item
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// Summary is a brief summary
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Description is a detailed description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Get is the GET operation
	Get *Operation `json:"get,omitempty" yaml:"get,omitempty"`

	// Put is the PUT operation
	Put *Operation `json:"put,omitempty" yaml:"put,omitempty"`

	// Post is the POST operation
	Post *Operation `json:"post,omitempty" yaml:"post,omitempty"`

	// Delete is the DELETE operation
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`

	// Options is the OPTIONS operation
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`

	// Head is the HEAD operation
	Head *Operation `json:"head,omitempty" yaml:"head,omitempty"`

	// Patch is the PATCH operation
	Patch *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`

	// Trace is the TRACE operation
	Trace *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`

	// Servers is a list of servers for this path
	Servers []Server `json:"servers,omitempty" yaml:"servers,omitempty"`

	// Parameters are parameters for all operations on this path
	Parameters []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// Operation represents an API operation.
type Operation struct {
	// Tags is a list of tags
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Summary is a brief summary
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`

	// Description is a detailed description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// ExternalDocs provides external documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`

	// OperationID is a unique identifier
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`

	// Parameters is a list of parameters
	Parameters []Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// RequestBody is the request body
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`

	// Responses is a map of responses
	Responses map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`

	// Callbacks is a map of callbacks
	Callbacks map[string]map[string]PathItem `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`

	// Deprecated indicates if the operation is deprecated
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// Security is a list of security requirements
	Security []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`

	// Servers is a list of servers
	Servers []Server `json:"servers,omitempty" yaml:"servers,omitempty"`
}

// Components holds reusable objects.
type Components struct {
	// Schemas is a map of schema objects
	Schemas map[string]*Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`

	// Responses is a map of response objects
	Responses map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`

	// Parameters is a map of parameter objects
	Parameters map[string]Parameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// Examples is a map of example objects
	Examples map[string]Example `json:"examples,omitempty" yaml:"examples,omitempty"`

	// RequestBodies is a map of request body objects
	RequestBodies map[string]RequestBody `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`

	// Headers is a map of header objects
	Headers map[string]Header `json:"headers,omitempty" yaml:"headers,omitempty"`

	// SecuritySchemes is a map of security scheme objects
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`

	// Links is a map of link objects
	Links map[string]Link `json:"links,omitempty" yaml:"links,omitempty"`

	// Callbacks is a map of callback objects
	Callbacks map[string]map[string]PathItem `json:"callbacks,omitempty" yaml:"callbacks,omitempty"`
}

// SecurityScheme represents a security scheme.
type SecurityScheme struct {
	// Type is the type of security scheme
	Type string `json:"type" yaml:"type"`

	// Description is a description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Name is the name of the header, query, or cookie parameter
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// In is the location of the API key (query, header, cookie)
	In string `json:"in,omitempty" yaml:"in,omitempty"`

	// Scheme is the HTTP authorization scheme
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// BearerFormat is the format of the bearer token
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`

	// Flows is OAuth2 flow definitions
	Flows *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`

	// OpenIDConnectURL is the OpenID Connect URL
	OpenIDConnectURL string `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

// OAuthFlows represents OAuth2 flow definitions.
type OAuthFlows struct {
	// Implicit is the implicit flow
	Implicit *OAuthFlow `json:"implicit,omitempty" yaml:"implicit,omitempty"`

	// Password is the password flow
	Password *OAuthFlow `json:"password,omitempty" yaml:"password,omitempty"`

	// ClientCredentials is the client credentials flow
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`

	// AuthorizationCode is the authorization code flow
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// OAuthFlow represents an OAuth2 flow.
type OAuthFlow struct {
	// AuthorizationURL is the authorization URL
	AuthorizationURL string `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`

	// TokenURL is the token URL
	TokenURL string `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`

	// RefreshURL is the refresh URL
	RefreshURL string `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`

	// Scopes is a map of available scopes
	Scopes map[string]string `json:"scopes" yaml:"scopes"`
}

// Link represents an OpenAPI link object.
type Link struct {
	// OperationRef is a reference to an operation
	OperationRef string `json:"operationRef,omitempty" yaml:"operationRef,omitempty"`

	// OperationID is the ID of an operation
	OperationID string `json:"operationId,omitempty" yaml:"operationId,omitempty"`

	// Parameters is a map of parameters
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`

	// RequestBody is the request body
	RequestBody interface{} `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`

	// Description is a description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Server is the server to use
	Server *Server `json:"server,omitempty" yaml:"server,omitempty"`
}

// Tag represents a tag object.
type Tag struct {
	// Name is the name of the tag
	Name string `json:"name" yaml:"name"`

	// Description is a description of the tag
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// ExternalDocs provides external documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}
