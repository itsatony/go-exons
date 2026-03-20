package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- MemorySpec ---

func TestMemorySpec_HasMemory(t *testing.T) {
	assert.False(t, (*MemorySpec)(nil).HasMemory())
	assert.False(t, (&MemorySpec{}).HasMemory())
	assert.True(t, (&MemorySpec{Scope: "test"}).HasMemory())
}

func TestMemorySpec_GetAutoRecall(t *testing.T) {
	val, ok := (*MemorySpec)(nil).GetAutoRecall()
	assert.False(t, ok)
	assert.False(t, val)

	val, ok = (&MemorySpec{}).GetAutoRecall()
	assert.False(t, ok)
	assert.False(t, val)

	tr := true
	val, ok = (&MemorySpec{AutoRecall: &tr}).GetAutoRecall()
	assert.True(t, ok)
	assert.True(t, val)

	fl := false
	val, ok = (&MemorySpec{AutoRecall: &fl}).GetAutoRecall()
	assert.True(t, ok)
	assert.False(t, val)
}

func TestMemorySpec_GetAutoRecord(t *testing.T) {
	val, ok := (*MemorySpec)(nil).GetAutoRecord()
	assert.False(t, ok)
	assert.False(t, val)

	tr := true
	val, ok = (&MemorySpec{AutoRecord: &tr}).GetAutoRecord()
	assert.True(t, ok)
	assert.True(t, val)
}

func TestMemorySpec_Clone(t *testing.T) {
	assert.Nil(t, (*MemorySpec)(nil).Clone())

	tr := true
	fl := false
	orig := &MemorySpec{
		Scope:      "test-scope",
		AutoRecall: &tr,
		AutoRecord: &fl,
		ReadScopes: []string{"global", "shared"},
	}

	clone := orig.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, orig.Scope, clone.Scope)

	// Verify deep copy — mutate clone, original unchanged
	*clone.AutoRecall = false
	assert.True(t, *orig.AutoRecall)

	clone.ReadScopes[0] = "mutated"
	assert.Equal(t, "global", orig.ReadScopes[0])
}

func TestMemorySpec_Clone_NilFields(t *testing.T) {
	orig := &MemorySpec{Scope: "minimal"}
	clone := orig.Clone()
	assert.Equal(t, "minimal", clone.Scope)
	assert.Nil(t, clone.AutoRecall)
	assert.Nil(t, clone.AutoRecord)
	assert.Nil(t, clone.ReadScopes)
}

// --- DispatchSpec ---

func TestDispatchSpec_HasTriggers(t *testing.T) {
	assert.False(t, (*DispatchSpec)(nil).HasTriggers())
	assert.False(t, (&DispatchSpec{}).HasTriggers())
	assert.True(t, (&DispatchSpec{TriggerKeywords: []string{"dns"}}).HasTriggers())
	assert.True(t, (&DispatchSpec{TriggerDescription: "handle DNS"}).HasTriggers())
}

func TestDispatchSpec_GetCostLimitUSD(t *testing.T) {
	val, ok := (*DispatchSpec)(nil).GetCostLimitUSD()
	assert.False(t, ok)
	assert.Equal(t, 0.0, val)

	val, ok = (&DispatchSpec{}).GetCostLimitUSD()
	assert.False(t, ok)

	cost := 0.50
	val, ok = (&DispatchSpec{CostLimitUSD: &cost}).GetCostLimitUSD()
	assert.True(t, ok)
	assert.Equal(t, 0.50, val)

	zero := 0.0
	val, ok = (&DispatchSpec{CostLimitUSD: &zero}).GetCostLimitUSD()
	assert.True(t, ok)
	assert.Equal(t, 0.0, val)
}

func TestDispatchSpec_Clone(t *testing.T) {
	assert.Nil(t, (*DispatchSpec)(nil).Clone())

	cost := 0.50
	orig := &DispatchSpec{
		TriggerKeywords:    []string{"dns", "domain"},
		TriggerDescription: "DNS tasks",
		CostLimitUSD:       &cost,
	}

	clone := orig.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, orig.TriggerDescription, clone.TriggerDescription)
	assert.Equal(t, 0.50, *clone.CostLimitUSD)

	// Verify deep copy
	clone.TriggerKeywords[0] = "mutated"
	assert.Equal(t, "dns", orig.TriggerKeywords[0])

	*clone.CostLimitUSD = 9.99
	assert.Equal(t, 0.50, *orig.CostLimitUSD)
}

