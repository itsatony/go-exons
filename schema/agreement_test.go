package schema_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	exons "github.com/itsatony/go-exons"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// agreementSchemaURL is the resource name the compiled schema is registered under.
// It is a label for the compiler, never fetched.
const agreementSchemaURL = "https://github.com/itsatony/go-exons/schema/exons.schema.json"

// Divergence reasons. A corpus row whose two verdicts differ must name one of these,
// so the set of ways the instruments may disagree is CLOSED and reviewed here rather
// than grown one row at a time.
const (
	// parserOnlyUniqueness: JSON Schema's uniqueItems compares whole items, so "ref
	// is unique within the list" is expressible only in Go.
	parserOnlyUniqueness = "parser-only: uniqueness of a key within a requirements list"
	// parserOnlyInputOrder: input_order must name declared inputs — a cross-field rule.
	parserOnlyInputOrder = "parser-only: input_order names only declared inputs"
	// schemaStricterClosed: every $def is additionalProperties:false while yaml.v3
	// ignores an unknown nested key.
	schemaStricterClosed = "schema-stricter: nested objects are closed, the parser ignores unknown keys"
	// schemaStricterType: the schema requires type; the parser defaults it to skill.
	schemaStricterType = "schema-stricter: type is required, the parser defaults it to skill"
)

// agreementCase is one document and the verdict EACH instrument must return for it.
type agreementCase struct {
	name        string
	frontmatter string
	wantSchema  bool // true = the schema accepts
	wantParser  bool // true = exons.Parse accepts
	// divergence names why the two verdicts differ; empty iff they agree.
	divergence string
}

// agreementHead is the minimal valid identity every row builds on unless it is
// testing identity itself.
const agreementHead = "name: doc\ndescription: a document\n"

func agree(name, fm string, accept bool) agreementCase {
	return agreementCase{name: name, frontmatter: fm, wantSchema: accept, wantParser: accept}
}

func diverge(name, fm string, schemaAccepts, parserAccepts bool, reason string) agreementCase {
	return agreementCase{name: name, frontmatter: fm, wantSchema: schemaAccepts, wantParser: parserAccepts, divergence: reason}
}

func agent(extra string) string  { return agreementHead + "type: agent\n" + extra }
func prompt(extra string) string { return agreementHead + "type: prompt\n" + extra }
func skill(extra string) string  { return agreementHead + "type: skill\n" + extra }

