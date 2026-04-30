// Package tools is the neutral tool registry shared by every front-end
// that drives ubtd through Claude-style "tool calls" — the AI planner
// inside `ubtctl ask` today, the MCP server inside `ubtctl mcp`, and
// future agent integrations.
//
// One Spec → many presentations. The Spec carries a name, a description,
// a JSON Schema (auto-derived from a Go struct via jsonschema tags), and
// a typed handler. Adapters in sibling packages translate the Spec into
// whatever envelope the caller expects (anthropic.BetaTool, MCP
// `tools/list` response, etc.).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Result is a tool's output, normalised across front-ends.
//
// Text is always set (it's what models render). IsError flags failures
// the model should treat as a returned-but-failed tool call rather than
// a transport error — e.g. "device not found" should set IsError=true
// and explain in Text, while a daemon disconnect should be a Go error
// returned from Handler.
type Result struct {
	Text    string
	IsError bool
}

// Handler is the runtime form of a tool: take a context, take the raw
// JSON Claude/MCP gave us, return a Result.
type Handler func(ctx context.Context, input json.RawMessage) (Result, error)

// Spec is the neutral description of a tool.
type Spec struct {
	Name        string
	Description string
	Schema      json.RawMessage // JSON Schema doc for the input
	Handler     Handler
	// Mutating marks tools whose side-effects reach the radio (or any
	// peer / external system). Replay tools (`ubtctl plan run`) gate
	// these behind explicit confirmation; the AI dry-run mode stubs them.
	Mutating bool
}

// New builds a Spec from a typed handler. The schema is derived from T
// via reflection — annotate fields with `jsonschema:"required,description=..."`
// the same way the Anthropic SDK's tool runner expects.
func New[T any](name, description string, h func(context.Context, T) (Result, error)) (Spec, error) {
	return NewMutating(name, description, false, h)
}

// NewMutating is like New but tags the resulting Spec as mutating. Use
// this for any tool that has visible side-effects (sends a packet, opens
// a connection, writes to a file). Replay tooling honours the flag.
func NewMutating[T any](name, description string, mutating bool, h func(context.Context, T) (Result, error)) (Spec, error) {
	schema, err := buildSchema[T]()
	if err != nil {
		return Spec{}, fmt.Errorf("build schema for %s: %w", name, err)
	}
	return Spec{
		Name:        name,
		Description: description,
		Schema:      schema,
		Mutating:    mutating,
		Handler: func(ctx context.Context, raw json.RawMessage) (Result, error) {
			var v T
			if len(raw) > 0 && string(raw) != "null" {
				if err := json.Unmarshal(raw, &v); err != nil {
					return Result{
						Text:    fmt.Sprintf("invalid arguments for %s: %v", name, err),
						IsError: true,
					}, nil
				}
			}
			return h(ctx, v)
		},
	}, nil
}

// MustNew panics on schema-generation failure. Use only in package-level
// initialisers where a panic crashes the process at start, not later.
func MustNew[T any](name, description string, h func(context.Context, T) (Result, error)) Spec {
	s, err := New(name, description, h)
	if err != nil {
		panic(err)
	}
	return s
}

func buildSchema[T any]() (json.RawMessage, error) {
	var zero T
	r := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
	}
	s := r.Reflect(zero)
	return json.Marshal(s)
}

// Registry is an ordered set of Specs with name-based lookup.
type Registry struct {
	specs []Spec
	index map[string]int
}

func NewRegistry(specs ...Spec) *Registry {
	r := &Registry{index: make(map[string]int, len(specs))}
	for _, s := range specs {
		r.Add(s)
	}
	return r
}

func (r *Registry) Add(s Spec) {
	if _, dup := r.index[s.Name]; dup {
		return
	}
	r.index[s.Name] = len(r.specs)
	r.specs = append(r.specs, s)
}

func (r *Registry) All() []Spec { return r.specs }

func (r *Registry) Get(name string) (Spec, bool) {
	i, ok := r.index[name]
	if !ok {
		return Spec{}, false
	}
	return r.specs[i], true
}