func TestDispatchSpec_Clone_NilFields(t *testing.T) {
	orig := &DispatchSpec{TriggerDescription: "minimal"}
	clone := orig.Clone()
	assert.Equal(t, "minimal", clone.TriggerDescription)
	assert.Nil(t, clone.TriggerKeywords)
	assert.Nil(t, clone.CostLimitUSD)
}

// --- VerificationCase ---

func TestVerificationCase_Clone(t *testing.T) {
	orig := VerificationCase{
		Name:           "test-case",
		Description:    "A test",
		Tags:           []string{"smoke", "dns"},
		Input:          map[string]any{"zone_id": "z1"},
		Prompt:         "List records",
		Expect:         &VerificationExpect{ToolCalls: []string{"dns_list"}},
		Ref:            "",
		TimeoutSeconds: 30,
	}

	clone := orig.Clone()
	assert.Equal(t, orig.Name, clone.Name)
	assert.Equal(t, orig.TimeoutSeconds, clone.TimeoutSeconds)

	// Verify deep copy
	clone.Tags[0] = "mutated"
	assert.Equal(t, "smoke", orig.Tags[0])

	clone.Input["zone_id"] = "mutated"
	assert.Equal(t, "z1", orig.Input["zone_id"])

	clone.Expect.ToolCalls[0] = "mutated"
	assert.Equal(t, "dns_list", orig.Expect.ToolCalls[0])
}

func TestVerificationCase_Clone_NilFields(t *testing.T) {
	orig := VerificationCase{Name: "minimal"}
	clone := orig.Clone()
	assert.Equal(t, "minimal", clone.Name)
	assert.Nil(t, clone.Tags)
	assert.Nil(t, clone.Input)
	assert.Nil(t, clone.Expect)
}

// --- VerificationExpect ---

func TestVerificationExpect_Clone(t *testing.T) {
	assert.Nil(t, (*VerificationExpect)(nil).Clone())

	orig := &VerificationExpect{
		ToolCalls:          []string{"search", "read"},
		ToolCallsAbsent:    []string{"delete"},
		OutputContains:     "found",
		OutputNotContains:  "error",
		OutputMatchesRegex: `^\d+`,
	}

	clone := orig.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, orig.OutputContains, clone.OutputContains)
	assert.Equal(t, orig.OutputMatchesRegex, clone.OutputMatchesRegex)

	// Verify deep copy
	clone.ToolCalls[0] = "mutated"
	assert.Equal(t, "search", orig.ToolCalls[0])

	clone.ToolCallsAbsent[0] = "mutated"
	assert.Equal(t, "delete", orig.ToolCallsAbsent[0])
}

func TestVerificationExpect_Clone_NilSlices(t *testing.T) {
	orig := &VerificationExpect{OutputContains: "hello"}
	clone := orig.Clone()
	assert.Equal(t, "hello", clone.OutputContains)
	assert.Nil(t, clone.ToolCalls)
	assert.Nil(t, clone.ToolCallsAbsent)
}

// --- RegistrySpec ---

func TestRegistrySpec_Clone(t *testing.T) {
	assert.Nil(t, (*RegistrySpec)(nil).Clone())

	orig := &RegistrySpec{
		Namespace: "dns-manager",
		Origin:    OriginInternal,
		Version:   "1.2.0",
	}

	clone := orig.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, orig.Namespace, clone.Namespace)
	assert.Equal(t, orig.Origin, clone.Origin)
	assert.Equal(t, orig.Version, clone.Version)
}

func TestRegistrySpec_Validate(t *testing.T) {
	assert.NoError(t, (*RegistrySpec)(nil).Validate())
	assert.NoError(t, (&RegistrySpec{}).Validate())
	assert.NoError(t, (&RegistrySpec{Origin: OriginInternal}).Validate())
	assert.NoError(t, (&RegistrySpec{Origin: OriginExternal}).Validate())
	assert.NoError(t, (&RegistrySpec{Origin: OriginUnknown}).Validate())

	err := (&RegistrySpec{Origin: "bogus"}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRegistryOrigin)
}

// --- SafetyConfig ---

