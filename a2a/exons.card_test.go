package a2a

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// conformantCard is a minimal card satisfying every A2A v1.0.1 required field.
func conformantCard() *AgentCard {
	return &AgentCard{
		Name:        "recipe-agent",
		Description: "Helps with recipes and cooking.",
		Version:     "1.2.0",
		SupportedInterfaces: []AgentInterface{
			{URL: "https://reg.example.com/@org/recipe-agent", ProtocolBinding: "HTTP+JSON", ProtocolVersion: "1.0"},
		},
		Capabilities:       &AgentCapabilities{},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Skills: []AgentSkill{
			{ID: "search", Name: "search", Description: "Searches recipes", Tags: []string{}},
		},
	}
}

func TestAgentCard_ToJSON_NilReceiver(t *testing.T) {
	var card *AgentCard
	data, err := card.ToJSON()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestAgentCard_ToJSONPretty_NilReceiver(t *testing.T) {
	var card *AgentCard
	data, err := card.ToJSONPretty()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestAgentCard_ToJSON_v101Shape(t *testing.T) {
	card := conformantCard()
	data, err := card.ToJSON()
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	// v1.0.1 field names (protojson camelCase) present.
	assert.Contains(t, parsed, "supportedInterfaces")
	assert.Contains(t, parsed, "defaultInputModes")
	assert.Contains(t, parsed, "skills")
	// Retired v0.3 shape must be gone.
	assert.NotContains(t, parsed, "url")
	assert.NotContains(t, parsed, "preferredTransport")
	assert.NotContains(t, parsed, "protocolVersion") // moved to per-interface
	assert.NotContains(t, parsed, "metadata")        // no top-level metadata in v1.0.1

	ifaces := parsed["supportedInterfaces"].([]any)
	require.Len(t, ifaces, 1)
	iface := ifaces[0].(map[string]any)
	assert.Equal(t, "HTTP+JSON", iface["protocolBinding"])
	assert.Equal(t, "1.0", iface["protocolVersion"])
}

func TestAgentCard_Validate_ConformantAndDefects(t *testing.T) {
	assert.Empty(t, conformantCard().Validate(), "a conformant card has no violations")

	var nilCard *AgentCard
	assert.NotEmpty(t, nilCard.Validate())

	bad := &AgentCard{} // everything missing
	fields := violationFields(bad.Validate())
	for _, want := range []string{"name", "description", "version", "supportedInterfaces", "capabilities", "defaultInputModes", "defaultOutputModes", "skills"} {
		assert.True(t, fields[want], "expected a violation for %q", want)
	}

	// Interface + skill field-level checks.
	partial := conformantCard()
	partial.SupportedInterfaces[0].ProtocolVersion = ""
	partial.Skills[0].Description = ""
	partial.Skills[0].Tags = nil
	fields = violationFields(partial.Validate())
	assert.True(t, fields["supportedInterfaces[0].protocolVersion"])
	assert.True(t, fields["skills[0].description"])
	assert.True(t, fields["skills[0].tags"])
}

func TestAgentCard_CanonicalPayload_ExcludesSignaturesAndIsStable(t *testing.T) {
	card := conformantCard()
	c1, err := card.CanonicalPayload()
	require.NoError(t, err)

	// Attaching a signature must not change the canonical payload (signatures excluded).
	card.AttachDetachedSignature("cHJvdGVjdGVk", []byte("rawsig"))
	c2, err := card.CanonicalPayload()
	require.NoError(t, err)
	assert.Equal(t, string(c1), string(c2), "canonical payload must exclude signatures")

	// JCS is deterministic across independent builds of an equal card.
	c3, err := conformantCard().CanonicalPayload()
	require.NoError(t, err)
	assert.Equal(t, string(c1), string(c3))

	// JCS sorts object keys lexicographically → keys appear in sorted order.
	s := string(c1)
	assert.Less(t, strings.Index(s, `"capabilities"`), strings.Index(s, `"description"`))
	assert.Less(t, strings.Index(s, `"name"`), strings.Index(s, `"version"`))
}

// TestAgentCard_DetachedJWS_RoundTrip signs the §8.4 signing input with a real
// Ed25519 key (as TesserAI would over the verbatim bytes) and verifies it back —
// proving the canonicalization + signing-input + attach + verify pipeline is a
// genuine RFC-7515 detached JWS.
func TestAgentCard_DetachedJWS_RoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	const kid = "content-key-1"
	const jku = "https://tsr.example.com/.well-known/tesserai-content-jwks.json"

	card := conformantCard()
	payload, err := card.CanonicalPayload()
	require.NoError(t, err)
	protectedB64, err := EncodeProtectedHeader(kid, jku)
	require.NoError(t, err)

	// The signer (TesserAI) signs the verbatim signing-input bytes.
	signingInput := JWSSigningInput(protectedB64, payload)
	sig := ed25519.Sign(priv, signingInput)
	card.AttachDetachedSignature(protectedB64, sig)

	// Protected header decodes to the expected JOSE header.
	hdrJSON, err := base64.RawURLEncoding.DecodeString(card.Signatures[0].Protected)
	require.NoError(t, err)
	var hdr map[string]any
	require.NoError(t, json.Unmarshal(hdrJSON, &hdr))
	assert.Equal(t, JWSAlgEdDSA, hdr["alg"])
	assert.Equal(t, JWSTypJOSE, hdr["typ"])
	assert.Equal(t, kid, hdr["kid"])
	assert.Equal(t, jku, hdr["jku"])

	// Verify: the injected verifier resolves the key by kid and runs ed25519.Verify.
	ok, err := card.VerifySignatures(func(h map[string]any, in, s []byte) bool {
		assert.Equal(t, kid, h["kid"])
		return ed25519.Verify(pub, in, s)
	})
	require.NoError(t, err)
	assert.True(t, ok, "a correctly signed card must verify")

	// Tamper with the card body → verification must fail (payload changed).
	tampered := *card
	tampered.Description = "TAMPERED"
	ok, err = tampered.VerifySignatures(func(_ map[string]any, in, s []byte) bool {
		return ed25519.Verify(pub, in, s)
	})
	require.NoError(t, err)
	assert.False(t, ok, "a tampered card must not verify")

	// Unsigned card → (false, nil).
	ok, err = conformantCard().VerifySignatures(func(_ map[string]any, _, _ []byte) bool { return true })
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAgentCard_VerifySignatures_StructuralDefects(t *testing.T) {
	card := conformantCard()
	card.Signatures = []AgentCardSignature{{Protected: "!!!not-base64!!!", Signature: "x"}}
	_, err := card.VerifySignatures(func(_ map[string]any, _, _ []byte) bool { return true })
	assert.Error(t, err)
}

func violationFields(vs []Violation) map[string]bool {
	fields := map[string]bool{}
	for _, v := range vs {
		fields[v.Field] = true
	}
	return fields
}
