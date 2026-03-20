package exons

import (
	"context"
	"sort"
	"strings"

	"github.com/itsatony/go-exons/a2a"
)

// A2A metadata keys for safety/dispatch enrichment.
const (
	A2AMetaKeySafetyGuardrails    = "safety.guardrails"
	A2AMetaKeySafetyDenyTools     = "safety.deny_tools"
	A2AMetaKeySafetyConfirmation  = "safety.require_confirmation_for"
	A2AMetaKeyDispatchDescription = "dispatch.trigger_description"
)

// A2ACardOptions configures how a Spec is compiled into an A2A Agent Card.
type A2ACardOptions struct {
	// URL is the agent's service endpoint (required — not derivable from Spec)
	URL string
	// ProviderOrganization is the organization name for the Agent Card's provider field
	ProviderOrganization string
	// ProviderURL is the organization's website URL
	ProviderURL string
	// Version overrides the agent version (defaults to registry metadata version, then "1.0.0")
	Version string
	// ProtocolVersion overrides the A2A protocol version (defaults to "0.3.0")
	ProtocolVersion string
	// DefaultInputModes overrides auto-detected input MIME types
	DefaultInputModes []string
	// DefaultOutputModes overrides auto-detected output MIME types
	DefaultOutputModes []string
	// SecuritySchemes defines inbound authentication configuration
	SecuritySchemes map[string]any
	// Security references which security schemes are required
	Security []map[string][]string
	// Capabilities overrides auto-detected capabilities
	Capabilities *a2a.Capabilities
	// Resolver resolves skill descriptions from external specs
	Resolver SpecResolver
}

// CompileAgentCard generates an A2A Agent Card from this Spec's configuration.
// This is a pure metadata transformation — no template execution occurs.
//
// The URL must be provided via options (it represents the deployment endpoint,
// which is not part of the Spec definition). Name is taken from the Spec
// and must not be empty.
//
// Skills are mapped from SkillRef entries. Descriptions are resolved via
// opts.Resolver when available; resolution failures are non-fatal (the skill
// appears with an empty description, following the same pattern as GenerateSkillsCatalog).
//
// Metadata enrichment:
//   - dispatch.TriggerKeywords → appended to each A2A skill's Tags
//   - registry.Version → used as agent card Version (if not overridden by opts)
//   - safety.Guardrails, safety.DenyTools, safety.RequireConfirmationFor → metadata
//   - dispatch.TriggerDescription → metadata under "dispatch.trigger_description"
//
// Metadata key spaces: a2a-prefixed extensions (e.g., "a2a.team") and metadata
// fields (e.g., "safety.guardrails") use disjoint key namespaces — no collisions.
//
// Streaming capability is auto-detected from execution.Config.Streaming.Enabled
// unless overridden via opts.Capabilities.
//
// Input modes are inferred from Spec.Inputs types (string->"text/plain",
// object->"application/json", etc.). Output modes are inferred from the
// execution modality. Both can be overridden via options.
func (s *Spec) CompileAgentCard(ctx context.Context, opts *A2ACardOptions) (*a2a.AgentCard, error) {
	if s == nil {
		return nil, NewA2AError(ErrMsgA2ACardNilSpec, nil)
	}
	if opts == nil || opts.URL == "" {
		return nil, NewA2AError(ErrMsgA2ACardMissingURL, nil)
	}
	if s.Name == "" {
		return nil, NewA2AError(ErrMsgA2ACardMissingName, nil)
	}

	card := &a2a.AgentCard{
		Name:            s.Name,
		Description:     s.Description,
		URL:             opts.URL,
		ProtocolVersion: A2AProtocolVersionDefault,
	}

	// Version: opts > registry metadata > default
	card.Version = a2aResolveVersion(s, opts)

	// Protocol version override
	if opts.ProtocolVersion != "" {
		card.ProtocolVersion = opts.ProtocolVersion
	}

	// Provider
	if opts.ProviderOrganization != "" {
		card.Provider = &a2a.Provider{
			Organization: opts.ProviderOrganization,
			URL:          opts.ProviderURL,
		}
	}

	// Capabilities: override or auto-detect streaming
	if opts.Capabilities != nil {
		card.Capabilities = opts.Capabilities
	} else {
		card.Capabilities = a2aAutoDetectCapabilities(s)
	}

	// Skills: map SkillRefs with optional resolver and dispatch tags
	card.Skills = a2aCompileSkills(ctx, s, opts.Resolver)

	// Input modes: override or auto-detect from Inputs
	if len(opts.DefaultInputModes) > 0 {
		card.DefaultInputModes = opts.DefaultInputModes
	} else {
		card.DefaultInputModes = a2aInferInputModes(s)
	}

	// Output modes: override or auto-detect from modality
	if len(opts.DefaultOutputModes) > 0 {
		card.DefaultOutputModes = opts.DefaultOutputModes
	} else {
		card.DefaultOutputModes = a2aInferOutputModes(s)
	}

	// Security passthrough
	card.SecuritySchemes = opts.SecuritySchemes
	card.Security = opts.Security

	// Metadata: a2a-prefixed extensions + safety/dispatch metadata
	card.Metadata = a2aBuildMetadata(s)

	return card, nil
}

// --- Internal helpers ---

// a2aResolveVersion determines the agent card version from options, registry metadata, or default.
func a2aResolveVersion(s *Spec, opts *A2ACardOptions) string {
	if opts.Version != "" {
		return opts.Version
	}
	if s.Registry != nil && s.Registry.Version != "" {
		return s.Registry.Version
	}
	return A2AVersionDefault
}

