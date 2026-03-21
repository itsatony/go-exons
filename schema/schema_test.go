package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// schemaPath returns the absolute path to exons.schema.json relative to this test file.
func schemaPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "exons.schema.json")
}

// loadSchema reads and parses the schema file, returning the top-level map.
func loadSchema(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	return schema
}

func TestSchemaIsValidJSON(t *testing.T) {
	schema := loadSchema(t)
	if schema == nil {
		t.Fatal("schema decoded to nil")
	}
}

func TestSchemaHasDraft202012(t *testing.T) {
	schema := loadSchema(t)

	dialect, ok := schema["$schema"].(string)
	if !ok {
		t.Fatal("$schema field is missing or not a string")
	}
	expected := "https://json-schema.org/draft/2020-12/schema"
	if dialect != expected {
		t.Errorf("$schema = %q, want %q", dialect, expected)
	}
}

func TestSchemaTopLevelStructure(t *testing.T) {
	schema := loadSchema(t)

	// Must be type: object
	if typ, _ := schema["type"].(string); typ != "object" {
		t.Errorf("top-level type = %q, want %q", typ, "object")
	}

	// Must have title
	if _, ok := schema["title"].(string); !ok {
		t.Error("missing title field")
	}

	// Must have required: [name, type]
	reqRaw, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("required field is missing or not an array")
	}
	requiredSet := make(map[string]bool, len(reqRaw))
	for _, v := range reqRaw {
		if s, ok := v.(string); ok {
			requiredSet[s] = true
		}
	}
	for _, field := range []string{"name", "type"} {
		if !requiredSet[field] {
			t.Errorf("required field %q is missing from required array", field)
		}
	}
}

func TestSchemaHasProperties(t *testing.T) {
	schema := loadSchema(t)

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("properties field is missing or not an object")
	}

	expectedProps := []string{
		"name", "description", "type", "execution",
		"inputs", "outputs", "sample",
		"skills", "tools", "constraints",
		"messages", "context", "credentials", "credential",
		"memory", "dispatch", "verifications", "registry", "safety",
	}
	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("missing property %q in schema", prop)
		}
	}
}

func TestSchemaHasDefs(t *testing.T) {
	schema := loadSchema(t)

	defs, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs field is missing or not an object")
	}

	expectedDefs := []string{
		"DocumentType", "InputDef", "OutputDef",
		"SkillRef", "ToolsConfig", "FunctionDef", "MCPServer",
		"ConstraintsConfig", "OperationalConstraints",
		"MessageTemplate", "CredentialRef",
		"MemorySpec", "DispatchSpec", "VerificationCase", "VerificationExpect",
		"RegistrySpec", "SafetyConfig",
		"ExecutionConfig", "ThinkingConfig", "ResponseFormat",
		"JSONSchemaSpec", "EnumConstraint", "GuidedDecoding",
		"ImageConfig", "AudioConfig", "EmbeddingConfig",
		"StreamingConfig", "AsyncConfig",
	}
	for _, def := range expectedDefs {
		if _, exists := defs[def]; !exists {
			t.Errorf("missing $defs/%s", def)
		}
	}
}

func TestSchemaDocumentTypeEnum(t *testing.T) {
	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	dt := defs["DocumentType"].(map[string]any)

	enumRaw, ok := dt["enum"].([]any)
	if !ok {
		t.Fatal("DocumentType enum missing")
	}

	expected := map[string]bool{"prompt": true, "skill": true, "agent": true}
	for _, v := range enumRaw {
		s, _ := v.(string)
		if !expected[s] {
			t.Errorf("unexpected DocumentType enum value: %q", s)
		}
		delete(expected, s)
	}
	for missing := range expected {
		t.Errorf("missing DocumentType enum value: %q", missing)
	}
}

