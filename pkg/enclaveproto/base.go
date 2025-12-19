package enclaveproto

import "encoding/json"

type Request struct {
	Type    string          `json:"type,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response[T any] struct {
	Success bool   `json:"success,omitempty"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}
