package exons

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopSpecResolver_AlwaysReturnsError(t *testing.T) {
	resolver := NewNoopSpecResolver()
	ctx := context.Background()

	spec, body, err := resolver.ResolveSpec(ctx, "any-slug", "v1")
	assert.Nil(t, spec)
	assert.Empty(t, body)
	require.Error(t, err)

	// Try another slug to verify it always errors
	spec2, body2, err2 := resolver.ResolveSpec(ctx, "other-slug", RefVersionLatest)
	assert.Nil(t, spec2)
	assert.Empty(t, body2)
	require.Error(t, err2)
}

func TestNoopSpecResolver_ImplementsInterface(t *testing.T) {
	var _ SpecResolver = NewNoopSpecResolver()
}

func TestMapSpecResolver_AddAndResolve(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	spec := &Spec{
		Name:        "test-skill",
		Description: "A test skill",
		Type:        DocumentTypeSkill,
	}
	body := "Hello {~exons.var name=\"user\" /~}"

	resolver.Add("test-skill", spec, body)

	resolved, resolvedBody, err := resolver.ResolveSpec(ctx, "test-skill", RefVersionLatest)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "test-skill", resolved.Name)
	assert.Equal(t, "A test skill", resolved.Description)
	assert.Equal(t, body, resolvedBody)
}

func TestMapSpecResolver_ClonesSpec(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	original := &Spec{
		Name:        "clone-test",
		Description: "Original description",
		Type:        DocumentTypeSkill,
	}
	body := "template body"

	resolver.Add("clone-test", original, body)

	// Modify the original after adding
	original.Description = "Modified description"

	// Resolve should return the original (cloned at add time), not the modified version
	resolved, _, err := resolver.ResolveSpec(ctx, "clone-test", RefVersionLatest)
	require.NoError(t, err)
	assert.Equal(t, "Original description", resolved.Description)

	// Modifying the resolved spec should not affect subsequent resolves
	resolved.Description = "Tampered description"

	resolved2, _, err := resolver.ResolveSpec(ctx, "clone-test", RefVersionLatest)
	require.NoError(t, err)
	assert.Equal(t, "Original description", resolved2.Description)
}

func TestMapSpecResolver_NotFound(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	spec, body, err := resolver.ResolveSpec(ctx, "nonexistent", "v1")
	assert.Nil(t, spec)
	assert.Empty(t, body)
	require.Error(t, err)
}

func TestMapSpecResolver_VersionIgnored(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	spec := &Spec{
		Name:        "versioned-skill",
		Description: "A versioned skill",
	}
	resolver.Add("versioned-skill", spec, "body")

	// Any version string should resolve to the same entry
	versions := []string{"", "v1", "v2", RefVersionLatest, "some-random-version"}
	for _, v := range versions {
		resolved, _, err := resolver.ResolveSpec(ctx, "versioned-skill", v)
		require.NoError(t, err, "version %q should resolve", v)
		assert.Equal(t, "versioned-skill", resolved.Name, "version %q", v)
	}
}

func TestMapSpecResolver_Remove(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	spec := &Spec{
		Name:        "removable",
		Description: "Will be removed",
	}
	resolver.Add("removable", spec, "body")

	// Verify it exists
	_, _, err := resolver.ResolveSpec(ctx, "removable", RefVersionLatest)
	require.NoError(t, err)

	// Remove it
	removed := resolver.Remove("removable")
	assert.True(t, removed)

	// Verify it's gone
	_, _, err = resolver.ResolveSpec(ctx, "removable", RefVersionLatest)
	require.Error(t, err)

	// Removing again returns false
	removed = resolver.Remove("removable")
	assert.False(t, removed)
}

func TestMapSpecResolver_Has(t *testing.T) {
	resolver := NewMapSpecResolver()

	assert.False(t, resolver.Has("missing"))

	resolver.Add("present", &Spec{Name: "present", Description: "exists"}, "body")
	assert.True(t, resolver.Has("present"))
	assert.False(t, resolver.Has("still-missing"))
}

func TestMapSpecResolver_Count(t *testing.T) {
	resolver := NewMapSpecResolver()

	assert.Equal(t, 0, resolver.Count())

	resolver.Add("one", &Spec{Name: "one", Description: "first"}, "body1")
	assert.Equal(t, 1, resolver.Count())

	resolver.Add("two", &Spec{Name: "two", Description: "second"}, "body2")
	assert.Equal(t, 2, resolver.Count())

	resolver.Remove("one")
	assert.Equal(t, 1, resolver.Count())
}

func TestMapSpecResolver_AddMulti(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	entries := map[string]MapSpecResolverEntry{
		"skill-a": {
			Spec: &Spec{Name: "skill-a", Description: "Skill A"},
			Body: "body-a",
		},
		"skill-b": {
			Spec: &Spec{Name: "skill-b", Description: "Skill B"},
			Body: "body-b",
		},
		"skill-c": {
			Spec: &Spec{Name: "skill-c", Description: "Skill C"},
			Body: "body-c",
		},
	}

	resolver.AddMulti(entries)
	assert.Equal(t, 3, resolver.Count())

	resolved, body, err := resolver.ResolveSpec(ctx, "skill-b", RefVersionLatest)
	require.NoError(t, err)
	assert.Equal(t, "skill-b", resolved.Name)
	assert.Equal(t, "body-b", body)

	// Verify the original entries are cloned (modify original, check stored)
	entries["skill-a"].Spec.Description = "Modified externally"
	resolvedA, _, err := resolver.ResolveSpec(ctx, "skill-a", RefVersionLatest)
	require.NoError(t, err)
	assert.Equal(t, "Skill A", resolvedA.Description)
}

func TestMapSpecResolver_ThreadSafety(t *testing.T) {
	resolver := NewMapSpecResolver()
	ctx := context.Background()

	// Pre-populate some entries
	for i := 0; i < 10; i++ {
		slug := fmt.Sprintf("skill-%d", i)
		resolver.Add(slug, &Spec{
			Name:        slug,
			Description: fmt.Sprintf("Skill %d", i),
		}, fmt.Sprintf("body-%d", i))
	}

	var wg sync.WaitGroup
	const goroutines = 20
	const iterations = 100

	// Concurrent reads
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slug := fmt.Sprintf("skill-%d", i%10)
				_, _, _ = resolver.ResolveSpec(ctx, slug, RefVersionLatest)
				_ = resolver.Has(slug)
				_ = resolver.Count()
			}
		}(g)
	}

	// Concurrent writes
	for g := 0; g < goroutines/2; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slug := fmt.Sprintf("dynamic-%d-%d", id, i)
				resolver.Add(slug, &Spec{
					Name:        slug,
					Description: fmt.Sprintf("Dynamic %d %d", id, i),
				}, "body")
			}
		}(g)
	}

	// Concurrent removes
	for g := 0; g < goroutines/4; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				slug := fmt.Sprintf("dynamic-%d-%d", id, i)
				resolver.Remove(slug)
			}
		}(g)
	}

	wg.Wait()

	// Verify original entries still accessible
	for i := 0; i < 10; i++ {
		slug := fmt.Sprintf("skill-%d", i)
		assert.True(t, resolver.Has(slug))
	}
}

func TestMapSpecResolver_ImplementsInterface(t *testing.T) {
	var _ SpecResolver = NewMapSpecResolver()
}
