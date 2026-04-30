package protocol

import (
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	in := &Envelope{
		ID:     "abc",
		Kind:   KindRequest,
		Method: MethodDiscover,
		Params: map[string]any{"transport": "rfcomm", "timeout_seconds": float64(5)},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, in); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.ID != in.ID || out.Kind != in.Kind || out.Method != in.Method {
		t.Fatalf("envelope mismatch: %+v vs %+v", out, in)
	}
	if got, want := out.Params["transport"], "rfcomm"; got != want {
		t.Fatalf("params: got %v want %v", got, want)
	}
}

func TestErrorEnvelope(t *testing.T) {
	in := &Envelope{
		ID:    "x",
		Kind:  KindResponse,
		Error: &Error{Code: CodeUnknownMethod, Message: "no such method"},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, in); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out.Error == nil || out.Error.Code != CodeUnknownMethod {
		t.Fatalf("error not preserved: %+v", out.Error)
	}
}
