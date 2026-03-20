package exons

import (
	"context"
	"math"
	"reflect"
	"strings"
	"sync"
)

// TemplateExecutor is the interface for executing nested templates.
// This allows resolvers to execute registered templates without full engine coupling.
type TemplateExecutor interface {
	// ExecuteTemplate executes a registered template by name with the given data.
	ExecuteTemplate(ctx context.Context, name string, data map[string]any) (string, error)
	// HasTemplate checks if a template is registered with the given name.
	HasTemplate(name string) bool
	// MaxDepth returns the configured maximum nesting depth.
	MaxDepth() int
	// GetTemplateSource retrieves the source string of a registered template.
	// Used for template inheritance resolution.
	GetTemplateSource(name string) (string, bool)
}

// SpecBodyResolver provides spec body lookup for reference resolution.
// This is the internal interface used by the ref resolver.
type SpecBodyResolver interface {
	// ResolveSpecBody looks up a spec by slug and version and returns its template body.
	ResolveSpecBody(ctx context.Context, slug string, version string) (string, error)
}

// Context provides access to template variables and execution state.
// It supports dot-notation path resolution (e.g., "user.profile.name")
// and hierarchical scoping through parent-child relationships.
//
// Thread safety: safe for concurrent reads. Mutations (Set, With* methods) use
// internal locking but callers should coordinate concurrent writes externally.
type Context struct {
	data         map[string]any
	parent       *Context
	mu           sync.RWMutex
	errorStrat   ErrorStrategy
	engine       TemplateExecutor // Optional engine reference for nested templates
	depth        int              // Current nesting depth for include operations
	specResolver SpecBodyResolver // Spec resolver for reference resolution
	refDepth     int              // Current reference resolution depth
	refChain     []string         // Chain of referenced spec slugs for circular detection
}

// NewContext creates a new execution context with the given data.
// If data is nil, an empty map is used.
func NewContext(data map[string]any) *Context {
	if data == nil {
		data = make(map[string]any)
	}
	return &Context{
		data:       data,
		errorStrat: ErrorStrategyThrow,
	}
}

// NewContextWithStrategy creates a context with a specific error strategy.
func NewContextWithStrategy(data map[string]any, strategy ErrorStrategy) *Context {
	ctx := NewContext(data)
	ctx.errorStrat = strategy
	return ctx
}

// Get retrieves a value by dot-notation path (e.g., "user.profile.name").
// Returns the value and true if found, or nil and false if not found.
func (c *Context) Get(path string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getPath(path)
}

// getPath resolves a dot-notation path without locking (internal use).
func (c *Context) getPath(path string) (any, bool) {
	if path == "" {
		return nil, false
	}

	parts := strings.Split(path, PathSeparator)
	var current any = c.data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				// Try parent context if not found
				if c.parent != nil {
					return c.parent.Get(path)
				}
				return nil, false
			}
			current = val
		case map[string]string:
			val, ok := v[part]
			if !ok {
				if c.parent != nil {
					return c.parent.Get(path)
				}
				return nil, false
			}
			current = val
		default:
			// Can't traverse further
			if c.parent != nil {
				return c.parent.Get(path)
			}
			return nil, false
		}
	}

	return current, true
}