func TestSafetyConfig_Clone(t *testing.T) {
	assert.Nil(t, (*SafetyConfig)(nil).Clone())

	orig := &SafetyConfig{
		Guardrails:             GuardrailsEnabled,
		RequireConfirmationFor: []string{"delete_record"},
		DenyTools:              []string{"write_file"},
	}

	clone := orig.Clone()
	require.NotNil(t, clone)
	assert.Equal(t, GuardrailsEnabled, clone.Guardrails)

	// Verify deep copy
	clone.RequireConfirmationFor[0] = "mutated"
	assert.Equal(t, "delete_record", orig.RequireConfirmationFor[0])

	clone.DenyTools[0] = "mutated"
	assert.Equal(t, "write_file", orig.DenyTools[0])
}

func TestSafetyConfig_Clone_NilSlices(t *testing.T) {
	orig := &SafetyConfig{Guardrails: GuardrailsDisabled}
	clone := orig.Clone()
	assert.Equal(t, GuardrailsDisabled, clone.Guardrails)
	assert.Nil(t, clone.RequireConfirmationFor)
	assert.Nil(t, clone.DenyTools)
}

func TestSafetyConfig_Validate(t *testing.T) {
	assert.NoError(t, (*SafetyConfig)(nil).Validate())
	assert.NoError(t, (&SafetyConfig{}).Validate())
	assert.NoError(t, (&SafetyConfig{Guardrails: GuardrailsEnabled}).Validate())
	assert.NoError(t, (&SafetyConfig{Guardrails: GuardrailsDisabled}).Validate())

	err := (&SafetyConfig{Guardrails: "maybe"}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSafetyGuardrails)
}

// --- Spec.Validate integration for metadata ---

func TestSpec_Validate_SafetyGuardrailsInvalid(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Type:        DocumentTypeAgent,
		Safety:      &SafetyConfig{Guardrails: "invalid"},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSafetyGuardrails)
}

func TestSpec_Validate_RegistryOriginInvalid(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Type:        DocumentTypeAgent,
		Registry:    &RegistrySpec{Origin: "invalid"},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgRegistryOrigin)
}

func TestSpec_Validate_MetadataValid(t *testing.T) {
	s := &Spec{
		Name:        "test-agent",
		Description: "test",
		Type:        DocumentTypeAgent,
		Memory:      &MemorySpec{Scope: "test-scope"},
		Dispatch:    &DispatchSpec{TriggerKeywords: []string{"dns"}},
		Registry:    &RegistrySpec{Origin: OriginInternal, Version: "1.0"},
		Safety:      &SafetyConfig{Guardrails: GuardrailsEnabled},
		Verifications: []VerificationCase{
			{Name: "test-case", Expect: &VerificationExpect{OutputContains: "ok"}},
		},
	}
	assert.NoError(t, s.Validate())
}

// --- MemorySpec.Validate ---

func TestMemorySpec_Validate(t *testing.T) {
	assert.NoError(t, (*MemorySpec)(nil).Validate())
	assert.NoError(t, (&MemorySpec{}).Validate())
	assert.NoError(t, (&MemorySpec{Scope: "valid-scope"}).Validate())

	err := (&MemorySpec{Scope: "INVALID SCOPE"}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMemoryInvalidScope)
}

func TestMemorySpec_Validate_ReadScopes(t *testing.T) {
	assert.NoError(t, (&MemorySpec{ReadScopes: []string{"global", "shared"}}).Validate())

	err := (&MemorySpec{ReadScopes: []string{"good", "BAD SCOPE"}}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMemoryReadScopeInvalid)
}

// --- DispatchSpec.Validate ---

func TestDispatchSpec_Validate(t *testing.T) {
	assert.NoError(t, (*DispatchSpec)(nil).Validate())
	assert.NoError(t, (&DispatchSpec{}).Validate())

	validCost := 50.0
	assert.NoError(t, (&DispatchSpec{CostLimitUSD: &validCost}).Validate())

	zeroCost := 0.0
	assert.NoError(t, (&DispatchSpec{CostLimitUSD: &zeroCost}).Validate())
}

func TestDispatchSpec_Validate_CostLimit(t *testing.T) {
	negative := -1.0
	err := (&DispatchSpec{CostLimitUSD: &negative}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgDispatchCostLimit)

	tooHigh := 1001.0
	err = (&DispatchSpec{CostLimitUSD: &tooHigh}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgDispatchCostLimit)
}

func TestDispatchSpec_Validate_EmptyKeyword(t *testing.T) {
	err := (&DispatchSpec{TriggerKeywords: []string{"dns", ""}}).Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgDispatchKeywordEmpty)
}

// --- VerificationCase.Validate ---

