// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"context"
	"fmt"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
)

// PythonParser provides Python AST parsing capabilities using tree-sitter.
type PythonParser struct {
	parser *sitter.Parser
}

// NewPythonParser creates a new Python parser.
func NewPythonParser() *PythonParser {
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	return &PythonParser{
		parser: parser,
	}
}

// ParsedPythonFile represents a parsed Python source file.
type ParsedPythonFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content []byte

	// Tree is the tree-sitter parse tree
	Tree *sitter.Tree

	// RootNode is the root node of the AST
	RootNode *sitter.Node

	// DecoratedFunctions contains extracted decorated function definitions
	DecoratedFunctions []PythonDecoratedFunction

	// Classes contains extracted class definitions
	Classes []PythonClass

	// PydanticModels contains extracted Pydantic model definitions
	PydanticModels []PydanticModel

	// Imports contains imported module names
	Imports []PythonImport
}

// PythonDecoratedFunction represents a function with decorators.
type PythonDecoratedFunction struct {
	// Name is the function name
	Name string

	// Decorators are the decorators applied to the function
	Decorators []PythonDecorator

	// Parameters are the function parameters
	Parameters []PythonParameter

	// ReturnType is the return type annotation if present
	ReturnType string

	// IsAsync indicates if the function is async
	IsAsync bool

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// PythonDecorator represents a decorator on a function or class.
type PythonDecorator struct {
	// Name is the decorator name (e.g., "app.route", "get")
	Name string

	// Arguments are the decorator arguments
	Arguments []string

	// KeywordArguments are keyword arguments (e.g., methods=['GET'])
	KeywordArguments map[string]string

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// PythonParameter represents a function parameter.
type PythonParameter struct {
	// Name is the parameter name
	Name string

	// Type is the type annotation if present
	Type string

	// Default is the default value if present
	Default string

	// IsRequired indicates if the parameter is required
	IsRequired bool
}

// PythonClass represents a class definition.
type PythonClass struct {
	// Name is the class name
	Name string

	// Bases are the base classes
	Bases []string

	// Decorators are the decorators applied to the class
	Decorators []PythonDecorator

	// Methods are the class methods
	Methods []PythonDecoratedFunction

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// PydanticModel represents a Pydantic model (BaseModel subclass).
type PydanticModel struct {
	// Name is the model name
	Name string

	// Fields are the model fields
	Fields []PydanticField

	// Line is the source line number
	Line int

	// Node is the tree-sitter node
	Node *sitter.Node
}

// PydanticField represents a field in a Pydantic model.
type PydanticField struct {
	// Name is the field name
	Name string

	// Type is the type annotation
	Type string

	// Default is the default value if present
	Default string

	// IsOptional indicates if the field is optional
	IsOptional bool

	// Description is the field description if present
	Description string
}

// PythonImport represents an import statement.
type PythonImport struct {
	// Module is the module being imported
	Module string

	// Names are the names imported from the module
	Names []string

	// Alias is the import alias if present
	Alias string

	// Line is the source line number
	Line int
}

// ParseSource parses Python source code from a string.
func (p *PythonParser) ParseSource(filename string, source string) (*ParsedPythonFile, error) {
	return p.Parse(filename, []byte(source))
}

// Parse parses Python source code from bytes.
func (p *PythonParser) Parse(filename string, content []byte) (*ParsedPythonFile, error) {
	tree, err := p.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Python: %w", err)
	}

	rootNode := tree.RootNode()
	if rootNode == nil {
		return nil, fmt.Errorf("failed to get root node")
	}

	pf := &ParsedPythonFile{
		Path:               filename,
		Content:            content,
		Tree:               tree,
		RootNode:           rootNode,
		DecoratedFunctions: []PythonDecoratedFunction{},
		Classes:            []PythonClass{},
		PydanticModels:     []PydanticModel{},
		Imports:            []PythonImport{},
	}

	// Extract definitions
	pf.Imports = p.ExtractImports(rootNode, content)
	pf.DecoratedFunctions = p.ExtractDecoratedFunctions(rootNode, content)
	pf.Classes = p.ExtractClasses(rootNode, content)
	pf.PydanticModels = p.ExtractPydanticModels(rootNode, content)

	return pf, nil
}

// ParseFile parses a Python source file from disk.
func (p *PythonParser) ParseFile(path string) (*ParsedPythonFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return p.Parse(path, content)
}

// ExtractImports extracts all import statements from the AST.
func (p *PythonParser) ExtractImports(rootNode *sitter.Node, content []byte) []PythonImport {
	var imports []PythonImport

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		switch node.Type() {
		case "import_statement":
			imp := p.parseImportStatement(node, content)
			if imp != nil {
				imports = append(imports, *imp)
			}
		case "import_from_statement":
			imp := p.parseImportFromStatement(node, content)
			if imp != nil {
				imports = append(imports, *imp)
			}
		}
		return true
	})

	return imports
}

