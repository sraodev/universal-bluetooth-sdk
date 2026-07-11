// Package mcp serves ubtctl's tool registry over the Model Context Protocol
// (https://modelcontextprotocol.io/). The wire format is JSON-RPC 2.0 over
// stdio with newline-delimited JSON, matching how MCP-aware editors and
// agents (Claude Desktop, Cursor, Zed, etc.) launch their servers.
//
// Every tool a client sees here is the same tools.Spec the in-process AI
// planner (`ubtctl ask`) uses — there is one source of truth and one
// execution path.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
)

// ProtocolVersion is the MCP revision this server speaks. Bump when we
// adopt newer revisions; clients negotiate via initialize.
const ProtocolVersion = "2025-03-26"

// Server is a single MCP session over a paired Reader/Writer. handleLine
// runs sequentially from Serve, so writes don't need synchronisation.
type Server struct {
	registry *tools.Registry
	log      *slog.Logger
	name     string
	version  string
}

func New(reg *tools.Registry, name, version string, log *slog.Logger) *Server {
	return &Server{registry: reg, name: name, version: version, log: log}
}

// Serve reads newline-delimited JSON-RPC messages from in and writes
// responses to out until the input is closed or ctx is cancelled.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleLine(ctx, out, line)
	}
	return scanner.Err()
}

// ---------------------------------------------------------------------------
// JSON-RPC envelope
// ---------------------------------------------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // absent → notification
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternal       = -32603
)

func (s *Server) handleLine(ctx context.Context, out io.Writer, line []byte) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.write(out, &rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: codeParseError, Message: err.Error()},
		})
		return
	}
	if req.JSONRPC != "2.0" {
		s.replyErr(out, req.ID, codeInvalidRequest, "jsonrpc must be \"2.0\"")
		return
	}

	// Notifications (no id) get no response, but we still dispatch them so
	// the client's `notifications/initialized` is properly observed.
	switch req.Method {
	case "initialize":
		s.handleInitialize(out, &req)
	case "notifications/initialized":
		// fire-and-forget — nothing to do beyond logging
		s.log.Debug("client initialized")
	case "ping":
		s.reply(out, req.ID, map[string]any{})
	case "tools/list":
		s.handleToolsList(out, &req)
	case "tools/call":
		s.handleToolsCall(ctx, out, &req)
	default:
		if len(req.ID) == 0 {
			return // unknown notification, silent drop
		}
		s.replyErr(out, req.ID, codeMethodNotFound, "method "+req.Method)
	}
}

// ---------------------------------------------------------------------------
// MCP method handlers
// ---------------------------------------------------------------------------

func (s *Server) handleInitialize(out io.Writer, req *rpcRequest) {
	s.reply(out, req.ID, map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	})
}

func (s *Server) handleToolsList(out io.Writer, req *rpcRequest) {
	specs := s.registry.All()
	list := make([]map[string]any, 0, len(specs))
	for _, sp := range specs {
		var schema any
		if err := json.Unmarshal(sp.Schema, &schema); err != nil {
			schema = map[string]any{"type": "object"}
		}
		list = append(list, map[string]any{
			"name":        sp.Name,
			"description": sp.Description,
			"inputSchema": schema,
		})
	}
	s.reply(out, req.ID, map[string]any{"tools": list})
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func (s *Server) handleToolsCall(ctx context.Context, out io.Writer, req *rpcRequest) {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.replyErr(out, req.ID, codeInvalidParams, err.Error())
		return
	}
	spec, ok := s.registry.Get(p.Name)
	if !ok {
		s.replyErr(out, req.ID, codeMethodNotFound, fmt.Sprintf("tool %q not registered", p.Name))
		return
	}
	res, err := spec.Handler(ctx, p.Arguments)
	if err != nil {
		// Underlying transport failure → JSON-RPC error.
		s.replyErr(out, req.ID, codeInternal, err.Error())
		return
	}
	s.reply(out, req.ID, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": res.Text},
		},
		"isError": res.IsError,
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (s *Server) reply(out io.Writer, id json.RawMessage, result any) {
	if len(id) == 0 {
		return // notification — no response
	}
	s.write(out, &rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) replyErr(out io.Writer, id json.RawMessage, code int, message string) {
	if len(id) == 0 {
		s.log.Warn("error from notification dropped", "code", code, "message", message)
		return
	}
	s.write(out, &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}

func (s *Server) write(out io.Writer, env *rpcResponse) {
	b, err := json.Marshal(env)
	if err != nil {
		s.log.Error("marshal response", "err", err)
		return
	}
	if _, err := out.Write(append(b, '\n')); err != nil {
		s.log.Error("write response", "err", err)
	}
}
