package client

import "encoding/json"

// Decode rehydrates a result map (as returned by Client.Call) into a typed
// destination. Lets callers stay typed without sprinkling json marshal/unmarshal
// throughout command handlers.
func Decode(src map[string]any, dst any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// toMap normalises an arbitrary value into the wire-friendly map[string]any
// shape every Envelope param/result field expects. nil and an empty map
// both produce a nil result, suppressing the field in JSON output.
func toMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
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