// parseImportStatement parses an import statement (import x).
func (p *PythonParser) parseImportStatement(node *sitter.Node, content []byte) *PythonImport {
	imp := &PythonImport{
		Line:  int(node.StartPoint().Row) + 1,
		Names: []string{},
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "dotted_name":
			imp.Module = child.Content(content)
			imp.Names = append(imp.Names, child.Content(content))
		case "aliased_import":
			// Handle "import x as y"
			for j := 0; j < int(child.ChildCount()); j++ {
				subChild := child.Child(j)
				if subChild.Type() == "dotted_name" && imp.Module == "" {
					imp.Module = subChild.Content(content)
					imp.Names = append(imp.Names, subChild.Content(content))
				} else if subChild.Type() == "identifier" {
					imp.Alias = subChild.Content(content)
				}
			}
		}
	}

	return imp
}

// parseImportFromStatement parses a from...import statement.
func (p *PythonParser) parseImportFromStatement(node *sitter.Node, content []byte) *PythonImport {
	imp := &PythonImport{
		Line:  int(node.StartPoint().Row) + 1,
		Names: []string{},
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "dotted_name", "relative_import":
			if imp.Module == "" {
				imp.Module = child.Content(content)
			}
		case "identifier":
			// Single name import
			imp.Names = append(imp.Names, child.Content(content))
		case "aliased_import":
			// Handle "from x import y as z"
			for j := 0; j < int(child.ChildCount()); j++ {
				subChild := child.Child(j)
				if subChild.Type() == "identifier" {
					imp.Names = append(imp.Names, subChild.Content(content))
				}
			}
		case "import_prefix":
			// Relative imports
			imp.Module = child.Content(content)
		case "wildcard_import":
			imp.Names = append(imp.Names, "*")
		}
	}

	return imp
}

// ExtractDecoratedFunctions extracts all decorated function definitions.
func (p *PythonParser) ExtractDecoratedFunctions(rootNode *sitter.Node, content []byte) []PythonDecoratedFunction {
	var functions []PythonDecoratedFunction

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "decorated_definition" {
			fn := p.parseDecoratedFunction(node, content)
			if fn != nil {
				functions = append(functions, *fn)
			}
			return false // Don't recurse into decorated definitions
		}
		return true
	})

	return functions
}

// parseDecoratedFunction parses a decorated function definition.
func (p *PythonParser) parseDecoratedFunction(node *sitter.Node, content []byte) *PythonDecoratedFunction {
	fn := &PythonDecoratedFunction{
		Line:       int(node.StartPoint().Row) + 1,
		Decorators: []PythonDecorator{},
		Parameters: []PythonParameter{},
		Node:       node,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "decorator":
			dec := p.parseDecorator(child, content)
			if dec != nil {
				fn.Decorators = append(fn.Decorators, *dec)
			}
		case "function_definition":
			p.parseFunctionDef(child, content, fn)
		}
	}

	if fn.Name == "" {
		return nil
	}

	return fn
}

