package client

import "encoding/json"

func marshalToMap(v any) map[string]any {
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