func agreementCorpus() []agreementCase {
	umlauts := func(n int) string { return strings.Repeat("ä", n) }
	return []agreementCase{
		// --- identity: aigentverse#72's headline — the schema accepted what Parse refused.
		agree("minimal agent", agent(""), true),
		agree("description missing", "name: doc\ntype: agent\n", false),
		agree("description empty", "name: doc\ndescription: \"\"\ntype: agent\n", false),
		// 1024 umlauts are 2048 bytes: under the old byte cap the parser refused a
		// description the schema's character maxLength accepted.
		agree("description 1024 non-ASCII characters", "name: doc\ndescription: "+umlauts(1024)+"\ntype: agent\n", true),
		agree("description 1025 non-ASCII characters", "name: doc\ndescription: "+umlauts(1025)+"\ntype: agent\n", false),
		diverge("type missing", agreementHead, false, true, schemaStricterType),

		// --- requirements: undeclared in the schema before v0.31.0, so ANY value passed it.
		agree("requirements with every field", agent(`requirements:
  mcp:
    - capability: dns-management
      credential_ref: cloudflare-api
      scope: org
  credentials:
    - ref: slack-bot
      provider: slack
      scope: user
  resources:
    - ref: product-docs
      kind: corpus
      access: write
      scope: per_call
      purpose: answers are grounded in the product documentation
`), true),
		agree("requirements empty mapping", agent("requirements: {}\n"), true),
		agree("mcp capability missing", agent("requirements:\n  mcp:\n    - scope: org\n"), false),
		agree("mcp scope out of vocabulary", agent("requirements:\n  mcp:\n    - capability: x\n      scope: global\n"), false),
		agree("credential ref empty", agent("requirements:\n  credentials:\n    - ref: \"\"\n"), false),
		agree("credential scope out of vocabulary", agent("requirements:\n  credentials:\n    - ref: r\n      scope: team\n"), false),
		agree("mcp capability 512 non-ASCII characters", agent("requirements:\n  mcp:\n    - capability: "+umlauts(512)+"\n"), true),
		agree("mcp capability 513 non-ASCII characters", agent("requirements:\n  mcp:\n    - capability: "+umlauts(513)+"\n"), false),
		diverge("mcp capability duplicated", agent("requirements:\n  mcp:\n    - capability: a\n    - capability: a\n"), true, false, parserOnlyUniqueness),
		diverge("credential entry with an unknown key", agent("requirements:\n  credentials:\n    - ref: r\n      credential_ref: r\n"), false, true, schemaStricterClosed),
		diverge("requirements unknown list", agent("requirements:\n  datasets:\n    - ref: r\n"), false, true, schemaStricterClosed),

		// --- requirements.resources (aigentverse#80).
		agree("resource minimal", agent("requirements:\n  resources:\n    - ref: product-docs\n      kind: corpus\n"), true),
		agree("resource kind with every token character", agent("requirements:\n  resources:\n    - ref: r\n      kind: vai.corpus_v2-beta\n"), true),
		agree("resource access read", agent("requirements:\n  resources:\n    - ref: r\n      kind: folder\n      access: read\n"), true),
		agree("resource ref missing", agent("requirements:\n  resources:\n    - kind: corpus\n"), false),
		agree("resource kind missing", agent("requirements:\n  resources:\n    - ref: r\n"), false),
		agree("resource kind uppercase", agent("requirements:\n  resources:\n    - ref: r\n      kind: Corpus\n"), false),
		agree("resource kind leading digit", agent("requirements:\n  resources:\n    - ref: r\n      kind: 2corpus\n"), false),
		agree("resource ref is a coordinate", agent("requirements:\n  resources:\n    - ref: s3://tenant-42/docs\n      kind: corpus\n"), false),
		agree("resource kind is a coordinate", agent("requirements:\n  resources:\n    - ref: r\n      kind: s3://corpus\n"), false),
		agree("resource access out of vocabulary", agent("requirements:\n  resources:\n    - ref: r\n      kind: corpus\n      access: admin\n"), false),
		agree("resource scope out of vocabulary", agent("requirements:\n  resources:\n    - ref: r\n      kind: corpus\n      scope: team\n"), false),
		agree("resource ref 512 non-ASCII characters", agent("requirements:\n  resources:\n    - ref: "+umlauts(512)+"\n      kind: corpus\n"), true),
		agree("resource ref 513 non-ASCII characters", agent("requirements:\n  resources:\n    - ref: "+umlauts(513)+"\n      kind: corpus\n"), false),
		agree("resource purpose 513 characters", agent("requirements:\n  resources:\n    - ref: r\n      kind: corpus\n      purpose: "+strings.Repeat("p", 513)+"\n"), false),
		diverge("resource ref duplicated", agent("requirements:\n  resources:\n    - ref: r\n      kind: corpus\n    - ref: r\n      kind: folder\n"), true, false, parserOnlyUniqueness),
		diverge("resource entry with an unknown key", agent("requirements:\n  resources:\n    - ref: r\n      kind: corpus\n      uri: x\n"), false, true, schemaStricterClosed),

		// --- the type-specific prohibitions, each with the AGENT control that proves
		// the prohibition is about the type and not about the block.
		agree("agent carries every prohibited-elsewhere block", agent(`skills:
  - slug: helper
tools:
  functions:
    - name: f
      description: a function
constraints:
  behavioral: [be careful]
memory:
  scope: mem
dispatch:
  trigger_keywords: [dns]
registry:
  namespace: reg
`), true),
		agree("prompt with skills", prompt("skills:\n  - slug: helper\n"), false),
		agree("prompt with an empty skills list", prompt("skills: []\n"), true),
		agree("prompt with tool functions", prompt("tools:\n  functions:\n    - name: f\n      description: a function\n"), false),
		agree("prompt with an empty tool functions list", prompt("tools:\n  functions: []\n"), true),
		agree("prompt with constraints", prompt("constraints:\n  behavioral: [be careful]\n"), false),
		agree("prompt with memory", prompt("memory:\n  scope: mem\n"), false),
		agree("prompt with dispatch", prompt("dispatch:\n  trigger_keywords: [dns]\n"), false),
		agree("prompt with registry", prompt("registry:\n  namespace: reg\n"), false),
		agree("skill with skills", skill("skills:\n  - slug: helper\n"), false),
		agree("skill with dispatch", skill("dispatch:\n  trigger_keywords: [dns]\n"), false),
		agree("skill with memory and registry", skill("memory:\n  scope: mem\nregistry:\n  namespace: reg\n"), true),

		// --- a parser-only cross-field rule, so the divergence vocabulary is exercised.
		diverge("input_order names an undeclared input", agent("inputs:\n  a:\n    type: string\ninput_order: [b]\n"), true, false, parserOnlyInputOrder),
	}
}