func TestSchemaProviderEnum(t *testing.T) {
	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	ec := defs["ExecutionConfig"].(map[string]any)
	props := ec["properties"].(map[string]any)
	provider := props["provider"].(map[string]any)

	enumRaw, ok := provider["enum"].([]any)
	if !ok {
		t.Fatal("ExecutionConfig.provider enum missing")
	}

	expected := map[string]bool{
		"openai": true, "anthropic": true, "google": true,
		"gemini": true, "vertex": true, "vllm": true,
		"azure": true, "mistral": true, "cohere": true,
	}
	for _, v := range enumRaw {
		s, _ := v.(string)
		if !expected[s] {
			t.Errorf("unexpected provider enum value: %q", s)
		}
		delete(expected, s)
	}
	for missing := range expected {
		t.Errorf("missing provider enum value: %q", missing)
	}
}

func TestSchemaNamePattern(t *testing.T) {
	schema := loadSchema(t)
	props := schema["properties"].(map[string]any)
	name := props["name"].(map[string]any)

	pattern, ok := name["pattern"].(string)
	if !ok {
		t.Fatal("name.pattern is missing")
	}
	if pattern != "^[a-z][a-z0-9-]*$" {
		t.Errorf("name.pattern = %q, want %q", pattern, "^[a-z][a-z0-9-]*$")
	}

	maxLen, ok := name["maxLength"].(float64)
	if !ok {
		t.Fatal("name.maxLength is missing")
	}
	if int(maxLen) != 64 {
		t.Errorf("name.maxLength = %v, want 64", maxLen)
	}
}

func TestSchemaGuardrailsEnum(t *testing.T) {
	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	sc := defs["SafetyConfig"].(map[string]any)
	props := sc["properties"].(map[string]any)
	guardrails := props["guardrails"].(map[string]any)

	enumRaw, ok := guardrails["enum"].([]any)
	if !ok {
		t.Fatal("SafetyConfig.guardrails enum missing")
	}

	expected := map[string]bool{"enabled": true, "disabled": true}
	for _, v := range enumRaw {
		s, _ := v.(string)
		if !expected[s] {
			t.Errorf("unexpected guardrails enum value: %q", s)
		}
		delete(expected, s)
	}
	for missing := range expected {
		t.Errorf("missing guardrails enum value: %q", missing)
	}
}

func TestSchemaOriginEnum(t *testing.T) {
	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)
	rs := defs["RegistrySpec"].(map[string]any)
	props := rs["properties"].(map[string]any)
	origin := props["origin"].(map[string]any)

	enumRaw, ok := origin["enum"].([]any)
	if !ok {
		t.Fatal("RegistrySpec.origin enum missing")
	}

	expected := map[string]bool{"internal": true, "external": true, "unknown": true}
	for _, v := range enumRaw {
		s, _ := v.(string)
		if !expected[s] {
			t.Errorf("unexpected origin enum value: %q", s)
		}
		delete(expected, s)
	}
	for missing := range expected {
		t.Errorf("missing origin enum value: %q", missing)
	}
}

func TestSchemaDefsHaveAdditionalPropertiesFalse(t *testing.T) {
	schema := loadSchema(t)
	defs := schema["$defs"].(map[string]any)

	// These $defs should have additionalProperties: false for strictness
	strictDefs := []string{
		"InputDef", "OutputDef", "SkillRef", "ToolsConfig", "FunctionDef",
		"MCPServer", "ConstraintsConfig", "OperationalConstraints",
		"MessageTemplate", "CredentialRef",
		"MemorySpec", "DispatchSpec", "VerificationCase", "VerificationExpect",
		"RegistrySpec", "SafetyConfig",
		"ExecutionConfig", "ThinkingConfig", "ResponseFormat",
		"JSONSchemaSpec", "EnumConstraint", "GuidedDecoding",
		"ImageConfig", "AudioConfig", "EmbeddingConfig",
		"StreamingConfig", "AsyncConfig",
	}

	for _, name := range strictDefs {
		def, ok := defs[name].(map[string]any)
		if !ok {
			t.Errorf("$defs/%s is missing or not an object", name)
			continue
		}
		ap, exists := def["additionalProperties"]
		if !exists {
			t.Errorf("$defs/%s missing additionalProperties", name)
			continue
		}
		if ap != false {
			t.Errorf("$defs/%s additionalProperties = %v, want false", name, ap)
		}
	}
}