// parseDecorator parses a decorator node.
func (p *PythonParser) parseDecorator(node *sitter.Node, content []byte) *PythonDecorator {
	dec := &PythonDecorator{
		Line:             int(node.StartPoint().Row) + 1,
		Arguments:        []string{},
		KeywordArguments: make(map[string]string),
		Node:             node,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			dec.Name = child.Content(content)
		case "attribute":
			// Handle @app.route style decorators
			dec.Name = child.Content(content)
		case "call":
			// Handle @decorator(...) style
			p.parseDecoratorCall(child, content, dec)
		}
	}

	return dec
}

// parseDecoratorCall parses a decorator call expression.
func (p *PythonParser) parseDecoratorCall(node *sitter.Node, content []byte, dec *PythonDecorator) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if dec.Name == "" {
				dec.Name = child.Content(content)
			}
		case "attribute":
			if dec.Name == "" {
				dec.Name = child.Content(content)
			}
		case "argument_list":
			p.parseDecoratorArguments(child, content, dec)
		}
	}
}

// parseDecoratorArguments parses decorator arguments.
func (p *PythonParser) parseDecoratorArguments(node *sitter.Node, content []byte, dec *PythonDecorator) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "string":
			// Positional string argument
			dec.Arguments = append(dec.Arguments, trimQuotes(child.Content(content)))
		case "keyword_argument":
			// Keyword argument like methods=['GET']
			key, value := p.parseKeywordArgument(child, content)
			if key != "" {
				dec.KeywordArguments[key] = value
			}
		case "identifier":
			dec.Arguments = append(dec.Arguments, child.Content(content))
		case "attribute":
			dec.Arguments = append(dec.Arguments, child.Content(content))
		}
	}
}

// parseKeywordArgument parses a keyword argument.
func (p *PythonParser) parseKeywordArgument(node *sitter.Node, content []byte) (string, string) {
	var key, value string
	foundEquals := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childType := child.Type()

		if childType == "identifier" && key == "" {
			key = child.Content(content)
			continue
		}

		if childType == "=" {
			foundEquals = true
			continue
		}

		// After finding key and =, the next node is the value
		if foundEquals && key != "" && value == "" {
			value = child.Content(content)
		}
	}

	return key, value
}

// parseFunctionDef parses a function definition node.
func (p *PythonParser) parseFunctionDef(node *sitter.Node, content []byte, fn *PythonDecoratedFunction) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			fn.Name = child.Content(content)
		case "parameters":
			fn.Parameters = p.parseParameters(child, content)
		case "type":
			fn.ReturnType = child.Content(content)
		case "async":
			fn.IsAsync = true
		}
	}
}

// parseParameters parses function parameters.
func (p *PythonParser) parseParameters(node *sitter.Node, content []byte) []PythonParameter {
	var params []PythonParameter

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			params = append(params, PythonParameter{
				Name:       child.Content(content),
				IsRequired: true,
			})
		case "typed_parameter":
			param := p.parseTypedParameter(child, content)
			if param != nil {
				params = append(params, *param)
			}
		case "default_parameter":
			param := p.parseDefaultParameter(child, content)
			if param != nil {
				params = append(params, *param)
			}
		case "typed_default_parameter":
			param := p.parseTypedDefaultParameter(child, content)
			if param != nil {
				params = append(params, *param)
			}
		}
	}

	return params
}

// parseTypedParameter parses a typed parameter (name: type).
func (p *PythonParser) parseTypedParameter(node *sitter.Node, content []byte) *PythonParameter {
	param := &PythonParameter{IsRequired: true}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if param.Name == "" {
				param.Name = child.Content(content)
			}
		case "type":
			param.Type = child.Content(content)
		}
	}

	if param.Name == "" {
		return nil
	}

	return param
}