// GetString retrieves a string value by path.
// Returns empty string if not found or not a string.
func (c *Context) GetString(path string) string {
	val, ok := c.Get(path)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// GetDefault retrieves a value by path with a fallback default.
func (c *Context) GetDefault(path string, defaultVal any) any {
	val, ok := c.Get(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetStringDefault retrieves a string value with a fallback default.
func (c *Context) GetStringDefault(path, defaultVal string) string {
	val, ok := c.Get(path)
	if !ok {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

// Set sets a value at the given path.
// Currently only supports simple keys (not nested paths).
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}

// Has checks if a value exists at the given path.
func (c *Context) Has(path string) bool {
	_, ok := c.Get(path)
	return ok
}

// Child creates a child context with additional data.
// The child inherits from the parent and can override values.
// Engine reference, depth, and reference tracking are propagated to child contexts.
// Returns interface{} to satisfy internal.ChildContextCreator interface.
func (c *Context) Child(data map[string]any) interface{} {
	if data == nil {
		data = make(map[string]any)
	}
	_, _, errStrat, eng, depth, specRes, refD, refC := c.readState()
	return &Context{
		data:         data,
		parent:       c,
		errorStrat:   errStrat,
		engine:       eng,
		depth:        depth,
		specResolver: specRes,
		refDepth:     refD,
		refChain:     refC,
	}
}

// Parent returns the parent context, or nil if this is a root context.
func (c *Context) Parent() *Context {
	return c.parent
}

// Data returns a deep copy of the context's direct data (not including parent).
// The copy is safe to modify without affecting the original context.
func (c *Context) Data() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return deepCopyMap(c.data)
}

// Keys returns a list of all top-level keys in this context (not including parent).
// This is used by the "did you mean?" suggestion system to find similar variable names.
func (c *Context) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// AllKeys returns a list of all top-level keys including parent contexts.
// Keys from this context take precedence over parent keys.
func (c *Context) AllKeys() []string {
	c.mu.RLock()
	keySet := make(map[string]bool, len(c.data))
	for k := range c.data {
		keySet[k] = true
	}
	c.mu.RUnlock()

	// Walk parent chain — each parent locks itself via Keys()
	for parent := c.parent; parent != nil; parent = parent.parent {
		for _, k := range parent.Keys() {
			keySet[k] = true
		}
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	return keys
}

// ErrorStrategy returns the current error handling strategy as an int.
// This allows the Context to satisfy the internal.ErrorStrategyAccessor interface.
func (c *Context) ErrorStrategy() int {
	return int(c.errorStrat)
}

// ErrorStrategyValue returns the current error handling strategy.
// Use this when you need the typed ErrorStrategy value.
func (c *Context) ErrorStrategyValue() ErrorStrategy {
	return c.errorStrat
}

// Engine returns the template executor if available, nil otherwise.
// This allows resolvers to execute nested templates.
// Returns interface{} to avoid import cycle issues with internal package.
func (c *Context) Engine() interface{} {
	return c.engine
}

// Depth returns the current nesting depth for template includes.
func (c *Context) Depth() int {
	return c.depth
}

// readState reads all fields under RLock. The returned data map is a reference,
// NOT a deep copy — callers must deep-copy it outside the lock.
func (c *Context) readState() (data map[string]any, parent *Context, errorStrat ErrorStrategy, engine TemplateExecutor, depth int, specResolver SpecBodyResolver, refDepth int, refChain []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data, c.parent, c.errorStrat, c.engine, c.depth, c.specResolver, c.refDepth, c.refChain
}

// WithEngine returns a new context with the given engine reference.
// This is typically called by the engine when starting template execution.
// The returned context has a deep copy of the data map for thread safety.
func (c *Context) WithEngine(engine TemplateExecutor) *Context {
	data, parent, errStrat, _, depth, specRes, refD, refC := c.readState()
	return &Context{
		data:         deepCopyMap(data),
		parent:       parent,
		errorStrat:   errStrat,
		engine:       engine,
		depth:        depth,
		specResolver: specRes,
		refDepth:     refD,
		refChain:     refC,
	}
}

// WithDepth returns a new context with the given depth.
// This is used when executing nested templates to track inclusion depth.
// The returned context has a deep copy of the data map for thread safety.
func (c *Context) WithDepth(depth int) *Context {
	data, parent, errStrat, eng, _, specRes, refD, refC := c.readState()
	return &Context{
		data:         deepCopyMap(data),
		parent:       parent,
		errorStrat:   errStrat,
		engine:       eng,
		depth:        depth,
		specResolver: specRes,
		refDepth:     refD,
		refChain:     refC,
	}
}

// WithSpecResolver returns a new context with the given spec resolver.
// This enables {~exons.ref~} tag functionality for spec composition.
// The returned context has a deep copy of the data map for thread safety.
func (c *Context) WithSpecResolver(resolver SpecBodyResolver) *Context {
	data, parent, errStrat, eng, depth, _, refD, refC := c.readState()
	return &Context{
		data:         deepCopyMap(data),
		parent:       parent,
		errorStrat:   errStrat,
		engine:       eng,
		depth:        depth,
		specResolver: resolver,
		refDepth:     refD,
		refChain:     refC,
	}
}

// WithRefDepth returns a new context with the given reference depth.
// This is used internally when resolving nested spec references.
func (c *Context) WithRefDepth(depth int) *Context {
	data, parent, errStrat, eng, d, specRes, _, refC := c.readState()
	return &Context{
		data:         deepCopyMap(data),
		parent:       parent,
		errorStrat:   errStrat,
		engine:       eng,
		depth:        d,
		specResolver: specRes,
		refDepth:     depth,
		refChain:     refC,
	}
}

// WithRefChain returns a new context with the given reference chain.
// This is used internally to track the chain of referenced spec slugs
// for circular reference detection.
func (c *Context) WithRefChain(chain []string) *Context {
	data, parent, errStrat, eng, d, specRes, refD, _ := c.readState()

	// Deep copy the chain to prevent modification
	var chainCopy []string
	if chain != nil {
		chainCopy = make([]string, len(chain))
		copy(chainCopy, chain)
	}

	return &Context{
		data:         deepCopyMap(data),
		parent:       parent,
		errorStrat:   errStrat,
		engine:       eng,
		depth:        d,
		specResolver: specRes,
		refDepth:     refD,
		refChain:     chainCopy,
	}
}

// SpecResolver returns the spec body resolver for reference resolution.
// Implements internal.SpecResolverAccessor interface.
// Returns interface{} to avoid import cycles with internal package.
func (c *Context) SpecResolver() interface{} {
	return c.specResolver
}

// RefDepth returns the current reference resolution depth.
// Implements internal.RefDepthAccessor interface.
func (c *Context) RefDepth() int {
	return c.refDepth
}

// RefChain returns the current chain of referenced spec slugs.
// Implements internal.RefChainAccessor interface.
func (c *Context) RefChain() []string {
	return c.refChain
}

// deepCopyValue recursively copies a value to ensure no shared references.
// Handles maps, slices, and basic types. Complex types (structs, pointers)
// are copied by value which may still share internal state.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case map[string]string:
		result := make(map[string]string, len(val))
		for k, v := range val {
			result[k] = v
		}
		return result
	case []any:
		return deepCopySlice(val)
	case []string:
		result := make([]string, len(val))
		copy(result, val)
		return result
	case []int:
		result := make([]int, len(val))
		copy(result, val)
		return result
	case []float64:
		result := make([]float64, len(val))
		copy(result, val)
		return result
	case []bool:
		result := make([]bool, len(val))
		copy(result, val)
		return result
	default:
		// For basic types (string, int, float, bool) and complex types
		// (structs, pointers), return as-is. Basic types are copied by value.
		// Complex types may still share internal state, which is acceptable
		// for template data that typically doesn't contain mutable structs.
		return val
	}
}

// deepCopyMap creates a deep copy of a map[string]any.
func deepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopySlice creates a deep copy of a []any slice.
func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = deepCopyValue(v)
	}
	return result
}

