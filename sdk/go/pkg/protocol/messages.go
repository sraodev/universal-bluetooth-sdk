// Package protocol defines the wire types shared by ubtd and ubt.
//
// The schema is mirrored in common/protocol/v1.proto. Keep them in sync.
package protocol

const (
	Version    = "1.0.0"
	MaxFrameKB = 16 * 1024
)

type Kind string

const (
	KindRequest  Kind = "request"
	KindResponse Kind = "response"
	KindEvent    Kind = "event"
)

const (
	MethodPing         = "Ping"
	MethodVersion      = "Version"
	MethodCapabilities = "Capabilities"
	MethodDiscover     = "Discover"
	MethodSend         = "Send"
	MethodStatus       = "Status"
)

const (
	CodeUnknownMethod  = "unknown_method"
	CodeNotImplemented = "not_implemented"
	CodeInvalidParams  = "invalid_params"
	CodeTransportError = "transport_error"
	CodeNotFound       = "not_found"
	CodeFrameTooLarge  = "frame_too_large"
	CodeInternal       = "internal"
)

type Envelope struct {
	ID     string         `json:"id"`
	Kind   Kind           `json:"kind"`
	Method string         `json:"method,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	Result map[string]any `json:"result,omitempty"`
	Error  *Error         `json:"error,omitempty"`
}

type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

type Device struct {
	Address    string            `json:"address"`
	Name       string            `json:"name"`
	Transport  string            `json:"transport"`
	RSSI       int               `json:"rssi"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type Capability struct {
	Transport string `json:"transport"`
	Discover  bool   `json:"discover"`
	Pair      bool   `json:"pair"`
	Stream    bool   `json:"stream"`
	MaxMTU    int    `json:"max_mtu"`
}

type PingResult struct {
	Pong             string `json:"pong"`
	ServerTimeUnixMs int64  `json:"server_time_unix_ms"`
}

type VersionResult struct {
	DaemonVersion   string `json:"daemon_version"`
	ProtocolVersion string `json:"protocol_version"`
}

type CapabilitiesResult struct {
	Capabilities []Capability `json:"capabilities"`
}

type DiscoverParams struct {
	Transport      string `json:"transport,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type SendParams struct {
	Address   string `json:"address"`
	Transport string `json:"transport"`
	Payload   []byte `json:"payload"`
	UUIDPort  int    `json:"uuid_port,omitempty"`
}

type SendResult struct {
	BytesSent     int64 `json:"bytes_sent"`
	LatencyMicros int64 `json:"latency_micros"`
}

type StatusResult struct {
	State          string   `json:"state"`
	ActiveSessions int      `json:"active_sessions"`
	Drivers        []string `json:"drivers"`
}
