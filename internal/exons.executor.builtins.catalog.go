package internal

import (
	"context"
)

// SkillsCatalogResolver handles the exons.skills_catalog built-in tag.
// It reads pre-generated skills catalog from the execution context.
type SkillsCatalogResolver struct{}

// NewSkillsCatalogResolver creates a new SkillsCatalogResolver.
func NewSkillsCatalogResolver() *SkillsCatalogResolver {
	return &SkillsCatalogResolver{}
}

// TagName returns the tag name for this resolver.
func (r *SkillsCatalogResolver) TagName() string {
	return TagNameSkillsCatalog
}

// Resolve returns the pre-generated skills catalog from the context.
func (r *SkillsCatalogResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	accessor, ok := execCtx.(ContextAccessor)
	if !ok {
		return "", NewBuiltinError(ErrMsgInvalidContext, TagNameSkillsCatalog)
	}

	// Read from context key where the catalog was injected
	val, found := accessor.Get(ContextKeySkills)
	if !found {
		return "", nil
	}

	if catalog, ok := val.(string); ok {
		return catalog, nil
	}

	return "", nil
}

// Validate checks attributes (no required attributes).
func (r *SkillsCatalogResolver) Validate(attrs Attributes) error {
	return nil
}

// ToolsCatalogResolver handles the exons.tools_catalog built-in tag.
// It reads pre-generated tools catalog from the execution context.
type ToolsCatalogResolver struct{}

// NewToolsCatalogResolver creates a new ToolsCatalogResolver.
func NewToolsCatalogResolver() *ToolsCatalogResolver {
	return &ToolsCatalogResolver{}
}

// TagName returns the tag name for this resolver.
func (r *ToolsCatalogResolver) TagName() string {
	return TagNameToolsCatalog
}

// Resolve returns the pre-generated tools catalog from the context.
func (r *ToolsCatalogResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	accessor, ok := execCtx.(ContextAccessor)
	if !ok {
		return "", NewBuiltinError(ErrMsgInvalidContext, TagNameToolsCatalog)
	}

	// Read from context key where the catalog was injected
	val, found := accessor.Get(ContextKeyTools)
	if !found {
		return "", nil
	}

	if catalog, ok := val.(string); ok {
		return catalog, nil
	}

	return "", nil
}

// Validate checks attributes (no required attributes).
func (r *ToolsCatalogResolver) Validate(attrs Attributes) error {
	return nil
}