// parseDefaultParameter parses a default parameter (name=default).
func (p *PythonParser) parseDefaultParameter(node *sitter.Node, content []byte) *PythonParameter {
	param := &PythonParameter{IsRequired: false}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if param.Name == "" {
				param.Name = child.Content(content)
			}
		default:
			if param.Name != "" && param.Default == "" && child.Type() != "=" {
				param.Default = child.Content(content)
			}
		}
	}

	if param.Name == "" {
		return nil
	}

	return param
}

// parseTypedDefaultParameter parses a typed default parameter (name: type = default).
func (p *PythonParser) parseTypedDefaultParameter(node *sitter.Node, content []byte) *PythonParameter {
	param := &PythonParameter{IsRequired: false}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if param.Name == "" {
				param.Name = child.Content(content)
			}
		case "type":
			param.Type = child.Content(content)
		default:
			if child.Type() != "=" && child.Type() != ":" && param.Name != "" && param.Type != "" && param.Default == "" {
				param.Default = child.Content(content)
			}
		}
	}

	if param.Name == "" {
		return nil
	}

	return param
}

// ExtractClasses extracts all class definitions from the AST.
func (p *PythonParser) ExtractClasses(rootNode *sitter.Node, content []byte) []PythonClass {
	var classes []PythonClass

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		switch node.Type() {
		case "class_definition":
			cls := p.parseClassDef(node, content)
			if cls != nil {
				classes = append(classes, *cls)
			}
			return false
		case "decorated_definition":
			// Check if it's a decorated class
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child.Type() == "class_definition" {
					cls := p.parseDecoratedClass(node, content)
					if cls != nil {
						classes = append(classes, *cls)
					}
					return false
				}
			}
		}
		return true
	})

	return classes
}

// parseClassDef parses a class definition node.
func (p *PythonParser) parseClassDef(node *sitter.Node, content []byte) *PythonClass {
	cls := &PythonClass{
		Line:       int(node.StartPoint().Row) + 1,
		Bases:      []string{},
		Decorators: []PythonDecorator{},
		Methods:    []PythonDecoratedFunction{},
		Node:       node,
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			cls.Name = child.Content(content)
		case "argument_list":
			// Base classes
			p.parseClassBases(child, content, cls)
		case "block":
			// Class body - extract methods
			cls.Methods = p.extractClassMethods(child, content)
		}
	}

	if cls.Name == "" {
		return nil
	}

	return cls
}

// parseDecoratedClass parses a decorated class definition.
func (p *PythonParser) parseDecoratedClass(node *sitter.Node, content []byte) *PythonClass {
	var cls *PythonClass
	decorators := []PythonDecorator{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "decorator":
			dec := p.parseDecorator(child, content)
			if dec != nil {
				decorators = append(decorators, *dec)
			}
		case "class_definition":
			cls = p.parseClassDef(child, content)
		}
	}

	if cls != nil {
		cls.Decorators = decorators
	}

	return cls
}

// parseClassBases parses class base classes.
func (p *PythonParser) parseClassBases(node *sitter.Node, content []byte, cls *PythonClass) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			cls.Bases = append(cls.Bases, child.Content(content))
		case "attribute":
			cls.Bases = append(cls.Bases, child.Content(content))
		}
	}
}

// extractClassMethods extracts methods from a class body.
func (p *PythonParser) extractClassMethods(node *sitter.Node, content []byte) []PythonDecoratedFunction {
	var methods []PythonDecoratedFunction

	p.walkNodes(node, func(n *sitter.Node) bool {
		switch n.Type() {
		case "function_definition":
			fn := &PythonDecoratedFunction{
				Line:       int(n.StartPoint().Row) + 1,
				Decorators: []PythonDecorator{},
				Parameters: []PythonParameter{},
				Node:       n,
			}
			p.parseFunctionDef(n, content, fn)
			if fn.Name != "" {
				methods = append(methods, *fn)
			}
			return false
		case "decorated_definition":
			fn := p.parseDecoratedFunction(n, content)
			if fn != nil {
				methods = append(methods, *fn)
			}
			return false
		}
		return true
	})

	return methods
}