// a2aAutoDetectCapabilities detects capabilities from the Spec's execution config.
func a2aAutoDetectCapabilities(s *Spec) *a2a.Capabilities {
	caps := &a2a.Capabilities{}
	if s.Execution != nil && s.Execution.Streaming != nil && s.Execution.Streaming.Enabled {
		caps.Streaming = true
	}
	return caps
}

// a2aCompileSkills maps SkillRef entries to A2A skills.
// Descriptions are resolved via SpecResolver (non-fatal on failure).
// Dispatch keywords are appended as tags to each skill.
func a2aCompileSkills(ctx context.Context, s *Spec, resolver SpecResolver) []a2a.Skill {
	if len(s.Skills) == 0 {
		return nil
	}

	// Collect dispatch keywords for tag enrichment
	var dispatchTags []string
	if s.Dispatch != nil && len(s.Dispatch.TriggerKeywords) > 0 {
		dispatchTags = s.Dispatch.TriggerKeywords
	}

	skills := make([]a2a.Skill, 0, len(s.Skills))

	for i := range s.Skills {
		ref := &s.Skills[i]

		skill := a2a.Skill{
			ID:   ref.Slug,
			Name: ref.Slug,
		}

		// Resolve description and name via SpecResolver (non-fatal)
		if resolver != nil && ref.Slug != "" {
			resolved, _, err := resolver.ResolveSpec(ctx, ref.Slug, "")
			if err == nil && resolved != nil {
				skill.Description = resolved.Description
				if resolved.Name != "" {
					skill.Name = resolved.Name
				}
				// Infer output modes from resolved spec's execution modality
				if resolved.Execution != nil && resolved.Execution.Modality != "" {
					if mime := modalityToMIME(resolved.Execution.Modality); mime != "" {
						skill.OutputModes = []string{mime}
					}
				}
			}
			// Resolution failures are non-fatal — skill appears with empty description
		}

		// Append dispatch keywords as tags
		if len(dispatchTags) > 0 {
			skill.Tags = make([]string, len(dispatchTags))
			copy(skill.Tags, dispatchTags)
		}

		skills = append(skills, skill)
	}

	return skills
}

// a2aInferInputModes infers MIME types from Spec.Inputs definitions.
func a2aInferInputModes(s *Spec) []string {
	if len(s.Inputs) == 0 {
		return []string{A2AMIMETextPlain}
	}

	mimeSet := make(map[string]bool)
	for _, def := range s.Inputs {
		if def == nil {
			continue
		}
		mime := inputTypeToMIME(def.Type)
		mimeSet[mime] = true
	}

	if len(mimeSet) == 0 {
		return []string{A2AMIMETextPlain}
	}

	return sortedStringKeys(mimeSet)
}

// a2aInferOutputModes infers output MIME types from the Spec's modality configuration.
func a2aInferOutputModes(s *Spec) []string {
	modality := ""
	if s.Execution != nil && s.Execution.Modality != "" {
		modality = s.Execution.Modality
	}

	if modality == "" {
		return []string{A2AMIMETextPlain}
	}

	mime := modalityToMIME(modality)
	if mime == "" {
		return []string{A2AMIMETextPlain}
	}
	return []string{mime}
}

// a2aBuildMetadata merges a2a-prefixed extensions and metadata fields.
// Extensions with the "a2a." prefix are included first, then metadata fields
// are added under unprefixed keys (e.g., "safety.guardrails", "dispatch.trigger_description").
// These key spaces do not overlap: extensions always have the "a2a." prefix,
// metadata keys never do.
func a2aBuildMetadata(s *Spec) map[string]any {
	meta := make(map[string]any)

	// Merge a2a-prefixed extensions
	for k, v := range s.Extensions {
		if strings.HasPrefix(k, ExtensionPrefixA2A) {
			meta[k] = v
		}
	}

	// Safety config enrichment
	if s.Safety != nil {
		if s.Safety.Guardrails != "" {
			meta[A2AMetaKeySafetyGuardrails] = s.Safety.Guardrails
		}
		if len(s.Safety.DenyTools) > 0 {
			meta[A2AMetaKeySafetyDenyTools] = s.Safety.DenyTools
		}
		if len(s.Safety.RequireConfirmationFor) > 0 {
			meta[A2AMetaKeySafetyConfirmation] = s.Safety.RequireConfirmationFor
		}
	}

	// Dispatch description enrichment
	if s.Dispatch != nil && s.Dispatch.TriggerDescription != "" {
		meta[A2AMetaKeyDispatchDescription] = s.Dispatch.TriggerDescription
	}

	if len(meta) == 0 {
		return nil
	}
	return meta
}

// modalityToMIME converts a modality constant to an A2A MIME type.
func modalityToMIME(modality string) string {
	switch modality {
	case ModalityText:
		return A2AMIMETextPlain
	case ModalityImage, ModalityImageEdit:
		return A2AMIMEImagePNG
	case ModalityAudioSpeech, ModalityAudioTranscription, ModalityMusic, ModalitySoundEffects:
		return A2AMIMEAudioMPEG
	case ModalityEmbedding:
		return A2AMIMEApplicationJSON
	case ModalityVideo:
		// Video generation APIs return structured JSON metadata (URLs, dimensions, status)
		// rather than raw binary video streams — application/json is the correct wire type.
		return A2AMIMEApplicationJSON
	default:
		return ""
	}
}

// inputTypeToMIME converts an InputDef type to an A2A MIME type.
func inputTypeToMIME(inputType string) string {
	switch inputType {
	case SchemaTypeString:
		return A2AMIMETextPlain
	case SchemaTypeObject, SchemaTypeArray:
		return A2AMIMEApplicationJSON
	case SchemaTypeNumber, SchemaTypeBoolean:
		return A2AMIMETextPlain
	default:
		return A2AMIMETextPlain
	}
}

// sortedStringKeys returns the keys of a map[string]bool as a sorted string slice.
func sortedStringKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
