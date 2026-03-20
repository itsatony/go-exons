package exons

// GetExtensions returns the full extensions map, or nil if empty.
func (s *Spec) GetExtensions() map[string]any {
	if s == nil {
		return nil
	}
	return s.Extensions
}

// GetExtension returns the value for the given extension key and whether it exists.
func (s *Spec) GetExtension(key string) (any, bool) {
	if s == nil || s.Extensions == nil {
		return nil, false
	}
	val, ok := s.Extensions[key]
	return val, ok
}

// HasExtension returns true if the given extension key exists.
// Note: HasExtensions() (plural, no key) is defined in exons.spec.go.
func (s *Spec) HasExtension(key string) bool {
	if s == nil || s.Extensions == nil {
		return false
	}
	_, ok := s.Extensions[key]
	return ok
}

// SetExtension sets the given extension key to the given value.
// Initializes the Extensions map if nil.
func (s *Spec) SetExtension(key string, value any) {
	if s == nil {
		return
	}
	if s.Extensions == nil {
		s.Extensions = make(map[string]any)
	}
	s.Extensions[key] = value
}

// RemoveExtension removes the given extension key.
func (s *Spec) RemoveExtension(key string) {
	if s == nil || s.Extensions == nil {
		return
	}
	delete(s.Extensions, key)
}

