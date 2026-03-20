package internal

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
)

// InternalResolver mirrors the public Resolver interface for internal use.
// This allows the internal package to work with resolvers without import cycles.
type InternalResolver interface {
	TagName() string
	Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error)
	Validate(attrs Attributes) error
}

// Registry manages resolver registration with first-come-wins semantics.
// It is thread-safe for concurrent read/write access.
type Registry struct {
	resolvers map[string]InternalResolver
	mu        sync.RWMutex
	logger    *slog.Logger
}

// NewRegistry creates a new resolver registry.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	logger.Debug(LogMsgRegistryCreated)
	return &Registry{
		resolvers: make(map[string]InternalResolver),
		logger:    logger,
	}
}

// Register adds a resolver to the registry.
// If a resolver for the same tag name already exists, returns an error
// but does not panic (first-come-wins semantics).
func (r *Registry) Register(resolver InternalResolver) error {
	if resolver == nil {
		return NewRegistryError(ErrMsgNilResolver, "")
	}

	tagName := resolver.TagName()
	if tagName == "" {
		return NewRegistryError(ErrMsgEmptyTagName, "")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.resolvers[tagName]; exists {
		// First-come-wins: log collision but don't panic
		r.logger.Warn(LogMsgResolverCollision,
			slog.String(LogFieldTagName, tagName),
			slog.String(LogFieldExisting, existing.TagName()),
		)
		return NewRegistryError(ErrMsgResolverAlreadyExists, tagName)
	}

	r.resolvers[tagName] = resolver
	r.logger.Debug(LogMsgResolverRegistered, slog.String(LogFieldTagName, tagName))
	return nil
}

// MustRegister adds a resolver and panics if registration fails.
// Use this for built-in resolvers that must always be available.
func (r *Registry) MustRegister(resolver InternalResolver) {
	if err := r.Register(resolver); err != nil {
		panic(err)
	}
}

// Get retrieves a resolver by tag name.
// Returns the resolver and true if found, or nil and false if not.
func (r *Registry) Get(tagName string) (InternalResolver, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resolver, exists := r.resolvers[tagName]
	return resolver, exists
}

// Has checks if a resolver is registered for the given tag name.
func (r *Registry) Has(tagName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.resolvers[tagName]
	return exists
}

// List returns all registered tag names in sorted order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.resolvers))
	for name := range r.resolvers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered resolvers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.resolvers)
}

// RegistryError represents a registry operation error
type RegistryError struct {
	Message string
	TagName string
}

// NewRegistryError creates a new registry error
func NewRegistryError(message, tagName string) *RegistryError {
	return &RegistryError{
		Message: message,
		TagName: tagName,
	}
}

// Error implements the error interface
func (e *RegistryError) Error() string {
	if e.TagName != StringValueEmpty {
		return fmt.Sprintf(ErrFmtTagMessage, e.Message, e.TagName)
	}
	return e.Message
}

// Registry error message constants
const (
	ErrMsgNilResolver           = "resolver cannot be nil"
	ErrMsgEmptyTagName          = "resolver tag name cannot be empty"
	ErrMsgResolverAlreadyExists = "resolver already registered for tag"
	ErrMsgResolverUnknown       = "no resolver registered for tag"
)

// Additional log field constants for registry
const (
	LogFieldTagName  = "tag_name"
	LogFieldExisting = "existing"
)
