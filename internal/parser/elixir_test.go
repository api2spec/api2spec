// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElixirParser_ExtractEctoSchemas(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedCount  int
		expectedModule string
		expectedTable  string
		expectedFields []string
	}{
		{
			name: "basic schema",
			content: `
defmodule MyApp.Schemas.User do
  use Ecto.Schema

  schema "users" do
    field :name, :string
    field :email, :string
    field :age, :integer
  end
end
`,
			expectedCount:  1,
			expectedModule: "MyApp.Schemas.User",
			expectedTable:  "users",
			expectedFields: []string{"name", "email", "age"},
		},
		{
			name: "schema with defaults",
			content: `
defmodule MyApp.Post do
  use Ecto.Schema

  schema "posts" do
    field :title, :string
    field :published, :boolean, default: false
    field :view_count, :integer, default: 0
  end
end
`,
			expectedCount:  1,
			expectedModule: "MyApp.Post",
			expectedTable:  "posts",
			expectedFields: []string{"title", "published", "view_count"},
		},
		{
			name: "embedded schema",
			content: `
defmodule MyApp.Error do
  use Ecto.Schema

  @primary_key false
  embedded_schema do
    field :code, :string
    field :message, :string
  end
end
`,
			expectedCount:  1,
			expectedModule: "MyApp.Error",
			expectedTable:  "",
			expectedFields: []string{"code", "message"},
		},
		{
			name: "no schema definition",
			content: `
defmodule MyApp.Controller do
  use Phoenix.Controller

  def index(conn, _params) do
    json(conn, %{})
  end
end
`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewElixirParser()
			pf := p.Parse("test.ex", []byte(tt.content))

			assert.Len(t, pf.Schemas, tt.expectedCount)

			if tt.expectedCount > 0 {
				schema := pf.Schemas[0]
				assert.Equal(t, tt.expectedModule, schema.ModuleName)
				assert.Equal(t, tt.expectedTable, schema.TableName)

				var fieldNames []string
				for _, f := range schema.Fields {
					fieldNames = append(fieldNames, f.Name)
				}
				assert.Equal(t, tt.expectedFields, fieldNames)
			}
		})
	}
}

func TestElixirParser_ExtractEctoFields_WithTypes(t *testing.T) {
	content := `
defmodule MyApp.CompleteSchema do
  use Ecto.Schema

  schema "items" do
    field :name, :string
    field :count, :integer
    field :price, :float
    field :active, :boolean
    field :data, :map
    field :created_at, :datetime
    field :uuid, :uuid
  end
end
`
	p := NewElixirParser()
	pf := p.Parse("test.ex", []byte(content))

	require.Len(t, pf.Schemas, 1)
	schema := pf.Schemas[0]

	expectedTypes := map[string]string{
		"name":       "string",
		"count":      "integer",
		"price":      "float",
		"active":     "boolean",
		"data":       "map",
		"created_at": "datetime",
		"uuid":       "uuid",
	}

	for _, field := range schema.Fields {
		expected, ok := expectedTypes[field.Name]
		require.True(t, ok, "unexpected field: %s", field.Name)
		assert.Equal(t, expected, field.Type, "field %s type mismatch", field.Name)
	}
}

func TestElixirParser_ExtractEctoFields_WithDefaults(t *testing.T) {
	content := `
defmodule MyApp.WithDefaults do
  use Ecto.Schema

  schema "items" do
    field :name, :string
    field :active, :boolean, default: true
    field :count, :integer, default: 0
  end
end
`
	p := NewElixirParser()
	pf := p.Parse("test.ex", []byte(content))

	require.Len(t, pf.Schemas, 1)
	schema := pf.Schemas[0]

	// Find fields by name
	var nameField, activeField, countField *EctoField
	for i := range schema.Fields {
		switch schema.Fields[i].Name {
		case "name":
			nameField = &schema.Fields[i]
		case "active":
			activeField = &schema.Fields[i]
		case "count":
			countField = &schema.Fields[i]
		}
	}

	require.NotNil(t, nameField)
	assert.False(t, nameField.HasDefault)

	require.NotNil(t, activeField)
	assert.True(t, activeField.HasDefault)
	assert.Equal(t, "true", activeField.Default)

	require.NotNil(t, countField)
	assert.True(t, countField.HasDefault)
	assert.Equal(t, "0", countField.Default)
}
