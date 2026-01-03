// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoParser_ParseSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "valid Go source",
			source: `package main

func main() {}`,
			wantErr: false,
		},
		{
			name:    "invalid Go source",
			source:  `package main func`,
			wantErr: true,
		},
		{
			name: "empty package",
			source: `package empty
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGoParser()
			pf, err := p.ParseSource("test.go", tt.source)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, pf)
				assert.NotNil(t, pf.AST)
			}
		})
	}
}

func TestGoParser_ExtractStructs(t *testing.T) {
	source := `package models

// User represents a user in the system.
type User struct {
	// ID is the unique identifier
	ID        string    ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\" validate:\"required\"`" + `
	Email     string    ` + "`json:\"email\" validate:\"required,email\"`" + `
	Age       int       ` + "`json:\"age\" validate:\"min=0,max=150\"`" + `
	IsActive  bool      ` + "`json:\"is_active\"`" + `
	Score     float64   ` + "`json:\"score,omitempty\"`" + `
	Metadata  *string   ` + "`json:\"metadata,omitempty\"`" + `
	Tags      []string  ` + "`json:\"tags\"`" + `
	Counts    map[string]int ` + "`json:\"counts\"`" + `
	Internal  string    ` + "`json:\"-\"`" + `
}

type Address struct {
	Street  string ` + "`json:\"street\"`" + `
	City    string ` + "`json:\"city\"`" + `
	ZipCode string ` + "`json:\"zip_code\"`" + `
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("models.go", source)
	require.NoError(t, err)

	structs := p.ExtractStructs(pf)
	require.Len(t, structs, 2)

	// Test User struct
	user := structs[0]
	assert.Equal(t, "User", user.Name)
	assert.Contains(t, user.Description, "User represents a user")
	require.Len(t, user.Fields, 10)

	// Test ID field
	idField := user.Fields[0]
	assert.Equal(t, "ID", idField.Name)
	assert.Equal(t, "id", idField.JSONName)
	assert.Equal(t, "string", idField.Type)
	assert.Equal(t, KindPrimitive, idField.TypeKind)
	assert.False(t, idField.IsRequired)

	// Test Name field with required validation
	nameField := user.Fields[1]
	assert.Equal(t, "Name", nameField.Name)
	assert.Equal(t, "name", nameField.JSONName)
	assert.True(t, nameField.IsRequired)

	// Test Email field with multiple validations
	emailField := user.Fields[2]
	assert.Equal(t, "Email", emailField.Name)
	assert.Equal(t, "email", emailField.JSONName)
	assert.True(t, emailField.IsRequired)
	assert.Equal(t, "true", emailField.ValidationTags["email"])

	// Test Age field with min/max validation
	ageField := user.Fields[3]
	assert.Equal(t, "Age", ageField.Name)
	assert.Equal(t, "age", ageField.JSONName)
	assert.Equal(t, "int", ageField.Type)
	assert.Equal(t, "0", ageField.ValidationTags["min"])
	assert.Equal(t, "150", ageField.ValidationTags["max"])

	// Test IsActive bool field
	activeField := user.Fields[4]
	assert.Equal(t, "IsActive", activeField.Name)
	assert.Equal(t, "is_active", activeField.JSONName)
	assert.Equal(t, "bool", activeField.Type)
	assert.Equal(t, KindPrimitive, activeField.TypeKind)

	// Test Score with omitempty
	scoreField := user.Fields[5]
	assert.Equal(t, "Score", scoreField.Name)
	assert.Equal(t, "score", scoreField.JSONName)
	assert.True(t, scoreField.Omitempty)

	// Test Metadata pointer field
	metaField := user.Fields[6]
	assert.Equal(t, "Metadata", metaField.Name)
	assert.Equal(t, "metadata", metaField.JSONName)
	assert.Equal(t, "*string", metaField.Type)
	assert.Equal(t, KindPointer, metaField.TypeKind)
	assert.True(t, metaField.IsPointer)
	assert.True(t, metaField.Omitempty)

	// Test Tags slice field
	tagsField := user.Fields[7]
	assert.Equal(t, "Tags", tagsField.Name)
	assert.Equal(t, "tags", tagsField.JSONName)
	assert.Equal(t, "[]string", tagsField.Type)
	assert.Equal(t, KindSlice, tagsField.TypeKind)
	assert.Equal(t, "string", tagsField.ElementType)

	// Test Counts map field
	countsField := user.Fields[8]
	assert.Equal(t, "Counts", countsField.Name)
	assert.Equal(t, "counts", countsField.JSONName)
	assert.Equal(t, "map[string]int", countsField.Type)
	assert.Equal(t, KindMap, countsField.TypeKind)
	assert.Equal(t, "string", countsField.KeyType)
	assert.Equal(t, "int", countsField.ElementType)

	// Test Internal field with json:"-"
	internalField := user.Fields[9]
	assert.Equal(t, "Internal", internalField.Name)
	assert.Equal(t, "-", internalField.JSONName)

	// Test Address struct
	addr := structs[1]
	assert.Equal(t, "Address", addr.Name)
	assert.Len(t, addr.Fields, 3)
}

func TestGoParser_ExtractStructs_Embedded(t *testing.T) {
	source := `package models

type BaseModel struct {
	ID        string ` + "`json:\"id\"`" + `
	CreatedAt string ` + "`json:\"created_at\"`" + `
}

type User struct {
	BaseModel
	Name string ` + "`json:\"name\"`" + `
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("models.go", source)
	require.NoError(t, err)

	structs := p.ExtractStructs(pf)
	require.Len(t, structs, 2)

	user := structs[1]
	assert.Equal(t, "User", user.Name)
	assert.Contains(t, user.Embedded, "BaseModel")
	assert.Len(t, user.Fields, 1)
	assert.Equal(t, "Name", user.Fields[0].Name)
}

func TestGoParser_ExtractStructs_NestedStruct(t *testing.T) {
	source := `package models

type Config struct {
	Database struct {
		Host string ` + "`json:\"host\"`" + `
		Port int    ` + "`json:\"port\"`" + `
	} ` + "`json:\"database\"`" + `
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("models.go", source)
	require.NoError(t, err)

	structs := p.ExtractStructs(pf)
	require.Len(t, structs, 1)

	config := structs[0]
	assert.Equal(t, "Config", config.Name)
	require.Len(t, config.Fields, 1)

	dbField := config.Fields[0]
	assert.Equal(t, "Database", dbField.Name)
	assert.Equal(t, "database", dbField.JSONName)
	assert.Equal(t, KindStruct, dbField.TypeKind)
	require.Len(t, dbField.NestedStruct, 2)

	assert.Equal(t, "Host", dbField.NestedStruct[0].Name)
	assert.Equal(t, "host", dbField.NestedStruct[0].JSONName)
	assert.Equal(t, "Port", dbField.NestedStruct[1].Name)
	assert.Equal(t, "port", dbField.NestedStruct[1].JSONName)
}

func TestGoParser_ExtractStructs_TimeType(t *testing.T) {
	source := `package models

import "time"

type Event struct {
	Name      string    ` + "`json:\"name\"`" + `
	StartTime time.Time ` + "`json:\"start_time\"`" + `
	EndTime   *time.Time ` + "`json:\"end_time,omitempty\"`" + `
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("models.go", source)
	require.NoError(t, err)

	structs := p.ExtractStructs(pf)
	require.Len(t, structs, 1)

	event := structs[0]
	require.Len(t, event.Fields, 3)

	startTime := event.Fields[1]
	assert.Equal(t, "StartTime", startTime.Name)
	assert.Equal(t, "start_time", startTime.JSONName)
	assert.Equal(t, "time.Time", startTime.Type)
	assert.Equal(t, KindTime, startTime.TypeKind)

	endTime := event.Fields[2]
	assert.Equal(t, "EndTime", endTime.Name)
	assert.Equal(t, "*time.Time", endTime.Type)
	assert.Equal(t, KindPointer, endTime.TypeKind)
	assert.True(t, endTime.Omitempty)
}

func TestGoParser_FindImports(t *testing.T) {
	source := `package main

import (
	"fmt"
	"net/http"
	chi "github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo"
	_ "github.com/lib/pq"
)
`

	p := NewGoParser()
	pf, err := p.ParseSource("main.go", source)
	require.NoError(t, err)

	imports := p.FindImports(pf)

	assert.Equal(t, "fmt", imports["fmt"])
	assert.Equal(t, "net/http", imports["http"])
	assert.Equal(t, "github.com/go-chi/chi/v5", imports["chi"])
	assert.Equal(t, "github.com/onsi/ginkgo", imports["."])
	assert.Equal(t, "github.com/lib/pq", imports["_"])
}

func TestGoParser_HasImport(t *testing.T) {
	source := `package main

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)
`

	p := NewGoParser()
	pf, err := p.ParseSource("main.go", source)
	require.NoError(t, err)

	assert.True(t, p.HasImport(pf, "github.com/go-chi/chi/v5"))
	assert.True(t, p.HasImport(pf, "net/http"))
	assert.False(t, p.HasImport(pf, "github.com/gin-gonic/gin"))
}

func TestGoParser_FindMethodCalls(t *testing.T) {
	source := `package main

import "github.com/go-chi/chi/v5"

func SetupRoutes(r chi.Router) {
	r.Get("/users", ListUsers)
	r.Post("/users", CreateUser)
	r.Get("/users/{id}", GetUser)
	r.Put("/users/{id}", UpdateUser)
	r.Delete("/users/{id}", DeleteUser)
	r.Patch("/users/{id}/status", UpdateStatus)
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("routes.go", source)
	require.NoError(t, err)

	// Find all HTTP method calls on router
	calls := p.FindMethodCalls(pf, "^r$", "^(Get|Post|Put|Delete|Patch|Head|Options)$")
	require.Len(t, calls, 6)

	// Verify first call
	assert.Equal(t, "r", calls[0].Receiver)
	assert.Equal(t, "Get", calls[0].Method)
	assert.Equal(t, "/users", calls[0].Arguments[0])
	assert.Equal(t, "ListUsers", calls[0].Arguments[1])

	// Verify POST call
	assert.Equal(t, "Post", calls[1].Method)
	assert.Equal(t, "/users", calls[1].Arguments[0])
	assert.Equal(t, "CreateUser", calls[1].Arguments[1])

	// Verify GET with path param
	assert.Equal(t, "Get", calls[2].Method)
	assert.Equal(t, "/users/{id}", calls[2].Arguments[0])
	assert.Equal(t, "GetUser", calls[2].Arguments[1])
}

func TestGoParser_FindMethodCalls_NestedRouters(t *testing.T) {
	source := `package main

import "github.com/go-chi/chi/v5"

func SetupRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", HealthCheck)
		r.Route("/v1", func(r chi.Router) {
			r.Get("/users", ListUsers)
		})
	})
}
`

	p := NewGoParser()
	pf, err := p.ParseSource("routes.go", source)
	require.NoError(t, err)

	// Find Route calls
	routeCalls := p.FindMethodCalls(pf, "^r$", "^Route$")
	assert.Len(t, routeCalls, 2)

	// Find Get calls
	getCalls := p.FindMethodCalls(pf, "^r$", "^Get$")
	assert.Len(t, getCalls, 2)
}

func TestStructField_ParseValidateTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		required bool
		values   map[string]string
	}{
		{
			name:     "required only",
			tag:      "required",
			required: true,
			values:   map[string]string{},
		},
		{
			name:     "required with email",
			tag:      "required,email",
			required: true,
			values:   map[string]string{"email": "true"},
		},
		{
			name:     "min max values",
			tag:      "min=1,max=100",
			required: false,
			values:   map[string]string{"min": "1", "max": "100"},
		},
		{
			name:     "complex validation",
			tag:      "required,min=1,max=255,email,url",
			required: true,
			values:   map[string]string{"min": "1", "max": "255", "email": "true", "url": "true"},
		},
		{
			name:     "uuid format",
			tag:      "uuid4",
			required: false,
			values:   map[string]string{"uuid4": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf := &StructField{ValidationTags: make(map[string]string)}
			sf.parseValidateTag(tt.tag)

			assert.Equal(t, tt.required, sf.IsRequired)
			for k, v := range tt.values {
				assert.Equal(t, v, sf.ValidationTags[k], "validation tag %s", k)
			}
		})
	}
}

func TestExtractStringLiteral(t *testing.T) {
	source := `package main

var (
	a = "hello"
	b = 42
	c = "world"
)
`

	p := NewGoParser()
	pf, err := p.ParseSource("test.go", source)
	require.NoError(t, err)

	// This is a basic test - in real usage, we'd extract from call expressions
	assert.NotNil(t, pf)
}

func TestTypeKind_Classification(t *testing.T) {
	source := `package test

import "time"

type Example struct {
	String      string
	Int         int
	Float       float64
	Bool        bool
	Slice       []string
	Map         map[string]int
	Pointer     *string
	Time        time.Time
	TimePtr     *time.Time
	Interface   interface{}
	Any         any
	Struct      SomeStruct
	InlineStruct struct { X int }
}

type SomeStruct struct{}
`

	p := NewGoParser()
	pf, err := p.ParseSource("test.go", source)
	require.NoError(t, err)

	structs := p.ExtractStructs(pf)
	require.Len(t, structs, 2)

	example := structs[0]
	require.Len(t, example.Fields, 13)

	expectations := []struct {
		name string
		kind TypeKind
	}{
		{"String", KindPrimitive},
		{"Int", KindPrimitive},
		{"Float", KindPrimitive},
		{"Bool", KindPrimitive},
		{"Slice", KindSlice},
		{"Map", KindMap},
		{"Pointer", KindPointer},
		{"Time", KindTime},
		{"TimePtr", KindPointer},
		{"Interface", KindInterface},
		{"Any", KindStruct}, // 'any' is an alias, parsed as identifier
		{"Struct", KindStruct},
		{"InlineStruct", KindStruct},
	}

	for i, exp := range expectations {
		t.Run(exp.name, func(t *testing.T) {
			field := example.Fields[i]
			assert.Equal(t, exp.name, field.Name)
			assert.Equal(t, exp.kind, field.TypeKind, "field %s", field.Name)
		})
	}
}
