package execution

import "testing"

// TestSerializedProvidersAreExactlyProviderFormatsArms keeps SerializedProviders honest in both
// directions: every listed provider has a ProviderFormat arm, and a provider outside the list has
// none — so the schema examples derived from it (go-exons#15) name what the code serializes.
func TestSerializedProvidersAreExactlyProviderFormatsArms(t *testing.T) {
	cfg := &Config{}
	for _, p := range SerializedProviders() {
		if _, err := cfg.ProviderFormat(p); err != nil {
			t.Errorf("SerializedProviders lists %q but ProviderFormat refuses it: %v", p, err)
		}
	}
	for _, p := range []string{"bedrock", "ollama", "xai", ""} {
		if _, err := cfg.ProviderFormat(p); err == nil {
			t.Errorf("ProviderFormat serializes %q, so SerializedProviders must list it", p)
		}
	}
}
