// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// ElixirParser provides Elixir parsing capabilities using regex patterns.
type ElixirParser struct{}

// NewElixirParser creates a new Elixir parser.
func NewElixirParser() *ElixirParser {
	return &ElixirParser{}
}

// ElixirModule represents an Elixir module definition.
type ElixirModule struct {
	// Name is the module name
	Name string

	// Functions are the module functions
	Functions []ElixirFunction

	// Uses are the use statements
	Uses []string

	// Aliases are the alias statements
	Aliases []string

	// Line is the source line number
	Line int
}

// ElixirFunction represents an Elixir function definition.
type ElixirFunction struct {
	// Name is the function name
	Name string

	// Arity is the number of parameters
	Arity int

	// Parameters are the function parameters
	Parameters []ElixirParameter

	// IsPublic indicates if the function is public (def vs defp)
	IsPublic bool

	// Line is the source line number
	Line int
}

// ElixirParameter represents a function parameter.
type ElixirParameter struct {
	// Name is the parameter name
	Name string

	// Default is the default value
	Default string

	// IsOptional indicates if the parameter is optional
	IsOptional bool
}

// ElixirRoute represents a Phoenix route definition.
type ElixirRoute struct {
	// Method is the HTTP method (get, post, put, delete, etc.)
	Method string

	// Path is the route path
	Path string

	// Controller is the controller module
	Controller string

	// Action is the controller action
	Action string

	// Line is the source line number
	Line int
}

// ElixirScope represents a Phoenix route scope.
type ElixirScope struct {
	// Path is the scope path
	Path string

	// Module is the scope module
	Module string

	// Pipes are the pipe_through declarations
	Pipes []string

	// Routes are the routes within this scope
	Routes []ElixirRoute

	// Line is the source line number
	Line int
}

// ElixirResource represents a Phoenix resources declaration.
type ElixirResource struct {
	// Path is the resource path
	Path string

	// Controller is the controller module
	Controller string

	// Only specifies which actions to include
	Only []string

	// Except specifies which actions to exclude
	Except []string

	// Line is the source line number
	Line int
}

// ParsedElixirFile represents a parsed Elixir source file.
type ParsedElixirFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Modules are the extracted module definitions
	Modules []ElixirModule

	// Routes are the extracted route definitions
	Routes []ElixirRoute

	// Scopes are the extracted scope definitions
	Scopes []ElixirScope

	// Resources are the extracted resource definitions
	Resources []ElixirResource
}

// Regex patterns for Elixir parsing
var (
	// Matches module definitions
	elixirModuleRegex = regexp.MustCompile(`(?m)defmodule\s+([\w.]+)`)

	// Matches use statements
	elixirUseRegex = regexp.MustCompile(`(?m)^\s*use\s+([\w.]+)`)

	// Matches alias statements
	elixirAliasRegex = regexp.MustCompile(`(?m)^\s*alias\s+([\w.]+)`)

	// Matches function definitions
	elixirFunctionRegex = regexp.MustCompile(`(?m)^\s*(def|defp)\s+(\w+)\s*(?:\(([^)]*)\))?`)

	// Matches Phoenix route definitions
	// get "/path", Controller, :action
	elixirRouteRegex = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|options|head)\s+"([^"]+)"\s*,\s*([\w.]+)\s*,\s*:(\w+)`)

	// Matches Phoenix scope definitions
	// scope "/path", Module do
	elixirScopeRegex = regexp.MustCompile(`(?m)scope\s+"([^"]+)"(?:\s*,\s*([\w.]+))?`)

	// Matches Phoenix resources definitions
	// resources "/users", UserController
	elixirResourcesRegex = regexp.MustCompile(`(?m)resources\s+"([^"]+)"\s*,\s*([\w.]+)`)

	// Matches pipe_through declarations
	elixirPipeThroughRegex = regexp.MustCompile(`(?m)pipe_through\s+(?::(\w+)|\[([^\]]+)\])`)
)

// Parse parses Elixir source code.
func (p *ElixirParser) Parse(filename string, content []byte) *ParsedElixirFile {
	src := string(content)
	pf := &ParsedElixirFile{
		Path:      filename,
		Content:   src,
		Modules:   []ElixirModule{},
		Routes:    []ElixirRoute{},
		Scopes:    []ElixirScope{},
		Resources: []ElixirResource{},
	}

	// Extract modules
	pf.Modules = p.extractModules(src)

	// Extract routes (for router files)
	pf.Routes = p.extractRoutes(src)
	pf.Scopes = p.extractScopes(src)
	pf.Resources = p.extractResources(src)

	return pf
}