// ExtractPydanticModels extracts Pydantic model definitions.
func (p *PythonParser) ExtractPydanticModels(rootNode *sitter.Node, content []byte) []PydanticModel {
	var models []PydanticModel

	classes := p.ExtractClasses(rootNode, content)

	// Build a map of class name to class definition for inheritance resolution
	classMap := make(map[string]PythonClass)
	for _, cls := range classes {
		classMap[cls.Name] = cls
	}

	// Build a set of known Pydantic model names
	// Start with direct BaseModel subclasses, then expand to include transitive inheritance
	pydanticClasses := make(map[string]bool)

	// First pass: find direct BaseModel subclasses
	for _, cls := range classes {
		if p.isDirectPydanticModel(cls) {
			pydanticClasses[cls.Name] = true
		}
	}

	// Second pass: find classes that inherit from known Pydantic models
	// Keep iterating until no new Pydantic models are found
	changed := true
	for changed {
		changed = false
		for _, cls := range classes {
			if pydanticClasses[cls.Name] {
				continue // Already known
			}
			for _, base := range cls.Bases {
				if pydanticClasses[base] {
					pydanticClasses[cls.Name] = true
					changed = true
					break
				}
			}
		}
	}

	// Third pass: extract the Pydantic models (own fields only)
	modelMap := make(map[string]*PydanticModel)
	for _, cls := range classes {
		if pydanticClasses[cls.Name] {
			model := p.parsePydanticModel(cls, rootNode, content)
			if model != nil {
				modelMap[model.Name] = model
			}
		}
	}

	// Fourth pass: resolve inherited fields in topological order
	// We need to process parent classes before child classes
	resolved := make(map[string]bool)
	for len(resolved) < len(modelMap) {
		for name, model := range modelMap {
			if resolved[name] {
				continue
			}
			cls := classMap[name]

			// Check if all parent Pydantic models are resolved
			allParentsResolved := true
			for _, base := range cls.Bases {
				if _, isPydantic := modelMap[base]; isPydantic {
					if !resolved[base] {
						allParentsResolved = false
						break
					}
				}
			}

			if allParentsResolved {
				inheritedFields := p.resolveInheritedFields(cls, modelMap)
				// Prepend inherited fields (they come before own fields)
				model.Fields = append(inheritedFields, model.Fields...)
				resolved[name] = true
			}
		}
	}

	// Collect models into slice
	for _, model := range modelMap {
		models = append(models, *model)
	}

	return models
}

// resolveInheritedFields gets all inherited fields from parent Pydantic models.
func (p *PythonParser) resolveInheritedFields(cls PythonClass, modelMap map[string]*PydanticModel) []PydanticField {
	var inheritedFields []PydanticField

	for _, base := range cls.Bases {
		if parentModel, ok := modelMap[base]; ok {
			// Include parent's fields (which already include its inherited fields after resolution)
			inheritedFields = append(inheritedFields, parentModel.Fields...)
		}
	}

	return inheritedFields
}

// isDirectPydanticModel checks if a class directly inherits from BaseModel or related classes.
func (p *PythonParser) isDirectPydanticModel(cls PythonClass) bool {
	for _, base := range cls.Bases {
		if base == "BaseModel" || strings.Contains(base, "BaseModel") ||
			base == "BaseSettings" || strings.Contains(base, "pydantic") {
			return true
		}
	}
	return false
}

// parsePydanticModel parses a Pydantic model from a class definition.
func (p *PythonParser) parsePydanticModel(cls PythonClass, rootNode *sitter.Node, content []byte) *PydanticModel {
	model := &PydanticModel{
		Name:   cls.Name,
		Fields: []PydanticField{},
		Line:   cls.Line,
		Node:   cls.Node,
	}

	// Find the class body and extract field definitions
	if cls.Node == nil {
		return model
	}

	p.walkNodes(cls.Node, func(node *sitter.Node) bool {
		if node.Type() == "block" {
			model.Fields = p.extractPydanticFields(node, content)
			return false
		}
		return true
	})

	return model
}

