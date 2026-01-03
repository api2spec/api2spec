// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/api2spec/api2spec/internal/parser"
	"github.com/api2spec/api2spec/pkg/types"
)

func TestGoSchemaExtractor_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		goType   string
		jsonType string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"int8", "integer"},
		{"int16", "integer"},
		{"int32", "integer"},
		{"int64", "integer"},
		{"uint", "integer"},
		{"uint8", "integer"},
		{"uint16", "integer"},
		{"uint32", "integer"},
		{"uint64", "integer"},
		{"float32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"byte", "integer"},
		{"rune", "integer"},
	}

	extractor := NewGoSchemaExtractor()

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			def := parser.StructDefinition{
				Name: "Test",
				Fields: []parser.StructField{
					{
						Name:     "Field",
						JSONName: "field",
						Type:     tt.goType,
						TypeKind: parser.KindPrimitive,
					},
				},
			}

			schema := extractor.ExtractFromStruct(def)
			require.NotNil(t, schema)
			require.NotNil(t, schema.Properties)
			require.Contains(t, schema.Properties, "field")
			assert.Equal(t, tt.jsonType, schema.Properties["field"].Type)
		})
	}
}

func TestGoSchemaExtractor_TimeType(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Event",
		Fields: []parser.StructField{
			{
				Name:     "CreatedAt",
				JSONName: "created_at",
				Type:     "time.Time",
				TypeKind: parser.KindTime,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	require.NotNil(t, schema.Properties["created_at"])
	assert.Equal(t, "string", schema.Properties["created_at"].Type)
	assert.Equal(t, "date-time", schema.Properties["created_at"].Format)
}

func TestGoSchemaExtractor_SliceType(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	tests := []struct {
		name        string
		elementType string
		wantType    string
		wantRef     string
	}{
		{"string slice", "string", "string", ""},
		{"int slice", "int", "integer", ""},
		{"struct slice", "User", "", "#/components/schemas/User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := parser.StructDefinition{
				Name: "Test",
				Fields: []parser.StructField{
					{
						Name:        "Items",
						JSONName:    "items",
						Type:        "[]" + tt.elementType,
						TypeKind:    parser.KindSlice,
						ElementType: tt.elementType,
					},
				},
			}

			schema := extractor.ExtractFromStruct(def)
			require.NotNil(t, schema.Properties["items"])
			assert.Equal(t, "array", schema.Properties["items"].Type)
			require.NotNil(t, schema.Properties["items"].Items)

			if tt.wantType != "" {
				assert.Equal(t, tt.wantType, schema.Properties["items"].Items.Type)
			}
			if tt.wantRef != "" {
				assert.Equal(t, tt.wantRef, schema.Properties["items"].Items.Ref)
			}
		})
	}
}

