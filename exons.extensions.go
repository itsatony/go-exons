package exons

import "encoding/json"

// Extension error messages
const (
	ErrMsgExtensionNotFound   = "extension key not found"
	ErrMsgExtensionCastFailed = "extension type conversion failed"
)

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

// GetExtensionAs converts the extension value for the given key into the target type T.
// It uses JSON marshal/unmarshal round-trip to convert map[string]any into a typed struct.
// Returns the zero value of T and an error if the key is not found or conversion fails.
func GetExtensionAs[T any](s *Spec, key string) (T, error) {
	var zero T
	if s == nil || s.Extensions == nil {
		return zero, NewSpecValidationError(ErrMsgExtensionNotFound, key)
	}

	val, ok := s.Extensions[key]
	if !ok {
		return zero, NewSpecValidationError(ErrMsgExtensionNotFound, key)
	}

	// JSON round-trip: marshal the raw value, then unmarshal into T
	data, err := json.Marshal(val)
	if err != nil {
		return zero, NewSpecValidationError(ErrMsgExtensionCastFailed, key)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, NewSpecValidationError(ErrMsgExtensionCastFailed, key)
	}

	return result, nil
}

// GetStandardFields returns a map of only the standard genspec fields
// that are set on this Spec: name, description, inputs, outputs, sample.
func (s *Spec) GetStandardFields() map[string]any {
	if s == nil {
		return nil
	}

	m := make(map[string]any)
	if s.Name != "" {
		m[SpecFieldName] = s.Name
	}
	if s.Description != "" {
		m[SpecFieldDescription] = s.Description
	}
	if len(s.Inputs) > 0 {
		m[SpecFieldInputs] = s.Inputs
	}
	if len(s.Outputs) > 0 {
		m[SpecFieldOutputs] = s.Outputs
	}
	if len(s.Sample) > 0 {
		m[SpecFieldSample] = s.Sample
	}

	return m
}

// GetExonsFields returns a map of only the go-exons extension fields
// that are set on this Spec: type, execution, skills, tools, context,
// constraints, messages, credentials, credential, genspec.
func (s *Spec) GetExonsFields() map[string]any {
	if s == nil {
		return nil
	}

	m := make(map[string]any)
	if s.Type != "" {
		m[SpecFieldType] = string(s.Type)
	}
	if s.Execution != nil {
		m[SpecFieldExecution] = s.Execution
	}
	if len(s.Skills) > 0 {
		m[SpecFieldSkills] = s.Skills
	}
	if s.Tools != nil {
		m[SpecFieldTools] = s.Tools
	}
	if len(s.Context) > 0 {
		m[SpecFieldContext] = s.Context
	}
	if s.Constraints != nil {
		m[SpecFieldConstraints] = s.Constraints
	}
	if len(s.Messages) > 0 {
		m[SpecFieldMessages] = s.Messages
	}
	if len(s.Credentials) > 0 {
		m[SpecFieldCredentials] = s.Credentials
	}
	if s.Credential != "" {
		m[SpecFieldCredential] = s.Credential
	}
	if s.GenSpec != nil {
		m[SpecFieldGenSpec] = s.GenSpec
	}

	return m
}
