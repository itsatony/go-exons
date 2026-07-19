package a2a

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/gowebpki/jcs"
)

// A2A §8.4 Agent Card signing. Cards are signed as RFC-7515 JSON Web Signatures
// with a **detached** payload: the JWS payload is the RFC-8785 (JCS) canonical form
// of the card with `signatures` removed, and the signature covers the JWS Signing
// Input `BASE64URL(protectedHeader) || '.' || BASE64URL(payload)`.
//
// This package is crypto-agnostic: it produces the canonical payload + signing input
// and assembles/decomposes the AgentCardSignature, but the actual signing/verifying
// key operations are injected by the caller (aigentverse routes them through the
// TesserAI content-signing seam — go-exons never holds a key).

// JWS protected-header parameters + values for A2A card signatures (§8.4.2).
const (
	// JWSAlgEdDSA is the JOSE algorithm identifier for Ed25519 (RFC 8037), the
	// algorithm TesserAI's content-signing key hierarchy uses.
	JWSAlgEdDSA = "EdDSA"
	// JWSTypJOSE is the recommended `typ` value for a card JWS.
	JWSTypJOSE = "JOSE"

	jwsHeaderAlg = "alg"
	jwsHeaderTyp = "typ"
	jwsHeaderKID = "kid"
	jwsHeaderJKU = "jku"
	jwsSeparator = "."
)

// Package-local errors (a2a/ cannot import the root exons package).
var (
	errCardNilForPayload = errors.New("a2a: cannot canonicalize a nil agent card")
	errCardNilForVerify  = errors.New("a2a: cannot verify signatures on a nil agent card")
	errSigBadProtected   = errors.New("a2a: signature has a malformed base64url protected header")
	errSigBadHeaderJSON  = errors.New("a2a: signature protected header is not valid JSON")
	errSigBadSignature   = errors.New("a2a: signature value is not valid base64url")
)

// CanonicalPayload returns the RFC-8785 (JCS) canonical JSON of the card with the
// `signatures` field excluded — the exact bytes a §8.4 JWS signs, and the exact
// bytes an independent verifier reconstructs. The struct's omitempty tags mean no
// explicit default values are emitted, so a verifier that strips defaults reproduces
// this canonical form byte for byte.
func (c *AgentCard) CanonicalPayload() ([]byte, error) {
	if c == nil {
		return nil, errCardNilForPayload
	}
	unsigned := *c
	unsigned.Signatures = nil
	raw, err := json.Marshal(&unsigned)
	if err != nil {
		return nil, err
	}
	return jcs.Transform(raw)
}

// EncodeProtectedHeader builds an EdDSA JWS Protected Header for a card signature
// and returns its base64url encoding (the value stored in AgentCardSignature.Protected).
// kid identifies the signing key; jku, when non-empty, is the JWKS URL a verifier
// fetches the key from. json.Marshal of the header map emits keys in a stable
// (sorted) order, so the encoding is deterministic.
func EncodeProtectedHeader(kid, jku string) (string, error) {
	hdr := map[string]any{
		jwsHeaderAlg: JWSAlgEdDSA,
		jwsHeaderTyp: JWSTypJOSE,
	}
	if kid != "" {
		hdr[jwsHeaderKID] = kid
	}
	if jku != "" {
		hdr[jwsHeaderJKU] = jku
	}
	b, err := json.Marshal(hdr)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// JWSSigningInput assembles the RFC-7515 §5.1 signing input from a base64url-encoded
// protected header and the raw (un-encoded) canonical payload:
// `ASCII(BASE64URL(header) || '.' || BASE64URL(payload))`.
func JWSSigningInput(protectedB64 string, payload []byte) []byte {
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	out := make([]byte, 0, len(protectedB64)+1+len(payloadB64))
	out = append(out, protectedB64...)
	out = append(out, jwsSeparator...)
	out = append(out, payloadB64...)
	return out
}

// AttachDetachedSignature appends a signature built from a base64url protected
// header and the raw signature bytes (which the caller obtained by signing the
// JWSSigningInput). Multiple signatures may be attached.
func (c *AgentCard) AttachDetachedSignature(protectedB64 string, sig []byte) {
	if c == nil {
		return
	}
	c.Signatures = append(c.Signatures, AgentCardSignature{
		Protected: protectedB64,
		Signature: base64.RawURLEncoding.EncodeToString(sig),
	})
}

// VerifySignatures verifies every signature on the card and reports whether the card
// is signed and all signatures are valid. `verify` receives the decoded protected
// header, the reconstructed JWS signing input, and the raw signature bytes; it
// returns true when the signature is valid under the key the header identifies (the
// caller resolves `kid`/`jku` and runs the Ed25519 check). Returns (false, nil) for
// an unsigned card and (false, nil) when any signature fails to verify; a structural
// defect (bad base64/JSON) is a non-nil error.
func (c *AgentCard) VerifySignatures(verify func(header map[string]any, signingInput, sig []byte) bool) (bool, error) {
	if c == nil {
		return false, errCardNilForVerify
	}
	if len(c.Signatures) == 0 {
		return false, nil
	}
	payload, err := c.CanonicalPayload()
	if err != nil {
		return false, err
	}
	for _, s := range c.Signatures {
		hdrJSON, err := base64.RawURLEncoding.DecodeString(s.Protected)
		if err != nil {
			return false, errSigBadProtected
		}
		var hdr map[string]any
		if err := json.Unmarshal(hdrJSON, &hdr); err != nil {
			return false, errSigBadHeaderJSON
		}
		sig, err := base64.RawURLEncoding.DecodeString(s.Signature)
		if err != nil {
			return false, errSigBadSignature
		}
		if !verify(hdr, JWSSigningInput(s.Protected, payload), sig) {
			return false, nil
		}
	}
	return true, nil
}