// GetInt retrieves an integer value by path.
// Handles int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64.
// Returns the value and true if found and convertible, or 0 and false otherwise.
func (c *Context) GetInt(path string) (int, bool) {
	val, ok := c.Get(path)
	if !ok {
		return 0, false
	}

	// Direct type assertion for common cases
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	}

	// Use reflection for other numeric types
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u := rv.Uint()
		if u > uint64(math.MaxInt) {
			return 0, false
		}
		return int(u), true
	case reflect.Float32, reflect.Float64:
		return int(rv.Float()), true
	}

	return 0, false
}

// GetIntDefault retrieves an integer value with a fallback default.
func (c *Context) GetIntDefault(path string, defaultVal int) int {
	val, ok := c.GetInt(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetFloat retrieves a float64 value by path.
// Handles float32, float64, and integer types.
// Returns the value and true if found and convertible, or 0 and false otherwise.
func (c *Context) GetFloat(path string) (float64, bool) {
	val, ok := c.Get(path)
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}

	// Use reflection for other numeric types
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	}

	return 0, false
}

// GetFloatDefault retrieves a float64 value with a fallback default.
func (c *Context) GetFloatDefault(path string, defaultVal float64) float64 {
	val, ok := c.GetFloat(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetBool retrieves a boolean value by path.
// Returns the value and true if found and is a bool, or false and false otherwise.
func (c *Context) GetBool(path string) (bool, bool) {
	val, ok := c.Get(path)
	if !ok {
		return false, false
	}
	if b, ok := val.(bool); ok {
		return b, true
	}
	return false, false
}

// GetBoolDefault retrieves a boolean value with a fallback default.
func (c *Context) GetBoolDefault(path string, defaultVal bool) bool {
	val, ok := c.GetBool(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetSlice retrieves a slice value by path.
// Returns the value and true if found and is a []any, or nil and false otherwise.
func (c *Context) GetSlice(path string) ([]any, bool) {
	val, ok := c.Get(path)
	if !ok {
		return nil, false
	}
	if s, ok := val.([]any); ok {
		return s, true
	}

	// Try to convert other slice types using reflection
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice {
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, true
	}

	return nil, false
}

// GetSliceDefault retrieves a slice value with a fallback default.
func (c *Context) GetSliceDefault(path string, defaultVal []any) []any {
	val, ok := c.GetSlice(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetMap retrieves a map value by path.
// Returns the value and true if found and is a map[string]any, or nil and false otherwise.
func (c *Context) GetMap(path string) (map[string]any, bool) {
	val, ok := c.Get(path)
	if !ok {
		return nil, false
	}
	if m, ok := val.(map[string]any); ok {
		return m, true
	}

	// Try to convert map[string]string
	if m, ok := val.(map[string]string); ok {
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result, true
	}

	return nil, false
}

// GetMapDefault retrieves a map value with a fallback default.
func (c *Context) GetMapDefault(path string, defaultVal map[string]any) map[string]any {
	val, ok := c.GetMap(path)
	if !ok {
		return defaultVal
	}
	return val
}

// GetStringSlice retrieves a []string value by path.
// Returns the value and true if found and convertible, or nil and false otherwise.
func (c *Context) GetStringSlice(path string) ([]string, bool) {
	val, ok := c.Get(path)
	if !ok {
		return nil, false
	}

	// Direct type assertion
	if s, ok := val.([]string); ok {
		return s, true
	}

	// Try to convert []any to []string
	if s, ok := val.([]any); ok {
		result := make([]string, 0, len(s))
		for _, v := range s {
			if str, ok := v.(string); ok {
				result = append(result, str)
			} else {
				return nil, false
			}
		}
		return result, true
	}

	return nil, false
}

// GetStringSliceDefault retrieves a []string value with a fallback default.
func (c *Context) GetStringSliceDefault(path string, defaultVal []string) []string {
	val, ok := c.GetStringSlice(path)
	if !ok {
		return defaultVal
	}
	return val
}