// extractPydanticFields extracts fields from a Pydantic model class body.
func (p *PythonParser) extractPydanticFields(node *sitter.Node, content []byte) []PydanticField {
	var fields []PydanticField

	p.walkNodes(node, func(n *sitter.Node) bool {
		// Look for type annotations (field: type) or assignments (field: type = default)
		if n.Type() == "expression_statement" {
			field := p.parseExpressionAsField(n, content)
			if field != nil {
				fields = append(fields, *field)
			}
			return false
		}
		return true
	})

	return fields
}

// parseExpressionAsField parses an expression statement as a Pydantic field.
func (p *PythonParser) parseExpressionAsField(node *sitter.Node, content []byte) *PydanticField {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "assignment":
			return p.parseAssignmentAsField(child, content)
		case "type":
			// Type annotation without assignment: field: type
			return p.parseTypeAnnotationAsField(node, content)
		}
	}
	return nil
}

// parseAssignmentAsField parses an assignment as a Pydantic field.
func (p *PythonParser) parseAssignmentAsField(node *sitter.Node, content []byte) *PydanticField {
	field := &PydanticField{}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "identifier":
			if field.Name == "" {
				field.Name = child.Content(content)
			}
		case "type":
			field.Type = child.Content(content)
			// Check if optional
			typeStr := child.Content(content)
			if strings.Contains(typeStr, "Optional") || strings.Contains(typeStr, "None") {
				field.IsOptional = true
			}
		default:
			if field.Name != "" && field.Default == "" && child.Type() != "=" && child.Type() != ":" {
				field.Default = child.Content(content)
				field.IsOptional = true
			}
		}
	}

	// Skip methods and special attributes
	if field.Name == "" || strings.HasPrefix(field.Name, "_") {
		return nil
	}

	return field
}

// parseTypeAnnotationAsField parses a type annotation as a Pydantic field.
func (p *PythonParser) parseTypeAnnotationAsField(node *sitter.Node, content []byte) *PydanticField {
	field := &PydanticField{IsOptional: false}

	p.walkNodes(node, func(n *sitter.Node) bool {
		switch n.Type() {
		case "identifier":
			if field.Name == "" {
				field.Name = n.Content(content)
			}
		case "type":
			field.Type = n.Content(content)
			// Check if optional
			typeStr := n.Content(content)
			if strings.Contains(typeStr, "Optional") || strings.Contains(typeStr, "None") {
				field.IsOptional = true
			}
		}
		return true
	})

	// Skip methods and special attributes
	if field.Name == "" || strings.HasPrefix(field.Name, "_") {
		return nil
	}

	return field
}

// FindCallExpressions finds all call expression nodes in the AST.
func (p *PythonParser) FindCallExpressions(rootNode *sitter.Node, content []byte) []*sitter.Node {
	var calls []*sitter.Node

	p.walkNodes(rootNode, func(node *sitter.Node) bool {
		if node.Type() == "call" {
			calls = append(calls, node)
		}
		return true
	})

	return calls
}

// GetCalleeText returns the callee text from a call expression.
func (p *PythonParser) GetCalleeText(node *sitter.Node, content []byte) string {
	if node.Type() != "call" {
		return ""
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "attribute" {
			return child.Content(content)
		}
	}

	return ""
}

// GetCallArguments returns the arguments from a call expression.
func (p *PythonParser) GetCallArguments(node *sitter.Node, content []byte) []*sitter.Node {
	var args []*sitter.Node

	if node.Type() != "call" {
		return args
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "argument_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				arg := child.Child(j)
				if arg.Type() != "," && arg.Type() != "(" && arg.Type() != ")" {
					args = append(args, arg)
				}
			}
			break
		}
	}

	return args
}

// ExtractStringLiteral extracts a string value from a string node.
func (p *PythonParser) ExtractStringLiteral(node *sitter.Node, content []byte) (string, bool) {
	if node == nil {
		return "", false
	}

	nodeType := node.Type()
	if nodeType != "string" && nodeType != "concatenated_string" {
		return "", false
	}

	text := node.Content(content)
	return trimQuotes(text), true
}

