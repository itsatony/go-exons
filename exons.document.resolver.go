package exons

import (
	"context"
	"sync"
)

// MapSpecResolverEntry holds a spec and its template body for the MapSpecResolver.
type MapSpecResolverEntry struct {
	Spec *Spec
	Body string
}

// NoopSpecResolver is a SpecResolver that always returns a not-found error.
// Use this as a placeholder when no real spec resolution is available.
type NoopSpecResolver struct{}

// NewNoopSpecResolver creates a new NoopSpecResolver.
func NewNoopSpecResolver() *NoopSpecResolver {
	return &NoopSpecResolver{}
}

// ResolveSpec always returns a not-found error for any slug and version.
func (r *NoopSpecResolver) ResolveSpec(_ context.Context, slug string, version string) (*Spec, string, error) {
	return nil, "", NewRefNotFoundError(slug, version)
}

// MapSpecResolver is a thread-safe, in-memory SpecResolver backed by a map.
// All lookups return cloned Spec instances to prevent mutation of stored data.
//
// Thread safety: MapSpecResolver is safe for concurrent access via sync.RWMutex.
type MapSpecResolver struct {
	mu      sync.RWMutex
	entries map[string]MapSpecResolverEntry
}

// NewMapSpecResolver creates a new empty MapSpecResolver.
func NewMapSpecResolver() *MapSpecResolver {
	return &MapSpecResolver{
		entries: make(map[string]MapSpecResolverEntry),
	}
}

// Add stores a spec and body under the given slug.
// The spec is cloned on storage to prevent external mutation.
func (r *MapSpecResolver) Add(slug string, spec *Spec, body string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[slug] = MapSpecResolverEntry{
		Spec: spec.Clone(),
		Body: body,
	}
}

// AddMulti stores multiple entries at once.
// Each spec is cloned on storage to prevent external mutation.
func (r *MapSpecResolver) AddMulti(entries map[string]MapSpecResolverEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for slug, entry := range entries {
		r.entries[slug] = MapSpecResolverEntry{
			Spec: entry.Spec.Clone(),
			Body: entry.Body,
		}
	}
}

// Remove deletes the entry for the given slug.
// Returns true if the entry existed and was removed, false otherwise.
func (r *MapSpecResolver) Remove(slug string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.entries[slug]
	if exists {
		delete(r.entries, slug)
	}
	return exists
}

// Has returns true if an entry exists for the given slug.
func (r *MapSpecResolver) Has(slug string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.entries[slug]
	return exists
}

// Count returns the number of stored entries.
func (r *MapSpecResolver) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// ResolveSpec looks up a spec by slug. The version parameter is accepted but
// currently ignored (all lookups resolve to the stored entry regardless of version).
// Returns a cloned Spec and the stored body, or a not-found error if the slug
// does not exist.
func (r *MapSpecResolver) ResolveSpec(_ context.Context, slug string, version string) (*Spec, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, exists := r.entries[slug]
	if !exists {
		return nil, "", NewRefNotFoundError(slug, version)
	}
	return entry.Spec.Clone(), entry.Body, nil
}
