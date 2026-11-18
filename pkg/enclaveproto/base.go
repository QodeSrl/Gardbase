package enclaveproto

import "encoding/json"

type Request struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Response[T any] struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Error   string `json:"error"`
}
