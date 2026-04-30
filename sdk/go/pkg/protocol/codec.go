package protocol

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxFrameBytes = MaxFrameKB * 1024

// WriteFrame serialises env as a length-prefixed JSON frame.
func WriteFrame(w io.Writer, env *Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	if len(body) > maxFrameBytes {
		return &Error{Code: CodeFrameTooLarge, Message: fmt.Sprintf("frame %d > %d bytes", len(body), maxFrameBytes)}
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(body)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// ReadFrame reads one length-prefixed JSON frame.
func ReadFrame(r io.Reader) (*Envelope, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 {
		return nil, errors.New("empty frame")
	}
	if n > maxFrameBytes {
		return nil, &Error{Code: CodeFrameTooLarge, Message: fmt.Sprintf("frame %d > %d bytes", n, maxFrameBytes)}
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	return &env, nil
}
