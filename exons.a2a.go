package exons

import (
	"context"
	"sort"
	"strings"

	"github.com/itsatony/go-exons/a2a"
)

// A2A metadata keys for safety/dispatch enrichment. On a v1.0.1 card these ride as
// `params` of a single go-exons AgentExtension (A2AExtensionURIGoExonsMetadata),
// because v1.0.1 has no top-level metadata field — capabilities.extensions[] is the
// spec's only vendor-data channel.
const (
	A2AMetaKeySafetyGuardrails    = "safety.guardrails"
	A2AMetaKeySafetyDenyTools     = "safety.deny_tools"
	A2AMetaKeySafetyConfirmation  = "safety.require_confirmation_for"
	A2AMetaKeyDispatchDescription = "dispatch.trigger_description"
)

// A2ACardOptions configures how a Spec is compiled into an A2A v1.0.1 Agent Card.
type A2ACardOptions struct {
	// SupportedInterfaces are the transport interfaces the card advertises. A2A
	// v1.0.1 requires at least one; a declaration-only publisher (aigentverse) passes
	// a single interface pointing at the registry definition URL. Any interface with
	// an empty ProtocolVersion is defaulted to opts.ProtocolVersion (else the package
	// default). Not derivable from the Spec — the deployment endpoint is not part of
	// the definition.
	SupportedInterfaces []a2a.AgentInterface
	// ProviderOrganization is the organization name for the provider field.
	ProviderOrganization string
	// ProviderURL is the organization's website URL.
	ProviderURL string
	// Version overrides the agent version (defaults to registry metadata version, then A2AVersionDefault).
	Version string
	// ProtocolVersion is the default per-interface A2A protocol version applied to
	// any supplied interface that omits one (defaults to A2AProtocolVersionDefault).
	ProtocolVersion string
	// DefaultInputModes overrides auto-detected input MIME types.
	DefaultInputModes []string
	// DefaultOutputModes overrides auto-detected output MIME types.
	DefaultOutputModes []string
	// SecuritySchemes defines inbound authentication configuration.
	SecuritySchemes map[string]any
	// SecurityRequirements references which security schemes are required.
	SecurityRequirements []map[string][]string
	// Capabilities overrides auto-detected capabilities (streaming). Extensions from
	// this value are preserved and merged with go-exons enrichment + opts.Extensions.
	Capabilities *a2a.AgentCapabilities
	// Extensions are additional protocol extensions to advertise (e.g. aigentverse's
	// provenance extension carrying did/content-hash/declaration-only).
	Extensions []a2a.AgentExtension
	// DocumentationURL / IconURL are optional passthroughs.
	DocumentationURL string
	IconURL          string
	// Resolver resolves skill descriptions from external specs.
	Resolver SpecResolver
}