func TestVerificationCase_Validate_Valid(t *testing.T) {
	vc := VerificationCase{
		Name:   "test-case",
		Expect: &VerificationExpect{OutputContains: "ok"},
	}
	assert.NoError(t, vc.Validate())
}

func TestVerificationCase_Validate_ValidWithRef(t *testing.T) {
	vc := VerificationCase{
		Name: "ref-case",
		Ref:  "external-test",
	}
	assert.NoError(t, vc.Validate())
}

func TestVerificationCase_Validate_NameRequired(t *testing.T) {
	vc := VerificationCase{Expect: &VerificationExpect{OutputContains: "ok"}}
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyNameRequired)
}

func TestVerificationCase_Validate_NameSlug(t *testing.T) {
	vc := VerificationCase{
		Name:   "INVALID NAME",
		Expect: &VerificationExpect{OutputContains: "ok"},
	}
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyNameInvalid)
}

func TestVerificationCase_Validate_RefAndExpectMutuallyExclusive(t *testing.T) {
	vc := VerificationCase{
		Name:   "both",
		Ref:    "external",
		Expect: &VerificationExpect{OutputContains: "ok"},
	}
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyRefAndExpect)
}

func TestVerificationCase_Validate_NoAssertions(t *testing.T) {
	// Neither Ref nor Expect
	vc := VerificationCase{Name: "empty"}
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyNoAssertions)

	// Expect with no assertions
	vc2 := VerificationCase{Name: "empty-expect", Expect: &VerificationExpect{}}
	err = vc2.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyNoAssertions)
}

func TestVerificationCase_Validate_Timeout(t *testing.T) {
	vc := VerificationCase{
		Name:           "timeout-case",
		Expect:         &VerificationExpect{OutputContains: "ok"},
		TimeoutSeconds: 30,
	}
	assert.NoError(t, vc.Validate())

	vc.TimeoutSeconds = 0 // zero means not set
	assert.NoError(t, vc.Validate())

	vc.TimeoutSeconds = 601
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyTimeout)
}

func TestVerificationCase_Validate_InvalidRegex(t *testing.T) {
	vc := VerificationCase{
		Name:   "regex-case",
		Expect: &VerificationExpect{OutputMatchesRegex: "[invalid"},
	}
	err := vc.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgVerifyRegexInvalid)
}

func TestVerificationCase_Validate_ValidRegex(t *testing.T) {
	vc := VerificationCase{
		Name:   "regex-case",
		Expect: &VerificationExpect{OutputMatchesRegex: `^\d+$`},
	}
	assert.NoError(t, vc.Validate())
}

// --- Spec.Validate type-gating for metadata ---

func TestSpec_Validate_PromptNoMemory(t *testing.T) {
	s := &Spec{
		Name: "test-prompt", Description: "test", Type: DocumentTypePrompt,
		Memory: &MemorySpec{Scope: "test"},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoMemory)
}

func TestSpec_Validate_PromptNoDispatch(t *testing.T) {
	s := &Spec{
		Name: "test-prompt", Description: "test", Type: DocumentTypePrompt,
		Dispatch: &DispatchSpec{TriggerKeywords: []string{"dns"}},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoDispatch)
}

func TestSpec_Validate_PromptNoRegistry(t *testing.T) {
	s := &Spec{
		Name: "test-prompt", Description: "test", Type: DocumentTypePrompt,
		Registry: &RegistrySpec{Namespace: "test"},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgPromptNoRegistry)
}

func TestSpec_Validate_SkillNoDispatch(t *testing.T) {
	s := &Spec{
		Name: "test-skill", Description: "test", Type: DocumentTypeSkill,
		Dispatch: &DispatchSpec{TriggerKeywords: []string{"dns"}},
	}
	err := s.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSkillNoDispatch)
}

func TestSpec_Validate_SkillAllowsMemoryAndRegistry(t *testing.T) {
	s := &Spec{
		Name: "test-skill", Description: "test", Type: DocumentTypeSkill,
		Memory:   &MemorySpec{Scope: "test"},
		Registry: &RegistrySpec{Origin: OriginInternal},
	}
	assert.NoError(t, s.Validate())
}

func TestSpec_Validate_PromptAllowsVerifications(t *testing.T) {
	s := &Spec{
		Name: "test-prompt", Description: "test", Type: DocumentTypePrompt,
		Verifications: []VerificationCase{
			{Name: "t1", Expect: &VerificationExpect{OutputContains: "ok"}},
		},
	}
	assert.NoError(t, s.Validate())
}
