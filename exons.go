// Package exons provides a declarative agent specification format for Go.
//
// Exons define what agents do — the functional coding regions of an agent's DNA.
// An .exons file describes a complete agent specification: identity, execution
// parameters, tools, memory, dispatch rules, verification cases, and more.
//
// The template syntax uses content-resistant {~...~} delimiters that work with
// any prompt content including code, XML, and JSON.
//
// # Quick Start
//
// Create an engine and execute a template:
//
//	engine := exons.MustNew()
//	result, err := engine.Execute(ctx, `Hello {~exons.var name="user" /~}!`, map[string]any{
//	    "user": "World",
//	})
//	// result: "Hello World!"
//
// # Parsing with YAML Frontmatter
//
// Templates can include YAML frontmatter that is parsed into a Spec:
//
//	source := `---
//	name: greeting
//	type: prompt
//	execution:
//	  provider: openai
//	  model: gpt-4o
//	---
//	{~exons.message role="user"~}
//	Hello {~exons.var name="name" /~}
//	{~/exons.message~}`
//
//	tmpl, err := engine.Parse(source)
//	spec := tmpl.Spec() // Access parsed frontmatter
//
// # Custom Resolvers
//
// Register custom tag handlers:
//
//	engine.Register(exons.NewResolverFunc("MyTag",
//	    func(ctx context.Context, execCtx *exons.Context, attrs exons.Attributes) (string, error) {
//	        name, _ := attrs.Get("name")
//	        return "Hello " + name, nil
//	    },
//	    nil,
//	))
//
// # Message Extraction
//
// Extract structured messages for LLM API calls:
//
//	messages, err := tmpl.ExecuteAndExtractMessages(ctx, data)
//	for _, msg := range messages {
//	    fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
//	}
//
// For more information, visit https://exons.ai
package exons