// CompileAgentCard generates an A2A v1.0.1 Agent Card from this Spec's configuration.
// This is a pure metadata transformation — no template execution occurs.
//
// Name is taken from the Spec and must not be empty. Transport endpoints are not part
// of the definition, so they come from opts.SupportedInterfaces; a declaration-only
// card supplies a single registry-definition interface. The returned card is not
// guaranteed to be fully conformant on its own (e.g. if the caller supplies no
// interfaces) — call AgentCard.Validate to check.
//
// Skills are mapped from SkillRef entries (descriptions resolved via opts.Resolver
// when available; resolution failures are non-fatal). When the Spec declares no
// skills, a single skill is synthesized from the agent itself so the required,
// non-empty skills[] is satisfied.
//
// Enrichment: dispatch.TriggerKeywords → each skill's Tags; registry.Version → the
// agent version; safety.* + dispatch.TriggerDescription + a2a-prefixed extensions →
// the params of a go-exons metadata AgentExtension. Streaming is auto-detected from
// execution.Config.Streaming.Enabled unless overridden.
func (s *Spec) CompileAgentCard(ctx context.Context, opts *A2ACardOptions) (*a2a.AgentCard, error) {
	if s == nil {
		return nil, NewA2AError(ErrMsgA2ACardNilSpec, nil)
	}
	if opts == nil {
		opts = &A2ACardOptions{}
	}
	if s.Name == "" {
		return nil, NewA2AError(ErrMsgA2ACardMissingName, nil)
	}

	defaultProtoVersion := opts.ProtocolVersion
	if defaultProtoVersion == "" {
		defaultProtoVersion = A2AProtocolVersionDefault
	}
	interfaces := make([]a2a.AgentInterface, 0, len(opts.SupportedInterfaces))
	for _, iface := range opts.SupportedInterfaces {
		if iface.ProtocolVersion == "" {
			iface.ProtocolVersion = defaultProtoVersion
		}
		interfaces = append(interfaces, iface)
	}

	card := &a2a.AgentCard{
		Name:                 s.Name,
		Description:          s.Description,
		SupportedInterfaces:  interfaces,
		Version:              a2aResolveVersion(s, opts),
		DocumentationURL:     opts.DocumentationURL,
		IconURL:              opts.IconURL,
		DefaultInputModes:    a2aInputModes(s, opts),
		DefaultOutputModes:   a2aOutputModes(s, opts),
		Skills:               a2aCompileSkills(ctx, s, opts.Resolver),
		SecuritySchemes:      opts.SecuritySchemes,
		SecurityRequirements: opts.SecurityRequirements,
	}

	// Provider (both url + organization are required by the spec when present).
	if opts.ProviderOrganization != "" || opts.ProviderURL != "" {
		card.Provider = &a2a.AgentProvider{
			Organization: opts.ProviderOrganization,
			URL:          opts.ProviderURL,
		}
	}

	card.Capabilities = a2aBuildCapabilities(s, opts)

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

// a2aBuildCapabilities assembles the AgentCapabilities: streaming (override or
// auto-detected) plus the merged extension list (go-exons enrichment + caller
// extensions + any extensions on an override capability set).
func a2aBuildCapabilities(s *Spec, opts *A2ACardOptions) *a2a.AgentCapabilities {
	caps := &a2a.AgentCapabilities{}
	if opts.Capabilities != nil {
		caps = opts.Capabilities
	} else if s.Execution != nil && s.Execution.Streaming != nil && s.Execution.Streaming.Enabled {
		streaming := true
		caps.Streaming = &streaming
	}

	var exts []a2a.AgentExtension
	exts = append(exts, caps.Extensions...)
	if meta := a2aBuildMetadata(s); len(meta) > 0 {
		exts = append(exts, a2a.AgentExtension{
			URI:    A2AExtensionURIGoExonsMetadata,
			Params: meta,
		})
	}
	exts = append(exts, opts.Extensions...)
	caps.Extensions = exts
	return caps
}

// a2aCompileSkills maps SkillRef entries to A2A v1.0.1 skills. Descriptions are
// resolved via SpecResolver (non-fatal on failure); an empty description falls back
// to the skill name so the required description is never blank. Tags are always a
// non-nil slice (dispatch keywords when present). When the Spec declares no skills,
// a single skill is synthesized from the agent itself.
func a2aCompileSkills(ctx context.Context, s *Spec, resolver SpecResolver) []a2a.AgentSkill {
	var dispatchTags []string
	if s.Dispatch != nil && len(s.Dispatch.TriggerKeywords) > 0 {
		dispatchTags = s.Dispatch.TriggerKeywords
	}
	tagsCopy := func() []string {
		out := make([]string, 0, len(dispatchTags))
		return append(out, dispatchTags...)
	}

	if len(s.Skills) == 0 {
		// Synthesize the agent's own capability so skills[] is present + non-empty.
		desc := s.Description
		if desc == "" {
			desc = s.Name
		}
		return []a2a.AgentSkill{{
			ID:          s.Name,
			Name:        s.Name,
			Description: desc,
			Tags:        tagsCopy(),
		}}
	}

	skills := make([]a2a.AgentSkill, 0, len(s.Skills))
	for i := range s.Skills {
		ref := &s.Skills[i]
		skill := a2a.AgentSkill{
			ID:   ref.Slug,
			Name: ref.Slug,
			Tags: tagsCopy(),
		}
		if resolver != nil && ref.Slug != "" {
			resolved, _, err := resolver.ResolveSpec(ctx, ref.Slug, "")
			if err == nil && resolved != nil {
				skill.Description = resolved.Description
				if resolved.Name != "" {
					skill.Name = resolved.Name
				}
				if resolved.Execution != nil && resolved.Execution.Modality != "" {
					if mime := modalityToMIME(resolved.Execution.Modality); mime != "" {
						skill.OutputModes = []string{mime}
					}
				}
			}
		}
		if skill.Description == "" {
			skill.Description = skill.Name
		}
		skills = append(skills, skill)
	}
	return skills
}

// a2aInputModes resolves the default input modes (override or auto-detected).
func a2aInputModes(s *Spec, opts *A2ACardOptions) []string {
	if len(opts.DefaultInputModes) > 0 {
		return opts.DefaultInputModes
	}
	return a2aInferInputModes(s)
}

// a2aOutputModes resolves the default output modes (override or auto-detected).
func a2aOutputModes(s *Spec, opts *A2ACardOptions) []string {
	if len(opts.DefaultOutputModes) > 0 {
		return opts.DefaultOutputModes
	}
	return a2aInferOutputModes(s)
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
		mimeSet[inputTypeToMIME(def.Type)] = true
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

// a2aBuildMetadata merges a2a-prefixed extensions and safety/dispatch enrichment into
// a single map, carried as the params of the go-exons metadata extension. Extensions
// with the "a2a." prefix are included first, then safety/dispatch keys (disjoint key
// spaces: extensions always carry the "a2a." prefix, metadata keys never do).
func a2aBuildMetadata(s *Spec) map[string]any {
	meta := make(map[string]any)
	for k, v := range s.Extensions {
		if strings.HasPrefix(k, ExtensionPrefixA2A) {
			meta[k] = v
		}
	}
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
		// Video generation APIs return structured JSON metadata (URLs, dimensions,
		// status) rather than raw binary video streams — application/json is correct.
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
