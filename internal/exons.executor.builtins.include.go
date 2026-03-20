package internal

import (
	"context"
)

// TemplateExecutor is the interface for executing nested templates.
// This is used by IncludeResolver and InheritanceResolver to execute registered templates.
type TemplateExecutor interface {
	ExecuteTemplate(ctx context.Context, name string, data map[string]any) (string, error)
	HasTemplate(name string) bool
	MaxDepth() int
	GetTemplateSource(name string) (string, bool)
}

// IncludeResolver handles the exons.include built-in tag.
// It executes registered templates and inserts their output.
type IncludeResolver struct{}

// NewIncludeResolver creates a new IncludeResolver.
func NewIncludeResolver() *IncludeResolver {
	return &IncludeResolver{}
}

// TagName returns the tag name for this resolver.
func (r *IncludeResolver) TagName() string {
	return TagNameInclude
}

// Resolve executes the referenced template and returns its output.
func (r *IncludeResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	// Get the template context accessor
	tmplCtx, ok := execCtx.(TemplateContextAccessor)
	if !ok {
		// Fall back to basic context accessor - no template support
		return "", NewBuiltinError(ErrMsgEngineNotAvailable, TagNameInclude)
	}

	// Get required 'template' attribute
	templateName, ok := attrs.Get(AttrTemplate)
	if !ok {
		return "", NewBuiltinError(ErrMsgMissingTemplateAttr, TagNameInclude)
	}

	// Get the engine from context
	engineInterface := tmplCtx.Engine()
	if engineInterface == nil {
		return "", NewBuiltinError(ErrMsgEngineNotAvailable, TagNameInclude)
	}

	engine, ok := engineInterface.(TemplateExecutor)
	if !ok {
		return "", NewBuiltinError(ErrMsgEngineNotAvailable, TagNameInclude)
	}

	// Check if template exists
	if !engine.HasTemplate(templateName) {
		return "", NewTemplateNotFoundWithHintError(templateName)
	}

	// Check depth limit
	currentDepth := tmplCtx.Depth()
	maxDepth := engine.MaxDepth()
	if maxDepth > 0 && currentDepth >= maxDepth {
		return "", NewBuiltinError(ErrMsgDepthExceeded, TagNameInclude)
	}

	// Build context data for child template
	childData := r.buildChildData(tmplCtx, attrs)

	// Execute the template
	// Note: The engine's ExecuteTemplate will create a new context with depth+1
	result, err := engine.ExecuteTemplate(ctx, templateName, childData)
	if err != nil {
		return "", NewBuiltinError(err.Error(), TagNameInclude)
	}

	return result, nil
}

// buildChildData creates the data map for the child template context.
func (r *IncludeResolver) buildChildData(tmplCtx TemplateContextAccessor, attrs Attributes) map[string]any {
	// Check for isolate mode
	isolate := attrs.GetDefault(AttrIsolate, AttrValueFalse) == AttrValueTrue

	childData := make(map[string]any)

	// If not isolated, we need to pass parent data
	// However, since we're using ExecuteTemplate which creates a fresh context,
	// we need to explicitly copy relevant data.
	// For now, we'll pass attributes as the child data and rely on the
	// engine to handle context inheritance if needed.

	// Check for 'with' attribute - scopes context to a path
	withPath, hasWith := attrs.Get(AttrWith)
	if hasWith && !isolate {
		// Get the value at the 'with' path and use it as root context
		val, found := tmplCtx.Get(withPath)
		if found {
			// If it's a map, use it directly
			if m, ok := val.(map[string]any); ok {
				for k, v := range m {
					childData[k] = v
				}
			} else {
				// Otherwise, put it under a special key
				childData[MetaKeyValue] = val
			}
		}
	}

	// Add all non-reserved attributes as context variables
	// Reserved attributes: template, with, isolate
	for _, key := range attrs.Keys() {
		if key == AttrTemplate || key == AttrWith || key == AttrIsolate {
			continue
		}
		val, _ := attrs.Get(key)
		childData[key] = val
	}

	// Include current depth for nested tracking
	// This will be incremented by the engine when creating the child context
	childData[MetaKeyParentDepth] = tmplCtx.Depth()

	return childData
}

// Validate checks that the required attributes are present.
func (r *IncludeResolver) Validate(attrs Attributes) error {
	if !attrs.Has(AttrTemplate) {
		return NewBuiltinError(ErrMsgMissingTemplateAttr, TagNameInclude)
	}
	return nil
}