// walkNodes walks all nodes in the tree, calling fn for each node.
// If fn returns false, it stops recursing into that node's children.
func (p *PythonParser) walkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
	if node == nil {
		return
	}

	if !fn(node) {
		return
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		p.walkNodes(node.Child(i), fn)
	}
}

// WalkNodes is a public method for walking nodes.
func (p *PythonParser) WalkNodes(node *sitter.Node, fn func(*sitter.Node) bool) {
	p.walkNodes(node, fn)
}

// IsSupported returns whether Python parsing is fully implemented.
func (p *PythonParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *PythonParser) SupportedExtensions() []string {
	return []string{".py", ".pyw"}
}

// Close cleans up parser resources.
func (p *PythonParser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// Close cleans up the parsed file resources.
func (pf *ParsedPythonFile) Close() {
	if pf.Tree != nil {
		pf.Tree.Close()
	}
}

// HasImport checks if the file has a specific import.
func (pf *ParsedPythonFile) HasImport(moduleName string) bool {
	for _, imp := range pf.Imports {
		if imp.Module == moduleName {
			return true
		}
		// Check for "from module import ..." style
		if strings.HasPrefix(imp.Module, moduleName) {
			return true
		}
		// Check for submodule imports
		if strings.Contains(imp.Module, moduleName) {
			return true
		}
	}
	return false
}

// HasImportedName checks if a specific name is imported from any module.
func (pf *ParsedPythonFile) HasImportedName(name string) bool {
	for _, imp := range pf.Imports {
		for _, n := range imp.Names {
			if n == name {
				return true
			}
		}
	}
	return false
}

// PythonTypeToOpenAPI converts a Python type to an OpenAPI type.
func PythonTypeToOpenAPI(pyType string) (openAPIType string, format string) {
	// Trim whitespace and handle Optional types
	pyType = strings.TrimSpace(pyType)
	pyType = strings.TrimPrefix(pyType, "Optional[")
	pyType = strings.TrimSuffix(pyType, "]")

	switch pyType {
	case "str", "string":
		return "string", ""
	case "int", "integer":
		return "integer", ""
	case "float":
		return "number", ""
	case "bool", "boolean":
		return "boolean", ""
	case "datetime", "datetime.datetime":
		return "string", "date-time"
	case "date", "datetime.date":
		return "string", "date"
	case "time", "datetime.time":
		return "string", "time"
	case "UUID", "uuid.UUID":
		return "string", "uuid"
	case "bytes":
		return "string", "binary"
	case "Any", "any":
		return "object", ""
	case "None", "NoneType":
		return "null", ""
	case "dict", "Dict":
		return "object", ""
	default:
		// Check for list types
		if strings.HasPrefix(pyType, "list[") || strings.HasPrefix(pyType, "List[") {
			return "array", ""
		}
		if strings.HasPrefix(pyType, "dict[") || strings.HasPrefix(pyType, "Dict[") {
			return "object", ""
		}
		// Assume it's a reference to another type/model
		return "object", ""
	}
}

// trimQuotes removes quotes from a string literal.
func trimQuotes(s string) string {
	// Handle triple quotes
	if strings.HasPrefix(s, `"""`) && strings.HasSuffix(s, `"""`) {
		return s[3 : len(s)-3]
	}
	if strings.HasPrefix(s, `'''`) && strings.HasSuffix(s, `'''`) {
		return s[3 : len(s)-3]
	}
	// Handle f-strings and other prefixes
	for _, prefix := range []string{"f", "r", "b", "u", "fr", "rf", "br", "rb"} {
		if strings.HasPrefix(s, prefix+`"`) || strings.HasPrefix(s, prefix+`'`) {
			s = s[len(prefix):]
			break
		}
	}
	// Handle single and double quotes
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
