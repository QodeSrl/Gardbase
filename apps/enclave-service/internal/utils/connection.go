package utils

import (
	"encoding/json"
	"log"
)

type Request struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key,omitempty"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func SendResponse(encoder *json.Encoder, res Response) {
	// encode the response as JSON and send it
	if err := encoder.Encode(res); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func SendError(encoder *json.Encoder, errMsg string) {
	response := Response{
		Success: false,
		Error:   errMsg,
	}
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}