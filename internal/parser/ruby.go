// SPDX-FileCopyrightText: 2026 api2spec
// SPDX-License-Identifier: FSL-1.1-MIT

// Package parser provides language-specific parsing capabilities.
package parser

import (
	"regexp"
	"strings"
)

// RubyParser provides Ruby parsing capabilities using regex patterns.
type RubyParser struct{}

// NewRubyParser creates a new Ruby parser.
func NewRubyParser() *RubyParser {
	return &RubyParser{}
}

// RubyClass represents a Ruby class definition.
type RubyClass struct {
	// Name is the class name
	Name string

	// SuperClass is the parent class
	SuperClass string

	// Modules are the included modules
	Modules []string

	// Methods are the class methods
	Methods []RubyMethod

	// Line is the source line number
	Line int
}

// RubyMethod represents a Ruby method definition.
type RubyMethod struct {
	// Name is the method name
	Name string

	// Parameters are the method parameters
	Parameters []RubyParameter

	// Line is the source line number
	Line int
}

// RubyParameter represents a method parameter.
type RubyParameter struct {
	// Name is the parameter name
	Name string

	// Default is the default value
	Default string

	// IsOptional indicates if the parameter is optional
	IsOptional bool

	// IsKeyword indicates if it's a keyword argument
	IsKeyword bool
}

// RubyRoute represents a Rails/Sinatra route definition.
type RubyRoute struct {
	// Method is the HTTP method (get, post, put, delete, etc.)
	Method string

	// Path is the route path
	Path string

	// Controller is the controller name
	Controller string

	// Action is the controller action
	Action string

	// Line is the source line number
	Line int
}

// RubyResource represents a Rails resources declaration.
type RubyResource struct {
	// Name is the resource name
	Name string

	// Controller is the controller (if specified)
	Controller string

	// Only specifies which actions to include
	Only []string

	// Except specifies which actions to exclude
	Except []string

	// Line is the source line number
	Line int
}

// RubyNamespace represents a Rails namespace declaration.
type RubyNamespace struct {
	// Name is the namespace name
	Name string

	// Routes are the routes within this namespace
	Routes []RubyRoute

	// Resources are the resources within this namespace
	Resources []RubyResource

	// Line is the source line number
	Line int
}

// ParsedRubyFile represents a parsed Ruby source file.
type ParsedRubyFile struct {
	// Path is the file path
	Path string

	// Content is the original source content
	Content string

	// Classes are the extracted class definitions
	Classes []RubyClass

	// Routes are the extracted route definitions
	Routes []RubyRoute

	// Resources are the extracted resource definitions
	Resources []RubyResource

	// Namespaces are the extracted namespace definitions
	Namespaces []RubyNamespace
}