// extractModules extracts module definitions from Elixir source.
func (p *ElixirParser) extractModules(src string) []ElixirModule {
	var modules []ElixirModule

	matches := elixirModuleRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		module := ElixirModule{
			Line:      line,
			Uses:      []string{},
			Aliases:   []string{},
			Functions: []ElixirFunction{},
		}

		// Extract module name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			module.Name = src[match[2]:match[3]]
		}

		// Find the module body
		moduleStart := match[0]
		moduleBody := p.findModuleBody(src[moduleStart:])
		if moduleBody != "" {
			module.Uses = p.extractUses(moduleBody)
			module.Aliases = p.extractAliases(moduleBody)
			module.Functions = p.extractFunctions(moduleBody, line)
		}

		if module.Name != "" {
			modules = append(modules, module)
		}
	}

	return modules
}

// findModuleBody finds the body of a module (between do and end).
func (p *ElixirParser) findModuleBody(src string) string {
	doIdx := strings.Index(src, " do")
	if doIdx == -1 {
		doIdx = strings.Index(src, "\n  do")
	}
	if doIdx == -1 {
		return ""
	}

	depth := 1
	start := doIdx + 3
	for i := start; i < len(src)-3; i++ {
		// Check for nested do...end blocks
		if i+3 < len(src) && src[i:i+3] == " do" || src[i:i+3] == "\ndo" {
			depth++
		}
		if i+4 < len(src) && (src[i:i+4] == "\nend" || src[i:i+4] == " end") {
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
	}
	return ""
}

// extractUses extracts use statements from Elixir source.
func (p *ElixirParser) extractUses(src string) []string {
	var uses []string
	matches := elixirUseRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			uses = append(uses, strings.TrimSpace(match[1]))
		}
	}
	return uses
}

// extractAliases extracts alias statements from Elixir source.
func (p *ElixirParser) extractAliases(src string) []string {
	var aliases []string
	matches := elixirAliasRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			aliases = append(aliases, strings.TrimSpace(match[1]))
		}
	}
	return aliases
}

// extractFunctions extracts function definitions from Elixir source.
func (p *ElixirParser) extractFunctions(src string, baseLineOffset int) []ElixirFunction {
	var functions []ElixirFunction

	matches := elixirFunctionRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		line := baseLineOffset + countLines(src[:match[0]])

		fn := ElixirFunction{
			Line:       line,
			Parameters: []ElixirParameter{},
		}

		// Check if public or private (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			fn.IsPublic = src[match[2]:match[3]] == "def"
		}

		// Extract function name (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			fn.Name = src[match[4]:match[5]]
		}

		// Extract parameters (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			paramStr := src[match[6]:match[7]]
			fn.Parameters = p.extractParameters(paramStr)
			fn.Arity = len(fn.Parameters)
		}

		if fn.Name != "" {
			functions = append(functions, fn)
		}
	}

	return functions
}

