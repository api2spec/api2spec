// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package types

// Schema represents an OpenAPI schema object.
// It follows the JSON Schema Specification with OpenAPI extensions.
type Schema struct {
	// Ref is a reference to another schema ($ref)
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// Type is the data type (string, number, integer, boolean, array, object)
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Format is the data format (date-time, email, uuid, etc.)
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Title is a short title for the schema
	Title string `json:"title,omitempty" yaml:"title,omitempty"`

	// Description is a detailed description of the schema
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Default is the default value
	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`

	// Example is an example value
	Example interface{} `json:"example,omitempty" yaml:"example,omitempty"`

	// Enum is a list of allowed values
	Enum []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`

	// Nullable indicates if the value can be null
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`

	// ReadOnly indicates the value is read-only
	ReadOnly bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`

	// WriteOnly indicates the value is write-only
	WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`

	// Deprecated indicates the schema is deprecated
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// --- String Validation ---

	// MinLength is the minimum string length
	MinLength *int `json:"minLength,omitempty" yaml:"minLength,omitempty"`

	// MaxLength is the maximum string length
	MaxLength *int `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`

	// Pattern is a regex pattern for string validation
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	// --- Numeric Validation ---

	// Minimum is the minimum numeric value
	Minimum *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`

	// Maximum is the maximum numeric value
	Maximum *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`

	// ExclusiveMinimum indicates if minimum is exclusive
	ExclusiveMinimum bool `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`

	// ExclusiveMaximum indicates if maximum is exclusive
	ExclusiveMaximum bool `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`

	// MultipleOf specifies the value must be a multiple of this number
	MultipleOf *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`

	// --- Array Validation ---

	// Items is the schema for array items
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`

	// MinItems is the minimum number of array items
	MinItems *int `json:"minItems,omitempty" yaml:"minItems,omitempty"`

	// MaxItems is the maximum number of array items
	MaxItems *int `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`

	// UniqueItems indicates if array items must be unique
	UniqueItems bool `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`

	// --- Object Validation ---

	// Properties maps property names to their schemas
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Required is a list of required property names
	Required []string `json:"required,omitempty" yaml:"required,omitempty"`

	// AdditionalProperties defines the schema for additional properties
	AdditionalProperties *Schema `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`

	// MinProperties is the minimum number of properties
	MinProperties *int `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`

	// MaxProperties is the maximum number of properties
	MaxProperties *int `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`

	// --- Composition ---

	// AllOf is a list of schemas that must all be valid
	AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`

	// OneOf is a list of schemas where exactly one must be valid
	OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`

	// AnyOf is a list of schemas where at least one must be valid
	AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`

	// Not is a schema that must not be valid
	Not *Schema `json:"not,omitempty" yaml:"not,omitempty"`

	// Discriminator is used for polymorphism
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`

	// --- OpenAPI Extensions ---

	// XML provides XML representation details
	XML *XML `json:"xml,omitempty" yaml:"xml,omitempty"`

	// ExternalDocs provides external documentation
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
}

// Discriminator is used for polymorphic schemas.
type Discriminator struct {
	// PropertyName is the name of the property used for discrimination
	PropertyName string `json:"propertyName" yaml:"propertyName"`

	// Mapping maps discriminator values to schema references
	Mapping map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
}

// XML provides XML-specific metadata.
type XML struct {
	// Name is the XML element name
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Namespace is the XML namespace
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Prefix is the XML namespace prefix
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`

	// Attribute indicates if this is an XML attribute
	Attribute bool `json:"attribute,omitempty" yaml:"attribute,omitempty"`

	// Wrapped indicates if array items should be wrapped
	Wrapped bool `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
}

// ExternalDocs provides external documentation.
type ExternalDocs struct {
	// Description is a description of the external documentation
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// URL is the URL of the external documentation
	URL string `json:"url" yaml:"url"`
}
