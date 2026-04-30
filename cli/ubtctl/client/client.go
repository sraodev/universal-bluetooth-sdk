// Package client wraps a duplex Unix socket connection to ubtd and exposes
// the request/response semantics defined in common/protocol/framing.md.
package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

type Client struct {
	conn net.Conn
	mu   sync.Mutex // serialises writes; reads happen on the caller goroutine
}

func Dial(socketPath string) (*Client, error) {
	c, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", socketPath, err)
	}
	return &Client{conn: c}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

// Call sends a single request and waits for the matching response,
// dropping events that arrive on the same id (for streaming, use Stream).
func (c *Client) Call(ctx context.Context, method string, params any) (map[string]any, error) {
	id, err := c.send(method, params)
	if err != nil {
		return nil, err
	}
	for {
		env, err := c.readNext(ctx)
		if err != nil {
			return nil, err
		}
		if env.ID != id {
			continue
		}
		if env.Kind == protocol.KindEvent {
			continue
		}
		if env.Error != nil {
			return nil, env.Error
		}
		return env.Result, nil
	}
}

// Stream sends a request and invokes onEvent for every event sharing the
// originating id, returning when the terminating response arrives.
func (c *Client) Stream(ctx context.Context, method string, params any, onEvent func(map[string]any)) error {
	id, err := c.send(method, params)
	if err != nil {
		return err
	}
	for {
		env, err := c.readNext(ctx)
		if err != nil {
			return err
		}
		if env.ID != id {
			continue
		}
		switch env.Kind {
		case protocol.KindEvent:
			onEvent(env.Params)
		case protocol.KindResponse:
			if env.Error != nil {
				return env.Error
			}
			return nil
		}
	}
}

func (c *Client) send(method string, params any) (string, error) {
	id, err := newID()
	if err != nil {
		return "", err
	}
	env := &protocol.Envelope{
		ID:     id,
		Kind:   protocol.KindRequest,
		Method: method,
		Params: paramsMap(params),
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := protocol.WriteFrame(c.conn, env); err != nil {
		return "", err
	}
	return id, nil
}

func (c *Client) readNext(ctx context.Context) (*protocol.Envelope, error) {
	if dl, ok := ctx.Deadline(); ok {
		_ = c.conn.SetReadDeadline(dl)
	} else {
		_ = c.conn.SetReadDeadline(time.Time{})
	}
	env, err := protocol.ReadFrame(c.conn)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return env, nil
}

func paramsMap(p any) map[string]any {
	if p == nil {
		return nil
	}
	if m, ok := p.(map[string]any); ok {
		return m
	}
	// Marshal-then-unmarshal; small structs only.
	return marshalToMap(p)
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