// extractParameters extracts parameters from a parameter string.
func (p *ElixirParser) extractParameters(src string) []ElixirParameter {
	var params []ElixirParameter

	if strings.TrimSpace(src) == "" {
		return params
	}

	// Split by comma
	paramStrs := strings.Split(src, ",")

	for _, paramStr := range paramStrs {
		paramStr = strings.TrimSpace(paramStr)
		if paramStr == "" {
			continue
		}

		param := ElixirParameter{}

		// Check for default value (\\)
		if idx := strings.Index(paramStr, "\\\\"); idx > 0 {
			param.Default = strings.TrimSpace(paramStr[idx+2:])
			param.IsOptional = true
			paramStr = strings.TrimSpace(paramStr[:idx])
		}

		// Extract parameter name (remove pattern matching)
		paramStr = strings.TrimPrefix(paramStr, "%")
		if idx := strings.Index(paramStr, "="); idx > 0 {
			paramStr = strings.TrimSpace(paramStr[:idx])
		}
		param.Name = paramStr

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// extractRoutes extracts Phoenix route definitions.
func (p *ElixirParser) extractRoutes(src string) []ElixirRoute {
	var routes []ElixirRoute

	matches := elixirRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := ElixirRoute{
			Line: line,
		}

		// Extract HTTP method (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			route.Method = strings.ToUpper(src[match[2]:match[3]])
		}

		// Extract path (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			route.Path = src[match[4]:match[5]]
		}

		// Extract controller (group 3)
		if match[6] >= 0 && match[7] >= 0 {
			route.Controller = src[match[6]:match[7]]
		}

		// Extract action (group 4)
		if match[8] >= 0 && match[9] >= 0 {
			route.Action = src[match[8]:match[9]]
		}

		if route.Method != "" && route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractScopes extracts Phoenix scope definitions.
func (p *ElixirParser) extractScopes(src string) []ElixirScope {
	var scopes []ElixirScope

	matches := elixirScopeRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		scope := ElixirScope{
			Line:   line,
			Pipes:  []string{},
			Routes: []ElixirRoute{},
		}

		// Extract scope path (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			scope.Path = src[match[2]:match[3]]
		}

		// Extract scope module (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			scope.Module = src[match[4]:match[5]]
		}

		// Find the scope body and extract routes
		scopeStart := match[0]
		scopeBody := p.findDoBlock(src[scopeStart:])
		if scopeBody != "" {
			scope.Routes = p.extractRoutes(scopeBody)
			scope.Pipes = p.extractPipeThroughs(scopeBody)
		}

		if scope.Path != "" {
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

// findDoBlock finds a do...end block.
func (p *ElixirParser) findDoBlock(src string) string {
	doIdx := strings.Index(src, " do")
	if doIdx == -1 {
		return ""
	}

	depth := 1
	start := doIdx + 4
	for i := start; i < len(src)-3; i++ {
		if i+3 < len(src) && (src[i:i+3] == " do" || strings.HasPrefix(src[i:], "\n  do")) {
			depth++
		}
		if strings.HasPrefix(src[i:], "\n  end") || strings.HasPrefix(src[i:], " end") {
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
	}
	return ""
}

// extractPipeThroughs extracts pipe_through declarations.
func (p *ElixirParser) extractPipeThroughs(src string) []string {
	var pipes []string

	matches := elixirPipeThroughRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			// Single atom
			pipes = append(pipes, match[1])
		} else if len(match) > 2 && match[2] != "" {
			// List of atoms
			list := strings.Split(match[2], ",")
			for _, item := range list {
				item = strings.TrimSpace(item)
				item = strings.TrimPrefix(item, ":")
				if item != "" {
					pipes = append(pipes, item)
				}
			}
		}
	}

	return pipes
}

// extractResources extracts Phoenix resources definitions.
func (p *ElixirParser) extractResources(src string) []ElixirResource {
	var resources []ElixirResource

	matches := elixirResourcesRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		resource := ElixirResource{
			Line:   line,
			Only:   []string{},
			Except: []string{},
		}

		// Extract path (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			resource.Path = src[match[2]:match[3]]
		}

		// Extract controller (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			resource.Controller = src[match[4]:match[5]]
		}

		if resource.Path != "" {
			resources = append(resources, resource)
		}
	}

	return resources
}

// IsSupported returns whether Elixir parsing is supported.
func (p *ElixirParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *ElixirParser) SupportedExtensions() []string {
	return []string{".ex", ".exs"}
}

// ExpandElixirResources expands a resources definition into individual routes.
func ExpandElixirResources(resource ElixirResource) []ElixirRoute {
	var routes []ElixirRoute

	// Standard Phoenix resource actions
	actions := []struct {
		method string
		path   string
		action string
	}{
		{"GET", "", "index"},
		{"GET", "/new", "new"},
		{"POST", "", "create"},
		{"GET", "/:id", "show"},
		{"GET", "/:id/edit", "edit"},
		{"PUT", "/:id", "update"},
		{"PATCH", "/:id", "update"},
		{"DELETE", "/:id", "delete"},
	}

	basePath := "/" + strings.TrimPrefix(resource.Path, "/")

	for _, a := range actions {
		// Check only/except filters
		if len(resource.Only) > 0 {
			found := false
			for _, o := range resource.Only {
				if o == a.action {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if len(resource.Except) > 0 {
			excluded := false
			for _, e := range resource.Except {
				if e == a.action {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		routes = append(routes, ElixirRoute{
			Method:     a.method,
			Path:       basePath + a.path,
			Controller: resource.Controller,
			Action:     a.action,
			Line:       resource.Line,
		})
	}

	return routes
}