// Regex patterns for Ruby parsing
var (
	// Matches class definitions
	rubyClassRegex = regexp.MustCompile(`(?m)^class\s+(\w+)(?:\s*<\s*(\w+))?`)

	// Matches module includes
	rubyIncludeRegex = regexp.MustCompile(`(?m)^\s*include\s+(\w+)`)

	// Matches method definitions
	rubyMethodRegex = regexp.MustCompile(`(?m)^\s*def\s+(\w+[?!]?)(?:\s*\(([^)]*)\))?`)

	// Matches Rails route definitions
	// get '/path', to: 'controller#action'
	rubyRouteToRegex = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|options|head)\s+['"]([^'"]+)['"](?:\s*,\s*to:\s*['"](\w+)#(\w+)['"])?`)

	// Matches Rails route definitions with controller array
	// get '/path', [Controller, :action]
	rubyRouteArrayRegex = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete)\s+['"]([^'"]+)['"]\s*,\s*\[?\s*(\w+)(?:::class)?\s*,\s*[:'"](\w+)`)

	// Matches resources declaration
	// resources :users
	rubyResourcesRegex = regexp.MustCompile(`(?m)^\s*resources?\s+:(\w+)`)

	// Matches api_resources declaration
	rubyAPIResourcesRegex = regexp.MustCompile(`(?m)^\s*api_resources?\s+:(\w+)`)

	// Matches namespace declaration
	// namespace :api do
	rubyNamespaceRegex = regexp.MustCompile(`(?m)^\s*namespace\s+:(\w+)`)

	// Matches Sinatra-style route definitions
	// get '/path' do
	rubySimpleRouteRegex = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|options|head)\s+['"]([^'"]+)['"]\s+do`)
)

// Parse parses Ruby source code.
func (p *RubyParser) Parse(filename string, content []byte) *ParsedRubyFile {
	src := string(content)
	pf := &ParsedRubyFile{
		Path:       filename,
		Content:    src,
		Classes:    []RubyClass{},
		Routes:     []RubyRoute{},
		Resources:  []RubyResource{},
		Namespaces: []RubyNamespace{},
	}

	// Extract classes
	pf.Classes = p.extractClasses(src)

	// Extract routes
	pf.Routes = p.extractRoutes(src)
	pf.Resources = p.extractResources(src)
	pf.Namespaces = p.extractNamespaces(src)

	return pf
}

// extractClasses extracts class definitions from Ruby source.
func (p *RubyParser) extractClasses(src string) []RubyClass {
	var classes []RubyClass

	matches := rubyClassRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		class := RubyClass{
			Line:    line,
			Modules: []string{},
			Methods: []RubyMethod{},
		}

		// Extract class name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			class.Name = src[match[2]:match[3]]
		}

		// Extract super class (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			class.SuperClass = src[match[4]:match[5]]
		}

		// Find the class body
		classStart := match[0]
		classBody := p.findClassBody(src[classStart:])
		if classBody != "" {
			class.Modules = p.extractIncludes(classBody)
			class.Methods = p.extractMethods(classBody, line)
		}

		if class.Name != "" {
			classes = append(classes, class)
		}
	}

	return classes
}

// findClassBody finds the body of a class (between class and end).
func (p *RubyParser) findClassBody(src string) string {
	// Find the end of class definition line
	lineEnd := strings.Index(src, "\n")
	if lineEnd == -1 {
		return ""
	}

	depth := 1
	start := lineEnd + 1
	i := start

	for i < len(src) {
		// Look for keywords that increase depth
		if p.isKeywordStart(src, i, "class") || p.isKeywordStart(src, i, "module") ||
			p.isKeywordStart(src, i, "def") || p.isKeywordStart(src, i, "do") ||
			p.isKeywordStart(src, i, "if") || p.isKeywordStart(src, i, "unless") ||
			p.isKeywordStart(src, i, "case") || p.isKeywordStart(src, i, "begin") {
			depth++
		}

		// Look for end keyword
		if p.isKeywordStart(src, i, "end") {
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
		i++
	}
	return ""
}

// isKeywordStart checks if a keyword starts at the given position.
func (p *RubyParser) isKeywordStart(src string, pos int, keyword string) bool {
	if pos+len(keyword) > len(src) {
		return false
	}
	// Check for preceding whitespace or line start
	if pos > 0 {
		prev := src[pos-1]
		if prev != ' ' && prev != '\t' && prev != '\n' {
			return false
		}
	}
	// Check for the keyword
	if src[pos:pos+len(keyword)] != keyword {
		return false
	}
	// Check for following whitespace or line end
	endPos := pos + len(keyword)
	if endPos < len(src) {
		next := src[endPos]
		if next != ' ' && next != '\t' && next != '\n' && next != '(' {
			return false
		}
	}
	return true
}

// extractIncludes extracts include statements from Ruby source.
func (p *RubyParser) extractIncludes(src string) []string {
	var includes []string
	matches := rubyIncludeRegex.FindAllStringSubmatch(src, -1)
	for _, match := range matches {
		if len(match) > 1 {
			includes = append(includes, strings.TrimSpace(match[1]))
		}
	}
	return includes
}

// extractMethods extracts method definitions from Ruby source.
func (p *RubyParser) extractMethods(src string, baseLineOffset int) []RubyMethod {
	var methods []RubyMethod

	matches := rubyMethodRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := baseLineOffset + countLines(src[:match[0]])

		method := RubyMethod{
			Line:       line,
			Parameters: []RubyParameter{},
		}

		// Extract method name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			method.Name = src[match[2]:match[3]]
		}

		// Extract parameters (group 2)
		if match[4] >= 0 && match[5] >= 0 {
			paramStr := src[match[4]:match[5]]
			method.Parameters = p.extractParameters(paramStr)
		}

		if method.Name != "" {
			methods = append(methods, method)
		}
	}

	return methods
}

// extractParameters extracts parameters from a parameter string.
func (p *RubyParser) extractParameters(src string) []RubyParameter {
	var params []RubyParameter

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

		param := RubyParameter{}

		// Check for default value
		if idx := strings.Index(paramStr, "="); idx > 0 {
			param.Default = strings.TrimSpace(paramStr[idx+1:])
			param.IsOptional = true
			paramStr = strings.TrimSpace(paramStr[:idx])
		}

		// Check for keyword argument
		if strings.HasSuffix(paramStr, ":") {
			param.IsKeyword = true
			paramStr = strings.TrimSuffix(paramStr, ":")
		}

		param.Name = paramStr

		if param.Name != "" {
			params = append(params, param)
		}
	}

	return params
}

// extractRoutes extracts Rails/Sinatra route definitions.
func (p *RubyParser) extractRoutes(src string) []RubyRoute {
	var routes []RubyRoute

	// Extract 'to:' style routes
	routes = append(routes, p.extractToRoutes(src)...)

	// Extract array style routes
	routes = append(routes, p.extractArrayRoutes(src)...)

	// Extract Sinatra style routes
	routes = append(routes, p.extractSinatraRoutes(src)...)

	return routes
}

// extractToRoutes extracts 'to:' style route definitions.
func (p *RubyParser) extractToRoutes(src string) []RubyRoute {
	var routes []RubyRoute

	matches := rubyRouteToRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := RubyRoute{
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

// extractArrayRoutes extracts array style route definitions.
func (p *RubyParser) extractArrayRoutes(src string) []RubyRoute {
	var routes []RubyRoute

	matches := rubyRouteArrayRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 10 {
			continue
		}

		line := countLines(src[:match[0]])

		route := RubyRoute{
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

// extractSinatraRoutes extracts Sinatra style route definitions.
func (p *RubyParser) extractSinatraRoutes(src string) []RubyRoute {
	var routes []RubyRoute

	matches := rubySimpleRouteRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}

		line := countLines(src[:match[0]])

		route := RubyRoute{
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

		if route.Method != "" && route.Path != "" {
			routes = append(routes, route)
		}
	}

	return routes
}

// extractResources extracts Rails resources definitions.
func (p *RubyParser) extractResources(src string) []RubyResource {
	var resources []RubyResource

	// Standard resources
	matches := rubyResourcesRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		resource := RubyResource{
			Line:   line,
			Only:   []string{},
			Except: []string{},
		}

		// Extract resource name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			resource.Name = src[match[2]:match[3]]
		}

		if resource.Name != "" {
			resources = append(resources, resource)
		}
	}

	// API resources
	apiMatches := rubyAPIResourcesRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range apiMatches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		resource := RubyResource{
			Line:   line,
			Only:   []string{},
			Except: []string{},
		}

		// Extract resource name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			resource.Name = src[match[2]:match[3]]
		}

		if resource.Name != "" {
			resources = append(resources, resource)
		}
	}

	return resources
}

// extractNamespaces extracts Rails namespace definitions.
func (p *RubyParser) extractNamespaces(src string) []RubyNamespace {
	var namespaces []RubyNamespace

	matches := rubyNamespaceRegex.FindAllStringSubmatchIndex(src, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		line := countLines(src[:match[0]])

		ns := RubyNamespace{
			Line:      line,
			Routes:    []RubyRoute{},
			Resources: []RubyResource{},
		}

		// Extract namespace name (group 1)
		if match[2] >= 0 && match[3] >= 0 {
			ns.Name = src[match[2]:match[3]]
		}

		// Find the namespace body
		nsStart := match[0]
		nsBody := p.findDoBlock(src[nsStart:])
		if nsBody != "" {
			ns.Routes = p.extractRoutes(nsBody)
			ns.Resources = p.extractResources(nsBody)
		}

		if ns.Name != "" {
			namespaces = append(namespaces, ns)
		}
	}

	return namespaces
}

// findDoBlock finds a do...end block.
func (p *RubyParser) findDoBlock(src string) string {
	doIdx := strings.Index(src, " do")
	if doIdx == -1 {
		return ""
	}

	depth := 1
	start := doIdx + 4
	i := start

	for i < len(src) {
		if p.isKeywordStart(src, i, "do") {
			depth++
		}
		if p.isKeywordStart(src, i, "end") {
			depth--
			if depth == 0 {
				return src[start:i]
			}
		}
		i++
	}
	return ""
}

// IsSupported returns whether Ruby parsing is supported.
func (p *RubyParser) IsSupported() bool {
	return true
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *RubyParser) SupportedExtensions() []string {
	return []string{".rb"}
}

// ExpandRubyResources expands a resources definition into individual routes.
func ExpandRubyResources(resource RubyResource) []RubyRoute {
	var routes []RubyRoute

	// Standard Rails resource actions
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
		{"DELETE", "/:id", "destroy"},
	}

	basePath := "/" + resource.Name

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

		controller := resource.Controller
		if controller == "" {
			controller = resource.Name
		}

		routes = append(routes, RubyRoute{
			Method:     a.method,
			Path:       basePath + a.path,
			Controller: controller,
			Action:     a.action,
			Line:       resource.Line,
		})
	}

	return routes
}
