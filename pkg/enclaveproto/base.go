package enclaveproto

import "encoding/json"

type Request struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Response[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}
