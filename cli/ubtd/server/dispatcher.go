// Package server hosts the UDS request dispatcher used by ubtd.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport"
)

type Server struct {
	log      *slog.Logger
	registry *transport.Registry
	commit   string
	version  string

	sessions atomic.Int64
}

func New(log *slog.Logger, registry *transport.Registry, daemonVersion, commit string) *Server {
	return &Server{
		log:      log,
		registry: registry,
		commit:   commit,
		version:  daemonVersion,
	}
}

// Serve accepts connections from ln until ctx is cancelled.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	var wg sync.WaitGroup
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				wg.Wait()
				return nil
			}
			return err
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			defer c.Close()
			s.handleConn(ctx, c)
		}(conn)
	}
}

func (s *Server) handleConn(ctx context.Context, c net.Conn) {
	s.sessions.Add(1)
	defer s.sessions.Add(-1)

	for {
		req, err := protocol.ReadFrame(c)
		if err != nil {
			return
		}
		if req.Kind != protocol.KindRequest {
			s.writeError(c, req.ID, protocol.CodeInvalidParams, "expected kind=request")
			continue
		}
		s.dispatch(ctx, c, req)
	}
}

func (s *Server) dispatch(ctx context.Context, c net.Conn, req *protocol.Envelope) {
	switch req.Method {
	case protocol.MethodPing:
		s.writeResult(c, req.ID, protocol.PingResult{
			Pong:             "pong",
			ServerTimeUnixMs: time.Now().UnixMilli(),
		})
	case protocol.MethodVersion:
		s.writeResult(c, req.ID, protocol.VersionResult{
			DaemonVersion:   s.version,
			ProtocolVersion: protocol.Version,
			Commit:          s.commit,
		})
	case protocol.MethodCapabilities:
		s.writeResult(c, req.ID, protocol.CapabilitiesResult{Capabilities: s.registry.Capabilities()})
	case protocol.MethodStatus:
		s.writeResult(c, req.ID, protocol.StatusResult{
			State:          "ready",
			ActiveSessions: int(s.sessions.Load()),
			Drivers:        s.registry.Names(),
		})
	case protocol.MethodDiscover:
		s.handleDiscover(ctx, c, req)
	case protocol.MethodSend:
		s.handleSend(ctx, c, req)
	default:
		s.writeError(c, req.ID, protocol.CodeUnknownMethod, fmt.Sprintf("method %q", req.Method))
	}
}

func (s *Server) handleDiscover(ctx context.Context, c net.Conn, req *protocol.Envelope) {
	var p protocol.DiscoverParams
	if err := decodeParams(req.Params, &p); err != nil {
		s.writeError(c, req.ID, protocol.CodeInvalidParams, err.Error())
		return
	}
	if p.Transport == "" {
		// Default to the first driver we have.
		caps := s.registry.Capabilities()
		if len(caps) == 0 {
			s.writeError(c, req.ID, protocol.CodeNotImplemented, "no drivers registered")
			return
		}
		p.Transport = caps[0].Transport
	}
	driver, ok := s.registry.ForTransport(p.Transport)
	if !ok {
		s.writeError(c, req.ID, protocol.CodeNotImplemented, "no driver for transport "+p.Transport)
		return
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 8
	}
	scanCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutSeconds)*time.Second)
	defer cancel()

	out := make(chan protocol.Device, 8)
	errCh := make(chan error, 1)
	go func() { errCh <- driver.Discover(scanCtx, p, out); close(out) }()

	for dev := range out {
		s.writeEvent(c, req.ID, protocol.MethodDiscover, dev)
	}
	if err := <-errCh; err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		s.writeDriverError(c, req.ID, err)
		return
	}
	s.writeResult(c, req.ID, map[string]any{})
}

func (s *Server) handleSend(ctx context.Context, c net.Conn, req *protocol.Envelope) {
	var p protocol.SendParams
	if err := decodeParams(req.Params, &p); err != nil {
		s.writeError(c, req.ID, protocol.CodeInvalidParams, err.Error())
		return
	}
	if p.Transport == "" {
		s.writeError(c, req.ID, protocol.CodeInvalidParams, "transport required")
		return
	}
	driver, ok := s.registry.ForTransport(p.Transport)
	if !ok {
		s.writeError(c, req.ID, protocol.CodeNotImplemented, "no driver for transport "+p.Transport)
		return
	}
	res, err := driver.Send(ctx, p)
	if err != nil {
		s.writeDriverError(c, req.ID, err)
		return
	}
	s.writeResult(c, req.ID, res)
}

// writeDriverError preserves the structured error code from a driver if it
// returned a *protocol.Error; otherwise falls back to CodeTransportError.
func (s *Server) writeDriverError(c net.Conn, id string, err error) {
	var pe *protocol.Error
	if errors.As(err, &pe) {
		s.writeError(c, id, pe.Code, pe.Message)
		return
	}
	s.writeError(c, id, protocol.CodeTransportError, err.Error())
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func decodeParams(raw map[string]any, dst any) error {
	if raw == nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

func toMap(v any) map[string]any {
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"_marshal_error": err.Error()}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"_unmarshal_error": err.Error()}
	}
	return m
}

func (s *Server) writeResult(c net.Conn, id string, result any) {
	if err := protocol.WriteFrame(c, &protocol.Envelope{
		ID:     id,
		Kind:   protocol.KindResponse,
		Result: toMap(result),
	}); err != nil {
		s.log.Warn("write result", "err", err)
	}
}

func (s *Server) writeEvent(c net.Conn, id, method string, payload any) {
	if err := protocol.WriteFrame(c, &protocol.Envelope{
		ID:     id,
		Kind:   protocol.KindEvent,
		Method: method,
		Params: toMap(payload),
	}); err != nil {
		s.log.Warn("write event", "err", err)
	}
}

func (s *Server) writeError(c net.Conn, id, code, message string) {
	if err := protocol.WriteFrame(c, &protocol.Envelope{
		ID:    id,
		Kind:  protocol.KindResponse,
		Error: &protocol.Error{Code: code, Message: message},
	}); err != nil {
		s.log.Warn("write error", "err", err)
	}
}
