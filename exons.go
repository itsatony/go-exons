// Package exons provides a declarative agent specification format for Go.
//
// Exons define what agents do — the functional coding regions of an agent's DNA.
// An .exons file describes a complete agent specification: identity, execution
// parameters, tools, memory, dispatch rules, verification cases, and more.
//
// The template syntax uses content-resistant {~...~} delimiters that work with
// any prompt content including code, XML, and JSON.
//
// File format:
//
//	---
//	name: my-agent
//	type: agent
//	execution:
//	  provider: anthropic
//	  model: claude-sonnet-4-6
//	---
//	{~exons.message role="system"~}
//	You are a helpful assistant.
//	{~/exons.message~}
//
// For more information, visit https://exons.ai
package exons