// compileAgreementSchema compiles the schema at path.
func compileAgreementSchema(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource(agreementSchemaURL, doc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	sch, err := c.Compile(agreementSchemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

// schemaAccepts validates frontmatter YAML against the compiled schema, going
// through JSON exactly as an editor or CI validator does.
func schemaAccepts(t *testing.T, sch *jsonschema.Schema, frontmatter string) (bool, error) {
	t.Helper()
	var value any
	if err := yaml.Unmarshal([]byte(frontmatter), &value); err != nil {
		t.Fatalf("corpus frontmatter is not YAML: %v", err)
	}
	asJSON, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("corpus frontmatter is not JSON-representable: %v", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(asJSON))
	if err != nil {
		t.Fatalf("re-decode instance: %v", err)
	}
	verr := sch.Validate(inst)
	return verr == nil, verr
}

// TestSchemaAndParserAgree is the durable fix for aigentverse#72: the published
// schema accepted documents exons.Parse refused (a missing description, a
// 1024-character non-ASCII description, and any requirements block at all), so an
// editor or a CI job showed green on a document every consumer would reject.
//
// Every corpus row states the verdict of BOTH instruments. Where they differ the
// row must name a reason from the closed divergence vocabulary above, and the
// only direction the #72 defect lives in — schema accepts, parser refuses — is
// admitted only for the parser-only rules JSON Schema cannot express.
//
// ⚠ It is the SCHEMA column that makes this a guard: removing `description` from
// the root's required list, or dropping the requirements $def, turns rows whose
// expected schema verdict is "refuse" into accepts, and the test goes red.
func TestSchemaAndParserAgree(t *testing.T) {
	sch := compileAgreementSchema(t, schemaPath(t))

	corpus := agreementCorpus()
	agreeing := 0
	for _, tc := range corpus {
		t.Run(tc.name, func(t *testing.T) {
			if (tc.wantSchema == tc.wantParser) != (tc.divergence == "") {
				t.Fatalf("row is inconsistent: a divergence reason is required exactly when the two verdicts differ")
			}
			if tc.wantSchema && !tc.wantParser && !strings.HasPrefix(tc.divergence, "parser-only:") {
				t.Fatalf("schema-accepts/parser-refuses is aigentverse#72's direction and is admitted only for a parser-only rule; got %q", tc.divergence)
			}

			gotSchema, verr := schemaAccepts(t, sch, tc.frontmatter)
			if gotSchema != tc.wantSchema {
				t.Errorf("schema accepted=%v, want %v (%v)", gotSchema, tc.wantSchema, verr)
			}

			doc := "---\n" + tc.frontmatter + "---\nbody\n"
			_, perr := exons.Parse([]byte(doc))
			if gotParser := perr == nil; gotParser != tc.wantParser {
				t.Errorf("exons.Parse accepted=%v, want %v (%v)", gotParser, tc.wantParser, perr)
			}
		})
		if tc.divergence == "" {
			agreeing++
		}
	}
	// Anti-vacuity: the corpus is mostly rows where the instruments must agree.
	if agreeing*2 < len(corpus) {
		t.Errorf("only %d of %d rows are agreement rows; the corpus has drifted into describing divergence", agreeing, len(corpus))
	}
}

// TestShippedReferenceDocumentPassesBothInstruments runs the one worked document
// the README points readers at through both instruments.
func TestShippedReferenceDocumentPassesBothInstruments(t *testing.T) {
	sch := compileAgreementSchema(t, schemaPath(t))
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(schemaPath(t)), "..", "examples", "dns-specialist.exons"))
	if err != nil {
		t.Fatalf("read reference document: %v", err)
	}
	if _, err := exons.Parse(raw); err != nil {
		t.Fatalf("exons.Parse refused the reference document: %v", err)
	}
	parts := strings.SplitN(string(raw), "\n---\n", 2)
	if len(parts) != 2 {
		t.Fatal("reference document has no closing frontmatter delimiter")
	}
	frontmatter := strings.TrimPrefix(parts[0], "---\n") + "\n"
	if ok, verr := schemaAccepts(t, sch, frontmatter); !ok {
		t.Fatalf("schema refused the reference document: %v", verr)
	}
}

// TestSchemaRequirementsBoundsAreTheGoConstants derives the schema's requirements
// bounds from the Go constants rather than restating them, so a change on either
// side fails here.
func TestSchemaRequirementsBoundsAreTheGoConstants(t *testing.T) {
	defs := loadSchema(t)["$defs"].(map[string]any)

	reqs := defs["SpecRequirements"].(map[string]any)["properties"].(map[string]any)
	for _, list := range []string{"mcp", "credentials", "resources"} {
		got, _ := reqs[list].(map[string]any)["maxItems"].(float64)
		if int(got) != exons.MaxRequirementEntries {
			t.Errorf("requirements.%s maxItems = %v, want exons.MaxRequirementEntries (%d)", list, got, exons.MaxRequirementEntries)
		}
	}

	bounded := map[string][]string{
		"MCPRequirement":        {"capability", "credential_ref"},
		"CredentialRequirement": {"ref", "provider"},
		"ResourceRequirement":   {"ref", "kind", "purpose"},
	}
	for def, fields := range bounded {
		props := defs[def].(map[string]any)["properties"].(map[string]any)
		for _, f := range fields {
			got, _ := props[f].(map[string]any)["maxLength"].(float64)
			if int(got) != exons.MaxRequirementFieldLen {
				t.Errorf("%s.%s maxLength = %v, want exons.MaxRequirementFieldLen (%d)", def, f, got, exons.MaxRequirementFieldLen)
			}
		}
	}

	resource := defs["ResourceRequirement"].(map[string]any)["properties"].(map[string]any)
	if got := resource["kind"].(map[string]any)["pattern"]; got != exons.ResourceKindPattern {
		t.Errorf("ResourceRequirement.kind pattern = %v, want exons.ResourceKindPattern %q", got, exons.ResourceKindPattern)
	}
	access := resource["access"].(map[string]any)["enum"].([]any)
	wantAccess := []any{"", exons.ResourceAccessRead, exons.ResourceAccessWrite}
	if len(access) != len(wantAccess) {
		t.Fatalf("ResourceRequirement.access enum = %v, want %v", access, wantAccess)
	}
	for i := range wantAccess {
		if access[i] != wantAccess[i] {
			t.Errorf("ResourceRequirement.access enum = %v, want %v", access, wantAccess)
		}
	}
	scope := defs["RequirementScope"].(map[string]any)["enum"].([]any)
	wantScope := []any{"", exons.RequirementScopeOrg, exons.RequirementScopeUser, exons.RequirementScopePerCall}
	if len(scope) != len(wantScope) {
		t.Fatalf("RequirementScope enum = %v, want %v", scope, wantScope)
	}
	for i := range wantScope {
		if scope[i] != wantScope[i] {
			t.Errorf("RequirementScope enum = %v, want %v", scope, wantScope)
		}
	}
}
