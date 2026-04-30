// Package tools is the neutral tool registry shared by every front-end
// that drives ubtd through Claude-style tool calls — the AI planner
// inside `ubtctl ask` today, the MCP server inside `ubtctl mcp`, and
// future agent integrations.
//
// One Spec → many presentations. Adapters in sibling packages translate
// the Spec into whatever envelope the caller expects (anthropic.BetaTool,
// MCP `tools/list` response, etc.).
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Result is a tool's output, normalised across front-ends.
type Result struct {
	Text    string
	IsError bool
}

// Handler is the runtime form of a tool.
type Handler func(ctx context.Context, input json.RawMessage) (Result, error)

// Spec is the neutral description of a tool. Mutating tools have side
// effects that reach the radio (or any peer / external system); replay
// gates them behind explicit confirmation.
type Spec struct {
	Name        string
	Description string
	Schema      json.RawMessage
	Handler     Handler
	Mutating    bool
}

// New builds a Spec from a typed handler. Schema is derived from T via
// reflection — annotate fields with `jsonschema:"required,description=..."`
// the same way the Anthropic SDK's tool runner expects.
//
// Panics on schema-build failure. That can only happen for unsupported
// Go types in T, which is a programmer error — fail loudly at startup
// rather than thread the error through every caller.
func New[T any](name, description string, h func(context.Context, T) (Result, error)) Spec {
	var zero T
	r := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
	}
	schema, err := json.Marshal(r.Reflect(zero))
	if err != nil {
		panic(fmt.Sprintf("tools.New(%s): %v", name, err))
	}
	return Spec{
		Name:        name,
		Description: description,
		Schema:      schema,
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
	}
}

// Registry is an ordered set of Specs with name-based lookup. Order is
// preserved so MCP tools/list and the AI tool list are deterministic.
type Registry struct {
	specs []Spec
	names map[string]int
}

func NewRegistry() *Registry {
	return &Registry{names: map[string]int{}}
}

func (r *Registry) Add(s Spec) {
	if _, dup := r.names[s.Name]; dup {
		return
	}
	r.names[s.Name] = len(r.specs)
	r.specs = append(r.specs, s)
}

func (r *Registry) All() []Spec { return r.specs }

func (r *Registry) Get(name string) (Spec, bool) {
	i, ok := r.names[name]
	if !ok {
		return Spec{}, false
	}
	return r.specs[i], true
}