func TestGoSchemaExtractor_MapType(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Config",
		Fields: []parser.StructField{
			{
				Name:        "Settings",
				JSONName:    "settings",
				Type:        "map[string]int",
				TypeKind:    parser.KindMap,
				KeyType:     "string",
				ElementType: "int",
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	require.NotNil(t, schema.Properties["settings"])
	assert.Equal(t, "object", schema.Properties["settings"].Type)
	require.NotNil(t, schema.Properties["settings"].AdditionalProperties)
	assert.Equal(t, "integer", schema.Properties["settings"].AdditionalProperties.Type)
}

func TestGoSchemaExtractor_PointerType(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "User",
		Fields: []parser.StructField{
			{
				Name:      "Nickname",
				JSONName:  "nickname",
				Type:      "*string",
				TypeKind:  parser.KindPointer,
				IsPointer: true,
			},
			{
				Name:      "Age",
				JSONName:  "age",
				Type:      "*int",
				TypeKind:  parser.KindPointer,
				IsPointer: true,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)

	// Pointer to string
	require.NotNil(t, schema.Properties["nickname"])
	assert.Equal(t, "string", schema.Properties["nickname"].Type)
	assert.True(t, schema.Properties["nickname"].Nullable)

	// Pointer to int
	require.NotNil(t, schema.Properties["age"])
	assert.Equal(t, "integer", schema.Properties["age"].Type)
	assert.True(t, schema.Properties["age"].Nullable)
}

func TestGoSchemaExtractor_StructReference(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Order",
		Fields: []parser.StructField{
			{
				Name:     "User",
				JSONName: "user",
				Type:     "User",
				TypeKind: parser.KindStruct,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	require.NotNil(t, schema.Properties["user"])
	assert.Equal(t, "#/components/schemas/User", schema.Properties["user"].Ref)
}

func TestGoSchemaExtractor_NestedStruct(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Config",
		Fields: []parser.StructField{
			{
				Name:     "Database",
				JSONName: "database",
				Type:     "struct{}",
				TypeKind: parser.KindStruct,
				NestedStruct: []parser.StructField{
					{
						Name:     "Host",
						JSONName: "host",
						Type:     "string",
						TypeKind: parser.KindPrimitive,
					},
					{
						Name:     "Port",
						JSONName: "port",
						Type:     "int",
						TypeKind: parser.KindPrimitive,
					},
				},
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	require.NotNil(t, schema.Properties["database"])
	assert.Equal(t, "object", schema.Properties["database"].Type)
	require.NotNil(t, schema.Properties["database"].Properties)
	assert.Equal(t, "string", schema.Properties["database"].Properties["host"].Type)
	assert.Equal(t, "integer", schema.Properties["database"].Properties["port"].Type)
}

func TestGoSchemaExtractor_RequiredFields(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "User",
		Fields: []parser.StructField{
			{
				Name:       "Name",
				JSONName:   "name",
				Type:       "string",
				TypeKind:   parser.KindPrimitive,
				IsRequired: true,
			},
			{
				Name:      "Email",
				JSONName:  "email",
				Type:      "string",
				TypeKind:  parser.KindPrimitive,
				Omitempty: true,
			},
			{
				Name:      "Age",
				JSONName:  "age",
				Type:      "*int",
				TypeKind:  parser.KindPointer,
				IsPointer: true,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	assert.Contains(t, schema.Required, "name")
	assert.NotContains(t, schema.Required, "email") // omitempty
	assert.NotContains(t, schema.Required, "age")   // pointer
}

func TestGoSchemaExtractor_ValidationTags(t *testing.T) {
	tests := []struct {
		name           string
		validationTags map[string]string
		checkSchema    func(*testing.T, *types.Schema)
	}{
		{
			name:           "email format",
			validationTags: map[string]string{"email": "true"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				assert.Equal(t, "email", s.Format)
			},
		},
		{
			name:           "url format",
			validationTags: map[string]string{"url": "true"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				assert.Equal(t, "uri", s.Format)
			},
		},
		{
			name:           "uuid format",
			validationTags: map[string]string{"uuid": "true"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				assert.Equal(t, "uuid", s.Format)
			},
		},
		{
			name:           "string min/max",
			validationTags: map[string]string{"min": "5", "max": "100"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				require.NotNil(t, s.MinLength)
				require.NotNil(t, s.MaxLength)
				assert.Equal(t, 5, *s.MinLength)
				assert.Equal(t, 100, *s.MaxLength)
			},
		},
		{
			name:           "alphanum pattern",
			validationTags: map[string]string{"alphanum": "true"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				assert.Equal(t, "^[a-zA-Z0-9]+$", s.Pattern)
			},
		},
		{
			name:           "oneof enum",
			validationTags: map[string]string{"oneof": "active pending inactive"},
			checkSchema: func(t *testing.T, s *types.Schema) {
				require.Len(t, s.Enum, 3)
				assert.Equal(t, "active", s.Enum[0])
				assert.Equal(t, "pending", s.Enum[1])
				assert.Equal(t, "inactive", s.Enum[2])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewGoSchemaExtractor()

			def := parser.StructDefinition{
				Name: "Test",
				Fields: []parser.StructField{
					{
						Name:           "Field",
						JSONName:       "field",
						Type:           "string",
						TypeKind:       parser.KindPrimitive,
						ValidationTags: tt.validationTags,
					},
				},
			}

			schema := extractor.ExtractFromStruct(def)
			require.NotNil(t, schema.Properties["field"])
			tt.checkSchema(t, schema.Properties["field"])
		})
	}
}

func TestGoSchemaExtractor_NumericValidation(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Numeric",
		Fields: []parser.StructField{
			{
				Name:           "Age",
				JSONName:       "age",
				Type:           "int",
				TypeKind:       parser.KindPrimitive,
				ValidationTags: map[string]string{"min": "0", "max": "150"},
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	ageSchema := schema.Properties["age"]

	require.NotNil(t, ageSchema.Minimum)
	require.NotNil(t, ageSchema.Maximum)
	assert.Equal(t, 0.0, *ageSchema.Minimum)
	assert.Equal(t, 150.0, *ageSchema.Maximum)
}

func TestGoSchemaExtractor_ArrayValidation(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "Container",
		Fields: []parser.StructField{
			{
				Name:           "Items",
				JSONName:       "items",
				Type:           "[]string",
				TypeKind:       parser.KindSlice,
				ElementType:    "string",
				ValidationTags: map[string]string{"min": "1", "max": "10"},
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	itemsSchema := schema.Properties["items"]

	require.NotNil(t, itemsSchema.MinItems)
	require.NotNil(t, itemsSchema.MaxItems)
	assert.Equal(t, 1, *itemsSchema.MinItems)
	assert.Equal(t, 10, *itemsSchema.MaxItems)
}

func TestGoSchemaExtractor_SkipsJSONIgnored(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name: "User",
		Fields: []parser.StructField{
			{
				Name:     "ID",
				JSONName: "id",
				Type:     "string",
				TypeKind: parser.KindPrimitive,
			},
			{
				Name:     "InternalField",
				JSONName: "-",
				Type:     "string",
				TypeKind: parser.KindPrimitive,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	assert.Contains(t, schema.Properties, "id")
	assert.NotContains(t, schema.Properties, "-")
	assert.NotContains(t, schema.Properties, "InternalField")
}

func TestGoSchemaExtractor_Description(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def := parser.StructDefinition{
		Name:        "User",
		Description: "User represents a system user.",
		Fields: []parser.StructField{
			{
				Name:        "Email",
				JSONName:    "email",
				Type:        "string",
				TypeKind:    parser.KindPrimitive,
				Description: "The user's email address.",
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)
	assert.Equal(t, "User", schema.Title)
	assert.Equal(t, "User represents a system user.", schema.Description)
	assert.Equal(t, "The user's email address.", schema.Properties["email"].Description)
}

func TestGoSchemaExtractor_Registry(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	def1 := parser.StructDefinition{
		Name: "User",
		Fields: []parser.StructField{
			{Name: "ID", JSONName: "id", Type: "string", TypeKind: parser.KindPrimitive},
		},
	}

	def2 := parser.StructDefinition{
		Name: "Order",
		Fields: []parser.StructField{
			{Name: "ID", JSONName: "id", Type: "string", TypeKind: parser.KindPrimitive},
		},
	}

	extractor.ExtractFromStruct(def1)
	extractor.ExtractFromStruct(def2)

	registry := extractor.Registry()
	assert.Equal(t, 2, registry.Count())
	assert.True(t, registry.Has("User"))
	assert.True(t, registry.Has("Order"))

	userSchema, ok := registry.Get("User")
	require.True(t, ok)
	assert.Equal(t, "User", userSchema.Title)
}

func TestGoSchemaExtractor_FullExample(t *testing.T) {
	extractor := NewGoSchemaExtractor()

	// Simulate parsing this struct:
	// type User struct {
	//     ID        string    `json:"id"`
	//     Name      string    `json:"name" validate:"required,min=1,max=255"`
	//     Email     string    `json:"email" validate:"required,email"`
	//     Age       *int      `json:"age,omitempty" validate:"min=0,max=150"`
	//     Tags      []string  `json:"tags"`
	//     Metadata  map[string]string `json:"metadata"`
	//     CreatedAt time.Time `json:"created_at"`
	// }

	def := parser.StructDefinition{
		Name:        "User",
		Description: "User represents a user in the system.",
		Fields: []parser.StructField{
			{
				Name:     "ID",
				JSONName: "id",
				Type:     "string",
				TypeKind: parser.KindPrimitive,
			},
			{
				Name:           "Name",
				JSONName:       "name",
				Type:           "string",
				TypeKind:       parser.KindPrimitive,
				IsRequired:     true,
				ValidationTags: map[string]string{"min": "1", "max": "255"},
			},
			{
				Name:           "Email",
				JSONName:       "email",
				Type:           "string",
				TypeKind:       parser.KindPrimitive,
				IsRequired:     true,
				ValidationTags: map[string]string{"email": "true"},
			},
			{
				Name:           "Age",
				JSONName:       "age",
				Type:           "*int",
				TypeKind:       parser.KindPointer,
				IsPointer:      true,
				Omitempty:      true,
				ValidationTags: map[string]string{"min": "0", "max": "150"},
			},
			{
				Name:        "Tags",
				JSONName:    "tags",
				Type:        "[]string",
				TypeKind:    parser.KindSlice,
				ElementType: "string",
			},
			{
				Name:        "Metadata",
				JSONName:    "metadata",
				Type:        "map[string]string",
				TypeKind:    parser.KindMap,
				KeyType:     "string",
				ElementType: "string",
			},
			{
				Name:     "CreatedAt",
				JSONName: "created_at",
				Type:     "time.Time",
				TypeKind: parser.KindTime,
			},
		},
	}

	schema := extractor.ExtractFromStruct(def)

	// Verify basic structure
	assert.Equal(t, "object", schema.Type)
	assert.Equal(t, "User", schema.Title)
	assert.Len(t, schema.Properties, 7)

	// Verify required fields
	assert.Contains(t, schema.Required, "name")
	assert.Contains(t, schema.Required, "email")
	assert.NotContains(t, schema.Required, "id")
	assert.NotContains(t, schema.Required, "age")

	// Verify ID
	assert.Equal(t, "string", schema.Properties["id"].Type)

	// Verify Name with validation
	name := schema.Properties["name"]
	assert.Equal(t, "string", name.Type)
	assert.Equal(t, 1, *name.MinLength)
	assert.Equal(t, 255, *name.MaxLength)

	// Verify Email with format
	email := schema.Properties["email"]
	assert.Equal(t, "string", email.Type)
	assert.Equal(t, "email", email.Format)

	// Verify Age pointer with validation
	age := schema.Properties["age"]
	assert.Equal(t, "integer", age.Type)
	assert.True(t, age.Nullable)

	// Verify Tags array
	tags := schema.Properties["tags"]
	assert.Equal(t, "array", tags.Type)
	assert.Equal(t, "string", tags.Items.Type)

	// Verify Metadata map
	metadata := schema.Properties["metadata"]
	assert.Equal(t, "object", metadata.Type)
	assert.Equal(t, "string", metadata.AdditionalProperties.Type)

	// Verify CreatedAt time
	createdAt := schema.Properties["created_at"]
	assert.Equal(t, "string", createdAt.Type)
	assert.Equal(t, "date-time", createdAt.Format)
}
